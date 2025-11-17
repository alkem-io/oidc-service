# Token Contracts: Enhanced Claims

**Date**: October 31, 2025  
**Feature**: Token Claims Enhancement  
**Branch**: 002-token-claims

## Overview

This document specifies the enhanced token claims that will be added to Access tokens and ID tokens. The existing OIDC service API endpoints remain unchanged; only the generated tokens will contain additional claims.

## Token Specifications

### Access Token Claims (Enhanced)

**Existing Claims**: Standard OAuth2/OIDC claims (unchanged)  
**New Claims**:

```json
{
  "given_name": "John",
  "family_name": "Doe"
}
```

**Schema**:
- `given_name` (string, optional): User's first name from Kratos `traits.name.first`
- `family_name` (string, optional): User's last name from Kratos `traits.name.last`

**Behavior**:
- Claims are omitted entirely if source data is null/empty
- Claims are UTF-8 encoded strings
- Claims follow OpenID Connect standard naming

### ID Token Claims (Enhanced)

**Existing Claims**: Standard OIDC claims (unchanged)  
**New Claims**:

```json
{
  "given_name": "Jane",
  "family_name": "Smith", 
  "email_verified": true,
  "accepted_terms": false
}
```

**Schema**:
- `given_name` (string, optional): User's first name from Kratos `traits.name.first`
- `family_name` (string, optional): User's last name from Kratos `traits.name.last`
- `email_verified` (boolean, optional): Email verification status from Kratos `verifiable_addresses`
- `accepted_terms` (boolean, optional): Terms acceptance from Kratos `traits.accepted_terms`

**Behavior**:
- Claims are omitted entirely if source data is null/undefined
- Boolean claims are explicit `true` or `false` values
- Claims follow OpenID Connect standard naming

## API Impact

### Endpoints

**No changes to existing API endpoints**. The OIDC service API contract remains unchanged:
- `/v1/oidc/login` - behavior unchanged
- `/v1/oidc/consent` - behavior unchanged  
- Health endpoints - unchanged (metrics endpoint removed Nov 2025)

### Error Responses

**Existing error responses remain unchanged**. New potential error scenarios:

**Token Generation Failure** (Internal):
- If token size limits exceeded due to additional claims
- Handled by existing error propagation mechanisms
- Returns appropriate HTTP error responses to client

## Client Integration

### Backward Compatibility

✅ **Fully backward compatible**
- Existing clients will continue to work without modification
- New claims are additive and optional
- No breaking changes to token structure or API

### New Client Capabilities

Clients can now access enhanced user profile information:

```javascript
// Access Token Claims (example)
const accessToken = parseJWT(tokenResponse.access_token);
const firstName = accessToken.given_name; // "John" or undefined
const lastName = accessToken.family_name; // "Doe" or undefined

// ID Token Claims (example)  
const idToken = parseJWT(tokenResponse.id_token);
const emailVerified = idToken.email_verified; // true, false, or undefined
const termsAccepted = idToken.accepted_terms; // true, false, or undefined
```

## Validation Rules

### Token Size Constraints
- Additional claims add approximately 200-300 bytes per token
- Well within typical JWT size limits (8KB+)
- Token generation will fail if size limits exceeded

### Data Validation
- String claims: Valid UTF-8, non-empty after trimming
- Boolean claims: Explicit `true` or `false` values
- Missing data: Claims omitted entirely (not null values)

## Security Considerations

### Personal Information
- Claims contain PII (personally identifiable information)
- Subject to existing token security policies
- Token lifetime and encryption remain unchanged

### Claim Authenticity
- Claims sourced from authoritative Kratos identity system
- Subject to existing token signing and validation
- No additional authentication required

## Testing Contract

### Access Token Validation
```yaml
# Contract test requirements
access_token_claims:
  given_name: 
    type: string
    required: false
    source: kratos_traits.name.first
  family_name:
    type: string  
    required: false
    source: kratos_traits.name.last
```

### ID Token Validation
```yaml
# Contract test requirements
id_token_claims:
  given_name:
    type: string
    required: false
    source: kratos_traits.name.first
  family_name:
    type: string
    required: false
    source: kratos_traits.name.last
  email_verified:
    type: boolean
    required: false
    source: kratos_verifiable_addresses
  accepted_terms:
    type: boolean
    required: false
    source: kratos_traits.accepted_terms
```