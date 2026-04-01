DROP INDEX IF EXISTS idx_github_links_project_id;
DROP INDEX IF EXISTS idx_task_files_project_id;
DROP INDEX IF EXISTS idx_epics_project_id;
DROP INDEX IF EXISTS idx_requirements_project_id;
DROP INDEX IF EXISTS idx_tasks_project_id;
DROP INDEX IF EXISTS idx_projects_repo_key;

ALTER TABLE github_links DROP CONSTRAINT IF EXISTS github_links_project_id_fkey;
ALTER TABLE task_files DROP CONSTRAINT IF EXISTS task_files_project_id_fkey;
ALTER TABLE epics DROP CONSTRAINT IF EXISTS epics_project_id_fkey;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_project_id_fkey;
ALTER TABLE requirements DROP CONSTRAINT IF EXISTS requirements_project_id_fkey;

ALTER TABLE github_links DROP COLUMN IF EXISTS project_id;
ALTER TABLE task_files DROP COLUMN IF EXISTS project_id;
ALTER TABLE epics DROP COLUMN IF EXISTS project_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS project_id;
ALTER TABLE requirements DROP COLUMN IF EXISTS project_id;

DROP TABLE IF EXISTS projects;
