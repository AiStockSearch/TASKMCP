-- Seed scenario:
-- "Авторизация и регистрация для мобильного приложения React Native"
--
-- Prereqs:
-- - golang-migrate: все .up.sql до 000007_memory_bank включительно (или полный `make migrate-up`)
-- - pgcrypto available for gen_random_uuid():
--     CREATE EXTENSION IF NOT EXISTS pgcrypto;
--
-- How to run:
--   psql "$DATABASE_URL" -v repo_key='acme/mobile-app' -f scripts/seed_rn_auth.sql

BEGIN;

-- 1) Ensure project for repo_key = :'repo_key'
WITH rk AS (
  SELECT :'repo_key'::text AS repo_key
),
parts AS (
  SELECT
    split_part(repo_key, '/', 1) AS repo_owner,
    split_part(repo_key, '/', 2) AS repo_name,
    repo_key
  FROM rk
),
ins AS (
  INSERT INTO projects (id, repo_owner, repo_name, repo_key)
  SELECT gen_random_uuid(), repo_owner, repo_name, repo_key
  FROM parts
  ON CONFLICT (repo_key) DO NOTHING
  RETURNING id
)
SELECT 1;

-- 2) Requirement + Epic + Tasks
WITH p AS (
  SELECT id AS project_id FROM projects WHERE repo_key = :'repo_key'
),
e AS (
  INSERT INTO epics (id, project_id, title, description, status)
  SELECT gen_random_uuid(), p.project_id, 'Mobile Auth MVP', 'MVP авторизации/регистрации для RN приложения', 'open'
  FROM p
  RETURNING id AS epic_id, project_id
),
r AS (
  INSERT INTO requirements (id, project_id, epic_id, title, spec_json, status)
  SELECT gen_random_uuid(),
         e.project_id,
         e.epic_id,
         'Авторизация и регистрация для мобильного приложения React Native',
         '{
            "platform":"react-native",
            "auth_methods":["email_password"],
            "definition_of_done":[
              "Пользователь может зарегистрироваться по email/password",
              "Пользователь может войти и остаётся залогинен после перезапуска приложения",
              "Logout работает и чистит токены",
              "Ошибки (невалидный email/пароль, сеть) отображаются корректно"
            ],
            "non_goals":["OAuth/SSO на первом этапе"],
            "notes":["Подготовить код для будущего добавления OAuth (Google/Apple)"]
          }'::jsonb,
         'todo'
  FROM e
  RETURNING id AS requirement_id, project_id, epic_id
),
t AS (
  INSERT INTO tasks (id, project_id, requirement_id, epic_id, title, description, status, priority)
  SELECT gen_random_uuid(), r.project_id, r.requirement_id, r.epic_id,
         x.title, x.description, 'todo', x.priority
  FROM r
  JOIN (VALUES
    (10, 'Определить auth flows и состояния', 'Описать login/signup/forgot, валидацию, ошибки, loading, навигацию между экранами.'),
    (20, 'Реализовать API-клиент и auth endpoints', 'Добавить HTTP клиент, модели запросов/ответов, методы signup/login/logout/refresh (если есть).'),
    (30, 'Сделать secure token storage', 'Сохранение access/refresh токенов (Keychain/Keystore), helper для чтения/очистки, logout.'),
    (40, 'Собрать UI: Login/Signup', 'Экраны входа/регистрации с валидацией, отображением ошибок, блокировкой кнопок при запросах.'),
    (50, 'Навигация и guard: auth stack vs app stack', 'Маршрутизация неавторизованных пользователей в auth stack; обработка logout.')
  ) AS x(priority, title, description) ON TRUE
  RETURNING id AS task_id, project_id, title
)
SELECT 1;

