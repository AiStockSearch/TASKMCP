# Writer Contract (LangChain → Vault)

Этот документ фиксирует **детерминированный write-side контракт** для LangChain‑оркестратора, который пишет требования/эпики/задачи в Vault (Postgres) так, чтобы операции были **повторяемыми (at‑least‑once safe)** и не создавали дублей.

Контекст слоёв 0–4 (интервью, декомпозиция, Vault, MCP, Cursor): [`architecture-five-layers.md`](architecture-five-layers.md).

## Source of truth

- **Брокер очереди** (например Redis) используется опционально: оркестратор [`orchestrator/`](../orchestrator/README.md) ставит задания `apply PlanBundle` в очередь; исполнитель применяет ту же транзакцию, что описана ниже.
- Vault (Postgres) — единственный source of truth для:
  - `projects`, `requirements`, `epics`, `tasks`, `task_files`
  - Memory Bank артефактов: `mb_documents`, `mb_document_versions`, `mb_state`, `mb_rules`
- Cursor читает/исполняет через MCP (Vault Bridge), но **не отвечает за создание** задач.

## Key constraints (dedup)

Миграция `migrations/000008_writer_constraints.up.sql` добавляет уникальности:

- `requirements`: UNIQUE `(project_id, title)`
- `tasks`: UNIQUE
  - `(project_id, requirement_id, title)` WHERE `requirement_id IS NOT NULL`
  - `(project_id, epic_id, title)` WHERE `epic_id IS NOT NULL`
- `task_files`: UNIQUE `(project_id, task_id, file_path)`
- `github_links`: UNIQUE `(project_id, entity_type, entity_id)`

Эти constraints обеспечивают безопасные retries при сетевых сбоях/повторах.

## Планируемый payload (PlanBundle)

LangChain planner должен выдавать JSON (пример: `examples/planbundle_rn_auth.json`) с ключами:
- `repo_key` (`owner/repo`)
- `requirement` (title/spec/status)
- `epic` (опционально)
- `tasks[]` (priority, link_to, task_files[])
- `memory_bank` (опционально: doc_writes[], state_json)

## Транзакция применения (рекомендуемый алгоритм)

Использовать одну транзакцию `READ COMMITTED`:

1) **Ensure project**\n
   - `INSERT INTO projects ... ON CONFLICT (repo_key) DO NOTHING`\n
   - `SELECT project_id`\n

2) **Epic (optional)**\n
   - если epic нужен: `INSERT INTO epics ... ON CONFLICT DO NOTHING` (если вы введёте dedup для epics)\n
   - получить `epic_id`\n

3) **Requirement**\n
   - `INSERT INTO requirements ... ON CONFLICT (project_id, title) DO UPDATE SET spec_json=EXCLUDED.spec_json, status=EXCLUDED.status`\n
   - получить `requirement_id`\n

4) **Tasks**\n
   - для каждой задачи:\n
     - `INSERT INTO tasks ... ON CONFLICT DO NOTHING` (dedup обеспечивают unique indexes)\n
     - (опционально) `UPDATE tasks SET description=?, priority=? WHERE ...` если хотите “upsert semantics”\n

5) **Task files**\n
   - для каждой `file_path`:\n
     - `INSERT INTO task_files ... ON CONFLICT DO NOTHING`\n

6) **Memory Bank (optional)**\n
   - документы: `mb_upsert_document` (через MCP) или прямой SQL в `mb_documents/mb_document_versions`\n
   - state: `mb_set_state` (через MCP) или прямой SQL\n

7) `COMMIT`

## Retry policy (at-least-once safe)

- Любая операция может быть повторена целиком.\n
- Стабильность обеспечивается:\n
  - `ON CONFLICT DO NOTHING/UPDATE`\n
  - unique indexes\n
- При ретраях **не должно** появляться дополнительных `requirements/tasks/task_files` дублей.

## Рекомендации (практика)

- Всегда использовать `project_id` scope, никогда не писать без него.\n
- Для задач держать `priority` плотной шкалой (10,20,30...) — удобно вставлять между.\n
- Для больших `spec_json` хранить `spec_hash` (опционально) и индексировать в KB.

