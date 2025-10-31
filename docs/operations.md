# Operations Runbook

## Maintenance Mode Toggle

Use the `OIDC_MAINTENANCE_MODE` environment variable to place the service in
maintenance. The toggle blocks challenge handling and returns HTTP 503 with an
optional `Retry-After` header derived from `OIDC_MAINTENANCE_RETRY`.

1. **Toggle via Docker Compose**
    - Run commands from the `alkem-io/server` repository root where the
      `quickstart-services.yml` stack is defined (or update `-f` to point to that
      file).
    - Persist the flag by appending `OIDC_MAINTENANCE_MODE=true` to your `.env`
      (or a compose override file) and then run:
    ```sh
    docker compose -f quickstart-services.yml up -d oidc-service
    ```
    - Alternatively, set it inline when recreating the service:
    ```sh
    OIDC_MAINTENANCE_MODE=true docker compose -f quickstart-services.yml up -d oidc-service
    ```
    In both cases the container must be recreated or restarted for the new
    environment to take effect.

2. **Toggle via Kubernetes**
   - Patch the deployment with `kubectl set env deployment/oidc-service OIDC_MAINTENANCE_MODE=true`.
   - Optionally set `OIDC_MAINTENANCE_RETRY=600` to emit a 10 minute retry hint.

3. **Validate state**
    - Ensure the service is healthy:
    ```sh
    curl -i http://OIDC_SERVICE_HOST/health/ready
    ```
    - Regenerate a Hydra login challenge using the procedure from Quickstart Step 6
      (or the equivalent `curl` + `awk` command) and invoke the login endpoint:
    ```sh
    curl -i "http://OIDC_SERVICE_HOST/v1/oidc/login?login_challenge=${LOGIN_CHALLENGE}"
    ```
    Expect `HTTP 503` with `{"error":"maintenance_mode"...}` while maintenance is
    enabled.

4. **Disable maintenance** by removing or setting the toggle to `false` and
   redeploying.

## Dependency and CI Audits

- **Go module audit (2025-10-30)**: `go list -u -m` surfaced
  `github.com/ory/client-go v1.22.7` as the latest stable release; the module was
  updated via `go get` and committed.
- **GitHub Actions audit (2025-10-30)**: `.github/workflows/build-deploy-k8s-*.yml`,
  `.github/workflows/build-release-docker-hub.yml`,
  `.github/workflows/schema-contract.yml`,
  `.github/workflows/trigger-e2e-tests.yml`, and
  `oidc-service/.github/workflows/container-build.yml` all pin current stable
  actions (`actions/checkout@v4`, `actions/setup-go@v5`, `actions/cache@v4`,
  `golangci/golangci-lint-action@v4`, `docker/setup-qemu-action@v3`,
  `docker/setup-buildx-action@v3`, `docker/build-push-action@v5`,
  `sigstore/cosign-installer@v3`). The syft container used for SBOM generation
  is pinned to `anchore/syft:v1.15.0`.
- **Re-run audit** after each release by executing `scripts/lint.sh`,
  reviewing the output of `go list -u -m all | grep '\['` for upgrades, and
  checking action release notes for new major versions.
