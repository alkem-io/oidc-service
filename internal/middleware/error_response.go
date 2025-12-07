package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

// WriteChallengeError serializes challenge-aware errors into the HTTP response.
func WriteChallengeError(w http.ResponseWriter, r *http.Request, err error) {
	if w == nil {
		return
	}

	logger := zap.NewNop()
	if r != nil {
		logger = Logger(r.Context())
	}

	var chErr challenge.Error
	if errors.As(err, &chErr) {
		payload := map[string]any{
			"error":       chErr.Code(),
			"message":     chErr.Error(),
			"challengeId": chErr.ChallengeID(),
			"timestamp":   chErr.Timestamp().Format(time.RFC3339),
		}
		if missing := chErr.MissingTraits(); len(missing) > 0 {
			payload["missingTraits"] = missing
		}

		logger.Error("challenge resolution failed",
			zap.String("challengeId", chErr.ChallengeID()),
			zap.String("errorCode", chErr.Code()),
			zap.Error(err),
		)

		writeJSON(w, chErr.StatusCode(), payload)
		return
	}

	logger.Error("challenge resolution failed with unexpected error", zap.Error(err))
	writeJSON(w, http.StatusInternalServerError, map[string]any{
		"error":       "internal_error",
		"message":     "unexpected error resolving challenge",
		"challengeId": "",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
