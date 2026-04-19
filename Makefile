GOOSE := $(shell go env GOPATH)/bin/goose
GOOSE_MIGRATIONS_DIR := cmd/migrate/migrations
GOOSE_SEEDS_DIR     := cmd/migrate/seeds
GOOSE_MOCK_DIR      := cmd/migrate/mocks
GOOSE_MIGRATION_TABLE := migration_tracking.goose_migrations
GOOSE_SEED_TABLE      := migration_tracking.seed_migrations
GOOSE_MOCK_TABLE      := migration_tracking.mock_migrations
GOOSE_DB_DSN := "host=$${DB_HOST:-localhost} port=$${DB_PORT:-5432} user=$${DB_USER:-postgres} password=$${DB_PASSWORD:-postgres} dbname=$${DB_NAME:-kbank_ecms} sslmode=$${DB_SSLMODE:-disable}"
POSTGRES_CONTAINER = $(shell docker compose ps -q postgres)
REDIS_CONTAINER = $(shell docker compose ps -q redis)

.PHONY: init build run dev-up dev-down migrate db-create-migration db-create-seed db-mock-create-sql db-mock-generate-decision-rule db-mock-generate-decision-rule-custom-out db-mock-data-sql-up db-mock-data-sql-down db-migration-status db-seed-status db-drop db-clear db-create db-reset test vet lint fmt clean install-hooks swagger swagger-server swagger-cms-delivery proto proto-install redis-set redis-seed-user-attrs

## Install protoc Go plugins
proto-install:
	@echo "Installing protoc-gen-lint..."
	go install github.com/yoheimuta/protolint/cmd/protolint@latest
	@echo "Installing protoc-gen-go..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@echo "Installing protoc-gen-go-grpc..."
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Protoc plugins installed."

## Generate Go code from protobuf definitions
proto:
	@echo "Generating protobuf code..."
	@mkdir -p internal/grpc/pb/cms_runtime/v1
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/cms_runtime/v1/runtime.proto
	@mv proto/cms_runtime/v1/runtime.pb.go internal/grpc/pb/cms_runtime/v1/
	@mv proto/cms_runtime/v1/runtime_grpc.pb.go internal/grpc/pb/cms_runtime/v1/
	@echo "Protobuf generation complete."

## Initialize workspace
init:
	@echo "Installing protoc plugins..."
	make proto-install
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

## Run the server locally
run:
	go run ./cmd/server/

## Run cms-delivery locally
run-cms-delivery:
	go run ./cmd/cms-delivery/

## Run cms-runtime locally
run-cms-runtime:
	go run ./cmd/cms-runtime/

## Generate Swagger documentation for all services
swagger: swagger-server swagger-cms-delivery

## Generate Swagger documentation for the server service
swagger-server:
	swag init -g cmd/server/main.go --output docs/swagger/server --packageName server --parseDependency --parseInternal

## Generate Swagger documentation for the cms-delivery service
swagger-cms-delivery:
	swag init -g cmd/cms-delivery/main.go --output docs/swagger/cmsdelivery --packageName cmsdelivery --parseDependency --parseInternal

## Build Docker Image
dev-build:
	docker build -t kbank-ems:latest .

## Start local dependencies (Postgres + Redis) via Docker Compose
dev-up:
	docker compose up -d --wait

## Stop local dependencies
dev-down:
	docker compose down

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

## Create a new goose mock file: make db-mock-create-sql name=<mock_name>
db-mock-create-sql:
	@if [ -z "$(name)" ]; then echo "Usage: make db-mock-create-sql name=<mock_name>"; exit 1; fi
	$(GOOSE) -dir $(GOOSE_MOCK_DIR) create $(name) sql

## Generate decision rule mock SQL using the Go generator.
## Usage:
##   make db-mock-generate-decision-rule
##   make db-mock-generate-decision-rule name=decision_rule_example_data count=100
db-mock-generate-decision-rule:
	go run ./scripts/mockdata -name=$${name:-decision_rule_example_data} -count=$${count:-50}

