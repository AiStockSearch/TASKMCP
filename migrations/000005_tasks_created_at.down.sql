DROP INDEX IF EXISTS idx_tasks_project_created_at;
ALTER TABLE tasks DROP COLUMN IF EXISTS created_at;
