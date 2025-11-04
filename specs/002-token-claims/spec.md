# Feature Specification: Token Claims Enhancement

**Feature Branch**: `002-token-claims`  
**Created**: October 31, 2025  
**Status**: Draft  
**Input**: User description: "Add claims to Access token and ID token. Both of them should also add claims 'given_name' and 'family_name', and ID token should also add claim 'email_verified' and 'accepted_terms' (boolean)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Enhanced Token Claims for User Profile Data (Priority: P1)

Client applications can access standardized user profile information (given name and family name) from both Access tokens and ID tokens, enabling consistent user identification and personalization across integrated services.

**Why this priority**: Core functionality that enables all downstream applications to properly identify and display user information consistently. This is foundational for user experience and system integration.

**Independent Test**: Can be fully tested by examining token contents after authentication and verifying that given_name and family_name claims are present in both token types with correct user data.

**Acceptance Scenarios**:

1. **Given** a user with first name "John" and last name "Doe" authenticates, **When** an Access token is issued, **Then** the token contains claims "given_name": "John" and "family_name": "Doe"
2. **Given** a user with first name "Jane" and last name "Smith" authenticates, **When** an ID token is issued, **Then** the token contains claims "given_name": "Jane" and "family_name": "Smith"
3. **Given** a user has no first or last name data, **When** tokens are issued, **Then** the given_name and family_name claims are omitted from the tokens

---

### User Story 2 - Compliance and Security Claims in ID Tokens (Priority: P2)

Client applications can determine user compliance and security status by examining the email_verified and accepted_terms claims in ID tokens, enabling appropriate security, legal compliance, and user experience decisions.

**Why this priority**: Important for security, legal compliance, and user trust, but secondary to basic user identification. Applications can function without this but may provide degraded security or user experience.

**Independent Test**: Can be fully tested by examining ID token contents and verifying that email_verified and accepted_terms claims accurately reflect the user's verification and terms acceptance status.

**Acceptance Scenarios**:

1. **Given** a user with a verified email address authenticates, **When** an ID token is issued, **Then** the token contains claim "email_verified": true
2. **Given** a user with an unverified email address authenticates, **When** an ID token is issued, **Then** the token contains claim "email_verified": false
3. **Given** a user who has accepted terms and conditions authenticates, **When** an ID token is issued, **Then** the token contains claim "accepted_terms": true
4. **Given** a user who has not accepted terms and conditions authenticates, **When** an ID token is issued, **Then** the token contains claim "accepted_terms": false
5. **Given** a user with no email address, **When** an ID token is issued, **Then** the email_verified claim is omitted from the token

---

### Edge Cases

- What happens when Kratos identity traits `name.first` or `name.last` are missing or null? (Claims will be omitted entirely)
- Unicode and special characters in name fields are preserved as valid UTF-8 strings in token claims
- What happens when Kratos email verification status cannot be determined or is unavailable? (email_verified claim will be omitted)
- What happens when `traits.accepted_terms` is null or missing from Kratos identity? (accepted_terms claim will be omitted)
- How does the system behave when token size limits are approached due to additional claims? (System will fail token generation to maintain data integrity)
- What occurs if the Kratos identity system is unavailable during token generation? (Token generation will fail entirely to maintain security and consistency)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST add "given_name" claim to both Access tokens and ID tokens containing the user's first name when available
- **FR-002**: System MUST add "family_name" claim to both Access tokens and ID tokens containing the user's last name when available  
- **FR-003**: System MUST add "email_verified" claim to ID tokens indicating whether the user's email address has been verified
- **FR-004**: System MUST add "accepted_terms" claim to ID tokens indicating whether the user has accepted terms and conditions (boolean value)
- **FR-005**: System MUST omit claims entirely when corresponding Kratos identity traits are missing or null
- **FR-006**: System MUST preserve unicode and special characters as valid UTF-8 strings in name field claims
- **FR-007**: System MUST ensure token claims are populated from Kratos identity traits as the authoritative user data source
- **FR-008**: System MUST maintain existing token functionality while adding new claims
- **FR-009**: System MUST fail token generation if total token size would exceed 4KB limit (browser cookie compatibility)
- **FR-010**: System MUST fail token generation if Kratos identity system is unavailable

### Key Entities

- **Kratos Identity**: Contains user traits including `name.first`, `name.last`, `email`, and `accepted_terms`; source of truth for token claims
- **Access Token**: OAuth2 token that will be enhanced with given_name and family_name claims derived from Kratos identity traits
- **ID Token**: OpenID Connect token that will be enhanced with given_name, family_name, email_verified, and accepted_terms claims derived from Kratos identity data

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of Access tokens issued contain given_name and family_name claims when user data is available
- **SC-002**: 100% of ID tokens issued contain given_name, family_name, email_verified, and accepted_terms claims when user data is available

- **SC-003**: Zero authentication failures caused by token claim additions
- **SC-004**: Client applications can successfully parse and utilize the new claims without breaking existing functionality

## Assumptions

- User profile data is stored in Kratos identity schema under `traits.name.first` and `traits.name.last` fields
- Email verification status is tracked in Kratos identity verifiable_addresses field
- Terms acceptance status is stored in Kratos identity schema under `traits.accepted_terms` field (boolean)
- Token size limits will not be exceeded by adding these standard claims
- Claims follow standard OpenID Connect naming conventions (given_name, family_name, email_verified, accepted_terms)
- No additional user consent is required for adding these standard profile claims
- Kratos identity data can be accessed during token generation process

## Data Mapping

The following Kratos identity schema fields will be mapped to token claims:

| Kratos Field | Token Claim | Token Type | Data Type |
|--------------|-------------|------------|-----------|
| `traits.name.first` | `given_name` | Access Token, ID Token | string |
| `traits.name.last` | `family_name` | Access Token, ID Token | string |
| Identity verifiable_addresses field | `email_verified` | ID Token | boolean |
| `traits.accepted_terms` | `accepted_terms` | ID Token | boolean |

## Clarifications

### Session 2025-10-31

- Q: How should the system handle missing Kratos identity traits for token claims? → A: Omit claims entirely when Kratos traits are missing/null
- Q: How should the system handle token size limits when adding new claims? → A: Fail token generation if size limits would be exceeded
- Q: How should the system behave when Kratos identity system is unavailable during token generation? → A: Fail token generation entirely if Kratos is unavailable
- Q: What is the exact source for email verification status in Kratos? → A: Identity verifiable_addresses field
- Q: Should performance impact monitoring be included for this lightweight change? → A: Omit performance impact monitoring

## Dependencies

- Access to Kratos identity data containing `traits.name.first` and `traits.name.last` fields
- Access to Kratos identity verifiable_addresses field for email verification status
- Access to Kratos identity `traits.accepted_terms` field for terms acceptance status
- Understanding of current token generation and claim inclusion mechanisms
- Integration with Kratos identity system for retrieving user traits during token issuance
