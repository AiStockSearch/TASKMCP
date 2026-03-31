package epics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

type EpicRow struct {
	ID          uuid.UUID
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
}

func (r *Repo) Create(ctx context.Context, projectID uuid.UUID, id uuid.UUID, title, desc string) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO epics (id, project_id, title, description, status)
VALUES ($1, $2, $3, NULLIF($4, ''), 'open')
`, id, projectID, title, desc)
	if err != nil {
		return fmt.Errorf("insert epic: %w", err)
	}
	return nil
}

func (r *Repo) List(ctx context.Context, projectID uuid.UUID, status string, limit, offset int) ([]EpicRow, error) {
	var (
		args []any
		where string
	)
	args = append(args, projectID)
	where = fmt.Sprintf("WHERE project_id = $%d", len(args))
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}
	args = append(args, limit)
	limitArg := len(args)
	args = append(args, offset)
	offsetArg := len(args)

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
SELECT id, title, COALESCE(description, ''), status, created_at
FROM epics
%s
ORDER BY created_at DESC, id DESC
LIMIT $%d OFFSET $%d
`, where, limitArg, offsetArg), args...)
	if err != nil {
		return nil, fmt.Errorf("query epics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []EpicRow
	for rows.Next() {
		var er EpicRow
		if err := rows.Scan(&er.ID, &er.Title, &er.Description, &er.Status, &er.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan epic: %w", err)
		}
		out = append(out, er)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate epics: %w", err)
	}
	return out, nil
}

func (r *Repo) LinkRequirement(ctx context.Context, projectID uuid.UUID, requirementID, epicID uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE requirements SET epic_id = $3 WHERE project_id = $1 AND id = $2`, projectID, requirementID, epicID)
	if err != nil {
		return false, fmt.Errorf("update requirement: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return n == 1, nil
}

func (r *Repo) LinkTask(ctx context.Context, projectID uuid.UUID, taskID, epicID uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE tasks SET epic_id = $3 WHERE project_id = $1 AND id = $2`, projectID, taskID, epicID)
	if err != nil {
		return false, fmt.Errorf("update task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return n == 1, nil
}

func (r *Repo) Exists(ctx context.Context, epicID uuid.UUID) (bool, error) {
	var one int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM epics WHERE id = $1`, epicID).Scan(&one)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("query epic exists: %w", err)
	}
	return true, nil
}

