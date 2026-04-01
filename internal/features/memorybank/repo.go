package memorybank

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Repo struct {
	db *sql.DB
}

type DocumentVersionItem struct {
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
	Hash      string `json:"content_hash"`
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func hashContent(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (r *Repo) GetDocument(ctx context.Context, projectID uuid.UUID, docKey string) (Document, bool, error) {
	var (
		docType    string
		titleStr   string
		currentVer int
		updatedAt  time.Time
		docID      uuid.UUID
		content    string
	)

	row := r.db.QueryRowContext(ctx, `
SELECT d.id, d.doc_type, COALESCE(d.title, ''), d.current_version, d.updated_at,
       v.content
FROM mb_documents d
JOIN mb_document_versions v
  ON v.document_id = d.id AND v.version = d.current_version
WHERE d.project_id = $1 AND d.doc_key = $2
`, projectID, docKey)

	if err := row.Scan(&docID, &docType, &titleStr, &currentVer, &updatedAt, &content); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Document{}, false, nil
		}
		return Document{}, false, fmt.Errorf("query mb document: %w", err)
	}
	_ = docID // reserved for future

	return Document{
		DocKey:    docKey,
		DocType:   docType,
		Title:     titleStr,
		Version:   currentVer,
		Content:   content,
		UpdatedAt: updatedAt.UTC().Format(time.RFC3339),
	}, true, nil
}

func (r *Repo) ListDocuments(ctx context.Context, projectID uuid.UUID, docType string, limit, offset int) ([]DocumentListItem, error) {
	var (
		args []any
		where string
	)
	args = append(args, projectID)
	where = fmt.Sprintf("WHERE project_id = $%d", len(args))
	if docType != "" {
		args = append(args, docType)
		where += fmt.Sprintf(" AND doc_type = $%d", len(args))
	}
	args = append(args, limit)
	limitArg := len(args)
	args = append(args, offset)
	offsetArg := len(args)

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
SELECT doc_key, doc_type, COALESCE(title, ''), current_version, updated_at
FROM mb_documents
%s
ORDER BY updated_at DESC, doc_key ASC
LIMIT $%d OFFSET $%d
`, where, limitArg, offsetArg), args...)
	if err != nil {
		return nil, fmt.Errorf("list mb documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []DocumentListItem
	for rows.Next() {
		var it DocumentListItem
		var updated time.Time
		if err := rows.Scan(&it.DocKey, &it.DocType, &it.Title, &it.Version, &updated); err != nil {
			return nil, fmt.Errorf("scan mb document: %w", err)
		}
		it.UpdatedAt = updated.UTC().Format(time.RFC3339)
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mb documents: %w", err)
	}
	return out, nil
}

func (r *Repo) UpsertDocument(ctx context.Context, projectID uuid.UUID, docKey, docType, title, content string) (UpsertDocumentResult, error) {
	h := hashContent(content)

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return UpsertDocumentResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var (
		docID         uuid.UUID
		currentVer    int
	)

	// Lock the document row if it exists.
	err = tx.QueryRowContext(ctx, `
SELECT id, current_version
FROM mb_documents
WHERE project_id = $1 AND doc_key = $2
FOR UPDATE
`, projectID, docKey).Scan(&docID, &currentVer)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return UpsertDocumentResult{}, fmt.Errorf("select mb document: %w", err)
		}
		// Insert new document.
		docID = uuid.New()
		_, err = tx.ExecContext(ctx, `
INSERT INTO mb_documents (id, project_id, doc_type, doc_key, title, current_version)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), 0)
`, docID, projectID, docType, docKey, title)
		if err != nil {
			return UpsertDocumentResult{}, fmt.Errorf("insert mb document: %w", err)
		}
		currentVer = 0
	}

	// If version exists and hash unchanged, do not create new version.
	if currentVer > 0 {
		var existingHash string
		err = tx.QueryRowContext(ctx, `
SELECT content_hash
FROM mb_document_versions
WHERE document_id = $1 AND version = $2
`, docID, currentVer).Scan(&existingHash)
		if err != nil {
			return UpsertDocumentResult{}, fmt.Errorf("select current version hash: %w", err)
		}
		if existingHash == h {
			// Still update title/doc_type (metadata) if needed.
			_, _ = tx.ExecContext(ctx, `
