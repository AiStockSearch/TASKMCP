CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY,
  repo_owner TEXT NOT NULL,
  repo_name TEXT NOT NULL,
  repo_key TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (repo_key)
);

ALTER TABLE requirements
  ADD COLUMN IF NOT EXISTS project_id UUID NULL;
ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS project_id UUID NULL;
ALTER TABLE epics
  ADD COLUMN IF NOT EXISTS project_id UUID NULL;
ALTER TABLE task_files
  ADD COLUMN IF NOT EXISTS project_id UUID NULL;
ALTER TABLE github_links
  ADD COLUMN IF NOT EXISTS project_id UUID NULL;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'requirements_project_id_fkey') THEN
    ALTER TABLE requirements
      ADD CONSTRAINT requirements_project_id_fkey
      FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tasks_project_id_fkey') THEN
    ALTER TABLE tasks
      ADD CONSTRAINT tasks_project_id_fkey
      FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'epics_project_id_fkey') THEN
    ALTER TABLE epics
      ADD CONSTRAINT epics_project_id_fkey
      FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'task_files_project_id_fkey') THEN
    ALTER TABLE task_files
      ADD CONSTRAINT task_files_project_id_fkey
      FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'github_links_project_id_fkey') THEN
    ALTER TABLE github_links
      ADD CONSTRAINT github_links_project_id_fkey
      FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_projects_repo_key ON projects(repo_key);
CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_requirements_project_id ON requirements(project_id);
CREATE INDEX IF NOT EXISTS idx_epics_project_id ON epics(project_id);
CREATE INDEX IF NOT EXISTS idx_task_files_project_id ON task_files(project_id);
CREATE INDEX IF NOT EXISTS idx_github_links_project_id ON github_links(project_id);
