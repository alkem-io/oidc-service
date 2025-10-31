# Data Model: Token Claims Enhancement

**Date**: October 31, 2025  
**Feature**: Token Claims Enhancement  
**Branch**: 002-token-claims

## Entity Overview

This feature enhances existing token generation by adding claims derived from Kratos identity data. No new entities are created; existing entities are extended with additional claim fields.

## Enhanced Entities

### 1. Token Claims Data Structure

**Entity**: `TokenClaims` (extends existing token generation in `internal/challenge`)

**Fields**:
| Field | Type | Source | Required | Description |
|-------|------|--------|----------|-------------|
| `given_name` | string | `traits.name.first` | No* | User's first name |
| `family_name` | string | `traits.name.last` | No* | User's last name |
| `email_verified` | boolean | `verifiable_addresses` | No* | Email verification status (ID token only) |
| `accepted_terms` | boolean | `traits.accepted_terms` | No* | Terms acceptance status (ID token only) |

*Required when source data is available; omitted when source data is missing/null

**Validation Rules**:
- `given_name`: Must be valid UTF-8 string if present
- `family_name`: Must be valid UTF-8 string if present  
- `email_verified`: Must be boolean value (true/false) if present
- `accepted_terms`: Must be boolean value (true/false) if present
- All claims must be omitted (not included) if source data is unavailable

**State Transitions**: N/A (stateless claim generation)

### 2. Kratos Identity Data Mapping

**Entity**: `KratosIdentityTraits` (existing entity, documented for reference)

**Relevant Fields**:
| Kratos Field | Token Claim | Extraction Logic |
|--------------|-------------|------------------|
| `traits.name.first` | `given_name` | Direct string mapping, omit if null/empty |
| `traits.name.last` | `family_name` | Direct string mapping, omit if null/empty |
| `verifiable_addresses[].verified` | `email_verified` | Boolean logic: true if any email address is verified |
| `traits.accepted_terms` | `accepted_terms` | Direct boolean mapping, omit if null/undefined |

**Validation Rules**:
- Source fields must be validated for type correctness before mapping
- Null, undefined, or empty string values result in claim omission
- Boolean fields must be explicit true/false values

### 3. Token Type Specifications

**Access Token Claims**:
- Include: `given_name`, `family_name`
- Exclude: `email_verified`, `accepted_terms`

**ID Token Claims**:
- Include: `given_name`, `family_name`, `email_verified`, `accepted_terms`

## Data Flow

```
Kratos Identity Traits
         ↓
Identity Mapper (internal/challenge)
         ↓
Token Claims Structure
         ↓
Token Generation (Access/ID Token)
         ↓
Client Application
```

## Error Handling

**Missing Source Data**: Omit corresponding claims from token  
**Invalid Source Data**: Log warning, omit claims, continue token generation  
**Kratos Unavailable**: Fail token generation entirely (existing behavior)  
**Token Size Exceeded**: Fail token generation with appropriate error

## Relationships

- **One-to-One**: Each Kratos identity maps to one set of token claims
- **Many-to-Many**: Claims can appear in multiple token types (Access/ID)
- **Dependencies**: Token claims depend on Kratos identity data availability

## Security Considerations

- Claims contain personally identifiable information (PII)
- Claims must not be logged in plaintext
- Claims follow existing token security and lifetime policies
- Email verification status is security-relevant information