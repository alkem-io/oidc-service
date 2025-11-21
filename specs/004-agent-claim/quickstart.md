# Quickstart – Agent ID Token Claim

## Prerequisites
- Go 1.25 toolchain
- Access to running Kratos, Hydra, and Alkemio services (resolver endpoint enabled with `agentId`)
- `OIDC_SERVICE_CONFIG` updated to point at the resolver host

## Steps
1. **Run tests**
   ```bash
   go test ./...
   golangci-lint run
   ```
2. **Start dependencies** (can reuse existing docker-compose stack)
   ```bash
   docker compose -f deployments/docker-compose/compose.yaml up hydra kratos alkemio
   ```
3. **Launch oidc-service**
   ```bash
   go run ./cmd/server
   ```
4. **Trigger login flow**
   - Authenticate via existing test client.
   - Ensure the Kratos identity used has a resolver mapping with both `userId` and `agentId`.
5. **Inspect tokens (manual path)**
   - Capture the ID/access tokens returned to the client (any JWT debugger works offline).
   - Decode the payload section and confirm:
     - `agent_id` equals the resolver’s `agentId` for both ID and access tokens.
     - `alkemio_user_id`, `given_name`, `family_name`, `email_verified`, and `accepted_terms` keep their prior semantics.
   - If `agent_id` is missing, re-check resolver data or logs for validation errors.
6. **Run automated validation (optional but recommended)**
   ```bash
   export OIDC_SERVICE_URL=http://localhost:8080
   export HYDRA_PUBLIC_URL=http://localhost:4444
   export KRATOS_ADMIN_URL=http://localhost:4434
   ./scripts/validate-token-claims.sh
   ```
   - The script provisions temporary identities, walks the Hydra login + consent flow, and fails fast if `agent_id` or any legacy claims are absent/mismatched in issued tokens.
   - Review the per-scenario summary at the end; all scenarios must pass before shipping.

## Troubleshooting
- Missing `agent_id`: confirm resolver mapping exists, returns valid UUIDs, and the automated script above succeeds end-to-end.
- Issuance failure: check structured logs for resolver errors (timeouts, invalid data).
- Local development: use `test/integration` helpers to stub resolver responses if upstream systems unavailable.
