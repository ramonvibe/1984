.PHONY: dev test

dev:
	bash scripts/dev.sh

test:
	go test ./...
