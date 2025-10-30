package challenge

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	hydraAdmin "github.com/ory/hydra-client-go/v2"
)

const (
	defaultRememberDuration = time.Hour
	defaultReadinessTimeout = 5 * time.Second
)

// HydraClient captures the Hydra Admin interactions required to resolve challenges.
type HydraClient interface {
	GetLoginRequest(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error)
	AcceptLoginRequest(
		ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2LoginRequest,
	) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error)
	GetConsentRequest(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error)
	AcceptConsentRequest(
		ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest,
	) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error)
}

// IdentityFetcher retrieves identity profiles given a Kratos identifier.
type IdentityFetcher interface {
	Fetch(ctx context.Context, identityID string) (*IdentityProfile, error)
}

// Options configures the challenge service orchestrator.
type Options struct {
	Hydra            HydraClient
	Identity         IdentityFetcher
	RememberFor      time.Duration
	Readiness        ReadinessState
	HydraProbe       ReadinessProbe
	KratosProbe      ReadinessProbe
	ReadinessTimeout time.Duration
}

type service struct {
	hydra             HydraClient
	identity          IdentityFetcher
	rememberFor       int64
	readinessDefaults ReadinessState
	hydraProbe        ReadinessProbe
	kratosProbe       ReadinessProbe
	readyTimeout      time.Duration
}

// ReadinessProbe evaluates upstream availability for readiness reporting.
type ReadinessProbe interface {
	Ready(ctx context.Context) error
	Version(ctx context.Context) (string, error)
}

const (
	readinessStatusOK       = "ok"
	readinessStatusUnknown  = "unknown"
	readinessStatusTimeout  = "timeout"
	readinessStatusError    = "unavailable"
	readinessResultReady    = "ready"
	readinessResultDegraded = "degraded"
)

// NewService constructs a challenge orchestrator that coordinates Hydra and Kratos calls.
// It validates that a Hydra client and an Identity fetcher are provided, applies sensible defaults for remember duration, readiness defaults, and readiness timeout, and returns an implementation of Service or an error if required dependencies are missing.
func NewService(opts Options) (Service, error) {
	if opts.Hydra == nil {
		return nil, errors.New("hydra client is required")
	}
	if opts.Identity == nil {
		return nil, errors.New("identity fetcher is required")
	}

	remember := opts.RememberFor
	if remember <= 0 {
		remember = defaultRememberDuration
	}
	if remember < time.Second {
		remember = time.Second
	}

	readiness := opts.Readiness
	if readiness.Status == "" {
		readiness.Status = "initializing"
	}
	if readiness.Hydra == "" {
		readiness.Hydra = "unknown"
	}
	if readiness.Kratos == "" {
		readiness.Kratos = "unknown"
	}

	timeout := opts.ReadinessTimeout
	if timeout <= 0 {
		timeout = defaultReadinessTimeout
	}

	return &service{
		hydra:             opts.Hydra,
		identity:          opts.Identity,
		rememberFor:       int64(remember.Seconds()),
		readinessDefaults: readiness,
		hydraProbe:        opts.HydraProbe,
		kratosProbe:       opts.KratosProbe,
		readyTimeout:      timeout,
	}, nil
}

