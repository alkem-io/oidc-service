package challenge

import "net/http"

// NewSessionRequiredError indicates a login flow requires a valid Kratos session.
func NewSessionRequiredError(challengeID string) *BaseError {
	return NewError(http.StatusUnauthorized, "session_required", "kratos session is required to complete login", challengeID, nil)
}

// NewSessionInvalidError indicates the provided Kratos session is invalid or expired.
func NewSessionInvalidError(challengeID string) *BaseError {
	return NewError(http.StatusUnauthorized, "session_invalid", "kratos session is invalid or expired", challengeID, nil)
}
