.PHONY: help migrate run-ingester run-processor run-api test docker-up docker-down

help:
	@echo "Available commands:"
	@echo "  make migrate         - Run database migrations"
	@echo "  make run-ingester    - Start ingester service"
	@echo "  make run-processor   - Start processor service"
	@echo "  make run-api         - Start API service"
	@echo "  make test            - Run tests"
	@echo "  make docker-up       - Start all services with Docker"
	@echo "  make docker-down     - Stop all Docker services"

migrate:
	@echo "Running migrations..."
	psql -h localhost -U indexer -d indexer -f database/migrations/001_initial_schema.sql

run-ingester:
	cd services/ingester && go run main.go

run-processor:
	cd services/processor && go run main.go

run-api:
	cd services/api && go run main.go

test:
	go test ./...

docker-up:
	docker-compose -f infrastructure/docker/docker-compose.yml up -d

docker-down:
	docker-compose -f infrastructure/docker/docker-compose.yml down
