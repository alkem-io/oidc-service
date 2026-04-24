package audit

// T013 — oidc-service audit emitter.
// Writes one JSON record per event to stdout in local-dev, structured log in k8s.
// Shape matches alkemio-server audit.ts and contracts/audit-event.md.
