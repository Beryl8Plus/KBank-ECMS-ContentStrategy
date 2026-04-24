GOOSE := goose
GOOSE_MIGRATIONS_DIR := cmd/migrate/migrations
GOOSE_SEEDS_DIR     := cmd/migrate/seeds
GOOSE_MOCK_DIR      := cmd/migrate/mocks
GOOSE_MIGRATION_TABLE := migration_tracking.goose_migrations
GOOSE_SEED_TABLE      := migration_tracking.seed_migrations
GOOSE_MOCK_TABLE      := migration_tracking.mock_migrations
GOOSE_DB_DSN := "host=$${DB_HOST:-localhost} port=$${DB_PORT:-5432} user=$${DB_USER:-postgres} password=$${DB_PASSWORD:-postgres} dbname=$${DB_NAME:-kbank_ecms} sslmode=$${DB_SSLMODE:-disable}"
ATLAS_DB_URL := "postgres://$${DB_USER:-postgres}:$${DB_PASSWORD:-postgres}@$${DB_HOST:-localhost}:$${DB_PORT:-5432}/$${DB_NAME:-kbank_ecms}?sslmode=$${DB_SSLMODE:-disable}&search_path=public"
ATLAS_EMPTY_URL := "postgres://$${DB_USER:-postgres}:$${DB_PASSWORD:-postgres}@$${DB_HOST:-localhost}:$${DB_PORT:-5432}/$${DB_NAME:-kbank_ecms}?sslmode=$${DB_SSLMODE:-disable}&search_path=migration_tracking"
POSTGRES_CONTAINER = $(shell docker compose ps -q postgres)
REDIS_CONTAINER = $(shell docker compose ps -q redis)

.PHONY: init build run run-svc-contstrat-delivery run-svc-contstrat-runtime dev-build dev-up dev-down migrate db-create-migration db-create-seed db-mock-create-sql db-mock-generate-decision-rule db-mock-generate-decision-rule-custom-out db-mock-data-sql-up db-mock-data-sql-down db-migration-status db-seed-status db-drop db-clear db-create db-reset db-schema-inspect db-schema-sql test vet lint fmt format-tags clean install-hooks swagger swagger-svc-contstrat-backoffice swagger-svc-contstrat-delivery proto proto-install redis-set redis-seed-user-attrs wire-gen

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
	@echo "Installing wire..."
	go install github.com/google/wire/cmd/wire@latest
	@echo "Installing git hooks..."
	make install-hooks
	@echo "Workspace initialization complete."

## Build the server binary
build: swagger
	go build -o bin/svc-contstrat-backoffice ./cmd/svc-contstrat-backoffice/

## Generate wire dependencies
wire-gen:
	wire gen ./cmd/svc-contstrat-backoffice
	wire gen ./cmd/svc-contstrat-delivery

## Run the server locally
run:
	go run ./cmd/svc-contstrat-backoffice/

## Run svc-contstrat-delivery locally
run-svc-contstrat-delivery:
	go run ./cmd/svc-contstrat-delivery/

## Run svc-contstrat-runtime locally
run-svc-contstrat-runtime:
	go run ./cmd/svc-contstrat-runtime/

## Generate Swagger documentation for all services
swagger: swagger-svc-contstrat-backoffice swagger-svc-contstrat-delivery

## Generate Swagger documentation for the server service
swagger-svc-contstrat-backoffice:
	swag init -g cmd/svc-contstrat-backoffice/main.go --output docs/swagger/svc-contstrat-backoffice --packageName svc_contstrat_backoffice --parseDependency --parseInternal --exclude cmd/svc-contstrat-delivery,cmd/svc-contstrat-runtime

## Generate Swagger documentation for the svc-contstrat-delivery service
swagger-svc-contstrat-delivery:
	swag init -g cmd/svc-contstrat-delivery/main.go --output docs/swagger/svc-contstrat-delivery --packageName svc_contstrat_delivery --parseDependency --parseInternal --exclude cmd/svc-contstrat-backoffice,cmd/svc-contstrat-runtime

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

