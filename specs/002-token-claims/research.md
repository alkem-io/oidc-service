# Research: Token Claims Enhancement

**Date**: October 31, 2025  
**Feature**: Token Claims Enhancement  
**Branch**: 002-token-claims

## Research Tasks

### 1. Token Generation Latency Impact Analysis

**Task**: Research performance impact of adding claims to token generation process  
**Status**: ✅ COMPLETE

**Decision**: Minimal performance impact expected  
**Rationale**: Adding 4 additional claims to existing token generation is a lightweight operation. Claims are simple string/boolean values extracted from already-available Kratos identity data. No additional network calls or heavy computation required.  
**Alternatives considered**: 
- Async claim population (rejected: adds complexity without significant benefit)
- Claim caching (rejected: premature optimization, adds state management complexity)

### 2. Current Token Generation Architecture

**Task**: Understand existing token generation and claim inclusion mechanisms  
**Status**: ✅ COMPLETE

**Decision**: Extend existing identity mapping in `internal/challenge` package  
**Rationale**: The service already has identity mapping logic in `identity_mapper.go` that converts Kratos identity data for token generation. This is the natural extension point for additional claims.  
**Alternatives considered**:
- New middleware for claim injection (rejected: violates domain-oriented architecture)
- Handler-level claim addition (rejected: violates thin handler principle)

### 3. Kratos Identity Schema Integration

**Task**: Research Kratos identity trait access patterns and field mapping  
**Status**: ✅ COMPLETE  

**Decision**: Use existing Kratos client patterns with trait field access  
**Rationale**: Service already accesses Kratos identity data through established client patterns. The required fields (`traits.name.first`, `traits.name.last`, `traits.accepted_terms`, verifiable_addresses) are standard Kratos identity schema fields.  
**Alternatives considered**:
- Direct Kratos API calls (rejected: bypasses existing client abstraction and timeout handling)
- Identity caching (rejected: adds complexity and potential data staleness issues)

### 4. Token Size Limit Handling

**Task**: Research token size constraints and graceful failure patterns  
**Status**: ✅ COMPLETE

**Decision**: Use existing error handling patterns with fail-fast approach  
**Rationale**: Adding 4 standard claims (approximately 200-300 bytes) to tokens is well within typical JWT size limits. If limits are exceeded, the existing error propagation from token generation will handle failures appropriately.  
**Alternatives considered**:
- Claim prioritization (rejected: all claims are required by specification)
- Dynamic claim inclusion (rejected: adds complexity and unpredictable behavior)

### 5. OpenID Connect Standard Compliance

**Task**: Verify claim naming and data type standards  
**Status**: ✅ COMPLETE

**Decision**: Use standard OpenID Connect claim names and types  
**Rationale**: The specified claims (`given_name`, `family_name`, `email_verified`, `accepted_terms`) follow OpenID Connect standards. `given_name` and `family_name` are standard profile claims, `email_verified` is a standard boolean claim, and `accepted_terms` follows boolean claim patterns.  
**Alternatives considered**:
- Custom claim names (rejected: reduces interoperability)
- Nested claim structure (rejected: not required and adds complexity)

### 6. Error Handling for Missing Kratos Data

**Task**: Research patterns for handling missing or null Kratos identity traits  
**Status**: ✅ COMPLETE

**Decision**: Omit claims when source data is missing/null  
**Rationale**: OpenID Connect allows optional claims. Omitting claims when data is unavailable is cleaner than including null values and prevents client applications from receiving potentially confusing null claims.  
**Alternatives considered**:
- Include claims with null values (rejected: can confuse client applications)
- Default values for missing claims (rejected: could provide misleading information)

## Summary

All research tasks completed. The implementation approach is clear:

1. **Architecture**: Extend existing `internal/challenge/identity_mapper.go` 
2. **Performance**: Minimal impact, no optimization needed
3. **Data Source**: Use existing Kratos client patterns
4. **Standards**: Follow standard OpenID Connect claim conventions
5. **Error Handling**: Omit claims when source data unavailable
6. **Testing**: Leverage existing contract and integration test patterns

No blocking issues identified. Ready to proceed to Phase 1 design.