## Generate decision rule mock SQL to an explicit output path.
## Usage:
##   make db-mock-generate-decision-rule-custom-out out=cmd/migrate/mock/20260417000000_decision_rule_example_data.sql
##   make db-mock-generate-decision-rule-custom-out out=cmd/migrate/mock/20260417000000_decision_rule_example_data.sql count=20
db-mock-generate-decision-rule-custom-out:
	@if [ -z "$(out)" ]; then echo "Usage: make db-mock-generate-decision-rule-custom-out out=<output_sql_path> [count=<mock_count>]"; exit 1; fi
	go run ./scripts/mockdata -out=$(out) -count=$${count:-50}

## Run mock data sql using goose (local only)
db-mock-data-sql-up:
	$(GOOSE) -dir $(GOOSE_MOCK_DIR) -table $(GOOSE_MOCK_TABLE) postgres $(GOOSE_DB_DSN) up

db-mock-data-sql-down:
	$(GOOSE) -dir $(GOOSE_MOCK_DIR) -table $(GOOSE_MOCK_TABLE) postgres $(GOOSE_DB_DSN) down

## Show schema migration status (migration_tracking.goose_migrations)
db-migration-status:
	$(GOOSE) -dir $(GOOSE_MIGRATIONS_DIR) -table $(GOOSE_MIGRATION_TABLE) postgres $(GOOSE_DB_DSN) status

## Show seed migration status (migration_tracking.seed_migrations)
db-seed-status:
	$(GOOSE) -dir $(GOOSE_SEEDS_DIR) -table $(GOOSE_SEED_TABLE) postgres $(GOOSE_DB_DSN) status

## Drop the database
db-drop:
	@echo "Dropping database $${DB_NAME:-kbank_ecms}..."
	@docker exec $(POSTGRES_CONTAINER) psql -U $${DB_USER:-postgres} -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$${DB_NAME:-kbank_ecms}' AND pid <> pg_backend_pid();"
	@docker exec $(POSTGRES_CONTAINER) dropdb -U $${DB_USER:-postgres} --if-exists $${DB_NAME:-kbank_ecms}

## Create the database
db-create:
	@echo "Creating database $${DB_NAME:-kbank_ecms}..."
	@docker exec $(POSTGRES_CONTAINER) createdb -U $${DB_USER:-postgres} $${DB_NAME:-kbank_ecms}

## Clear all tables in the database (truncate rows, keep schema)
db-clear:
	@echo "Clearing all tables in $${DB_NAME:-kbank_ecms}..."
	@docker exec $(POSTGRES_CONTAINER) psql -U $${DB_USER:-postgres} -d $${DB_NAME:-kbank_ecms} -t -c "SELECT 'TRUNCATE ' || string_agg(quote_ident(tablename), ', ') || ' CASCADE;' FROM pg_tables WHERE schemaname = 'public' HAVING count(*) > 0;" | \
		xargs -I {} docker exec $(POSTGRES_CONTAINER) psql -U $${DB_USER:-postgres} -d $${DB_NAME:-kbank_ecms} -c "{}"

## Reset the database (drop → create → migrate)
db-reset: db-drop db-create migrate

## Set a key/value in the local Redis container: make redis-set key=<key> val=<value>
redis-set:
	@if [ -z "$(key)" ] || [ -z "$(val)" ]; then echo "Usage: make redis-set key=<key> val=<value>"; exit 1; fi
	@docker exec $(REDIS_CONTAINER) redis-cli SET "$(key)" "$(val)"

## Seed Redis with sample CIS user attribute data (10 test users covering all segments/regions/risk levels)
redis-seed-user-attrs:
	@docker exec -i $(REDIS_CONTAINER) sh < scripts/seed-redis-user-attrs.sh

## Run all tests
test:
	go test ./...

## Run go vet
vet:
	go vet ./...

## Run golangci-lint
lint:
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
	chmod +x .githooks/pre-commit
	chmod +x .githooks/commit-msg
	cp .githooks/pre-commit .git/hooks/pre-commit
	cp .githooks/commit-msg .git/hooks/commit-msg
	@echo "Git hooks installed successfully."

## Remove build artifacts
clean:
	rm -rf bin/