UPDATE mb_documents
SET doc_type = $3, title = NULLIF($4, ''), updated_at = now()
WHERE id = $1 AND project_id = $2
`, docID, projectID, docType, title)

			if err := tx.Commit(); err != nil {
				return UpsertDocumentResult{}, fmt.Errorf("commit: %w", err)
			}
			return UpsertDocumentResult{DocID: docID.String(), Version: currentVer}, nil
		}
	}

	newVer := currentVer + 1
	_, err = tx.ExecContext(ctx, `
INSERT INTO mb_document_versions (id, document_id, version, content, content_hash)
VALUES ($1, $2, $3, $4, $5)
`, uuid.New(), docID, newVer, content, h)
	if err != nil {
		return UpsertDocumentResult{}, fmt.Errorf("insert mb document version: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
UPDATE mb_documents
SET doc_type = $3, title = NULLIF($4, ''), current_version = $5, updated_at = now()
WHERE id = $1 AND project_id = $2
`, docID, projectID, docType, title, newVer)
	if err != nil {
		return UpsertDocumentResult{}, fmt.Errorf("update mb document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return UpsertDocumentResult{}, fmt.Errorf("commit: %w", err)
	}
	return UpsertDocumentResult{DocID: docID.String(), Version: newVer}, nil
}

func (r *Repo) getDocumentID(ctx context.Context, projectID uuid.UUID, docKey string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `
SELECT id
FROM mb_documents
WHERE project_id = $1 AND doc_key = $2
`, projectID, docKey).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.UUID{}, false, nil
		}
		return uuid.UUID{}, false, fmt.Errorf("query document id: %w", err)
	}
	return id, true, nil
}

func (r *Repo) ListVersions(ctx context.Context, projectID uuid.UUID, docKey string, limit, offset int) ([]DocumentVersionItem, bool, error) {
	docID, ok, err := r.getDocumentID(ctx, projectID, docKey)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT version, created_at, content_hash
FROM mb_document_versions
WHERE document_id = $1
ORDER BY version DESC
LIMIT $2 OFFSET $3
`, docID, limit, offset)
	if err != nil {
		return nil, true, fmt.Errorf("list versions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []DocumentVersionItem
	for rows.Next() {
		var it DocumentVersionItem
		var created time.Time
		if err := rows.Scan(&it.Version, &created, &it.Hash); err != nil {
			return nil, true, fmt.Errorf("scan version: %w", err)
		}
		it.CreatedAt = created.UTC().Format(time.RFC3339)
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, true, fmt.Errorf("iterate versions: %w", err)
	}
	return out, true, nil
}

func (r *Repo) GetDocumentVersion(ctx context.Context, projectID uuid.UUID, docKey string, version int) (Document, bool, error) {
	docID, ok, err := r.getDocumentID(ctx, projectID, docKey)
	if err != nil {
		return Document{}, false, err
	}
	if !ok {
		return Document{}, false, nil
	}

	var (
		docType   string
		titleStr  string
		updatedAt time.Time
		content   string
	)
	row := r.db.QueryRowContext(ctx, `
SELECT d.doc_type, COALESCE(d.title, ''), d.updated_at, v.content
FROM mb_documents d
JOIN mb_document_versions v ON v.document_id = d.id
WHERE d.project_id = $1 AND d.doc_key = $2 AND v.version = $3
`, projectID, docKey, version)
	if err := row.Scan(&docType, &titleStr, &updatedAt, &content); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Document{}, false, nil
		}
		return Document{}, false, fmt.Errorf("query document version: %w", err)
	}
	_ = docID
	return Document{
		DocKey:    docKey,
		DocType:   docType,
		Title:     titleStr,
		Version:   version,
		Content:   content,
		UpdatedAt: updatedAt.UTC().Format(time.RFC3339),
	}, true, nil
}

func (r *Repo) GetState(ctx context.Context, projectID uuid.UUID) (string, bool, error) {
	var state string
	err := r.db.QueryRowContext(ctx, `SELECT state_json::text FROM mb_state WHERE project_id = $1`, projectID).Scan(&state)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "{}", false, nil
		}
		return "", false, fmt.Errorf("query mb state: %w", err)
	}
	return state, true, nil
}

func (r *Repo) SetState(ctx context.Context, projectID uuid.UUID, stateJSON string) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO mb_state (project_id, state_json, updated_at)
VALUES ($1, $2::jsonb, now())
ON CONFLICT (project_id) DO UPDATE
SET state_json = EXCLUDED.state_json, updated_at = now()
`, projectID, stateJSON)
	if err != nil {
		return fmt.Errorf("upsert mb state: %w", err)
	}
	return nil
}

