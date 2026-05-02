.PHONY: init build run dev-build dev-up dev-down test vet lint fmt format-tags clean install-hooks swagger swagger-format swagger-server wire-gen

## Initialize workspace
init:
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing swag..."
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Installing wire..."
	go install github.com/google/wire/cmd/wire@latest
	@echo "Installing gci..."
	go install github.com/daixiang0/gci@latest
	@echo "Installing git hooks..."
	make install-hooks
	@echo "Workspace initialization complete."
	make swagger
	@echo "Running go mod tidy..."
	go mod tidy
	@echo "Workspace is ready for development."

tidy:
	go mod tidy
	@echo "go mod tidy completed."

## Build the server binary
build: swagger
	go build -o bin/server ./cmd/server/

## Generate wire dependencies
wire-gen:
	wire gen ./cmd/server

## Run the server locally
run:
	go run ./cmd/server/

## Generate Swagger documentation for all services
swagger: swagger-server
	@echo "Swagger documentation generated for all services."
	$(MAKE) swagger-format
	@echo "Swagger documentation formatted."

## Generate Swagger documentation for the server service
swagger-server:
	swag init -g cmd/server/main.go --output cmd/server/docs --packageName server --parseDependency --parseInternal

## Swagger formatting for all generated docs
swagger-format:
	swag fmt

## Build Docker Image
dev-build:
	docker build -t kbank-ecms-content-strategy:latest .

## Start local dependencies (Postgres + Redis) via Docker Compose
dev-up:
	docker compose up -d --wait

## Stop local dependencies
dev-down:
	docker compose down

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
	gci write --skip-generated -s standard -s default -s "prefix(kbank-ecms)" .
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
