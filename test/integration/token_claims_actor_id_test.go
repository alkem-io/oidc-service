package integration_test

import (
	"testing"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

func TestTokenClaimsActorIDMapping(t *testing.T) {
	actorID := "8f88b965-0f0d-4eb9-8d68-f565f5c81a6e"
	claims := &challenge.TokenClaims{}

	idToken := claims.ToIDTokenMap()
	if _, ok := idToken["alkemio_actor_id"]; ok {
		t.Fatalf("alkemio_actor_id should be absent when claim unset")
	}

	accessToken := claims.ToAccessTokenMap()
	if _, ok := accessToken["alkemio_actor_id"]; ok {
		t.Fatalf("alkemio_actor_id should be absent from access token map when unset")
	}

	claims.AlkemioActorID = stringPtrActor(actorID)

	idToken = claims.ToIDTokenMap()
	if idVal, ok := idToken["alkemio_actor_id"].(string); !ok || idVal != actorID {
		t.Fatalf("expected id token alkemio_actor_id=%s, got %v", actorID, idToken["alkemio_actor_id"])
	}

	accessToken = claims.ToAccessTokenMap()
	if accessVal, ok := accessToken["alkemio_actor_id"].(string); !ok || accessVal != actorID {
		t.Fatalf("expected access token alkemio_actor_id=%s, got %v", actorID, accessToken["alkemio_actor_id"])
	}
}

func stringPtrActor(value string) *string {
	return &value
}
