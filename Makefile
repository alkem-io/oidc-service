GO ?= go
BIN_DIR ?= bin
SERVICE_BIN := $(BIN_DIR)/oidc-service
DOCSTRINGCOV_BIN := $(BIN_DIR)/docstringcov
DOCSTRINGCOV_FLAGS ?=

.PHONY: build build-service build-docstringcov test lint docstring-coverage

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

# Run static analysis via golangci-lint.
lint:
	golangci-lint run

# Run the docstring coverage CLI and write docs/coverage.json.
docstring-coverage: build-docstringcov
	$(DOCSTRINGCOV_BIN) $(DOCSTRINGCOV_FLAGS)
