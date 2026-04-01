-- Core Vault tables required before epics/projects migrations.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS requirements (
  id UUID PRIMARY KEY,
  title TEXT NOT NULL,
  spec_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  status VARCHAR NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tasks (
  id UUID PRIMARY KEY,
  requirement_id UUID NULL REFERENCES requirements(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  description TEXT NULL,
  status VARCHAR NOT NULL,
  priority INT NOT NULL DEFAULT 100
);

CREATE TABLE IF NOT EXISTS task_files (
  id UUID PRIMARY KEY,
  task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  file_path TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (task_id, file_path)
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_priority_id ON tasks(status, priority, id);
