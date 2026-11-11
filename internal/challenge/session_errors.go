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

// NewSessionAAL2RequiredError indicates the Kratos session must complete its second factor before login.
func NewSessionAAL2RequiredError(challengeID string) *BaseError {
	return NewError(
		http.StatusUnauthorized, "session_aal2_required", "kratos session requires a second factor", challengeID, nil,
	)
}
