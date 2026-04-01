DROP INDEX IF EXISTS idx_requirements_epic_id;
DROP INDEX IF EXISTS idx_tasks_epic_id;

ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_epic_id_fkey;
ALTER TABLE requirements DROP CONSTRAINT IF EXISTS requirements_epic_id_fkey;

ALTER TABLE tasks DROP COLUMN IF EXISTS epic_id;
ALTER TABLE requirements DROP COLUMN IF EXISTS epic_id;

DROP TABLE IF EXISTS epics;
