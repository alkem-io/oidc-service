package challenge

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// FlowType enumerates supported Hydra challenge flows.
type FlowType string

const (
	// FlowLogin represents the login challenge flow.
	FlowLogin FlowType = "login"
	// FlowConsent represents the consent challenge flow.
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
	TokenClaims  *TokenClaims
}

// TokenClaims holds the enhanced claim fields to be included in tokens.
type TokenClaims struct {
	GivenName      *string `json:"given_name,omitempty"`
	FamilyName     *string `json:"family_name,omitempty"`
	EmailVerified  *bool   `json:"email_verified,omitempty"`
	AcceptedTerms  *bool   `json:"accepted_terms,omitempty"`
	AlkemioUserID  *string `json:"alkemio_user_id,omitempty"`
	AlkemioAgentID *string `json:"agent_id,omitempty"`
}

// IsEmpty returns true if no claims are set.
func (tc *TokenClaims) IsEmpty() bool {
	return tc.GivenName == nil && tc.FamilyName == nil && tc.EmailVerified == nil && tc.AcceptedTerms == nil && tc.AlkemioUserID == nil && tc.AlkemioAgentID == nil
}

// ToAccessTokenMap converts claims appropriate for Access tokens to a map.
func (tc *TokenClaims) ToAccessTokenMap() map[string]any {
	claims := make(map[string]any)
	if tc.GivenName != nil {
		claims["given_name"] = *tc.GivenName
	}
	if tc.FamilyName != nil {
		claims["family_name"] = *tc.FamilyName
	}
	if tc.AlkemioUserID != nil {
		claims["alkemio_user_id"] = *tc.AlkemioUserID
	}
	if tc.AlkemioAgentID != nil {
		claims["agent_id"] = *tc.AlkemioAgentID
	}
	return claims
}

// ToIDTokenMap converts claims appropriate for ID tokens to a map.
func (tc *TokenClaims) ToIDTokenMap() map[string]any {
	claims := make(map[string]any)
	if tc.GivenName != nil {
		claims["given_name"] = *tc.GivenName
	}
	if tc.FamilyName != nil {
		claims["family_name"] = *tc.FamilyName
	}
	if tc.EmailVerified != nil {
		claims["email_verified"] = *tc.EmailVerified
	}
	if tc.AcceptedTerms != nil {
		claims["accepted_terms"] = *tc.AcceptedTerms
	}
	if tc.AlkemioUserID != nil {
		claims["alkemio_user_id"] = *tc.AlkemioUserID
	}
	if tc.AlkemioAgentID != nil {
		claims["agent_id"] = *tc.AlkemioAgentID
	}
	return claims
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
	// ResolveLogin completes the login challenge.
	ResolveLogin(ctx context.Context, challengeID string) (*Resolution, error)
	// ResolveConsent completes the consent challenge.
	ResolveConsent(ctx context.Context, challengeID string) (*Resolution, error)
	// Readiness checks the service health.
	Readiness(ctx context.Context) ReadinessState
}

// Error models domain failures surfaced to HTTP.
type Error interface {
	error
	// StatusCode returns the HTTP status code.
	StatusCode() int
	// Code returns the error code.
	Code() string
	// ChallengeID returns the challenge ID.
	ChallengeID() string
	// MissingTraits returns the missing traits.
	MissingTraits() []string
	// Timestamp returns the error timestamp.
	Timestamp() time.Time
}

// BaseError provides a canonical domain error implementation.
type BaseError struct {
	status        int
	code          string
	message       string
	challengeID   string
	missingTraits []string
	timestamp     time.Time
}

// NewError constructs a BaseError instance with the supplied metadata.
func NewError(status int, code, message, challengeID string, missing []string) *BaseError {
	return &BaseError{
		status:        status,
		code:          code,
		message:       message,
		challengeID:   challengeID,
		missingTraits: append([]string(nil), missing...),
		timestamp:     time.Now().UTC(),
	}
}

// Error returns the canonical error message.
func (e *BaseError) Error() string {
	return e.message
}

// StatusCode exposes the HTTP status associated with the error.
func (e *BaseError) StatusCode() int {
	return e.status
}

// Code returns the stable machine-readable error code.
func (e *BaseError) Code() string {
	return e.code
}

// ChallengeID returns the Hydra challenge identifier related to the error, if any.
func (e *BaseError) ChallengeID() string {
	return e.challengeID
}

// MissingTraits returns a copy of the traits that were absent from the identity payload.
func (e *BaseError) MissingTraits() []string {
	return append([]string(nil), e.missingTraits...)
}

// Timestamp indicates when the error instance was created.
func (e *BaseError) Timestamp() time.Time {
	return e.timestamp
}

// NewMissingTraitsError reports that Kratos omitted required identity traits.
func NewMissingTraitsError(challengeID string, traits []string) *BaseError {
	return NewError(http.StatusBadRequest, "missing_traits", "identity is missing required traits", challengeID, traits)
}

// NewHydraFailureError reports failures communicating with Hydra.
func NewHydraFailureError(challengeID, message string) *BaseError {
	if message == "" {
		message = "failed to resolve hydra challenge"
	}
	return NewError(http.StatusInternalServerError, "hydra_failure", message, challengeID, nil)
}

// NewInvalidChallengeError indicates Hydra rejected the provided challenge identifier.
func NewInvalidChallengeError(challengeID string) *BaseError {
	return NewError(http.StatusNotFound, "invalid_challenge", "challenge not found", challengeID, nil)
}

// NewMaintenanceError signals to callers that the service is intentionally unavailable.
func NewMaintenanceError(message string) *BaseError {
	if message == "" {
		message = "service is in maintenance mode"
	}
	return NewError(http.StatusServiceUnavailable, "maintenance_mode", message, "", nil)
}

// NewKratosFailureError reports failures retrieving or validating identity data from Kratos.
func NewKratosFailureError(challengeID, message string) *BaseError {
	if message == "" {
		message = "failed to resolve identity traits"
	}
	return NewError(http.StatusInternalServerError, "kratos_failure", message, challengeID, nil)
}

// NewAlkemioIdentityMissingError reports that the Alkemio resolver did not find the identity mapping.
func NewAlkemioIdentityMissingError(challengeID string) *BaseError {
	return NewError(http.StatusForbidden, "alkemio_identity_missing", "alkemio user mapping not found", challengeID, nil)
}

// NewAlkemioResolutionError reports a general failure talking to the Alkemio resolver.
func NewAlkemioResolutionError(challengeID, message string) *BaseError {
	if strings.TrimSpace(message) == "" {
		message = "failed to resolve alkemio user id"
	}
	return NewError(http.StatusBadGateway, "alkemio_resolution_failed", message, challengeID, nil)
}
