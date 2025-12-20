.PHONY: help setup setup-full docker-up docker-down migrate migrate-down db-shell db-seed db-clear-seeds redis-cli kafka-topics clean logs status run-ingester run-processor run-api test generate-seeds explore-rpc

help:
	@echo "🚀 Blockchain Indexer - Development Commands"
	@echo ""
	@echo "⭐ Quick Start:"
	@echo "  make setup-full      - Complete setup with sample data (recommended)"
	@echo "  make setup           - Setup infrastructure + migrations only"
	@echo ""
	@echo "Setup & Infrastructure:"
	@echo "  make docker-up       - Start all infrastructure (postgres, redis, kafka)"
	@echo "  make docker-down     - Stop all infrastructure"
	@echo "  make logs            - View all container logs"
	@echo "  make status          - Check status of all services"
	@echo ""
	@echo "Database Migrations (golang-migrate):"
	@echo "  make migrate-up      - Apply all pending migrations"
	@echo "  make migrate-down    - Rollback last migration"
	@echo "  make migrate-status  - Show current migration version"
	@echo "  make migrate-create  - Create new migration file"
	@echo "  make migrate-force   - Force set migration version (recovery)"
	@echo "  make db-shell        - Open PostgreSQL shell"
	@echo "  make db-reset        - Drop all tables and re-migrate (⚠️  DELETES DATA)"
	@echo "  make db-seed         - Load sample data for development"
	@echo "  make db-clear-seeds  - Clear sample data (keeps real data)"
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
	@echo ""
	@echo "📚 Documentation:"
	@echo "  SANDBOX_SETUP.md     - Complete developer setup guide"
	@echo "  DATABASE_GUIDE.md    - Database management & migrations"
	@echo "  PROGRESS_TRACKING.md - Project status & roadmap"
	@echo ""
	@echo "📚 Documentation:"
	@echo "  docs/DEPLOYMENT.md   - Production deployment & Supabase setup"
	@echo "  docs/DEVELOPMENT_STATUS.md - Current progress & roadmap"

# Initial setup (infrastructure + migrations only)
setup: docker-up wait-for-services migrate-up
	@echo "✅ Setup complete! Infrastructure is ready."
	@echo ""
	@echo "Next steps:"
	@echo "  1. Load sample data (optional): make db-seed"
	@echo "  2. Start API: make run-api"
	@echo "  3. Start UI: cd web && npm install && npm run dev"
	@echo "  4. Access UI: http://localhost:5173"
	@echo ""
	@echo "Optional - Start ingester (requires RPC keys in .env):"
	@echo "  make run-ingester"
	@echo ""
	@echo "📚 Full guide: SANDBOX_SETUP.md"

# Complete setup with sample data (recommended for new developers)
setup-full: docker-up wait-for-services migrate-up db-seed
	@echo "✅ Setup complete! Infrastructure + sample data loaded."
	@echo ""
	@echo "🎉 Your sandbox is ready with sample blockchain data!"
	@echo ""
	@echo "Start developing:"
	@echo "  Terminal 1: make run-api"
	@echo "  Terminal 2: cd web && npm install && npm run dev"
	@echo ""
	@echo "Then visit: http://localhost:5173"
	@echo ""
	@echo "📚 See SANDBOX_SETUP.md for full guide"


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

# Database commands (using golang-migrate)
MIGRATION_PATH=database/migrations
DATABASE_URL ?= postgresql://indexer:password@localhost:5432/indexer?sslmode=disable

check-migrate:
	@which migrate > /dev/null || (echo "❌ golang-migrate not found!" && echo "" && echo "Install it:" && echo "  macOS: brew install golang-migrate" && echo "  Linux: curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz && sudo mv migrate /usr/local/bin/" && echo "" && echo "See SANDBOX_SETUP.md for details" && exit 1)

migrate-up: check-migrate
	@echo "📈 Applying migrations..."
	@migrate -path $(MIGRATION_PATH) -database "$(DATABASE_URL)" up
	@echo "✅ Migrations complete"

migrate-down:
	@echo "📉 Rolling back last migration..."
	@migrate -path $(MIGRATION_PATH) -database "$(DATABASE_URL)" down 1
	@echo "✅ Rollback complete"

migrate-status:
	@echo "📊 Migration status:"
	@migrate -path $(MIGRATION_PATH) -database "$(DATABASE_URL)" version

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir $(MIGRATION_PATH) -seq $$name
	@echo "✅ Created migration files"

migrate-force:
	@read -p "Force version to: " version; \
	migrate -path $(MIGRATION_PATH) -database "$(DATABASE_URL)" force $$version

# Legacy commands (kept for backwards compatibility)
migrate: migrate-up

db-shell:
	@echo "🐘 Opening PostgreSQL shell..."
	@docker exec -it indexer-postgres psql -U indexer -d indexer

db-reset:
	@echo "⚠️  This will DELETE ALL DATA. Are you sure? [y/N] " && read ans && [ $${ans:-N} = y ]
	@migrate -path $(MIGRATION_PATH) -database "$(DATABASE_URL)" drop -f
	@echo "✅ Database dropped"
	@make migrate-up

db-seed:
	@echo "🌱 Seeding database with sample data..."
	@for seed in database/seeds/*.sql; do \
		echo "  Loading $$(basename $$seed)..."; \
		docker exec -i indexer-postgres psql -U indexer -d indexer < $$seed || exit 1; \
	done
	@echo "✅ Seeding complete"

db-clear-seeds:
	@echo "🧹 Clearing seed data (blocks < 1000)..."
	@docker exec indexer-postgres psql -U indexer -d indexer -c "DELETE FROM logs WHERE tx_hash IN (SELECT tx_hash FROM transactions WHERE block_number < 1000);"
	@docker exec indexer-postgres psql -U indexer -d indexer -c "DELETE FROM transactions WHERE block_number < 1000;"
	@docker exec indexer-postgres psql -U indexer -d indexer -c "DELETE FROM blocks WHERE block_number < 1000;"
	@echo "✅ Seed data cleared"

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

# Generate seed data from real blockchain
generate-seeds:
	@if [ -z "$$ETH_RPC_URL" ]; then \
		echo "Using free public RPC (slower)..."; \
		export ETH_RPC_URL="https://eth.llamarpc.com"; \
		cd scripts && go mod download && ETH_RPC_URL="https://eth.llamarpc.com" go run explore_rpc.go --generate-seeds; \
	else \
		cd scripts && go mod download && go run explore_rpc.go --generate-seeds; \
	fi
	@echo ""
	@echo "✅ Seed file generated!"
	@echo "📁 Location: database/seeds/001_sample_blocks.sql"
	@echo "🚀 Run: make db-seed"

# Cleanup
clean:
	@echo "🧹 Cleaning up..."
	docker compose -f infrastructure/docker/docker-compose.yml down -v
	@echo "✅ Cleanup complete"