func (r *Repo) ListRules(ctx context.Context, projectID *uuid.UUID, scopePrefix string, enabledOnly bool, limit, offset int) ([]Rule, error) {
	var (
		args []any
		where []string
	)
	if projectID != nil {
		args = append(args, *projectID)
		where = append(where, fmt.Sprintf("(project_id = $%d OR project_id IS NULL)", len(args)))
	} else {
		where = append(where, "project_id IS NULL")
	}
	if enabledOnly {
		where = append(where, "enabled = true")
	}
	if scopePrefix != "" {
		args = append(args, scopePrefix+"%")
		where = append(where, fmt.Sprintf("scope LIKE $%d", len(args)))
	}

	args = append(args, limit)
	limitArg := len(args)
	args = append(args, offset)
	offsetArg := len(args)

	query := `
SELECT id, project_id, scope, priority, title, content, enabled, updated_at
FROM mb_rules
`
	if len(where) > 0 {
		query += "WHERE " + joinAnd(where) + "\n"
	}
	query += "ORDER BY priority ASC, project_id NULLS LAST, id ASC\n"
	query += fmt.Sprintf("LIMIT $%d OFFSET $%d", limitArg, offsetArg)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mb rules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Rule
	for rows.Next() {
		var (
			id uuid.UUID
			pid sql.Null[uuid.UUID]
			scope string
			priority int
			title string
			content string
			enabled bool
			updated time.Time
		)
		if err := rows.Scan(&id, &pid, &scope, &priority, &title, &content, &enabled, &updated); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		var pidStr *string
		if pid.Valid {
			s := pid.V.String()
			pidStr = &s
		}
		out = append(out, Rule{
			ID:        id.String(),
			ProjectID: pidStr,
			Scope:     scope,
			Priority:  priority,
			Title:     title,
			Content:   content,
			Enabled:   enabled,
			UpdatedAt: updated.UTC().Format(time.RFC3339),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rules: %w", err)
	}
	return out, nil
}

func (r *Repo) UpsertRule(ctx context.Context, projectID *uuid.UUID, ruleID *uuid.UUID, scope string, priority int, title string, content string, enabled bool) (uuid.UUID, error) {
	id := uuid.New()
	if ruleID != nil {
		id = *ruleID
	}

	_, err := r.db.ExecContext(ctx, `
INSERT INTO mb_rules (id, project_id, scope, priority, title, content, enabled, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (id) DO UPDATE
SET project_id = EXCLUDED.project_id,
    scope = EXCLUDED.scope,
    priority = EXCLUDED.priority,
    title = EXCLUDED.title,
    content = EXCLUDED.content,
    enabled = EXCLUDED.enabled,
    updated_at = now()
`, id, projectID, scope, priority, title, content, enabled)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("upsert rule: %w", err)
	}
	return id, nil
}

func (r *Repo) SetRuleEnabled(ctx context.Context, projectID *uuid.UUID, id uuid.UUID, enabled bool) (bool, error) {
	var res sql.Result
	var err error
	if projectID != nil {
		res, err = r.db.ExecContext(ctx, `UPDATE mb_rules SET enabled = $3, updated_at = now() WHERE id = $1 AND (project_id = $2 OR project_id IS NULL)`, id, *projectID, enabled)
	} else {
		res, err = r.db.ExecContext(ctx, `UPDATE mb_rules SET enabled = $2, updated_at = now() WHERE id = $1 AND project_id IS NULL`, id, enabled)
	}
	if err != nil {
		return false, fmt.Errorf("update rule: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return n == 1, nil
}

func joinAnd(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	s := parts[0]
	for i := 1; i < len(parts); i++ {
		s += " AND " + parts[i]
	}
	return s
}