## Generate decision rule mock SQL + Redis seed script using the Go generator.
## The Redis seed script (scripts/seed-redis-user-attrs.sh) is always regenerated
## so its attribute UUIDs stay in sync with the SQL output.
## Usage:
##   make db-mock-generate-decision-rule
##   make db-mock-generate-decision-rule name=decision_rule_example_data count=100
db-mock-generate-decision-rule:
	go run ./scripts/mockdata -name=$${name:-decision_rule_example_data} -count=$${count:-50}

## Run mock data sql using goose (local only)
db-mock-data-sql-up:
	@docker exec $(POSTGRES_CONTAINER) psql -q -U $${DB_USER:-postgres} -d $${DB_NAME:-kbank_ecms} -c \
		"INSERT INTO $(GOOSE_MOCK_TABLE) (version_id, is_applied) SELECT 0, true WHERE NOT EXISTS (SELECT 1 FROM $(GOOSE_MOCK_TABLE) WHERE version_id = 0);"
	@output="$$( $(GOOSE) -dir $(GOOSE_MOCK_DIR) -table $(GOOSE_MOCK_TABLE) postgres $(GOOSE_DB_DSN) up 2>&1 )"; status=$$?; \
	printf '%s\n' "$$output"; \
	if [ $$status -ne 0 ] && ! printf '%s\n' "$$output" | grep -q "goose run: no next version found"; then \
		exit $$status; \
	fi

db-mock-data-sql-down:
	@docker exec $(POSTGRES_CONTAINER) psql -q -U $${DB_USER:-postgres} -d $${DB_NAME:-kbank_ecms} -c \
		"INSERT INTO $(GOOSE_MOCK_TABLE) (version_id, is_applied) SELECT 0, true WHERE NOT EXISTS (SELECT 1 FROM $(GOOSE_MOCK_TABLE) WHERE version_id = 0);"
	@output="$$( $(GOOSE) -dir $(GOOSE_MOCK_DIR) -table $(GOOSE_MOCK_TABLE) postgres $(GOOSE_DB_DSN) down 2>&1 )"; status=$$?; \
	printf '%s\n' "$$output"; \
	if [ $$status -ne 0 ] && ! printf '%s\n' "$$output" | grep -q "goose run: no next version found"; then \
		exit $$status; \
	fi

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

## Clear all tables in the database (truncate rows, keep schema).
## Also resets the mock migration tracking table so db-mock-data-sql-up can be re-run.
db-clear:
	@echo "Clearing all tables in $${DB_NAME:-kbank_ecms}..."
	@docker exec $(POSTGRES_CONTAINER) psql -U $${DB_USER:-postgres} -d $${DB_NAME:-kbank_ecms} -tAc \
		"SELECT 'TRUNCATE ' || string_agg(format('%I.%I', schemaname, tablename), ', ') || ' CASCADE;' FROM pg_tables WHERE schemaname = 'public' OR (schemaname = 'migration_tracking' AND tablename = 'mock_migrations') HAVING count(*) > 0;" | \
		docker exec -i $(POSTGRES_CONTAINER) psql -q -U $${DB_USER:-postgres} -d $${DB_NAME:-kbank_ecms}
	@docker exec $(POSTGRES_CONTAINER) psql -q -U $${DB_USER:-postgres} -d $${DB_NAME:-kbank_ecms} -c \
		"INSERT INTO $(GOOSE_MOCK_TABLE) (version_id, is_applied) VALUES (0, true);"
	@echo "Done. All rows cleared."

## Reset the database (drop → create → migrate)
db-reset: db-drop db-create migrate

## Inspect the current database schema and output in Atlas HCL format
db-schema-inspect:
	@echo "Inspecting database schema..."
	atlas schema inspect -u $(ATLAS_DB_URL)

## Generate SQL DDL from the current database state
db-schema-sql:
	@echo "Generating SQL schema from database..."
	@if [ ! -d "cmd/migrate/migrations-prod" ]; then mkdir -p cmd/migrate/migrations-prod; fi
	@atlas schema diff --from $(ATLAS_EMPTY_URL) --to $(ATLAS_DB_URL) > cmd/migrate/migrations-prod/database_schema.sql
	@echo "Schema saved to cmd/migrate/migrations-prod/database_schema.sql"

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