func (s *service) ResolveLogin(ctx context.Context, challengeID string) (*Resolution, error) {
	challengeID = strings.TrimSpace(challengeID)
	if challengeID == "" {
		return nil, NewError(http.StatusBadRequest, "missing_challenge", "login challenge is required", "", nil)
	}

	req, resp, err := s.hydra.GetLoginRequest(ctx, challengeID)
	if err != nil {
		return nil, mapHydraError(FlowLogin, challengeID, resp, err)
	}
	if req == nil {
		return nil, NewHydraFailureError(challengeID, "hydra returned empty login request")
	}

	if req.GetSkip() {
		contextData := readLoginContext(req)
		subject := strings.TrimSpace(req.GetSubject())

		if ctxID := readIdentityID(contextData); ctxID != "" {
			subject = ctxID
		}

		if subject == "" {
			return nil, NewHydraFailureError(challengeID, "hydra login request missing subject for skip flow")
		}

		if !isKratosIdentityID(subject) {
			if provider := IdentityHintProviderFromContext(ctx); provider != nil {
				identityID, err := provider.IdentityHint(ctx)
				if err != nil {
					return nil, mapIdentityHintError(challengeID, err)
				}

				identityID = strings.TrimSpace(identityID)
				if identityID == "" {
					return nil, NewSessionInvalidError(challengeID)
				}

				if !isKratosIdentityID(identityID) {
					return nil, NewSessionInvalidError(challengeID)
				}

				subject = identityID
			}
		}

		if !isKratosIdentityID(subject) {
			return nil, NewSessionInvalidError(challengeID)
		}

		contextData = ensureLoginContextIdentity(contextData, subject)

		payload := hydraAdmin.NewAcceptOAuth2LoginRequest(subject)
		payload.SetRemember(true)
		payload.SetRememberFor(s.rememberFor)
		if contextMap, ok := contextData.(map[string]any); ok {
			payload.SetContext(contextMap)
		}

		redirect, resp, err := s.hydra.AcceptLoginRequest(ctx, challengeID, payload)
		if err != nil {
			return nil, mapHydraError(FlowLogin, challengeID, resp, err)
		}

		return resolutionFromRedirect(challengeID, redirect)
	}

	identityID := strings.TrimSpace(req.GetSubject())
	if identityID == "" {
		provider := IdentityHintProviderFromContext(ctx)
		if provider == nil {
			return nil, NewSessionRequiredError(challengeID)
		}

		hintID, err := provider.IdentityHint(ctx)
		if err != nil {
			return nil, mapIdentityHintError(challengeID, err)
		}

		identityID = strings.TrimSpace(hintID)
		if identityID == "" {
			return nil, NewSessionInvalidError(challengeID)
		}
	}

	profile, err := s.identity.Fetch(ctx, identityID)
	if err != nil {
		return nil, mapIdentityError(challengeID, err)
	}

	payload := hydraAdmin.NewAcceptOAuth2LoginRequest(profile.ID)
	payload.SetRemember(true)
	payload.SetRememberFor(s.rememberFor)
	payload.SetContext(buildLoginContext(profile))

	redirect, resp, err := s.hydra.AcceptLoginRequest(ctx, challengeID, payload)
	if err != nil {
		return nil, mapHydraError(FlowLogin, challengeID, resp, err)
	}

	return resolutionFromRedirect(challengeID, redirect)
}

func (s *service) ResolveConsent(ctx context.Context, challengeID string) (*Resolution, error) {
	challengeID = strings.TrimSpace(challengeID)
	if challengeID == "" {
		return nil, NewError(http.StatusBadRequest, "missing_challenge", "consent challenge is required", "", nil)
	}

	req, resp, err := s.hydra.GetConsentRequest(ctx, challengeID)
	if err != nil {
		return nil, mapHydraError(FlowConsent, challengeID, resp, err)
	}
	if req == nil {
		return nil, NewHydraFailureError(challengeID, "hydra returned empty consent request")
	}

	identityID := extractIdentityID(req)
	if identityID == "" {
		return nil, NewHydraFailureError(challengeID, "consent request missing identity reference")
	}

	profile, err := s.identity.Fetch(ctx, identityID)
	if err != nil {
		return nil, mapIdentityError(challengeID, err)
	}

	payload := hydraAdmin.NewAcceptOAuth2ConsentRequest()
	if scopes := req.GetRequestedScope(); len(scopes) > 0 {
		payload.SetGrantScope(scopes)
	}
	payload.SetRemember(true)
	payload.SetRememberFor(s.rememberFor)
	payload.SetContext(buildConsentContext(profile))

	session := hydraAdmin.NewAcceptOAuth2ConsentRequestSession()
	session.SetIdToken(buildIDTokenClaims(profile))
	session.SetAccessToken(buildAccessTokenClaims(profile))
	payload.SetSession(*session)

	redirect, resp, err := s.hydra.AcceptConsentRequest(ctx, challengeID, payload)
	if err != nil {
		return nil, mapHydraError(FlowConsent, challengeID, resp, err)
	}

	return resolutionFromRedirect(challengeID, redirect)
}

