# MCP Vault Bridge (Cursor ↔ Postgres «Vault»)

MCP-сервер на **Go** со **stdio**-транспортом: Cursor запускает его как subprocess. Это тонкий мост между ассистентом и PostgreSQL: задачи, эпики, GitHub Issues (GitHub App), база знаний (**pgvector** + FTS), **Memory Bank** и multi-project по `repo_key`.

**Сквозная архитектура (LangChain → Vault → MCP → Cursor):** см. [`docs/architecture-five-layers.md`](docs/architecture-five-layers.md).

## Зачем он нужен

- **Бэклог и исполнение**: взять следующую задачу (`SKIP LOCKED`), закрыть с отчётом, повесить контекстные файлы.
- **Multi-project**: изоляция по `repo_key` (`owner/repo`) и таблице `projects`.
- **RAG**: чанки + embeddings в Postgres; гибридный поиск `FTS + pgvector`.
- **Write-side**: данные создаёт внешний оркестратор (например LangChain) прямо в SQL; контракт и dedup — в `docs/writer-contract.md`.

## Локальная разработка (Docker)

Образ Postgres: **`pgvector/pgvector:pg16`** (расширение `vector` для миграций).

```bash
make up              # Postgres + Redis; хост: Postgres 5433, Redis 6380 (чтобы не занимать локальный 6379)
# свой порт Redis: make up REDIS_HOST_PORT=6379
make migrate-up      # golang-migrate (контейнер migrate)
make seed-rn-auth    # пример сценария RN auth (после успешных миграций)
```

Полезные цели Makefile:

| Команда | Назначение |
|--------|------------|
| `make up` | Поднять Postgres и **Redis** (`POSTGRES_HOST_PORT`, по умолчанию Redis на хосте **6380**; переопределение: `REDIS_HOST_PORT=6379`) |
| `make down` | Остановить compose и удалить volume |
| `make logs` | Логи Postgres |
| `make migrate-up` | Накатить все `*.up.sql` |
| `make migrate-down` | Откатить **одну** миграцию (`POSTGRES_URL`) |
| `make seed-rn-auth` | `psql -f scripts/seed_rn_auth.sql` |
| `make psql-url` | Напечатать текущий `POSTGRES_URL` |
| `make orchestrator-up` | Сборка и запуск **orchestrator-api** + **orchestrator-worker** (профиль `orchestrator`, нужен `WEBHOOK_SECRET`) |
| `make apply-plan-bundle` | `POST` PlanBundle (`PLAN_BUNDLE_FILE`, `ORCHESTRATOR_URL`; `WEBHOOK_SECRET` по умолчанию `dev-change-me`, как в compose) |
| `make orch-webhook-secret` | Показать `WEBHOOK_SECRET` из контейнера `orchestrator-api` (сверка при **401 invalid token**) |

### Cursor: Architect-Inquisitor → PlanBundle

В репозитории есть проектная команда [`.cursor/commands/architect-inquisitor.md`](.cursor/commands/architect-inquisitor.md): диалог «инквизитор» в Cursor, в конце — **только JSON** по [`docs/writer-contract.md`](docs/writer-contract.md) (ориентир [`examples/planbundle_rn_auth.json`](examples/planbundle_rn_auth.json)). Сохраняйте пакет в [`plans/`](plans/). Правило для таких файлов: [`.cursor/rules/plan-bundle.mdc`](.cursor/rules/plan-bundle.mdc).

После запуска API оркестратора: `make apply-plan-bundle` подставляет `WEBHOOK_SECRET` из корневого `.env` (как контейнер оркестратора), иначе `dev-change-me`. Случайный `export WEBHOOK_SECRET=…` в shell **не** подменяет `.env` при вызове make (раньше плейсхолдер из документации давал **401**).

```bash
# пример пакета из репозитория (файл в plans/ сначала нужно создать, напр. через команду Architect-Inquisitor)
make apply-plan-bundle
make apply-plan-bundle PLAN_BUNDLE_FILE=plans/planbundle-my-feature.json
```

Если ответ **401 invalid token**: сверьте `make orch-webhook-secret` с тем, что в `.env`; при необходимости явно: `make apply-plan-bundle WEBHOOK_SECRET='…'`. **Для `orchestrator-up`:** если в shell был `export WEBHOOK_SECRET=…`, при `docker compose` он перебивает `.env` — поэтому `make orchestrator-up` сбрасывает его на время вызова compose; после смены `.env` выполните `make orchestrator-up` ещё раз.

