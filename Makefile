.PHONY: up down logs migrate-up migrate-down seed-rn-auth psql-url orchestrator-up apply-plan-bundle orch-webhook-secret

POSTGRES_HOST_PORT ?= 5433
# Хостовый порт Redis (по умолчанию 6380, чтобы не конфликтовать с локальным Redis на 6379).
REDIS_HOST_PORT ?= 6380
POSTGRES_URL ?= postgres://vault:vault@127.0.0.1:$(POSTGRES_HOST_PORT)/vault?sslmode=disable

up:
	POSTGRES_HOST_PORT=$(POSTGRES_HOST_PORT) REDIS_HOST_PORT=$(REDIS_HOST_PORT) docker compose up -d postgres redis

down:
	docker compose down -v

logs:
	docker compose logs -f --tail=200

migrate-up:
	docker compose run --rm migrate

migrate-down:
	docker run --rm -v "$(PWD)/migrations:/migrations" migrate/migrate:v4.17.1 \
		-path /migrations -database "$(POSTGRES_URL)" down 1

seed-rn-auth:
	psql "$(POSTGRES_URL)" -v repo_key='acme/mobile-app' -f scripts/seed_rn_auth.sql

psql-url:
	@echo "$(POSTGRES_URL)"

# Поднять FastAPI + arq worker (те же POSTGRES_HOST_PORT / REDIS_HOST_PORT, что и у `make up`)
# Сбрасываем WEBHOOK_SECRET в окружении подпроцесса: иначе случайный export в shell перетирает значение из `.env`.
orchestrator-up:
	POSTGRES_HOST_PORT=$(POSTGRES_HOST_PORT) REDIS_HOST_PORT=$(REDIS_HOST_PORT) \
		env -u WEBHOOK_SECRET docker compose --profile orchestrator up -d --build orchestrator-api orchestrator-worker

# Значение из корневого `.env`, иначе dev-change-me (как у docker compose для оркестратора).
# Используем `:=`, не `?=`: иначе случайный `export WEBHOOK_SECRET=…` в shell перебивает .env (в т.ч. плейсхолдер вроде «ваш-секрет»).
# Явная передача всё ещё работает: `make apply-plan-bundle WEBHOOK_SECRET='…'` (приоритет над Makefile).
WEBHOOK_SECRET := $(shell val=$$(grep -E '^WEBHOOK_SECRET=' .env 2>/dev/null | sed 's/^WEBHOOK_SECRET=//' | tr -d '\r'); echo "$${val:-dev-change-me}")

# Отправить PlanBundle в оркестратор (запущенный orchestrator-api, см. README)
PLAN_BUNDLE_FILE ?= examples/planbundle_rn_auth.json
ORCHESTRATOR_URL ?= http://127.0.0.1:8080

orch-webhook-secret:
	@docker compose exec -T orchestrator-api printenv WEBHOOK_SECRET 2>/dev/null || (echo 'orchestrator-api не запущен' && exit 1)

apply-plan-bundle:
	@test -f "$(PLAN_BUNDLE_FILE)" || (echo "File not found: $(PLAN_BUNDLE_FILE) — создайте файл или укажите пример: PLAN_BUNDLE_FILE=examples/planbundle_rn_auth.json" && exit 1)
	curl -sS -w '\nHTTP %{http_code}\n' -X POST "$(ORCHESTRATOR_URL)/hooks/apply-plan-bundle" \
		-H "Authorization: Bearer $(WEBHOOK_SECRET)" \
		-H "Content-Type: application/json" \
		-d @"$(PLAN_BUNDLE_FILE)"