func (s *service) Readiness(ctx context.Context) ReadinessState {
	ctx, cancel := s.withReadinessTimeout(ctx)
	defer cancel()

	hydraStatus, hydraVersion := probeStatus(ctx, s.hydraProbe)
	kratosStatus, kratosVersion := probeStatus(ctx, s.kratosProbe)

	status := readinessResultReady
	if hydraStatus != readinessStatusOK || kratosStatus != readinessStatusOK {
		status = readinessResultDegraded
	}

	version := firstNonEmpty(hydraVersion, kratosVersion, s.readinessDefaults.Version)

	return ReadinessState{
		Status:  status,
		Hydra:   hydraStatus,
		Kratos:  kratosStatus,
		Version: version,
	}
}

func (s *service) withReadinessTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	timeout := s.readyTimeout
	if timeout <= 0 {
		timeout = defaultReadinessTimeout
	}

	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) <= timeout {
			return ctx, func() {}
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	return ctx, cancel
}

// probeStatus determines the readiness status and version string reported by a readiness probe.
// 
// It returns the probe status as one of the readiness status constants and the probe's version
// (trimmed). If the probe is nil it returns readinessStatusUnknown and an empty version.
// If Ready returns an error it returns a classified error status and an empty version.
// If Version returns an error it returns readinessStatusOK and an empty version.
func probeStatus(ctx context.Context, probe ReadinessProbe) (string, string) {
	if probe == nil {
		return readinessStatusUnknown, ""
	}

	if err := probe.Ready(ctx); err != nil {
		return classifyReadinessError(err), ""
	}

	version, err := probe.Version(ctx)
	if err != nil {
		return readinessStatusOK, ""
	}

	return readinessStatusOK, strings.TrimSpace(version)
}

// classifyReadinessError maps an error to a readiness status string.
// If err is nil it returns readinessStatusOK.
// If err is context.DeadlineExceeded, context.Canceled, or a net.Error with Timeout() it returns readinessStatusTimeout.
// For any other non-nil error it returns readinessStatusError.
func classifyReadinessError(err error) string {
	if err == nil {
		return readinessStatusOK
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return readinessStatusTimeout
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return readinessStatusTimeout
	}

	return readinessStatusError
}

// firstNonEmpty returns the first value from values that is non-empty after
// trimming surrounding whitespace. If no value yields a non-empty string, an
// empty string is returned.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// resolutionFromRedirect validates a Hydra redirect response and returns a Resolution
// containing the trimmed redirect URL.
// It returns a Hydra failure error if the redirect object is nil or if the redirect URL is empty.
func resolutionFromRedirect(challengeID string, redirect *hydraAdmin.OAuth2RedirectTo) (*Resolution, error) {
	if redirect == nil {
		return nil, NewHydraFailureError(challengeID, "hydra response missing redirect url")
	}

	target := strings.TrimSpace(redirect.GetRedirectTo())
	if target == "" {
		return nil, NewHydraFailureError(challengeID, "hydra response contained empty redirect url")
	}

	return &Resolution{RedirectURL: target}, nil
}

// mapHydraError maps an HTTP response and/or error from a Hydra request into a package-level error describing the challenge failure.
// It returns an InvalidChallengeError for 404 or 410 responses, an `invalid_challenge` error for 400 responses, and otherwise a HydraFailureError that includes the flow and any error text.
func mapHydraError(flow FlowType, challengeID string, resp *http.Response, err error) error {
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusNotFound, http.StatusGone:
			return NewInvalidChallengeError(challengeID)
		case http.StatusBadRequest:
			return NewError(
				http.StatusBadRequest, "invalid_challenge", fmt.Sprintf("hydra rejected %s challenge", flow), challengeID, nil,
			)
		}
	}

	message := fmt.Sprintf("hydra %s request failed", flow)
	if err != nil && err.Error() != "" {
		message = message + ": " + err.Error()
	}

	return NewHydraFailureError(challengeID, message)
}

