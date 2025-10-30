package challenge

import "net/http"

// NewSessionRequiredError constructs a ChallengeError indicating that a valid Kratos session is required to complete login for the given challenge ID.
func NewSessionRequiredError(challengeID string) *ChallengeError {
	return NewError(http.StatusUnauthorized, "session_required", "kratos session is required to complete login", challengeID, nil)
}

// NewSessionInvalidError returns a ChallengeError indicating the provided Kratos session is invalid or expired.
// The returned error has HTTP status 401 and error code "session_invalid"; challengeID is attached to the error.
func NewSessionInvalidError(challengeID string) *ChallengeError {
	return NewError(http.StatusUnauthorized, "session_invalid", "kratos session is invalid or expired", challengeID, nil)
}