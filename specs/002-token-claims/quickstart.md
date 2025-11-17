# Quickstart: Token Claims Enhancement

**Date**: October 31, 2025  
**Feature**: Token Claims Enhancement  
**Branch**: 002-token-claims

## Overview

This quickstart guide helps you understand and test the enhanced token claims feature that adds user profile information (`given_name`, `family_name`) to both Access tokens and ID tokens, plus compliance information (`email_verified`, `accepted_terms`) to ID tokens.

## What's New

### Access Tokens
- ✅ `given_name`: User's first name  
- ✅ `family_name`: User's last name

### ID Tokens  
- ✅ `given_name`: User's first name
- ✅ `family_name`: User's last name
- ✅ `email_verified`: Email verification status (boolean)
- ✅ `accepted_terms`: Terms acceptance status (boolean)

## Prerequisites

1. **Running OIDC Service**: Version with token claims enhancement
2. **Kratos Identity System**: With user identity data
3. **Test User**: With name and email data in Kratos

## Quick Test

### 1. Authenticate a User

Use your existing OIDC flow to authenticate a test user:

```bash
# Example using curl (adjust URLs for your setup)
curl -X GET "https://oidc-service.local/v1/oidc/login?login_challenge=YOUR_CHALLENGE"
```

### 2. Examine Token Claims

Decode the returned tokens to verify new claims:

```javascript
// Using a JWT decoder library or online tool
const accessToken = jwt.decode(tokenResponse.access_token);
console.log('Access Token Claims:', {
  given_name: accessToken.given_name,
  family_name: accessToken.family_name
});

const idToken = jwt.decode(tokenResponse.id_token);
console.log('ID Token Claims:', {
  given_name: idToken.given_name,
  family_name: idToken.family_name,
  email_verified: idToken.email_verified,
  accepted_terms: idToken.accepted_terms
});
```

### 3. Expected Results

**Complete User Data**:
```json
{
  "given_name": "John",
  "family_name": "Doe", 
  "email_verified": true,
  "accepted_terms": false
}
```

**Partial User Data** (missing first name):
```json
{
  "family_name": "Doe",
  "email_verified": true,
  "accepted_terms": false
}
```

**Minimal User Data** (only email verified):
```json
{
  "email_verified": true
}
```

## Kratos Identity Setup

### Required Kratos Schema Fields

Ensure your Kratos identity schema includes:

```json
{
  "traits": {
    "name": {
      "first": "John",
      "last": "Doe"  
    },
    "accepted_terms": false
  },
  "verifiable_addresses": [
    {
      "value": "john@example.com",
      "verified": true
    }
  ]
}
```

### Test Data Creation

Create test users with varying data completeness:

```bash
# Complete user data
kratos identities create --schema-id default --traits '{
  "name": {"first": "John", "last": "Doe"},
  "email": "john@example.com", 
  "accepted_terms": true
}'

# Partial user data (no last name)
kratos identities create --schema-id default --traits '{
  "name": {"first": "Jane"},
  "email": "jane@example.com",
  "accepted_terms": false
}'

# Minimal user data (no names)
kratos identities create --schema-id default --traits '{
  "email": "minimal@example.com"
}'
```

## Testing Scenarios

### Scenario 1: Complete User Profile
1. Authenticate user with complete name data
2. Verify both tokens contain `given_name` and `family_name`
3. Verify ID token contains email and terms status

### Scenario 2: Partial User Profile  
1. Authenticate user missing some name fields
2. Verify only available claims appear in tokens
3. Verify missing claims are omitted (not null)

### Scenario 3: Minimal User Profile
1. Authenticate user with minimal identity data
2. Verify tokens work without profile claims
3. Verify service maintains backward compatibility

## Troubleshooting

### Claims Not Appearing

**Check Kratos Identity Data**:
```bash
kratos identities get YOUR_USER_ID
```

**Verify Field Mapping**:
- `given_name` ← `traits.name.first`
- `family_name` ← `traits.name.last`  
- `email_verified` ← `verifiable_addresses[].verified`
- `accepted_terms` ← `traits.accepted_terms`

### Token Size Issues

If tokens are rejected due to size:
1. Check total claim data size
2. Verify JWT size limits in client/server configuration
3. Consider reducing other custom claims if present

### Backward Compatibility

Existing clients should work unchanged:
- New claims are optional and additive
- No breaking changes to token structure
- API endpoints remain unchanged

## Development Integration

### Client-Side Usage

```javascript
// Safe claim access with fallbacks
function getUserDisplayName(idToken) {
  const firstName = idToken.given_name || '';
  const lastName = idToken.family_name || '';
  return `${firstName} ${lastName}`.trim() || 'User';
}

function checkUserCompliance(idToken) {
  const emailVerified = idToken.email_verified ?? false;
  const termsAccepted = idToken.accepted_terms ?? false;
  
  return {
    canAccessSensitiveData: emailVerified,
    requiresTermsAcceptance: !termsAccepted
  };
}
```

### Server-Side Validation

```go
// Example token claim validation
type TokenClaims struct {
    GivenName     string `json:"given_name,omitempty"`
    FamilyName    string `json:"family_name,omitempty"`
    EmailVerified *bool  `json:"email_verified,omitempty"`
    AcceptedTerms *bool  `json:"accepted_terms,omitempty"`
}

func validateUserClaims(claims TokenClaims) error {
    // Claims are optional - validate only if present
    if claims.EmailVerified != nil && !*claims.EmailVerified {
        return errors.New("email verification required")
    }
    return nil
}
```

## Next Steps

1. **Integration Testing**: Test with your specific client applications
2. **Monitoring**: Review structured logs for token size and generation performance (metrics removed Nov 2025)  
3. **User Experience**: Update UI to display enhanced user information
4. **Compliance**: Verify enhanced claims meet your security requirements

## Support

- **Logs**: Check OIDC service logs for claim generation details
- **Observability**: Tail zap logs for claim extraction diagnostics (metrics removed Nov 2025)
- **Documentation**: See `data-model.md` and `contracts/` for detailed specifications