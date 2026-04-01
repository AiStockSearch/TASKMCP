CREATE TABLE IF NOT EXISTS mb_documents (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  doc_type TEXT NOT NULL,
  doc_key TEXT NOT NULL,
  title TEXT NULL,
  current_version INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (project_id, doc_key)
);

CREATE TABLE IF NOT EXISTS mb_document_versions (
  id UUID PRIMARY KEY,
  document_id UUID NOT NULL REFERENCES mb_documents(id) ON DELETE CASCADE,
  version INT NOT NULL,
  content TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (document_id, version)
);

CREATE INDEX IF NOT EXISTS idx_mb_docs_project_type ON mb_documents(project_id, doc_type, updated_at);
CREATE INDEX IF NOT EXISTS idx_mb_doc_versions_doc_created ON mb_document_versions(document_id, created_at);

CREATE TABLE IF NOT EXISTS mb_state (
  project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
  state_json JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mb_rules (
  id UUID PRIMARY KEY,
  project_id UUID NULL REFERENCES projects(id) ON DELETE CASCADE,
  scope TEXT NOT NULL,
  priority INT NOT NULL,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  enabled BOOL NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mb_rules_project_scope_priority
  ON mb_rules(project_id, enabled, scope, priority);
CREATE INDEX IF NOT EXISTS idx_mb_rules_global_scope_priority
  ON mb_rules(enabled, scope, priority);
