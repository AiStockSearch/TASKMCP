-- +migrate Up
CREATE TABLE IF NOT EXISTS github_links (
  id UUID PRIMARY KEY,
  entity_type TEXT NOT NULL, -- epic|requirement|task
  entity_id UUID NOT NULL,
  repo_owner TEXT NOT NULL,
  repo_name TEXT NOT NULL,
  issue_number INT NOT NULL,
  issue_url TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_type, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_github_links_entity ON github_links(entity_type, entity_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_github_links_entity;
DROP TABLE IF EXISTS github_links;
