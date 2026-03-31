-- +migrate Up
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
CREATE INDEX IF NOT EXISTS idx_tasks_status_priority_id ON tasks(status, priority, id);

-- +migrate Down
DROP INDEX IF EXISTS idx_tasks_status_priority_id;
DROP INDEX IF EXISTS idx_requirements_epic_id;
DROP INDEX IF EXISTS idx_tasks_epic_id;

ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_epic_id_fkey;
ALTER TABLE requirements DROP CONSTRAINT IF EXISTS requirements_epic_id_fkey;

ALTER TABLE tasks DROP COLUMN IF EXISTS epic_id;
ALTER TABLE requirements DROP COLUMN IF EXISTS epic_id;

DROP TABLE IF EXISTS epics;
