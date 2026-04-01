package epics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

func (r *Repo) AddTasksToEpic(ctx context.Context, projectID uuid.UUID, epicID uuid.UUID, taskIDs []uuid.UUID) (updated int64, existing []uuid.UUID, err error) {
	// Fetch existing tasks within project
	rows, err := r.db.QueryContext(ctx, `
SELECT id
FROM tasks
WHERE project_id = $1 AND id = ANY($2)
`, projectID, pq.Array(taskIDs))
	if err != nil {
		return 0, nil, fmt.Errorf("query existing tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return 0, nil, fmt.Errorf("scan existing task id: %w", err)
		}
		existing = append(existing, id)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("iterate existing tasks: %w", err)
	}

	if len(existing) == 0 {
		return 0, nil, nil
	}

	res, err := r.db.ExecContext(ctx, `
UPDATE tasks
SET epic_id = $3
WHERE project_id = $1 AND id = ANY($2)
`, projectID, pq.Array(existing), epicID)
	if err != nil {
		return 0, existing, fmt.Errorf("update tasks epic_id: %w", err)
	}
	updated, err = res.RowsAffected()
	if err != nil {
		return 0, existing, fmt.Errorf("rows affected: %w", err)
	}
	return updated, existing, nil
}

type EpicTaskRow struct {
	ID            uuid.UUID
	RequirementID *uuid.UUID
	EpicID        *uuid.UUID
	Title         string
	Description   string
	Status        string
	Priority      int
	FilePaths     []string
}

func (r *Repo) ListEpicTasks(ctx context.Context, projectID uuid.UUID, epicID uuid.UUID, includeFiles bool) ([]EpicTaskRow, error) {
	if includeFiles {
		rows, err := r.db.QueryContext(ctx, `
SELECT t.id, t.requirement_id, t.epic_id, t.title, COALESCE(t.description, ''), t.status, t.priority,
       COALESCE(array_agg(tf.file_path ORDER BY tf.file_path) FILTER (WHERE tf.file_path IS NOT NULL), '{}') AS file_paths
FROM tasks t
LEFT JOIN task_files tf
  ON tf.project_id = t.project_id AND tf.task_id = t.id
WHERE t.project_id = $1 AND t.epic_id = $2
GROUP BY t.id, t.requirement_id, t.epic_id, t.title, t.description, t.status, t.priority
ORDER BY t.priority ASC, t.id ASC
`, projectID, epicID)
		if err != nil {
			return nil, fmt.Errorf("query epic tasks: %w", err)
		}
		defer func() { _ = rows.Close() }()

		var out []EpicTaskRow
		for rows.Next() {
			var (
				id        uuid.UUID
				reqID     sql.Null[uuid.UUID]
				epID      sql.Null[uuid.UUID]
				title     string
				desc      string
				status    string
				priority  int
				paths     pq.StringArray
			)
			if err := rows.Scan(&id, &reqID, &epID, &title, &desc, &status, &priority, &paths); err != nil {
				return nil, fmt.Errorf("scan epic task: %w", err)
			}
			row := EpicTaskRow{
				ID:          id,
				Title:       title,
				Description: desc,
				Status:      status,
				Priority:    priority,
				FilePaths:   []string(paths),
			}
			if reqID.Valid {
				v := reqID.V
				row.RequirementID = &v
			}
			if epID.Valid {
				v := epID.V
				row.EpicID = &v
			}
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate epic tasks: %w", err)
		}
		return out, nil
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT id, requirement_id, epic_id, title, COALESCE(description, ''), status, priority
FROM tasks
WHERE project_id = $1 AND epic_id = $2
ORDER BY priority ASC, id ASC
`, projectID, epicID)
	if err != nil {
		return nil, fmt.Errorf("query epic tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []EpicTaskRow
	for rows.Next() {
		var (
			id        uuid.UUID
			reqID     sql.Null[uuid.UUID]
			epID      sql.Null[uuid.UUID]
			title     string
			desc      string
			status    string
			priority  int
		)
		if err := rows.Scan(&id, &reqID, &epID, &title, &desc, &status, &priority); err != nil {
			return nil, fmt.Errorf("scan epic task: %w", err)
		}
		row := EpicTaskRow{
			ID:          id,
			Title:       title,
			Description: desc,
			Status:      status,
			Priority:    priority,
		}
		if reqID.Valid {
			v := reqID.V
			row.RequirementID = &v
		}
		if epID.Valid {
			v := epID.V
			row.EpicID = &v
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate epic tasks: %w", err)
	}
	return out, nil
}

