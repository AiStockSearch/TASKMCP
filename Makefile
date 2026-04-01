.PHONY: up down logs migrate-up migrate-down seed-rn-auth psql-url

POSTGRES_HOST_PORT ?= 5433
POSTGRES_URL ?= postgres://vault:vault@127.0.0.1:$(POSTGRES_HOST_PORT)/vault?sslmode=disable

up:
	POSTGRES_HOST_PORT=$(POSTGRES_HOST_PORT) docker compose up -d postgres

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

