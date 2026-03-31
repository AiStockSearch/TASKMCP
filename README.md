# MCP Vault Bridge (Cursor ↔ Postgres “Vault”)

Этот репозиторий содержит **MCP Server на Go**, который Cursor запускает как **stdio subprocess**.  
Сервер выступает “мостом” между Cursor AI и PostgreSQL (“Vault”): выдаёт задачи из бэклога, отмечает выполнение, связывает задачи с файлами, управляет Epic’ами, умеет создавать GitHub Issue по задаче (через GitHub App), и хранит контекст/документы в Postgres для RAG (pgvector).

## Зачем он нужен

- **Автономный пайплайн разработки**: Cursor может получать следующую задачу, видеть backlog, закрывать задачи с отчётом.
- **Multi-project**: задачи/эпики/контекст разделяются по проектам (проект = `repo_key` вида `owner/repo`).
- **RAG контекст**: можно складывать Markdown/решения в Postgres + pgvector и делать semantic search по embedding-векторам (embeddings генерируются снаружи).

## Быстрый старт

### 1) Переменные окружения

Создайте `.env` рядом с бинарём (или задайте env в настройках Cursor):

```env
DATABASE_URL=postgres://user:pass@127.0.0.1:5432/dbname?sslmode=disable

# Опционально: дефолтный проект для вызовов без repo_key
DEFAULT_REPO_KEY=org/repo

# Опционально: размерность embedding (должна совпадать с vector(N) в миграции)
KB_EMBEDDING_DIM=1536

# Опционально: GitHub App (для github_create_issue_for_task)
GITHUB_APP_ID=123
GITHUB_APP_INSTALLATION_ID=456
# один из вариантов:
GITHUB_APP_PRIVATE_KEY_PEM="-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
# или:
GITHUB_APP_PRIVATE_KEY_PATH=/abs/path/to/private-key.pem
```

### 2) Сборка

```bash
go build -o mcp-vault-bridge .
```

### 3) Подключение к Cursor

- Cursor → **Settings → Features → MCP → Command**
- Укажите абсолютный путь до бинаря: `.../mcp-vault-bridge`
- Убедитесь, что `DATABASE_URL` доступен (через `.env` в working dir или env переменные).

Важно: сервер пишет логи в **stderr** и общается с Cursor по **stdout** (stdio MCP).

## Миграции БД

SQL миграции лежат в каталоге `migrations/`:

- `0001_epics.sql` — `epics` + `epic_id` в `requirements/tasks`
- `0002_github_links.sql` — `github_links`
- `0003_projects.sql` — `projects` + `project_id` во всех сущностях
- `0004_tasks_created_at.sql` — `tasks.created_at` (для `order=created_at`)
- `0005_pgvector_kb.sql` — `pgvector` + `documents/document_chunks` (embedding `vector(1536)`)

Накатывайте любым мигратором (например `golang-migrate`), в правильном порядке.

## Multi-project: как задаётся проект

Проект идентифицируется как `repo_key = "owner/repo"`.

- Почти все tools принимают `repo_key` (опционально).
- Если `repo_key` не передан, используется `DEFAULT_REPO_KEY` из env.
- При первом обращении проект автоматически создаётся в таблице `projects`.

## MCP tools (основные)

### Tasks
- `get_next_task(repo_key?)` → берёт `todo` задачу с минимальным `priority`, переводит в `in_progress`, возвращает `file_paths`.
- `complete_task(repo_key?, task_id, report)` → строго завершает только `in_progress` (ставит `done` + append отчёта в `description`).
- `add_context_file(repo_key?, task_id, file_path)` → идемпотентно добавляет запись в `task_files`.
- `list_tasks(repo_key?, status?, requirement_id?, epic_id?, limit?, offset?, order?, include_files?)`
- `get_task(repo_key?, task_id)`

### Epics
- `create_epic(repo_key?, title, description?)`
- `list_epics(repo_key?, status?, limit?, offset?)`
- `link_requirement_to_epic(repo_key?, requirement_id, epic_id)`
- `link_task_to_epic(repo_key?, task_id, epic_id)`

### GitHub Issues (GitHub App)
- `github_get_issue_link(repo_key?, entity_type, entity_id)` → читает связь из Postgres (без GitHub API).
- `github_create_issue_for_task(repo_key?, task_id, repo_owner, repo_name, title_override?, body_mode?)`  
  Создаёт Issue через GitHub API и сохраняет ссылку в `github_links`.

### Knowledge Base (pgvector, embeddings снаружи)
- `kb_chunk_markdown(text, max_chars?, overlap_chars?)`  
  Heading-aware чанкинг Markdown (без embeddings).
- `kb_upsert_document_chunks(repo_key?, source_type?, source_path, title?, full_text, chunks[])`  
  Upsert документа и чанков, **ожидает embeddings в chunks**.
- `kb_search_context(repo_key?, query_embedding[], top_k?)`  
  Semantic search по pgvector.

## Рекомендуемый RAG pipeline (embeddings снаружи)

1) Взять Markdown/решение (например файл в репозитории).
2) Вызвать `kb_chunk_markdown` → получить `chunks[]` (content + metadata).
3) Сгенерировать embeddings для каждого `chunk.content` во внешнем оркестраторе (LangChain/скрипт/сервис).
4) Вызвать `kb_upsert_document_chunks` и записать чанки в Postgres.
5) Для запроса: эмбеддить query → `kb_search_context` → topK чанков → собрать prompt.

## Ограничения и заметки

- В миграции `document_chunks.embedding` задан как `vector(1536)`. Если вы используете embedding модель с другой размерностью — обновите миграцию/схему и `KB_EMBEDDING_DIM`.
- `order=created_at` для задач требует накатывания `0004_tasks_created_at.sql`.

