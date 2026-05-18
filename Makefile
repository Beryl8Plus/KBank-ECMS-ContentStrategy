.PHONY: init build run dev-build dev-up dev-down test vet lint install-golangci-lint fmt format-tags clean install-hooks swagger swagger-format swagger-server tidy update-pkg


## Install golangci-lint v2 (cross-platform: macOS/Linux via install script, Windows via winget)
install-golangci-lint:
ifeq ($(OS),Windows_NT)
	@echo "Installing golangci-lint v2 for Windows..."
	winget install golangci.golangci-lint
else
	@echo "Installing golangci-lint v2..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(shell go env GOPATH)/bin
endif

## Initialize workspace
init:
	@echo "Installing golangci-lint..."
	$(MAKE) install-golangci-lint
	@echo "Installing swag..."
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Installing gci..."
	go install github.com/daixiang0/gci@latest
	@echo "Installing git hooks..."
	make install-hooks
	@echo "Running go mod tidy..."
	go mod tidy
	@echo "Workspace initialization complete."

tidy:
	go mod tidy
	@echo "go mod tidy completed."

## Update all dependencies to their latest minor/patch versions
update-pkg:
	go get -u ./...
	go mod tidy
	@echo "Packages updated successfully."

## Build the server binary
build: swagger
	go build -o bin/server ./cmd/server/


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
