# GitHub Copilot Instructions

- You are assisting on the `alkem-io/oidc-service` Go backend.
- Follow Spec Kit workflows located under `.specify/`.
- Default toolchain: Go 1.25, chi router, zap logging.
- Prefer `go test ./...` and `golangci-lint run` for validation steps.
- When editing tasks, update `specs/**/tasks.md` with `[X]` markers once complete.
- Respect constitution requirements for observability, security, and deterministic containers.
- Feature numbering is global: every new branch, Spec Kit directory, and checklist must use the next shared integer (e.g., `004-agent-claim`) rather than restarting counts per contributor.

## Active Technologies
- Go 1.25 (from copilot instructions and constitution) + Chi router, Zap logging, Hydra client, Kratos client (002-token-claims)
- N/A (reads from existing Kratos identity system) (002-token-claims)
- Go 1.25 + internal Hydra/Kratos clients, internal HTTP server/middleware, Zap logging (003-token-claims-oidc)
- N/A (feature reads from existing Kratos and Alkemio server; no new storage) (003-token-claims-oidc)
- Go 1.25 (per constitution/toolchain) + Chi router, Zap logging, internal Hydra/Kratos clients, identity resolver REST dependency (004-agent-claim)
- N/A (stateless token orchestration only) (004-agent-claim)
- Go 1.25 (per Spec Kit toolchain) + Internal CLI `cmd/docstringcov` built on Go stdlib `go/ast` + `go/packages`, invoked via `make docstring-coverage` and GitHub Actions job; `golangci-lint` remains for complementary lint rules (005-docstring-coverage)
- Git-tracked JSON artifacts in `docs/coverage/coverage-latest.json` plus rotating history under `docs/coverage/history/` (retain last 20 runs) (005-docstring-coverage)

## Recent Changes
- 002-token-claims: Added Go 1.25 (from copilot instructions and constitution) + Chi router, Zap logging, Hydra client, Kratos client
