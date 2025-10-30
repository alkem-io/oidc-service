package challenge

import (
	"context"
	"net/http"
	"time"
)

// FlowType enumerates supported Hydra challenge flows.
type FlowType string

const (
	FlowLogin   FlowType = "login"
	FlowConsent FlowType = "consent"
)

// Resolution captures the outcome of a challenge orchestration.
type Resolution struct {
	RedirectURL string
}

// HydraChallenge contains data required to resolve a login or consent challenge.
type HydraChallenge struct {
	ID        string
	Flow      FlowType
	Subject   string
	ClientID  string
	Skip      bool
	RequestID string
}

// IdentityProfile models traits retrieved from Kratos for Hydra acceptance.
type IdentityProfile struct {
	ID           string
	Email        string
	DisplayName  string
	MatrixUserID string
	Traits       map[string]any
}

// ReadinessState reflects upstream availability for health checks.
type ReadinessState struct {
	Status  string
	Hydra   string
	Kratos  string
	Version string
}

// Service defines behaviour for resolving Hydra challenges.
type Service interface {
	ResolveLogin(ctx context.Context, challengeID string) (*Resolution, error)
	ResolveConsent(ctx context.Context, challengeID string) (*Resolution, error)
	Readiness(ctx context.Context) ReadinessState
}

// Error models domain failures surfaced to HTTP.
type Error interface {
	error
	StatusCode() int
	Code() string
	ChallengeID() string
	MissingTraits() []string
	Timestamp() time.Time
}

// ChallengeError provides a canonical domain error implementation.
type ChallengeError struct {
	status        int
	code          string
	message       string
	challengeID   string
	missingTraits []string
	timestamp     time.Time
}

// timestamp in UTC.
func NewError(status int, code, message, challengeID string, missing []string) *ChallengeError {
	return &ChallengeError{
		status:        status,
		code:          code,
		message:       message,
		challengeID:   challengeID,
		missingTraits: append([]string(nil), missing...),
		timestamp:     time.Now().UTC(),
	}
}

func (e *ChallengeError) Error() string {
	return e.message
}

func (e *ChallengeError) StatusCode() int {
	return e.status
}

func (e *ChallengeError) Code() string {
	return e.code
}

func (e *ChallengeError) ChallengeID() string {
	return e.challengeID
}

func (e *ChallengeError) MissingTraits() []string {
	return append([]string(nil), e.missingTraits...)
}

func (e *ChallengeError) Timestamp() time.Time {
	return e.timestamp
}

// NewMissingTraitsError creates a ChallengeError indicating the identity is missing required traits.
// The produced error has HTTP status 400, code "missing_traits", and includes the provided challenge ID and missing trait names.
func NewMissingTraitsError(challengeID string, traits []string) *ChallengeError {
	return NewError(http.StatusBadRequest, "missing_traits", "identity is missing required traits", challengeID, traits)
}

// NewHydraFailureError creates a ChallengeError representing an internal Hydra resolution failure.
// If message is empty, the error message defaults to "failed to resolve hydra challenge".
func NewHydraFailureError(challengeID, message string) *ChallengeError {
	if message == "" {
		message = "failed to resolve hydra challenge"
	}
	return NewError(http.StatusInternalServerError, "hydra_failure", message, challengeID, nil)
}

// NewInvalidChallengeError returns a ChallengeError indicating the requested challenge was not found.
// The error has HTTP status 404 and domain code "invalid_challenge".
func NewInvalidChallengeError(challengeID string) *ChallengeError {
	return NewError(http.StatusNotFound, "invalid_challenge", "challenge not found", challengeID, nil)
}

// NewMaintenanceError creates a ChallengeError representing that the service is in maintenance mode.
// If message is empty, the default message "service is in maintenance mode" is used.
func NewMaintenanceError(message string) *ChallengeError {
	if message == "" {
		message = "service is in maintenance mode"
	}
	return NewError(http.StatusServiceUnavailable, "maintenance_mode", message, "", nil)
}

// NewKratosFailureError creates a ChallengeError for a Kratos (identity) resolution failure for the given challenge ID.
// If message is empty, the error message defaults to "failed to resolve identity traits".
// The returned error has HTTP status 500 and code "kratos_failure".
func NewKratosFailureError(challengeID, message string) *ChallengeError {
	if message == "" {
		message = "failed to resolve identity traits"
	}
	return NewError(http.StatusInternalServerError, "kratos_failure", message, challengeID, nil)
}