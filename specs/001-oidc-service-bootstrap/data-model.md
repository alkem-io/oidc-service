# Data Model & Domain Notes: OIDC Service Bootstrap

## Domain Entities

### HydraChallenge

Represents the payload retrieved from Hydra Admin API when resolving login or consent challenges.

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| id | string | Path parameter | Hydra challenge identifier |
| subject | string | Hydra response | End-user ID, may be empty prior to login |
| request_url | string | Hydra response | Used only for auditing |
| redirect_to | string | Hydra response | Final redirect location returned to the client |
| requested_scopes | []string | Hydra response | Consent story only |
| request_context | map[string]any | Hydra response | Holds tenant + UI hints |

### LoginResult

Internal struct handed to presenters to build HTTP responses.

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| redirect_to | string | Hydra accept response | Sent to client |
| session | KratosSession | Derived | Used to capture identity metadata |
| remember | bool | Hydra accept response | Mirrors Hydra behaviour |
| remember_for | time.Duration | Hydra accept response | TTL in seconds |

### KratosSession

Represents the authenticated identity resolved from Kratos.

| Field | Type | Notes |
|-------|------|-------|
| id | string | UUID from Kratos |
| identity_id | string | Underlying identity reference |
| email | string | Extracted from traits |
| expires_at | time.Time | Used to detect stale sessions |
| active | bool | Derived from Kratos payload |

### MaintenanceState

Tracks whether the service is allowing traffic.

| Field | Type | Notes |
|-------|------|-------|
| active | bool | When true, requests short-circuit |
| reason | string | Optional operator-supplied message |
| toggled_by | string | Operator identifier (header) |
| toggled_at | time.Time | UTC timestamp |

## Configuration Surface

| Env Var | Default | Description |
|---------|---------|-------------|
| `OIDC_HYDRA_ADMIN_URL` | none | Base URL for Hydra Admin API |
| `OIDC_KRATOS_PUBLIC_URL` | none | Internal base URL for Kratos public API (service-to-service) |
| `OIDC_KRATOS_BROWSER_URL` | none | External base URL for Kratos browser flows returned to clients |
| `OIDC_WEB_BASE_URL` | none | Public web origin used to derive login return URL when explicit override absent |
| `OIDC_LOGIN_RETURN_BASE_URL` | derived | External URL used for `return_to` when redirecting to Kratos login (defaults to `${OIDC_WEB_BASE_URL}/oidc/login`) |
| `OIDC_HTTP_LISTEN_ADDR` | `:8080` | HTTP bind address |
| `OIDC_REQUEST_TIMEOUT` | `5s` | Outbound request timeout |
| `OIDC_SESSION_COOKIE` | `ory_kratos_session` | Kratos session cookie name |
| `OIDC_MAINTENANCE_SHARED_SECRET` | none | Secret required to toggle maintenance |
| `OIDC_LOG_LEVEL` | `info` | zap log level |

## External Interfaces

- **Hydra Admin**: `GET /oauth2/auth/requests/login`, `PUT /oauth2/auth/requests/login/accept`, equivalent consent endpoints.
- **Kratos**: `GET /sessions/whoami` with session cookie header.
- **Health**: `/health/live`, `/health/ready` returning JSON state snapshots.

## Error Catalogue

| Code | HTTP | Description |
|------|------|-------------|
| `INVALID_CHALLENGE` | 400 | Hydra indicates the challenge is unknown or expired |
| `HYDRA_UNAVAILABLE` | 503 | Hydra Admin API unreachable or timed out |
| `KRATOS_SESSION_INVALID` | 401 | Kratos session cookie missing or inactive |
| `MAINTENANCE_MODE_ENABLED` | 503 | Maintenance overlay is active |
| `CONFIGURATION_ERROR` | 500 | Service failed required configuration loading |

## Observability

- **Logs**: Fields include `component`, `challenge_id`, `redirect_to`, `status`, `maintenance_active`. These structured logs replaced the earlier Prometheus metrics requirement (removed Nov 2025) and now serve as the authoritative operational signal.

## Data Flow Overview

1. HTTP request enters `chi` router.
2. Maintenance middleware checks `MaintenanceState`; may short-circuit.
3. Handler binds DTO, validates challenge ID format.
4. Hydra client fetches challenge metadata.
5. Kratos client resolves session (login only).
6. Hydra client accepts/denies challenge.
7. Presenter logs outcome and sends JSON response (metrics removed per Nov 2025 decision).
8. Health endpoints aggregate component checkers and emit JSON.
