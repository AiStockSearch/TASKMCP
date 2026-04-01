ALTER TABLE requirements ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE epics ENABLE ROW LEVEL SECURITY;
ALTER TABLE github_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE mb_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE mb_state ENABLE ROW LEVEL SECURITY;
ALTER TABLE mb_rules ENABLE ROW LEVEL SECURITY;

-- When app.project_id is unset/empty, policies allow all rows (local dev / single-tenant).
-- When set to a UUID, rows are restricted to that project_id.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'requirements' AND policyname = 'p_requirements_project') THEN
    EXECUTE $p$
CREATE POLICY p_requirements_project ON requirements
USING (
  COALESCE(NULLIF(btrim(current_setting('app.project_id', true)), ''), '') = ''
  OR project_id = NULLIF(btrim(current_setting('app.project_id', true)), '')::uuid
)$p$;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'tasks' AND policyname = 'p_tasks_project') THEN
    EXECUTE $p$
CREATE POLICY p_tasks_project ON tasks
USING (
  COALESCE(NULLIF(btrim(current_setting('app.project_id', true)), ''), '') = ''
  OR project_id = NULLIF(btrim(current_setting('app.project_id', true)), '')::uuid
)$p$;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'task_files' AND policyname = 'p_task_files_project') THEN
    EXECUTE $p$
CREATE POLICY p_task_files_project ON task_files
USING (
  COALESCE(NULLIF(btrim(current_setting('app.project_id', true)), ''), '') = ''
  OR project_id = NULLIF(btrim(current_setting('app.project_id', true)), '')::uuid
)$p$;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'epics' AND policyname = 'p_epics_project') THEN
    EXECUTE $p$
CREATE POLICY p_epics_project ON epics
USING (
  COALESCE(NULLIF(btrim(current_setting('app.project_id', true)), ''), '') = ''
  OR project_id = NULLIF(btrim(current_setting('app.project_id', true)), '')::uuid
)$p$;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'github_links' AND policyname = 'p_github_links_project') THEN
    EXECUTE $p$
CREATE POLICY p_github_links_project ON github_links
USING (
  COALESCE(NULLIF(btrim(current_setting('app.project_id', true)), ''), '') = ''
  OR project_id = NULLIF(btrim(current_setting('app.project_id', true)), '')::uuid
)$p$;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'documents' AND policyname = 'p_documents_project') THEN
    EXECUTE $p$
CREATE POLICY p_documents_project ON documents
USING (
  COALESCE(NULLIF(btrim(current_setting('app.project_id', true)), ''), '') = ''
  OR project_id = NULLIF(btrim(current_setting('app.project_id', true)), '')::uuid
)$p$;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'document_chunks' AND policyname = 'p_document_chunks_project') THEN
    EXECUTE $p$
CREATE POLICY p_document_chunks_project ON document_chunks
USING (
  COALESCE(NULLIF(btrim(current_setting('app.project_id', true)), ''), '') = ''
  OR project_id = NULLIF(btrim(current_setting('app.project_id', true)), '')::uuid
)$p$;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'mb_documents' AND policyname = 'p_mb_documents_project') THEN
    EXECUTE $p$
CREATE POLICY p_mb_documents_project ON mb_documents
USING (
  COALESCE(NULLIF(btrim(current_setting('app.project_id', true)), ''), '') = ''
  OR project_id = NULLIF(btrim(current_setting('app.project_id', true)), '')::uuid
)$p$;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'mb_state' AND policyname = 'p_mb_state_project') THEN
    EXECUTE $p$
CREATE POLICY p_mb_state_project ON mb_state
USING (
  COALESCE(NULLIF(btrim(current_setting('app.project_id', true)), ''), '') = ''
  OR project_id = NULLIF(btrim(current_setting('app.project_id', true)), '')::uuid
)$p$;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'mb_rules' AND policyname = 'p_mb_rules_project') THEN
    EXECUTE $p$
CREATE POLICY p_mb_rules_project ON mb_rules
USING (
  project_id IS NULL
  OR COALESCE(NULLIF(btrim(current_setting('app.project_id', true)), ''), '') = ''
  OR project_id = NULLIF(btrim(current_setting('app.project_id', true)), '')::uuid
)$p$;
  END IF;
END $$;
