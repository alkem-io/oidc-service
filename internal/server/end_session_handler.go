package server

// T031 — end_session_endpoint handler.
// Validates id_token_hint against Hydra JWKS + iss/exp, derives client_id from
// aud/azp, validates post_logout_redirect_uri against the RP's registered list,
// calls DELETE /admin/oauth2/auth/sessions/consent?client=<id>&subject=<sub>,
// emits session.end_session audit, and 302s to post_logout_redirect_uri.