// mapIdentityError maps identity-fetching errors to the service's Kratos/identity error types
// scoped to the provided challengeID.
//
// - *MissingTraitsError is converted to a MissingTraitsError that preserves the missing traits.
// - *IdentityNotFoundError and *IdentityLookupError are converted to Kratos failure errors
//   containing the original error message.
// - Any other error is converted to a generic Kratos failure error containing the error text.
func mapIdentityError(challengeID string, err error) error {
	switch e := err.(type) {
	case *MissingTraitsError:
		return NewMissingTraitsError(challengeID, e.Traits)
	case *IdentityNotFoundError:
		return NewKratosFailureError(challengeID, e.Error())
	case *IdentityLookupError:
		return NewKratosFailureError(challengeID, e.Error())
	default:
		return NewKratosFailureError(challengeID, err.Error())
	}
}

// mapIdentityHintError converts errors returned while obtaining an identity hint
// into the appropriate service-level error for the specified challengeID.
// If err is nil, it returns nil. If err matches ErrIdentitySessionRequired it
// returns NewSessionRequiredError(challengeID). If err matches
// ErrIdentitySessionInvalid it returns NewSessionInvalidError(challengeID).
// For any other error it returns NewKratosFailureError(challengeID, err.Error()).
func mapIdentityHintError(challengeID string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrIdentitySessionRequired) {
		return NewSessionRequiredError(challengeID)
	}
	if errors.Is(err, ErrIdentitySessionInvalid) {
		return NewSessionInvalidError(challengeID)
	}
	return NewKratosFailureError(challengeID, err.Error())
}

const identityContextKey = "identity_id"

// buildLoginContext builds a map containing identity context for login flows.
// The map includes `identity_id`, `email`, and `display_name`. If present,
// `matrix_user_id` is added and `traits` are included as a deep clone to avoid
// sharing mutable state.
func buildLoginContext(profile *IdentityProfile) map[string]any {
	context := map[string]any{
		identityContextKey: profile.ID,
		"email":            profile.Email,
		"display_name":     profile.DisplayName,
	}
	if profile.MatrixUserID != "" {
		context["matrix_user_id"] = profile.MatrixUserID
	}
	if len(profile.Traits) > 0 {
		context["traits"] = cloneTraits(profile.Traits)
	}
	return context
}

// and, when available, the Matrix user id under the "matrix_user_id" key.
func buildConsentContext(profile *IdentityProfile) map[string]any {
	context := map[string]any{
		identityContextKey: profile.ID,
	}
	if profile.MatrixUserID != "" {
		context["matrix_user_id"] = profile.MatrixUserID
	}
	return context
}

// buildIDTokenClaims constructs a set of ID token claims from the given IdentityProfile.
// It includes the `email` and `display_name` claims and adds `matrix_user_id` when the profile contains one.
//
// The returned map contains the claim names as keys and their corresponding values.
func buildIDTokenClaims(profile *IdentityProfile) map[string]any {
	claims := map[string]any{
		"email":        profile.Email,
		"display_name": profile.DisplayName,
	}
	if profile.MatrixUserID != "" {
		claims["matrix_user_id"] = profile.MatrixUserID
	}
	return claims
}

// buildAccessTokenClaims builds a map of claim values for an access token from the given IdentityProfile.
// It includes "email", "display_name", and the identity id under the identityContextKey; it adds "matrix_user_id" when present
// and includes a deep-cloned "traits" entry if the profile contains traits.
func buildAccessTokenClaims(profile *IdentityProfile) map[string]any {
	claims := map[string]any{
		"email":            profile.Email,
		"display_name":     profile.DisplayName,
		identityContextKey: profile.ID,
	}
	if profile.MatrixUserID != "" {
		claims["matrix_user_id"] = profile.MatrixUserID
	}
	if len(profile.Traits) > 0 {
		claims["traits"] = cloneTraits(profile.Traits)
	}
	return claims
}

// cloneTraits creates and returns a deep copy of the provided traits map.
// If src is nil, cloneTraits returns nil.
func cloneTraits(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	return cloneTraitMap(src)
}

