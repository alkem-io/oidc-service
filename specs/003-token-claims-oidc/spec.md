# Feature Specification: Alkemio User ID Token Claim

**Feature Branch**: `003-token-claims-oidc`  
**Created**: 2025-11-17  
**Status**: Draft  
**Input**: User description: "I need to add to the token claims in oidc-service yet another claim, which  Alkemio userID, which can be resolved from Kratos ID by using REST EP /rest/internal/identity/resolve in the alkemio server."

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.
  
  Assign priorities (P1, P2, P3, etc.) to each story, where P1 is the most critical.
  Think of each story as a standalone slice of functionality that can be:
  - Developed independently
  - Tested independently
  - Deployed independently
  - Demonstrated to users independently
-->

### User Story 1 - Include Alkemio user ID in tokens (Priority: P1)

When an Alkemio user signs in via identity provider and receives tokens issued by oidc-service, the tokens must include an additional claim containing their Alkemio user ID, resolved from their Kratos authentication ID using the Alkemio server.

**Why this priority**: Downstream Alkemio services depend on the Alkemio user ID to authorize actions and correlate activity; exposing it directly in the token avoids repeated identity lookups on every request.

**Independent Test**: This can be fully tested by authenticating a user whose Kratos identity is linked to an Alkemio user and verifying that the returned tokens contain the correct Alkemio user ID claim for successful resolutions and that token issuance fails when resolution does not succeed.

**Acceptance Scenarios**:

1. **Given** a user with a valid Kratos session and a corresponding Alkemio user ID resolvable via `POST /rest/internal/identity/resolve` (with their Kratos authentication ID, both represented as UUIDs), **When** they obtain access and ID tokens from oidc-service, **Then** both tokens include a claim with their Alkemio user ID matching the `userId` value from the Alkemio server response (the claim is never scoped by client or flow—if resolution succeeds, it must be emitted).
2. **Given** a user with a valid Kratos session but no Alkemio user ID mapping (the identity resolution endpoint returns 404), **When** they request tokens from oidc-service, **Then** token issuance fails and the user cannot receive tokens until a valid Alkemio user mapping exists.

---

### User Story 2 - Handle identity resolution failures gracefully (Priority: P2)

When oidc-service cannot resolve an Alkemio user ID from the Kratos authentication ID (for example, due to network errors, timeouts, or server-side failures), it must fail token issuance in a predictable way that preserves security and gives operators enough information to diagnose issues without exposing sensitive details to end users.

**Why this priority**: The new dependency on the Alkemio server identity resolution endpoint introduces a potential point of failure that must not degrade authentication reliability or leak internal error details to clients.

**Independent Test**: This can be tested by simulating failures in the `/rest/internal/identity/resolve` endpoint (e.g., 5xx responses, timeouts) and verifying that token issuance fails in a controlled way while recording appropriate operational signals.

**Acceptance Scenarios**:

1. **Given** a valid Kratos session and a temporary failure when calling `/rest/internal/identity/resolve`, **When** oidc-service attempts to issue tokens, **Then** it fails token issuance in a controlled way (without issuing tokens lacking the Alkemio user ID claim) and records an operational signal (e.g., structured log or metric) for troubleshooting.

---

[Add more user stories as needed, each with an assigned priority]

### Edge Cases

None identified beyond the main user stories at this time.

## Requirements *(mandatory)*

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right functional requirements.
-->

### Functional Requirements

- **FR-001**: System MUST include an Alkemio user ID claim in tokens issued for authenticated users when a corresponding Alkemio user ID can be resolved from the user's Kratos authentication ID via the Alkemio server.
- **FR-002**: System MUST call the Alkemio server identity resolution endpoint (`POST /rest/internal/identity/resolve`) using the Kratos authentication ID associated with the current session to obtain the Alkemio user ID before finalizing token claims.
- **FR-003**: System MUST prevent token issuance when the identity resolution endpoint returns 404 for the provided authentication ID, so that only users fully represented in the Alkemio platform can obtain tokens.
- **FR-004**: System MUST fail token issuance when the identity resolution call fails due to network, timeout, or server error, while ensuring that the failure is operationally observable.
- **FR-005**: System MUST ensure that the Alkemio user ID claim is always added to identity and access tokens for every successful issuance once a mapping exists—there is no client or flow-based scoping.
- **FR-006**: System MUST record structured logs and/or metrics for identity resolution attempts, including success, not-found, and failure cases, without logging sensitive identifiers in plain form.
- **FR-007**: Kratos authentication IDs supplied to the resolver and Alkemio user IDs returned from `/rest/internal/identity/resolve` MUST be valid UUIDs; invalid identifiers MUST cause token issuance to fail.

### Key Entities *(include if feature involves data)*

- **Kratos Identity**: Represents the identity of a user in the authentication system, identified by a Kratos ID used as input to the identity resolution.
- **Alkemio User**: Represents the user in the Alkemio platform, identified by an Alkemio user ID that is exposed in the new token claim when resolution succeeds.
- **Token Claims Set**: Represents the collection of claims included in tokens issued by oidc-service, extended to include the Alkemio user ID claim for eligible tokens.

## Success Criteria *(mandatory)*

<!--
  ACTION REQUIRED: Define measurable success criteria.
  These must be technology-agnostic and measurable.
-->

### Measurable Outcomes

- **SC-003**: All identity resolution failures that affect claim population are captured in structured logs or metrics that allow operators to identify affected users or flows without exposing sensitive identifiers.
- **SC-004**: Downstream Alkemio services can rely on the Alkemio user ID token claim to perform authorization decisions, reducing the need for additional identity resolution calls for the targeted flows after rollout.
