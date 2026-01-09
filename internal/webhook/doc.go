// Package webhook provides HTTP handlers for Kratos webhook integrations.
//
// The primary endpoint is POST /webhooks/kratos/post-login which resolves
// Alkemio identity claims (actor_id, agent_id) for a given Kratos identity
// and returns them in a format Kratos uses to update identity.metadata_public.
//
// This webhook is called by Kratos after successful login or registration
// flows when configured with response.parse: true.
package webhook
