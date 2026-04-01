package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

type TaskRow struct {
	ID            uuid.UUID
	RequirementID *uuid.UUID
	EpicID        *uuid.UUID
	Title         string
	Description   string
	Status        string
	Priority      int
}

const (
	qGetNextTodo = `
SELECT id, title, description
FROM tasks
WHERE project_id = $1 AND status = 'todo'
ORDER BY priority ASC, id ASC
FOR UPDATE SKIP LOCKED
LIMIT 1
`
	qSetTaskInProgress = `UPDATE tasks SET status = 'in_progress' WHERE project_id = $1 AND id = $2`
	qTaskFilesForTask  = `
SELECT file_path
FROM task_files
WHERE project_id = $1 AND task_id = $2
ORDER BY file_path ASC
`
)

func (r *Repo) GetNextTodoAndLock(ctx context.Context, projectID uuid.UUID) (TaskRow, []string, bool, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return TaskRow{}, nil, false, fmt.Errorf("db begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var row TaskRow
	err = tx.QueryRowContext(ctx, qGetNextTodo, projectID).Scan(&row.ID, &row.Title, &row.Description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskRow{}, nil, false, nil
		}
		return TaskRow{}, nil, false, fmt.Errorf("scan next task: %w", err)
	}

	if _, err := tx.ExecContext(ctx, qSetTaskInProgress, projectID, row.ID); err != nil {
		return TaskRow{}, nil, false, fmt.Errorf("update task status: %w", err)
	}

	rows, err := tx.QueryContext(ctx, qTaskFilesForTask, projectID, row.ID)
	if err != nil {
		return TaskRow{}, nil, false, fmt.Errorf("query task files: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return TaskRow{}, nil, false, fmt.Errorf("scan task file: %w", err)
		}
		paths = append(paths, p)
	}
	if err := rows.Err(); err != nil {
		return TaskRow{}, nil, false, fmt.Errorf("iterate task files: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return TaskRow{}, nil, false, fmt.Errorf("commit tx: %w", err)
	}
	return row, paths, true, nil
}

const qCompleteTask = `
UPDATE tasks
SET status = 'done',
    description = COALESCE(description, '') || $3
WHERE project_id = $1 AND id = $2 AND status = 'in_progress'
`

func (r *Repo) CompleteInProgress(ctx context.Context, projectID uuid.UUID, taskID uuid.UUID, appendix string) (bool, error) {
	res, err := r.db.ExecContext(ctx, qCompleteTask, projectID, taskID, appendix)
	if err != nil {
		return false, fmt.Errorf("update task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return n == 1, nil
}

const qAddContextFile = `
INSERT INTO task_files (id, project_id, task_id, file_path)
SELECT $1, $2, $3, $4
WHERE NOT EXISTS (
  SELECT 1 FROM task_files WHERE project_id = $2 AND task_id = $3 AND file_path = $4
)
`

func (r *Repo) AddContextFile(ctx context.Context, projectID uuid.UUID, taskID uuid.UUID, filePath string, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO task_files (id, project_id, task_id, file_path)
SELECT $1, $2, $3, $4
WHERE NOT EXISTS (
  SELECT 1 FROM task_files WHERE project_id = $2 AND task_id = $3 AND file_path = $4
)
`, id, projectID, taskID, filePath)
	if err != nil {
		return fmt.Errorf("insert task_files: %w", err)
	}
	return nil
}

type ListOptions struct {
	ProjectID     uuid.UUID
	Status        string
	RequirementID *uuid.UUID
	EpicID        *uuid.UUID
	Limit         int
	Offset        int
	Order         string
}

func (r *Repo) List(ctx context.Context, opt ListOptions) ([]TaskRow, error) {
	var (
		args  []any
		where []string
	)
	args = append(args, opt.ProjectID)
	where = append(where, fmt.Sprintf("project_id = $%d", len(args)))
	if opt.Status != "" {
		args = append(args, opt.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if opt.RequirementID != nil {
		args = append(args, *opt.RequirementID)
		where = append(where, fmt.Sprintf("requirement_id = $%d", len(args)))
	}
	if opt.EpicID != nil {
		args = append(args, *opt.EpicID)
		where = append(where, fmt.Sprintf("epic_id = $%d", len(args)))
	}

	orderBy := "priority ASC, id ASC"
	if opt.Order == "created_at" {
		orderBy = "created_at ASC, id ASC"
	}

	query := `
SELECT id, requirement_id, epic_id, title, description, status, priority
FROM tasks
`
	if len(where) > 0 {
		query += "WHERE " + joinAnd(where) + "\n"
	}
	query += "ORDER BY " + orderBy + "\n"

	args = append(args, opt.Limit)
	query += fmt.Sprintf("LIMIT $%d\n", len(args))
	args = append(args, opt.Offset)
	query += fmt.Sprintf("OFFSET $%d\n", len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []TaskRow
	for rows.Next() {
		var (
			id            uuid.UUID
			reqID         sql.Null[uuid.UUID]
			epID          sql.Null[uuid.UUID]
			title         string
			desc          sql.NullString
			status        string
			priority      int
		)
		if err := rows.Scan(&id, &reqID, &epID, &title, &desc, &status, &priority); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tr := TaskRow{
			ID:          id,
			Title:       title,
			Description: desc.String,
			Status:      status,
			Priority:    priority,
		}
		if reqID.Valid {
			v := reqID.V
			tr.RequirementID = &v
		}
		if epID.Valid {
			v := epID.V
			tr.EpicID = &v
		}
		out = append(out, tr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return out, nil
}

func (r *Repo) Get(ctx context.Context, projectID uuid.UUID, taskID uuid.UUID) (TaskRow, bool, error) {
	var (
		reqID    sql.Null[uuid.UUID]
		epID     sql.Null[uuid.UUID]
		title    string
		desc     sql.NullString
		status   string
		priority int
	)
	row := r.db.QueryRowContext(ctx, `
SELECT requirement_id, epic_id, title, description, status, priority
FROM tasks
WHERE project_id = $1 AND id = $2
`, projectID, taskID)
	if err := row.Scan(&reqID, &epID, &title, &desc, &status, &priority); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskRow{}, false, nil
		}
		return TaskRow{}, false, fmt.Errorf("query task: %w", err)
	}
	tr := TaskRow{
		ID:          taskID,
		Title:       title,
		Description: desc.String,
		Status:      status,
		Priority:    priority,
	}
	if reqID.Valid {
		v := reqID.V
		tr.RequirementID = &v
	}
	if epID.Valid {
		v := epID.V
		tr.EpicID = &v
	}
	return tr, true, nil
}

func (r *Repo) FilesByTaskIDs(ctx context.Context, projectID uuid.UUID, taskIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	out := make(map[uuid.UUID][]string, len(taskIDs))
	for _, id := range taskIDs {
		out[id] = nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT task_id, file_path
FROM task_files
WHERE project_id = $1 AND task_id = ANY($2)
ORDER BY task_id ASC, file_path ASC
`, projectID, pq.Array(taskIDs))
	if err != nil {
		return nil, fmt.Errorf("query task_files: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var tid uuid.UUID
		var p string
		if err := rows.Scan(&tid, &p); err != nil {
			return nil, fmt.Errorf("scan task_files: %w", err)
		}
		out[tid] = append(out[tid], p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task_files: %w", err)
	}
	return out, nil
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

