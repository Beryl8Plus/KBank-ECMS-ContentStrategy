GOOSE := $(shell go env GOPATH)/bin/goose
GOOSE_MIGRATIONS_DIR := cmd/migrate/migrations
GOOSE_SEEDS_DIR     := cmd/migrate/seeds
GOOSE_MIGRATION_TABLE := migration_tracking.goose_migrations
GOOSE_SEED_TABLE      := migration_tracking.seed_migrations
GOOSE_DB_DSN := "host=$${DB_HOST:-localhost} port=$${DB_PORT:-5432} user=$${DB_USER:-postgres} password=$${DB_PASSWORD:-postgres} dbname=$${DB_NAME:-kbank_ecms} sslmode=$${DB_SSLMODE:-disable}"

.PHONY: init build run dev-up dev-down migrate db-create-migration db-create-seed db-migration-status db-seed-status db-drop db-clear db-create db-reset test vet lint fmt clean install-hooks swagger

## Initialize workspace
init:
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing swag..."
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Installing goose..."
	go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo "Installing git hooks..."
	make install-hooks
	@echo "Workspace initialization complete."

## Build the server binary
build: swagger
	go build -o bin/server ./cmd/server/

## Generate Swagger documentation
swagger:
	swag init -g cmd/server/main.go --output docs/swagger --parseDependency --parseInternal

## Start local dependencies (Postgres + Redis) via Docker Compose
dev-up:
	docker compose up -d --wait

## Stop local dependencies
dev-down:
	docker compose down

## Run the server locally
run:
	go run ./cmd/server/

## Run database migration
migrate:
	go run ./cmd/migrate/

## Create a new goose schema migration file: make db-create-migration name=<migration_name>
db-create-migration:
	@if [ -z "$(name)" ]; then echo "Usage: make db-create-migration name=<migration_name>"; exit 1; fi
	$(GOOSE) -dir $(GOOSE_MIGRATIONS_DIR) create $(name) sql

## Create a new goose seed file: make db-create-seed name=<seed_name>
db-create-seed:
	@if [ -z "$(name)" ]; then echo "Usage: make db-create-seed name=<seed_name>"; exit 1; fi
	$(GOOSE) -dir $(GOOSE_SEEDS_DIR) create $(name) sql

## Show schema migration status (migration_tracking.goose_migrations)
db-migration-status:
	$(GOOSE) -dir $(GOOSE_MIGRATIONS_DIR) -table $(GOOSE_MIGRATION_TABLE) postgres $(GOOSE_DB_DSN) status

## Show seed migration status (migration_tracking.seed_migrations)
db-seed-status:
	$(GOOSE) -dir $(GOOSE_SEEDS_DIR) -table $(GOOSE_SEED_TABLE) postgres $(GOOSE_DB_DSN) status

## Drop the database
db-drop:
	@echo "Dropping database $${DB_NAME:-kbank_ecms}..."
	@docker exec kbank-ecms-backend-postgres-1 psql -U $${DB_USER:-postgres} -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$${DB_NAME:-kbank_ecms}' AND pid <> pg_backend_pid();"
	@docker exec kbank-ecms-backend-postgres-1 dropdb -U $${DB_USER:-postgres} --if-exists $${DB_NAME:-kbank_ecms}

## Create the database
db-create:
	@echo "Creating database $${DB_NAME:-kbank_ecms}..."
	@docker exec kbank-ecms-backend-postgres-1 createdb -U $${DB_USER:-postgres} $${DB_NAME:-kbank_ecms}

## Clear all tables in the database (truncate rows, keep schema)
db-clear:
	@echo "Clearing all tables in $${DB_NAME:-kbank_ecms}..."
	@docker exec kbank-ecms-backend-postgres-1 psql -U $${DB_USER:-postgres} -d $${DB_NAME:-kbank_ecms} -t -c "SELECT 'TRUNCATE ' || string_agg(quote_ident(tablename), ', ') || ' CASCADE;' FROM pg_tables WHERE schemaname = 'public' HAVING count(*) > 0;" | \
		xargs -I {} docker exec kbank-ecms-backend-postgres-1 psql -U $${DB_USER:-postgres} -d $${DB_NAME:-kbank_ecms} -c "{}"

## Reset the database (drop → create → migrate)
db-reset: db-drop db-create migrate

## Run all tests
test:
	go test ./...

## Run go vet
vet:
	go vet ./...

## Run golangci-lint
lint: swagger
	golangci-lint run ./...

## Format GORM and JSON tags in entities
format-tags:
	go run scripts/format_tags.go

## Run all formatters
fmt:
	go fmt ./...
	make format-tags

## Install git hooks
install-hooks:
	chmod +x scripts/commit-lint.sh
	chmod +x scripts/conventional-pr.sh
	chmod +x .githooks/pre-commit
	chmod +x .githooks/commit-msg
	cp .githooks/pre-commit .git/hooks/pre-commit
	cp .githooks/commit-msg .git/hooks/commit-msg
	@echo "Git hooks installed successfully."

## Remove build artifacts
clean:
	rm -rf bin/
