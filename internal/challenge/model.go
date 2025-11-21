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

// NewError constructs a ChallengeError instance with the supplied metadata.
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

// Helper constructors for common failure modes.
func NewMissingTraitsError(challengeID string, traits []string) *ChallengeError {
	return NewError(http.StatusBadRequest, "missing_traits", "identity is missing required traits", challengeID, traits)
}

func NewHydraFailureError(challengeID, message string) *ChallengeError {
	if message == "" {
		message = "failed to resolve hydra challenge"
	}
	return NewError(http.StatusInternalServerError, "hydra_failure", message, challengeID, nil)
}

func NewInvalidChallengeError(challengeID string) *ChallengeError {
	return NewError(http.StatusNotFound, "invalid_challenge", "challenge not found", challengeID, nil)
}

func NewMaintenanceError(message string) *ChallengeError {
	if message == "" {
		message = "service is in maintenance mode"
	}
	return NewError(http.StatusServiceUnavailable, "maintenance_mode", message, "", nil)
}

func NewKratosFailureError(challengeID, message string) *ChallengeError {
	if message == "" {
		message = "failed to resolve identity traits"
	}
	return NewError(http.StatusInternalServerError, "kratos_failure", message, challengeID, nil)
}

func NewAlkemioIdentityMissingError(challengeID string) *ChallengeError {
	return NewError(http.StatusForbidden, "alkemio_identity_missing", "alkemio user mapping not found", challengeID, nil)
}

func NewAlkemioResolutionError(challengeID, message string) *ChallengeError {
	if strings.TrimSpace(message) == "" {
		message = "failed to resolve alkemio user id"
	}
	return NewError(http.StatusBadGateway, "alkemio_resolution_failed", message, challengeID, nil)
}
