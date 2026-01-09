# Feature: DB-First Claims Resolution with API Fallback

**Status**: Implemented
**Branch**: `006-db-claims-lookup`
**Date**: 2026-01-08

## Overview

The OIDC service resolves Alkemio identity claims (UserID, AgentID) from a local PostgreSQL database before falling back to the HTTP API. This reduces latency and load on the Alkemio server.

## Architecture

```
CompositeResolver
    ├─ DatabaseResolver (try first)
    │     └─ SQLC queries → pgxpool → PostgreSQL
    │
    └─ IdentityResolver (fallback)
          └─ HTTP API → Alkemio Server
```

## User Stories

### US1: Fast Claims Resolution via Database (P1)
When a user's authenticationID exists in the database, claims are resolved directly without API calls.

### US2: API Fallback for Unknown Users (P2)
When authenticationID is not in the database, the system falls back to the Alkemio API.

### US3: Graceful Database Failure Handling (P3)
Database errors trigger API fallback with warning logs. Authentication continues during DB outages.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_HOST` | `localhost` | PostgreSQL host |
| `DATABASE_PORT` | `5432` | PostgreSQL port |
| `DATABASE_NAME` | `alkemio` | Database name |
| `DATABASE_USERNAME` | `synapse` | Database user |
| `DATABASE_PASSWORD` | `synapse` | Database password |
| `DATABASE_TIMEOUT` | `5s` | Connection timeout |

## Database Schema

Uses existing `user` table:

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Alkemio User ID |
| `authenticationID` | UUID | Kratos identity ID (lookup key) |
| `agentId` | UUID | Alkemio Agent ID |

## Files

- `internal/alkemio/db.go` - Database pool initialization
- `internal/alkemio/db_resolver.go` - DatabaseResolver implementation
- `internal/alkemio/composite_resolver.go` - CompositeResolver (DB-first, API fallback)
- `internal/alkemio/queries/` - SQLC generated code
- `internal/config/config.go` - Database configuration fields
- `cmd/server/main.go` - Wiring

## Dependencies

- `github.com/jackc/pgx/v5` v5.8.0 - PostgreSQL driver
- `sqlc` v1.30.0 - SQL code generation (build tool)
