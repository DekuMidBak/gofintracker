BUF_CACHE_DIR := $(CURDIR)/.cache/buf
MIGRATE_IMAGE := migrate/migrate:v4.18.3
MIGRATE_NETWORK ?= gofintracker_default
MIGRATIONS_DIR := $(CURDIR)/migrations
MIGRATE_STEPS ?= 1

USERS_DATABASE_DSN ?= postgres://gofintracker:gofintracker@postgres:5432/users_db?sslmode=disable
TRANSACTIONS_DATABASE_DSN ?= postgres://gofintracker:gofintracker@postgres:5432/transactions_db?sslmode=disable
ANALYTICS_DATABASE_DSN ?= postgres://gofintracker:gofintracker@postgres:5432/analytics_db?sslmode=disable
USERS_TEST_DATABASE_DSN ?= postgres://gofintracker:gofintracker@localhost:5433/users_db?sslmode=disable

.PHONY: test test-integration test-integration-users tidy fmt proto proto-lint up down logs ps migrate-create migrate-up migrate-up-users migrate-up-transactions migrate-up-analytics migrate-down-users migrate-down-transactions migrate-down-analytics

test:
	go test ./...

test-integration: test-integration-users

test-integration-users:
	USERS_TEST_DATABASE_DSN="$(USERS_TEST_DATABASE_DSN)" go test ./internal/user/postgres -count=1

tidy:
	go mod tidy

fmt:
	go fmt ./...

proto:
	@if find proto -name '*.proto' -print -quit | grep -q .; then \
		BUF_CACHE_DIR=$(BUF_CACHE_DIR) buf generate; \
	else \
		echo "No proto files to generate yet"; \
	fi

proto-lint:
	@if find proto -name '*.proto' -print -quit | grep -q .; then \
		BUF_CACHE_DIR=$(BUF_CACHE_DIR) buf lint; \
	else \
		echo "No proto files to lint yet"; \
	fi

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps

migrate-create:
	@if [ -z "$(SERVICE)" ] || [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create SERVICE=user-service NAME=create_users"; \
		exit 1; \
	fi
	docker run --rm \
		-v "$(MIGRATIONS_DIR):/migrations" \
		$(MIGRATE_IMAGE) \
		create -ext sql -dir /migrations/$(SERVICE) -seq $(NAME)

migrate-up: migrate-up-users migrate-up-transactions migrate-up-analytics

migrate-up-users:
	docker run --rm \
		--network $(MIGRATE_NETWORK) \
		-v "$(MIGRATIONS_DIR):/migrations:ro" \
		$(MIGRATE_IMAGE) \
		-path=/migrations/user-service \
		-database="$(USERS_DATABASE_DSN)" \
		up

migrate-up-transactions:
	docker run --rm \
		--network $(MIGRATE_NETWORK) \
		-v "$(MIGRATIONS_DIR):/migrations:ro" \
		$(MIGRATE_IMAGE) \
		-path=/migrations/transaction-service \
		-database="$(TRANSACTIONS_DATABASE_DSN)" \
		up

migrate-up-analytics:
	docker run --rm \
		--network $(MIGRATE_NETWORK) \
		-v "$(MIGRATIONS_DIR):/migrations:ro" \
		$(MIGRATE_IMAGE) \
		-path=/migrations/analytics-service \
		-database="$(ANALYTICS_DATABASE_DSN)" \
		up

migrate-down-users:
	docker run --rm \
		--network $(MIGRATE_NETWORK) \
		-v "$(MIGRATIONS_DIR):/migrations:ro" \
		$(MIGRATE_IMAGE) \
		-path=/migrations/user-service \
		-database="$(USERS_DATABASE_DSN)" \
		down $(MIGRATE_STEPS)

migrate-down-transactions:
	docker run --rm \
		--network $(MIGRATE_NETWORK) \
		-v "$(MIGRATIONS_DIR):/migrations:ro" \
		$(MIGRATE_IMAGE) \
		-path=/migrations/transaction-service \
		-database="$(TRANSACTIONS_DATABASE_DSN)" \
		down $(MIGRATE_STEPS)

migrate-down-analytics:
	docker run --rm \
		--network $(MIGRATE_NETWORK) \
		-v "$(MIGRATIONS_DIR):/migrations:ro" \
		$(MIGRATE_IMAGE) \
		-path=/migrations/analytics-service \
		-database="$(ANALYTICS_DATABASE_DSN)" \
		down $(MIGRATE_STEPS)
