# Key Decisions – Agent ID Token Claim (Final)

## Decision 1: Claim Population Layer
- **Decision**: Populate `agent_id` within `internal/challenge` claim assembly before Hydra token finalization.
- **Rationale**: Keeps HTTP handlers thin, reuses existing Kratos→Alkemio resolution path, and ensures both ID/access tokens share a single source of truth.
- **Alternatives Considered**:
  - Add claim in handlers (`internal/server`): rejected because it would duplicate logic for login and consent flows.
  - Post-process Hydra tokens: rejected since Hydra signatures must be deterministic before response; mutation afterwards is not supported.

## Decision 2: UUID Validation Strategy
- **Decision**: Treat `agentId` as valid only when it matches canonical UUID format (lowercase hex, hyphenated) and fail issuance otherwise.
- **Rationale**: Mirrors existing `userId` validation, prevents inconsistent identifiers in downstream authorization systems, and surfaces configuration issues early.
- **Alternatives Considered**:
  - Accept any non-empty string: rejected because it could leak malformed IDs into tokens and break policy engines.
  - Normalize arbitrary casing: rejected to avoid masking upstream data quality problems.

## Decision 3: Test Coverage Scope
- **Decision**: Extend contract, integration, and unit tests to cover successful claim emission plus resolver not-found/invalid-data rejections.
- **Rationale**: Contract tests guarantee the public token schema, integration tests exercise resolver wiring, and unit tests keep mapper logic fast; together they satisfy constitution testing mandates.
- **Alternatives Considered**:
  - Unit tests only: rejected because contract behaviour (claims in real tokens) would remain unverified.
  - Contract tests only: rejected because internal validation logic would lack fast regression coverage.
