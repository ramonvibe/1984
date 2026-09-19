.PHONY: dev test up down logs

dev:
	bash scripts/dev.sh

test:
	go test ./...

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f app
