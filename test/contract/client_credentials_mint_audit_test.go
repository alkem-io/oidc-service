package contract_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/audit"
	"github.com/alkem-io/oidc-service/internal/server"
)

// TestTokenMintAuditEmitsServiceClientActorOnClientCredentials — FR-021 /
// FR-026 (T024). When Hydra calls the token-hook with
// `requester.grant_types` containing `client_credentials`, the emitted
// `token.mint` audit row MUST carry `actor_type: "service-client"`.
//
// Replaces the prior plan's `client_credentials_scope_gate_test.go`: per
// research.md R-9 addendum, `client_credentials` never reaches the consent
// path (machine-to-machine flows skip consent entirely), so the grant-type
// discriminator lives in the token-mint webhook, not in
// `internal/challenge/service.go`.
func TestTokenMintAuditEmitsServiceClientActorOnClientCredentials(t *testing.T) {
	t.Parallel()

	events := emitTokenMintAndCaptureAudit(t, hookPayload{
		Session: hookSessionInput{
			Subject:     "analytics-ingestion-pipeline",
			AccessToken: map[string]any{},
			IDToken:     map[string]any{},
		},
		Requester: hookRequesterInput{
			ClientID:      "analytics-ingestion-pipeline",
			GrantedScopes: []string{"platform:read", "analytics:write"},
			GrantTypes:    []string{"client_credentials"},
		},
	})

	mint := findEventByType(t, events, "token.mint")
	require.Equal(t, "service-client", mint["actor_type"],
		"client_credentials grant must emit actor_type=service-client")
	require.Equal(t, "success", mint["outcome"])
	require.Equal(t, "analytics-ingestion-pipeline", mint["client_id"])
}

// TestTokenMintAuditEmitsUserActorOnAuthorizationCode — FR-021 / FR-026 (T024).
// The default branch (every grant other than `client_credentials`) MUST tag
// the mint with `actor_type: "user"`. Covers `authorization_code` (the
// browser login flow's first mint) and `refresh_token` (silent re-issuance)
// via two sub-tests.
func TestTokenMintAuditEmitsUserActorOnAuthorizationCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		grantType string
	}{
		{name: "AuthorizationCode", grantType: "authorization_code"},
		{name: "RefreshToken", grantType: "refresh_token"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			events := emitTokenMintAndCaptureAudit(t, hookPayload{
				Session: hookSessionInput{
					Subject:     "a9b6d4c2-68c2-4c60-920e-0f2b77e9c123",
					AccessToken: map[string]any{},
					IDToken:     map[string]any{},
				},
				Requester: hookRequesterInput{
					ClientID:      "alkemio-client-web",
					GrantedScopes: []string{"openid", "profile", "email"},
					GrantTypes:    []string{tc.grantType},
				},
			})

			mint := findEventByType(t, events, "token.mint")
			require.Equal(t, "user", mint["actor_type"],
				"%s grant must emit actor_type=user (not service-client)", tc.grantType)
			require.Equal(t, "success", mint["outcome"])
		})
	}
}

// --- shared scaffolding ------------------------------------------------------

// hookPayload mirrors the subset of Hydra's token-hook envelope that the
// handler under test decodes. Kept local to this file so the test stays
// hermetic against future schema growth on the handler side.
type hookPayload struct {
	Session   hookSessionInput   `json:"session"`
	Requester hookRequesterInput `json:"requester"`
}

type hookSessionInput struct {
	AccessToken map[string]any `json:"access_token,omitempty"`
	IDToken     map[string]any `json:"id_token,omitempty"`
	Subject     string         `json:"subject"`
}

type hookRequesterInput struct {
	ClientID        string   `json:"client_id"`
	GrantedScopes   []string `json:"granted_scopes"`
	GrantedAudience []string `json:"granted_audience,omitempty"`
	GrantTypes      []string `json:"grant_types"`
}

// staticActorResolver returns "" + nil so the handler exits the resolver
// branch via the "no claim to inject" path and proceeds to the success
// response (where the mint audit fires). The token-mint discriminator is
// orthogonal to actor-id resolution; we want to isolate that branch under test.
type staticActorResolver struct{}

func (staticActorResolver) Resolve(_ context.Context, _ string, _ []string) (string, error) {
	return "", nil
}

// emitTokenMintAndCaptureAudit POSTs a synthetic Hydra token-hook payload to
// the real handler with an injected audit emitter writing to an in-memory
// buffer, then parses the captured JSON-line events back into maps for
// per-test assertions. Returns the decoded events in emission order.
func emitTokenMintAndCaptureAudit(t *testing.T, payload hookPayload) []map[string]any {
	t.Helper()

	var buf bytes.Buffer
	emitter := audit.NewEmitter(&buf)
	handler := server.NewTokenHookHandler(zap.NewNop(), emitter, staticActorResolver{})

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/oidc/token-hook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"token-hook must respond 200 on the success path; body=%s", rec.Body.String())

	return decodeAuditLines(t, buf.Bytes())
}

// decodeAuditLines splits the captured emitter output on newlines and decodes
// each non-empty line as a JSON object. The audit Emitter guarantees one
// record per `\n`-terminated line (see internal/audit/audit.go::Emit).
func decodeAuditLines(t *testing.T, raw []byte) []map[string]any {
	t.Helper()

	var out []map[string]any
	for _, line := range bytes.Split(raw, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var ev map[string]any
		require.NoError(t, json.Unmarshal(line, &ev), "audit line not valid JSON: %s", line)
		out = append(out, ev)
	}
	return out
}

// findEventByType locates the first event with the given `event_type` and
// fails the test if none is present. Keeps the per-test assertion blocks
// terse while still surfacing the captured payload on failure for debugging.
func findEventByType(t *testing.T, events []map[string]any, eventType string) map[string]any {
	t.Helper()

	for _, ev := range events {
		if et, _ := ev["event_type"].(string); et == eventType {
			return ev
		}
	}
	t.Fatalf("no event with event_type=%q in captured audit stream: %#v", eventType, events)
	return nil
}
