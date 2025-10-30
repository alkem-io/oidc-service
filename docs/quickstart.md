# OIDC Service Quickstart

This guide walks through running the standalone OIDC Challenge Service locally
alongside the Synapse + Hydra + Kratos stack used in Alkemio development.

## Prerequisites

- Docker Desktop 4.30+ (Docker Engine 24+) with Compose v2
- Access to pull `docker.io/alkemio/oidc-service`
- The Alkemio server quickstart stack (`quickstart-services.yml`) from the
  root of the `alkem-io/server` repository
- A configuration file based on `configs/env.sample`

## 1. Pull required images

> The following Docker Compose commands assume you are running them from the
> `alkem-io/server` repository root where `quickstart-services.yml` lives. If
> you're inside `oidc-service/`, run `cd ..` first or adjust the `-f` path.

```sh
docker compose -f quickstart-services.yml pull hydra kratos mailslurper synapse
docker pull docker.io/alkemio/oidc-service:latest
```

## 2. Prepare environment configuration

```sh
cp configs/env.sample .env.oidc
# Edit .env.oidc with Hydra/Kratos URLs, admin tokens, cookie domain, and maintenance toggle
```

## 3. Launch supporting services

```sh
docker compose -f quickstart-services.yml up -d hydra kratos mailslurper synapse
```

## 4. Run the OIDC Service container

```sh
docker run --rm \
  --env-file .env.oidc \
  --name oidc-service \
  -p 8085:8080 \
  docker.io/alkemio/oidc-service:latest
```

## 5. Verify readiness

```sh
curl -sSf http://localhost:8085/health/ready | jq
```

A healthy service returns a payload similar to:

```json
{"status":"ready","hydra":"ok","kratos":"ok"}
```

## 6. Exercise the login flow

```sh
curl -i "http://localhost:8085/v1/oidc/login?login_challenge=test"
```

Expect an HTTP `302` redirect when Hydra accepts the challenge, or structured
`4xx` errors when the challenge is invalid or missing identity traits.

## 7. Toggle maintenance mode

```sh
docker run --rm \
  --env-file .env.oidc \
  -e OIDC_MAINTENANCE_MODE=true \
  -p 8085:8080 \
  docker.io/alkemio/oidc-service:latest
curl -i http://localhost:8085/v1/oidc/login?login_challenge=test
```

During maintenance the service responds with HTTP `503` and a JSON body
explaining the outage window.

## 8. Inspect Prometheus metrics

```sh
curl -s http://localhost:8085/metrics | grep oidc_challenge_latency_seconds
```

Latency histograms and counters confirm the metrics endpoint is wired for
observability dashboards.

## 9. GitHub Actions overview

The legacy `ci.yml` and `release.yml` jobs have been removed. The service now
ships with the following workflows under `.github/workflows/`:

- `pr-build.yml` — builds the container image for pull requests against any path.
- `build-release-docker-hub.yml` — publishes `docker.io/alkemio/oidc-service`
  when a release is published.
- `build-deploy-k8s-dev-hetzner.yml` — builds the image and deploys to the dev
  Hetzner cluster on pushes to `develop`.
- `build-deploy-k8s-sandbox-hetzner.yml` — manual deployment to the sandbox
  Hetzner environment via `workflow_dispatch`.
- `build-deploy-k8s-test-hetzner.yml` — manual deployment to the test Hetzner
  environment via `workflow_dispatch`.
- `schema-contract.yml` — lints the OpenAPI contract and diff-checks pull
  requests touching `contracts/` or documentation.
- `trigger-e2e-tests.yml` — notifies the external `alkem-io/test-suites`
  repository to execute end-to-end verification on release publication.

Consult the repository secrets to ensure Docker registry and Travis API tokens
are available before invoking the release or deployment workflows.

## Troubleshooting

- Readiness showing `unreachable` indicates Hydra or Kratos is misconfigured—
verify URLs and admin tokens in `.env.oidc`.
- `INVALID_CHALLENGE` 404 responses mean the challenge has expired in Hydra; use
`hydra tokens introspect` to confirm state.
- Set `OIDC_LOG_LEVEL=debug` to gain additional zap logs when investigating
failures.