-- 3) Task files (примерный список)
WITH p AS (
  SELECT id AS project_id FROM projects WHERE repo_key = :'repo_key'
),
t AS (
  SELECT id AS task_id, title
  FROM tasks
  WHERE project_id = (SELECT project_id FROM p)
)
INSERT INTO task_files (id, project_id, task_id, file_path)
SELECT gen_random_uuid(), p.project_id, t.task_id, f.file_path
FROM p
JOIN t ON TRUE
JOIN LATERAL (
  SELECT unnest(CASE t.title
    WHEN 'Определить auth flows и состояния' THEN ARRAY[
      'apps/mobile/src/navigation/AuthStack.tsx',
      'apps/mobile/src/screens/LoginScreen.tsx',
      'apps/mobile/src/screens/SignupScreen.tsx'
    ]
    WHEN 'Реализовать API-клиент и auth endpoints' THEN ARRAY[
      'apps/mobile/src/api/httpClient.ts',
      'apps/mobile/src/api/authApi.ts',
      'apps/mobile/src/api/types.ts'
    ]
    WHEN 'Сделать secure token storage' THEN ARRAY[
      'apps/mobile/src/auth/tokenStore.ts',
      'apps/mobile/src/auth/authService.ts'
    ]
    WHEN 'Собрать UI: Login/Signup' THEN ARRAY[
      'apps/mobile/src/screens/LoginScreen.tsx',
      'apps/mobile/src/screens/SignupScreen.tsx'
    ]
    WHEN 'Навигация и guard: auth stack vs app stack' THEN ARRAY[
      'apps/mobile/src/navigation/RootNavigator.tsx',
      'apps/mobile/src/navigation/AuthStack.tsx'
    ]
    ELSE ARRAY[]::text[]
  END) AS file_path
) AS f ON TRUE
ON CONFLICT DO NOTHING;

-- 4) Memory Bank docs + state
WITH p AS (
  SELECT id AS project_id FROM projects WHERE repo_key = :'repo_key'
),
docs AS (
  SELECT * FROM (VALUES
    ('memory-bank/tasks.md', 'tasks', 'Backlog', '## Backlog\n\n- [ ] Auth & Registration (RN)\n'),
    ('memory-bank/activeContext.md', 'activeContext', 'Active Context', 'Фокус: Mobile Auth MVP (RN).\n\nСледующий шаг: взять задачу с наивысшим приоритетом и выполнить.\n'),
    ('memory-bank/progress.md', 'progress', 'Progress', 'Старт проекта.\n')
  ) AS v(doc_key, doc_type, title, content)
),
ins_docs AS (
  INSERT INTO mb_documents (id, project_id, doc_type, doc_key, title, current_version)
  SELECT gen_random_uuid(), p.project_id, docs.doc_type, docs.doc_key, docs.title, 0
  FROM p
  JOIN docs ON TRUE
  ON CONFLICT (project_id, doc_key) DO NOTHING
  RETURNING id, doc_key
),
all_docs AS (
  SELECT d.id, d.doc_key, d.project_id, docs.content, d.current_version
  FROM mb_documents d
  JOIN p ON p.project_id = d.project_id
  JOIN docs ON docs.doc_key = d.doc_key
),
ins_versions AS (
  INSERT INTO mb_document_versions (id, document_id, version, content, content_hash)
  SELECT gen_random_uuid(), all_docs.id, all_docs.current_version + 1, all_docs.content,
         encode(digest(all_docs.content, 'sha256'), 'hex')
  FROM all_docs
  WHERE all_docs.current_version = 0
  ON CONFLICT DO NOTHING
),
upd_docs AS (
  UPDATE mb_documents d
  SET current_version = 1, updated_at = now()
  FROM p
  WHERE d.project_id = p.project_id AND d.doc_key IN (SELECT doc_key FROM docs) AND d.current_version = 0
  RETURNING d.id
)
INSERT INTO mb_state (project_id, state_json, updated_at)
SELECT p.project_id,
       '{"phase":"PLAN","current_requirement_title":"Авторизация и регистрация для мобильного приложения React Native"}'::jsonb,
       now()
FROM p
ON CONFLICT (project_id) DO UPDATE
SET state_json = EXCLUDED.state_json, updated_at = now();

COMMIT;

