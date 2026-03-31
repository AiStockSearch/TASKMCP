-- +migrate Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS documents (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_type TEXT NOT NULL,          -- markdown|decision|file
  source_path TEXT NOT NULL,          -- e.g. README.md
  title TEXT NULL,
  content_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (project_id, source_type, source_path, content_hash)
);

-- IMPORTANT: set embedding dimension to match your embedding model (e.g. 1536).
-- This migration uses a placeholder dimension 1536.
CREATE TABLE IF NOT EXISTS document_chunks (
  id UUID PRIMARY KEY,
  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  chunk_index INT NOT NULL,
  content TEXT NOT NULL,
  embedding vector(1536) NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (document_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_documents_project ON documents(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_document_chunks_project ON document_chunks(project_id, created_at);

-- For similarity search. ivfflat requires ANALYZE and works best with many rows.
CREATE INDEX IF NOT EXISTS idx_document_chunks_embedding_cosine
  ON document_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- +migrate Down
DROP INDEX IF EXISTS idx_document_chunks_embedding_cosine;
DROP INDEX IF EXISTS idx_document_chunks_project;
DROP INDEX IF EXISTS idx_documents_project;
DROP TABLE IF EXISTS document_chunks;
DROP TABLE IF EXISTS documents;
DROP EXTENSION IF EXISTS vector;