### Оркестратор (отдельный брокер Redis)

Каталог [`orchestrator/`](orchestrator/README.md): **FastAPI** принимает PlanBundle (`POST /hooks/apply-plan-bundle`), кладёт задание в **Redis**, воркер **arq** применяет пакет к Vault по [`docs/writer-contract.md`](docs/writer-contract.md).

```bash
make up && make migrate-up
# положите WEBHOOK_SECRET в корневой `.env` (репозиторий); не полагайтесь на export в shell — он перетирал бы .env
make orchestrator-up
# POST http://localhost:8080/hooks/apply-plan-bundle (+ Authorization: Bearer …)
```

Строка подключения по умолчанию:

`postgres://vault:vault@127.0.0.1:5433/vault?sslmode=disable`

Переопределение: `make up POSTGRES_HOST_PORT=5432` или `make seed-rn-auth POSTGRES_URL='postgres://...'`.

**Важно:** миграции оформлены для **golang-migrate** (`migrations/NNNNNN_name.up.sql` / `.down.sql`). Если `migrate-up` падал на середине, проще сбросить состояние: `make down` (с `-v` удалит данные) и повторить цикл.

## Переменные окружения

Создайте `.env` в корне репозитория или задайте переменные в настройках Cursor.

При запуске MCP **рабочий каталог** часто не совпадает с корнем репо: бинарий поднимает `.env`, идя **вверх от cwd**, пока не найдёт файл с `DATABASE_URL`. При необходимости укажите путь явно: `MCP_VAULT_BRIDGE_DOTENV=/abs/path/to/TASKMCP/.env`.

```env
# Обязательно для tools, которые ходят в БД
DATABASE_URL=postgres://vault:vault@127.0.0.1:5433/vault?sslmode=disable

# Если tool вызван без repo_key
DEFAULT_REPO_KEY=org/repo

# Размерность вектора (должна совпадать с vector(N) в миграции, по умолчанию 1536)
KB_EMBEDDING_DIM=1536

# GitHub App — для github_create_issue_for_task
GITHUB_APP_ID=123
GITHUB_APP_INSTALLATION_ID=456
GITHUB_APP_PRIVATE_KEY_PEM="-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
# или:
GITHUB_APP_PRIVATE_KEY_PATH=/abs/path/to/private-key.pem
```

Сервер **не обязан** падать при недоступной БД на старте; при ошибке подключения tools вернут понятную ошибку до восстановления `DATABASE_URL`.

## Сборка и Cursor

```bash
go build -o mcp-vault-bridge .
```

- Cursor → **Settings → Features → MCP** → command: абсолютный путь к `mcp-vault-bridge`.
- Логи — в **stderr**, протокол MCP — **stdout**.

## Миграции

Каталог `migrations/`, формат **golang-migrate**:

| Версия | Содержание |
|--------|------------|
| `000001` | Базовые `requirements`, `tasks`, `task_files`, `pgcrypto` |
| `000002` | `epics`, `epic_id` |
| `000003` | `github_links` |
| `000004` | `projects`, `project_id` |
| `000005` | `tasks.created_at`, индекс под `order=created_at` |
| `000006` | `pgvector`, `documents`, `document_chunks` |
| `000007` | Memory Bank: `mb_documents`, версии, `mb_state`, `mb_rules` |
| `000008` | Unique-индексы для idempotent write-side |
| `000009` | FTS: `content_tsv`, GIN |
| `000010` | RLS (см. раздел Security) |

## Multi-project (`repo_key`)

- Почти все инструменты принимают опциональный `repo_key` (`owner/repo`).
- Если не передан — используется `DEFAULT_REPO_KEY`.
- При первом обращении строка в `projects` создаётся автоматически.

## MCP tools

### Tasks

| Tool | Назначение |
|------|------------|
| `get_next_task` | `todo` → `in_progress`, приоритет, файлы, при наличии связи — `requirement_id`, `requirement_title`, `spec_json` (DoD/ограничения из Vault) |
| `complete_task` | Только из `in_progress` → `done`, append отчёта |
| `add_context_file` | Идемпотентная связь `task_files` |
| `list_tasks` | Фильтры, пагинация, `order` = `priority` \| `created_at`, `include_files`, опционально `include_requirement_spec` (`requirement_title` + `spec_json`) |
| `get_task` | Одна задача + файлы; те же поля требования (`requirement_title`, `spec_json`); GitHub link при наличии |

