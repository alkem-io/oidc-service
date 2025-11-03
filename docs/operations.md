# Operations Runbook
## Login redirect behavior

When a request reaches `/oidc/login` without a valid Kratos session, the service
does not render JSON to the browser. Instead, it redirects the user-agent to the
Kratos browser login endpoint with a `return_to` parameter pointing back to
`${OIDC_WEB_BASE_URL}/oidc/login?...`.

- If `OIDC_KRATOS_BROWSER_URL` is set, it is used to build the Kratos login URL.
- If it is not set, the service infers scheme/host from `X-Forwarded-Proto` and
   `X-Forwarded-Host` (as provided by Traefik) and uses a loopback-safe fallback
   to avoid invalid hosts during local development.

This ensures end-users never see JSON error payloads like `session_required` in
their browser during normal flows.


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
   curl -i "http://OIDC_SERVICE_HOST/oidc/login?login_challenge=${LOGIN_CHALLENGE}"
    ```
    Expect `HTTP 503` with `{"error":"maintenance_mode"...}` while maintenance is
    enabled.

4. **Disable maintenance** by removing or setting the toggle to `false` and
   redeploying.

## Token Claims Monitoring

The service enhances OIDC tokens with user profile and compliance claims. Monitor these key metrics and behaviors:

### Metrics to Monitor

- **Token claim generation rates**: `token_claims_total` and `token_claims_generated` histograms track claim extraction success/failure
- **Token claim types**: Monitor both Access token claims (`given_name`, `family_name`) and ID token claims (`given_name`, `family_name`, `email_verified`, `accepted_terms`)
- **Claim extraction errors**: Watch for spikes in failed claim extractions which may indicate Kratos identity data issues

### Expected Claim Behavior

1. **Name Claims** (`given_name`, `family_name`):
   - **Source**: Kratos identity `traits.name.first` and `traits.name.last`
   - **Tokens**: Both Access tokens and ID tokens
   - **Omission**: Claims omitted if source data missing or invalid UTF-8
   - **Validation**: 255 character limit, UTF-8 validation, printable characters only

2. **Email Verification** (`email_verified`):
   - **Source**: Kratos identity `verifiable_addresses` array
   - **Tokens**: ID tokens only
   - **Logic**: `true` if any email address is verified, `false` if addresses exist but none verified, omitted if no email addresses
   - **Monitoring**: High false rates may indicate email verification workflow issues

3. **Terms Acceptance** (`accepted_terms`):
   - **Source**: Kratos identity `traits.accepted_terms` 
   - **Tokens**: ID tokens only
   - **Types**: Boolean values only, strings like "true"/"false" are parsed, invalid types cause omission
   - **Compliance**: Critical for legal compliance - monitor omission rates

### Troubleshooting Token Claims

1. **Missing Claims**:
   - Check Kratos identity data structure in admin API
   - Verify `traits.name.first`, `traits.name.last`, `traits.accepted_terms` fields
   - Validate `verifiable_addresses` array for email verification
   - Review service logs for extraction errors with identity context

2. **Invalid UTF-8 Names**:
   - Service logs will show UTF-8 validation failures
   - Claims will be omitted for safety
   - Check Kratos identity schema validation

3. **Email Verification Issues**:
   - Verify Kratos email verification workflow is functioning
   - Check `verifiable_addresses[].verified` boolean values
   - Monitor for addresses with `via: "email"` but `verified: false`

4. **Terms Acceptance Tracking**:
   - Ensure Kratos identity schema includes `accepted_terms` boolean field
   - Monitor for type mismatches (strings instead of booleans)
   - Verify terms acceptance UI updates Kratos correctly

### Log Analysis

Search logs for token claim operations:
```bash
# Successful claim extraction
grep "added enhanced claims" /var/log/oidc-service.log

# Token claim extraction errors
grep "failed to extract" /var/log/oidc-service.log

# Identity lookup issues
grep "identity lookup" /var/log/oidc-service.log
```

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
