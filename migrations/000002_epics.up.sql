CREATE TABLE IF NOT EXISTS epics (
  id UUID PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT NULL,
  status VARCHAR NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE requirements
  ADD COLUMN IF NOT EXISTS epic_id UUID NULL;

ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS epic_id UUID NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'requirements_epic_id_fkey'
  ) THEN
    ALTER TABLE requirements
      ADD CONSTRAINT requirements_epic_id_fkey
      FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE SET NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'tasks_epic_id_fkey'
  ) THEN
    ALTER TABLE tasks
      ADD CONSTRAINT tasks_epic_id_fkey
      FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE SET NULL;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_tasks_epic_id ON tasks(epic_id);
CREATE INDEX IF NOT EXISTS idx_requirements_epic_id ON requirements(epic_id);
