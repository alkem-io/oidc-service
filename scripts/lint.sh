#!/usr/bin/env bash
set -euo pipefail

# Run linting and unit tests for the OIDC service.
GOLANGCI_LINT=${GOLANGCI_LINT:-golangci-lint}
"${GOLANGCI_LINT}" run ./...
go test ./...
