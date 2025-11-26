.PHONY: help setup docker-up docker-down migrate migrate-down db-shell redis-cli kafka-topics clean logs status run-ingester run-processor run-api test

help:
	@echo "🚀 Blockchain Indexer - Development Commands"
	@echo ""
	@echo "Setup & Infrastructure:"
	@echo "  make setup           - Complete initial setup (docker + migrations)"
	@echo "  make docker-up       - Start all infrastructure (postgres, redis, kafka)"
	@echo "  make docker-down     - Stop all infrastructure"
	@echo "  make logs            - View all container logs"
	@echo "  make status          - Check status of all services"
	@echo ""
	@echo "Database:"
	@echo "  make migrate         - Run database migrations"
	@echo "  make migrate-down    - Rollback last migration"
	@echo "  make db-shell        - Open PostgreSQL shell"
	@echo "  make db-reset        - Reset database (WARNING: deletes all data)"
	@echo ""
	@echo "Development:"
	@echo "  make run-ingester    - Start ingester service (stops existing first)"
	@echo "  make run-processor   - Start processor service (stops existing first)"
	@echo "  make run-api         - Start API service (stops existing first)"
	@echo "  make stop-services   - Stop all running services"
	@echo "  make test            - Run all tests"
	@echo "  make explore-rpc     - Explore RPC data and validate schemas"
	@echo ""
	@echo "Utilities:"
	@echo "  make redis-cli       - Open Redis CLI"
	@echo "  make kafka-topics    - List Kafka topics"
	@echo "  make clean           - Clean up all data and stop services"

# Initial setup
setup: docker-up wait-for-services migrate
	@echo "✅ Setup complete! Infrastructure is ready."
	@echo ""
	@echo "Quick start:"
	@echo "  1. Update .env with your RPC API keys"
	@echo "  2. make run-ingester"
	@echo ""
	@echo "Web UIs available at:"
	@echo "  - Kafka UI: http://localhost:8080"
	@echo "  - pgAdmin: http://localhost:5050"

# Docker commands
docker-up:
	@echo "🐳 Starting infrastructure..."
	docker compose -f infrastructure/docker/docker-compose.yml up -d
	@echo "✅ Containers started"

docker-down:
	@echo "🛑 Stopping infrastructure..."
	docker compose -f infrastructure/docker/docker-compose.yml down

logs:
	docker compose -f infrastructure/docker/docker-compose.yml logs -f

status:
	@echo "📊 Service Status:"
	@docker ps --filter "name=indexer-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
	@echo ""
	@echo "Go Services:"
	@ps aux | grep -E "services/(ingester|api|processor)" | grep -v grep | awk '{print $$2, $$11, $$12, $$13}' || echo "  No Go services running"

wait-for-services:
	@echo "⏳ Waiting for services to be ready..."
	@sleep 5
	@docker exec indexer-postgres pg_isready -U indexer > /dev/null 2>&1 || (echo "❌ PostgreSQL not ready" && exit 1)
	@docker exec indexer-redis redis-cli ping > /dev/null 2>&1 || (echo "❌ Redis not ready" && exit 1)
	@echo "✅ All services ready"

# Database commands
migrate:
	@echo "🔄 Running database migrations..."
	@PGPASSWORD=password psql -h localhost -U indexer -d indexer -f database/migrations/001_initial_schema.sql
	@echo "✅ Migrations complete"

migrate-down:
	@echo "⏮️  Rolling back migrations..."
	@echo "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" | PGPASSWORD=password psql -h localhost -U indexer -d indexer
	@echo "✅ Rollback complete"

db-shell:
	@echo "🐘 Opening PostgreSQL shell..."
	@PGPASSWORD=password psql -h localhost -U indexer -d indexer

db-reset: migrate-down migrate
	@echo "✅ Database reset complete"

# Utility commands
redis-cli:
	@docker exec -it indexer-redis redis-cli

kafka-topics:
	@docker exec indexer-kafka rpk topic list

kafka-create-topics:
	@echo "Creating Kafka topics..."
	@docker exec indexer-kafka rpk topic create raw.blocks -p 3 -r 1
	@docker exec indexer-kafka rpk topic create parsed.events -p 3 -r 1
	@docker exec indexer-kafka rpk topic create system.reorg -p 1 -r 1
	@echo "✅ Topics created"

# Service commands
stop-services:
	@echo "🛑 Stopping all services..."
	@pkill -f "go run main.go" || true
	@pkill -f "services/ingester" || true
	@pkill -f "services/api" || true
	@pkill -f "services/processor" || true
	@echo "✅ All services stopped"

run-ingester:
	@echo "🔍 Checking for existing ingester processes..."
	@pkill -f "services/ingester.*go run" 2>/dev/null || true
	@sleep 1
	@if [ ! -d "services/ingester" ]; then echo "❌ Ingester service not yet created. Run setup first."; exit 1; fi
	@if [ -f .env ]; then export $$(cat .env | grep -v '^#' | xargs) && cd services/ingester && go run main.go; else echo "⚠️  No .env file found. Copy .env.example to .env and configure your RPC URLs."; exit 1; fi

run-processor:
	@pkill -f "services/processor.*go run" 2>/dev/null || true
	@sleep 1
	@if [ ! -d "services/processor" ]; then echo "❌ Processor service not yet created. Run setup first."; exit 1; fi
	cd services/processor && go run main.go

run-api:
	@pkill -f "services/api.*go run" 2>/dev/null || true
	@sleep 1
	@if [ ! -d "services/api" ]; then echo "❌ API service not yet created. Run setup first."; exit 1; fi
	cd services/api && go run main.go

test:
	go test ./...

# RPC Exploration
explore-rpc:
	@if [ -z "$$ETH_RPC_URL" ]; then \
		echo "❌ ETH_RPC_URL not set"; \
		echo ""; \
		echo "Get a free API key:"; \
		echo "  - Alchemy: https://www.alchemy.com/"; \
		echo "  - Infura: https://infura.io/"; \
		echo ""; \
		echo "Then run:"; \
		echo '  export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY"'; \
		echo "  make explore-rpc"; \
		exit 1; \
	fi
	@echo "🔍 Exploring RPC data..."
	cd scripts && go mod download && go run explore_rpc.go

# Cleanup
clean:
	@echo "🧹 Cleaning up..."
	docker compose -f infrastructure/docker/docker-compose.yml down -v
	@echo "✅ Cleanup complete"
