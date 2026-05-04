BUF_CACHE_DIR := $(CURDIR)/.cache/buf

.PHONY: test tidy fmt proto proto-lint up down logs ps

test:
	go test ./...

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
