package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/alkem-io/oidc-service/internal/challenge"
	"go.uber.org/zap"
)

// WriteChallengeError serializes a challenge-aware error into an HTTP JSON response.
// 
// If w is nil the function is a no-op. If r is non-nil a request-scoped logger is used;
// otherwise a no-op logger is used. When err implements challenge.Error the response
// JSON contains the fields "error" (error code), "message", "challengeId", "timestamp"
// and, if present, "missingTraits", and the HTTP status is taken from the challenge error.
// For all other errors the function logs the unexpected error and writes a 500 response
// with an "internal_error" payload including an empty "challengeId" and the current UTC timestamp.
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

// writeJSON writes the given payload to w as JSON, sets the Content-Type header to "application/json", and writes the provided HTTP status code; any JSON encoding error is ignored.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}