### Epics

| Tool | Назначение |
|------|------------|
| `create_epic` | Создать эпик |
| `list_epics` | Список, фильтр `status` |
| `link_requirement_to_epic` | `requirements.epic_id` |
| `link_task_to_epic` | `tasks.epic_id` |
| `epic_add_tasks` | Массово привязать `task_ids[]` к эпику |
| `epic_list_tasks` | Задачи эпика, опционально `include_files` |

### GitHub Issues (GitHub App)

| Tool | Назначение |
|------|------------|
| `github_get_issue_link` | Чтение `github_links` из БД |
| `github_create_issue_for_task` | Создание Issue в API + сохранение связи |

### Knowledge Base

| Tool | Назначение |
|------|------------|
| `kb_chunk_markdown` | Чанкинг Markdown (без embeddings) |
| `kb_upsert_document_chunks` | Upsert документа и чанков **с embeddings** |
| `kb_search_context` | Поиск по вектору |
| `kb_hybrid_search` | `query_text` + опционально `query_embedding`, веса FTS/vector |

### Memory Bank (`mb_*`)

Документы версионируются по хешу контента. Поле **`doc_type`** при upsert: например `activeContext`, `progress`, `plan`, `adr`, `tasks`, `refactor_plan`, `reflection`, `archive` (см. описание tool `mb_upsert_document`).

| Tool | Назначение |
|------|------------|
| `mb_get_document` | Текущая версия по `doc_key` |
| `mb_upsert_document` | Запись с версионированием |
| `mb_list_documents` | Список с фильтром `doc_type` |
| `mb_list_versions` | История версий документа |
| `mb_get_document_version` | Конкретная версия |
| `mb_get_state` / `mb_set_state` | JSON-состояние проекта |
| `mb_rules_list` | Атомарные правила (проект + глобальные / только глобальные) |
| `mb_rules_upsert` | Создать/обновить правило |
| `mb_rules_enable` / `mb_rules_disable` | Включить/выключить |
| `mb_rules_apply_preview` | Сборка «rules pack» по префиксам `scopes[]` |

**Preflight-пример:** `mb_rules_list` + `mb_get_document` для `activeContext` / `plan` перед `get_next_task`.

## RAG pipeline (рекомендуемый)

1. Источник текста (файл, ADR, решение).
2. `kb_chunk_markdown` → список чанков.
3. Внешне: embeddings для каждого `chunk.content`.
4. `kb_upsert_document_chunks`.
5. Запрос: эмбеддинг запроса + при необходимости ключевые слова → **`kb_hybrid_search`** (или только `kb_search_context`).

## Write-side (LangChain и др.)

- Правила dedup и порядок транзакции: **`docs/writer-contract.md`**.
- Примеры PlanBundle: `examples/planbundle_rn_auth.json`, `examples/planbundle_rn_chat_gemini.json`; сид: `scripts/seed_rn_auth.sql`.

## Ограничения

- В схеме **`vector(1536)`**; другая размерность — правка миграции `000006` и `KB_EMBEDDING_DIM`.
- Для **`order=created_at`** нужна миграция `000005`.

## Security: RLS (`000010`)

Включён **Row Level Security** с политиками по `project_id`:

- Если **`app.project_id`** не задан или пустой — строки **не скрываются** (удобно для локальной разработки).
- Для изоляции в shared-кластере на сессию:  
  `SET app.project_id = '<uuid проекта>'`  
  Видны строки этого проекта; глобальные правила `mb_rules` с `project_id IS NULL` остаются доступны по политике.

## Типичные проблемы

| Симптом | Причина |
|---------|---------|
| `Bind ... 5432 failed` | Порт занят локальным Postgres — используйте `POSTGRES_HOST_PORT=5433` (по умолчанию в Makefile). |
| `error: first .: file does not exist` (migrate) | Неверные имена файлов; в репозитории используются только `*.up.sql` / `*.down.sql`. |
| `extension "vector" is not available` | Образ без pgvector; в compose должен быть `pgvector/pgvector:pg16`. |
| `relation "projects" does not exist` после сида | Не накатили миграции (`make migrate-up` завершился с ошибкой). |
| `role "vault" does not exist` | `psql` подключился к **чужому** Postgres на том же порту — проверьте `POSTGRES_URL` и порт compose. |
