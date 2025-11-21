package integration_test

import (
	"testing"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

func TestTokenClaimsAgentIDMapping(t *testing.T) {
	agentID := "8f88b965-0f0d-4eb9-8d68-f565f5c81a6e"
	claims := &challenge.TokenClaims{}

	idToken := claims.ToIDTokenMap()
	if _, ok := idToken["agent_id"]; ok {
		t.Fatalf("agent_id should be absent when claim unset")
	}

	accessToken := claims.ToAccessTokenMap()
	if _, ok := accessToken["agent_id"]; ok {
		t.Fatalf("agent_id should be absent from access token map when unset")
	}

	claims.AlkemioAgentID = stringPtrAgent(agentID)

	idToken = claims.ToIDTokenMap()
	if idVal, ok := idToken["agent_id"].(string); !ok || idVal != agentID {
		t.Fatalf("expected id token agent_id=%s, got %v", agentID, idToken["agent_id"])
	}

	accessToken = claims.ToAccessTokenMap()
	if accessVal, ok := accessToken["agent_id"].(string); !ok || accessVal != agentID {
		t.Fatalf("expected access token agent_id=%s, got %v", agentID, accessToken["agent_id"])
	}
}

func stringPtrAgent(value string) *string {
	return &value
}
