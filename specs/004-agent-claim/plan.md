# Implementation Retrospective: Agent ID Token Claim

**Status:** Complete (2025‑11‑21)  
**Spec:** [spec.md](./spec.md)

## Scope Recap
- Surface `agent_id` alongside `alkemio_user_id` in both ID and access tokens by plumbing the resolver output through `internal/challenge`.
- Enforce canonical UUID validation for `agentId`; failed validations abort login/consent with explicit challenge errors.
- Backfill contract/integration/unit tests plus documentation so the new claim is observable and supportable.

## Delivery Notes
- Code touches stayed inside existing domain packages (`internal/challenge`, `internal/server`) and their test suites; no new configuration or services were introduced.
- Structured logging now records resolver success/failure with masked IDs, matching constitution observability requirements.
- Quickstart + operations docs describe how to verify `agent_id` manually or via `scripts/validate-token-claims.sh`.

## Verification History
- `golangci-lint run ./...`
- `go test ./...`
- `scripts/lint.sh`
- `scripts/validate-token-claims.sh` (happy/unhappy paths)

## Open Items
- None; future teams can rerun the validation script after dependency upgrades.
