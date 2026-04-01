ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_tasks_project_created_at ON tasks(project_id, created_at);
