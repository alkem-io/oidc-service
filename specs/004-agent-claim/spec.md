# Agent ID Token Claim — Delivered Summary

**Status:** Complete (2025‑11‑21)  
**Branch:** `004-agent-claim`

## Outcome
- Both ID and access tokens now include an `agent_id` claim sourced from the Alkemio resolver response (`agentId`).
- Resolver failures, missing mappings, or malformed UUIDs block login/consent resolution with explicit `alkemio_*` challenge errors; no tokens are minted in those cases.
- Structured logs mask identifiers but record whether mappings resolved, failed, or were rejected for invalid data.

## Implementation Highlights
- Claim logic lives inside `internal/challenge` so handlers stay thin and Hydra responses remain deterministic.
- `identity_mapper` validates that both `userId` and `agentId` are canonical UUIDs before claims are attached.
- Contract tests (`test/contract/*claims_test.go`) assert the public token surface, while integration tests cover resolver success, not-found, invalid-agent, and maintenance scenarios.
- Docs (`quickstart.md`, `docs/operations.md`) explain how to verify `agent_id` manually and via `scripts/validate-token-claims.sh`.

## Validation
- `golangci-lint run ./...`
- `go test ./...`
- `scripts/lint.sh`
- `scripts/validate-token-claims.sh` (happy + unhappy paths exercised against local stack)

## Follow-ups
- None. Continue to monitor resolver health via existing logs and rerun the validation script when upgrading Hydra/Kratos.
