package challenge

// T033 — refresh-time re-resolver.
// Reads Kratos metadata_public.alkemio_actor_id; on claim-absent + alkemio scope
// POSTs /rest/internal/identity/resolve once against alkemio-server and re-reads.
// Still-absent or stamp-fail returns temporarily_unavailable, emits
// refresh.missing_alkemio_actor_id audit, never rotates without the claim
// (FR-006 / FR-006a).
