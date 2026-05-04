.PHONY: test tidy fmt up down logs ps

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps
