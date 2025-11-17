# Research: Alkemio User ID Token Claim

**Feature**: `003-token-claims-oidc`
**Date**: 2025-11-17

## Decisions

### Identity Resolution Client

- **Decision**: Reuse existing HTTP client patterns from `internal/kratos`/`internal/hydra` for calling the Alkemio server REST endpoint `POST /rest/internal/identity/resolve`, via a small dedicated helper or client wrapper.
- **Rationale**: Keeps external integration behavior (timeouts, error mapping, logging discipline) consistent with existing clients and aligns with the constitution's secure external integrations principle.
- **Alternatives considered**:
- Call the Alkemio server directly from the challenge service using `net/http` without a dedicated helper (rejected: would duplicate timeout and error handling logic).
- Introduce a generic external service client abstraction (rejected for now: overkill for a single endpoint; can be refactored later if more endpoints appear).

### Behavior for Missing Mapping

- **Decision**: Fail token issuance when the identity resolution endpoint returns 404 (no Kratos identity or no user mapping for the provided authentication ID).
- **Rationale**: Ensures that only users fully represented in the Alkemio platform receive tokens, avoiding ambiguous authorization or partially onboarded identities. The 404 outcome from `/rest/internal/identity/resolve` is treated as "no valid identity" for the purposes of token issuance.
- **Alternatives considered**:
- Issue tokens without the claim (rejected: downstream services might treat such users inconsistently and assumptions about the claims presence would break).
- Use a placeholder value (e.g., `"unknown"`) (rejected: would require all consumers to special-case the value and still leaves ambiguity).

### Behavior for Multiple Mappings

- **Decision**: Fail token issuance when multiple Alkemio user IDs are returned for a single Kratos ID.
- **Rationale**: Prevents ambiguous authorization decisions and forces underlying data inconsistencies to be resolved promptly.
- **Alternatives considered**:
- Deterministically pick one mapping (e.g., lowest ID) (rejected: could silently authorize as the wrong user).
- Omit the claim but issue the token (rejected: downstream services would not see the problem and might behave unexpectedly).

### Tokens Carrying the Claim

- **Decision**: Include the Alkemio user ID claim in both access tokens and ID tokens for the targeted flows, using the `userId` value returned by `/rest/internal/identity/resolve` as `alkemio_user_id`.
- **Rationale**: Ensures both backend services and any clients consuming ID tokens have access to the same canonical identifier, simplifying authorization decisions and matching the REST contract.
- **Alternatives considered**:
- Access tokens only (rejected: some clients may rely on ID tokens for user identity context).
- ID tokens only (rejected: backend services frequently rely on access token claims).

### Observability

- **Decision**: Emit structured logs and metrics for resolution attempts, categorized into success, not-found, and failure, including correlation identifiers but avoiding full IDs where not necessary.
- **Rationale**: Supports SC-003 without leaking sensitive identifiers and aligns with the observability principle.
- **Alternatives considered**:
- Log only failures (rejected: harder to understand baseline behavior and success rate).
- Include raw identifiers in logs (rejected: increases risk of sensitive data exposure).
