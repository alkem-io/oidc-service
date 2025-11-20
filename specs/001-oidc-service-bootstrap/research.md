# Phase 0 Research: OIDC Service Bootstrap

## Problem Statement

The legacy NestJS server bundled OIDC controllers that tightly coupled Hydra, Kratos, and Alkemio platform concerns. We need a focused Go service that orchestrates Hydra login/consent flows, enforces maintenance controls, and exposes health endpoints while remaining aligned with the platform's spec-driven workflow.

## Existing Landscape

- Hydra provides Admin APIs for login and consent challenge retrieval/acceptance.
- Kratos exposes REST endpoints for session resolution using HTTP cookies.
- The Alkemio platform routes `/oidc/*` traffic (with legacy support for `/api/public/rest/oidc/*`) through Traefik and Oathkeeper to this service.
- Contract tests already exist for login, consent, and health flows in `test/contract` and serve as executable specification.

## Key Questions

1. What response schema do downstream consumers expect for login and consent endpoints? Answered by `contracts/openapi.yaml` – JSON payload with `redirect_to`, `remember`, and `remember_for` fields.
2. How do we distinguish maintenance mode failures from Hydra/Kratos errors? Solution: dedicated maintenance middleware emitting `MAINTENANCE_MODE_ENABLED` error codes.
3. What configuration surface is mandatory at bootstrap? Hydra admin URL, Kratos public URL, session cookie name, maintenance shared secret, timeout budgets.
4. How do we observe production behaviour without Prometheus? Answer: rely on structured Zap logs with correlation IDs and log shipping via the platform's existing stack (decision ratified Nov 2025).

## Research Findings

- Hydra Go client must be wrapped to inject timeouts and structured error translation.
- Kratos session lookup only needs the `/sessions/whoami` endpoint; no direct database writes required.
- Maintenance toggles persist in-memory with optional Redis backing not required for MVP; process-level state plus audit logging sufficed.
- Health readiness should include checks for Hydra, Kratos, database connectivity (via Kratos session repository), and maintenance flag state.

## Open Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Hydra latency spikes | Login flow degradation | Add timeout + retry-with-backoff wrapper and instrument structured logs for latency diagnostics |
| Kratos cookie drift | Users locked out | Make session cookie name configurable and add unit tests covering overrides |
| Maintenance toggle loss on pod restart | Unexpected traffic acceptance | Integrate optional persistent backing store in future; for now document restart procedure and include in runbook |
| Spec drift between OpenAPI and handlers | Contract regressions | Enforce contract tests in CI and include spec updates in review checklist |

## Decision Log (Phase 0)

- ✅ Use chi router for lightweight HTTP handling.
- ✅ Store configuration defaults in `configs/.env.example` and load via `envconfig`.
- ✅ Keep maintenance toggle in-memory for MVP with audit logs to stdout.
- ✅ Observability: rely on structured Zap logs; Prometheus metrics were removed in Nov 2025 to reduce operational surface area.
