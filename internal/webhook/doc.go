// Package webhook provides HTTP handlers for Kratos webhook integrations.
//
// The primary endpoint is POST /webhooks/kratos/post-login which resolves
// Alkemio identity claims (alkemio_actor_id, alkemio_agent_id) for a given
// Kratos identity and stores them in identity.metadata_public via the Kratos
// Admin API (PATCH /admin/identities/{id}).
//
// This webhook is called by Kratos after email verification and login flows.
// It uses the Admin API because Kratos response.parse only works for
// registration and settings flows, not for login or verification.
package webhook
