BIN_DIR ?= bin
SERVICE_BIN := $(BIN_DIR)/oidc-service
DOCSTRINGCOV_BIN := $(BIN_DIR)/docstringcov
DOCSTRINGCOV_FLAGS ?=

# Go commands
GO?=go
GOTEST=$(GO) test
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOVET=$(GO) vet
GOFMT=$(GO) fmt
SQLC?=sqlc


.PHONY: build build-service build-docstringcov test lint fmt generate sqlc docstring-coverage

# Build all binaries.
build: build-service build-docstringcov

# Build the main OIDC service binary.
build-service:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(SERVICE_BIN) ./cmd/server

# Build the docstring coverage CLI binary.
build-docstringcov:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(DOCSTRINGCOV_BIN) ./cmd/docstringcov

# Run the full Go test suite.
test:
	$(GO) test ./...

# Lint the code
.PHONY: lint
lint:
	@echo "Linting..."
	$(GOVET) ./...
	# Check if golangci-lint is installed
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping advanced linting"; \
	fi

# Format the code
.PHONY: fmt
fmt:
	@echo "Formatting..."
	$(GOFMT) ./...

# Generate SQLC code from SQL queries.
sqlc:
	@echo "Generating SQLC code..."
	@if command -v $(SQLC) >/dev/null; then \
		$(SQLC) generate; \
	else \
		echo "sqlc not found. Install with: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest"; \
		exit 1; \
	fi

# Run all code generation (currently just SQLC).
generate: sqlc

# Run the docstring coverage CLI and write docs/coverage.json.
docstring-coverage: build-docstringcov
	$(DOCSTRINGCOV_BIN) $(DOCSTRINGCOV_FLAGS)