// cloneTraitMap returns a deep copy of the provided traits map.
// If src is nil, cloneTraitMap returns nil. Nested maps, slices, and primitive
// values are cloned so the result does not share mutable references with src.
func cloneTraitMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	clone := make(map[string]any, len(src))
	for key, value := range src {
		clone[key] = cloneTraitValue(value)
	}
	return clone
}

// cloneTraitSlice returns a deep copy of the provided slice of trait values.
// If src is nil, cloneTraitSlice returns nil.
func cloneTraitSlice(src []any) []any {
	if src == nil {
		return nil
	}

	clone := make([]any, len(src))
	for i, value := range src {
		clone[i] = cloneTraitValue(value)
	}
	return clone
}

// cloneTraitValue returns a deep copy of the provided trait value.
// Maps and slices are cloned recursively to avoid shared references; other values are returned unchanged.
func cloneTraitValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneTraitMap(typed)
	case []any:
		return cloneTraitSlice(typed)
	default:
		return typed
	}
}

// extractIdentityID extracts the identity ID associated with an OAuth2 consent request.
// It prefers an identity ID stored in the request's context under the "identity_id" key;
// if none is present, it falls back to the request subject (trimmed). An empty string is
// returned when the request is nil or no identity ID can be determined.
func extractIdentityID(req *hydraAdmin.OAuth2ConsentRequest) string {
	if req == nil {
		return ""
	}

	if ctx := req.GetContext(); ctx != nil {
		if id := readIdentityID(ctx); id != "" {
			return id
		}
	}

	return strings.TrimSpace(req.GetSubject())
}

// readIdentityID extracts the identity ID from a context-like value.
//
// If ctx is a map[string]any and contains the key identityContextKey, it returns
// the value converted to a string with surrounding whitespace trimmed. For any
// other input or if the key is absent, it returns an empty string.
func readIdentityID(ctx interface{}) string {
	switch value := ctx.(type) {
	case map[string]any:
		if id, ok := value[identityContextKey]; ok {
			return strings.TrimSpace(fmt.Sprint(id))
		}
	}
	return ""
}

// ensureLoginContextIdentity returns a context object that includes the given identity ID.
// If the provided identityID is empty (after trimming) the original ctx is returned.
// If ctx is a map[string]any and already contains the same identity_id value, the original map is returned.
// If ctx is a map[string]any and does not contain the identity_id (or contains a different value), a shallow copy
// of the map is returned with identity_id set to the trimmed identityID.
// For nil or non-map ctx values a new map containing only identity_id is returned.
func ensureLoginContextIdentity(ctx interface{}, identityID string) interface{} {
	identityID = strings.TrimSpace(identityID)
	if identityID == "" {
		return ctx
	}

	switch value := ctx.(type) {
	case map[string]any:
		if existing, ok := value[identityContextKey]; ok {
			existingID := strings.TrimSpace(fmt.Sprint(existing))
			if existingID == identityID {
				return value
			}
		}

		clone := make(map[string]any, len(value)+1)
		for key, val := range value {
			clone[key] = val
		}
		clone[identityContextKey] = identityID
		return clone
	case nil:
		return map[string]any{identityContextKey: identityID}
	default:
		return map[string]any{identityContextKey: identityID}
	}
}

// readLoginContext returns the "context" value from req.AdditionalProperties when present.
// If req is nil, AdditionalProperties is nil, or the "context" key is missing, it returns nil.
func readLoginContext(req *hydraAdmin.OAuth2LoginRequest) interface{} {
	if req == nil {
		return nil
	}

	if req.AdditionalProperties == nil {
		return nil
	}

	if ctx, ok := req.AdditionalProperties["context"]; ok {
		return ctx
	}

	return nil
}

// isKratosIdentityID reports whether the provided value is a Kratos identity ID (a UUID).
// It returns true if value is a non-empty valid UUID, false otherwise.
func isKratosIdentityID(value string) bool {
	if value == "" {
		return false
	}

	if _, err := uuid.Parse(value); err != nil {
		return false
	}

	return true
}