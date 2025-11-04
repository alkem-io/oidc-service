# GitHub Copilot Instructions

- You are assisting on the `alkem-io/oidc-service` Go backend.
- Follow Spec Kit workflows located under `.specify/`.
- Default toolchain: Go 1.25, chi router, zap logging, Prometheus metrics.
- Prefer `go test ./...` and `golangci-lint run` for validation steps.
- When editing tasks, update `specs/**/tasks.md` with `[X]` markers once complete.
- Respect constitution requirements for observability, security, and deterministic containers.

## Active Technologies
- Go 1.25 (from copilot instructions and constitution) + Chi router, Zap logging, Prometheus metrics, Hydra client, Kratos client (002-token-claims)
- N/A (reads from existing Kratos identity system) (002-token-claims)

## Recent Changes
- 002-token-claims: Added Go 1.25 (from copilot instructions and constitution) + Chi router, Zap logging, Prometheus metrics, Hydra client, Kratos client
