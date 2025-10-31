#!/usr/bin/env bash
set -euo pipefail

# Placeholder for OpenAPI generation and validation logic.
# Update this script when code generation tooling is introduced.
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
OPENAPI_FILE="${ROOT_DIR}/contracts/openapi.yaml"

if [ ! -f "${OPENAPI_FILE}" ]; then
  echo "OpenAPI file not found at ${OPENAPI_FILE}" >&2
  exit 1
fi

echo "Validate OpenAPI specification at ${OPENAPI_FILE}".
