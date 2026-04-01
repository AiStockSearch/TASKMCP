# Vault orchestrator (FastAPI + Redis + arq)

Внешний write-side для [writer-contract.md](../docs/writer-contract.md): вебхук принимает **PlanBundle** (как `examples/planbundle_rn_auth.json`), кладёт задание в **Redis**, воркер **arq** применяет изменения в Postgres Vault в одной транзакции.

## Переменные окружения

| Переменная | Описание |
|------------|----------|
| `DATABASE_URL` | Postgres Vault (как у MCP) |
| `REDIS_URL` | Брокер, например `redis://127.0.0.1:6379/0` |
| `WEBHOOK_SECRET` | Общий секрет; заголовок `Authorization: Bearer <secret>` |
| `API_HOST` | По умолчанию `0.0.0.0` |
| `API_PORT` | По умолчанию `8080` |

## Локальный запуск

```bash
cd orchestrator
python -m venv .venv && source .venv/bin/activate
pip install -e .
# Терминал 1: Redis и Postgres уже подняты (make up)
arq vault_orchestrator.jobs.WorkerSettings
# Терминал 2:
uvicorn vault_orchestrator.main:app --host 0.0.0.0 --port 8080
```

Пример запроса:

```bash
curl -sS -X POST http://127.0.0.1:8080/hooks/apply-plan-bundle \
  -H "Authorization: Bearer $WEBHOOK_SECRET" \
  -H "Content-Type: application/json" \
  -d @../examples/planbundle_rn_auth.json
```

Ответ: `202` и `{"job_id":"...", "status":"queued"}`.

## Docker

Из корня репозитория:

```bash
export WEBHOOK_SECRET='your-secret'
docker compose --profile orchestrator up -d --build
```

Сервисы: `orchestrator-api` (:8080), `orchestrator-worker`, `redis`, `postgres`.
