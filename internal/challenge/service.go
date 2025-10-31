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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

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

func buildConsentContext(profile *IdentityProfile) map[string]any {
	context := map[string]any{
		identityContextKey: profile.ID,
	}
	if profile.MatrixUserID != "" {
		context["matrix_user_id"] = profile.MatrixUserID
	}
	return context
}

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

func cloneTraits(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	return cloneTraitMap(src)
}

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

func readIdentityID(ctx interface{}) string {
	switch value := ctx.(type) {
	case map[string]any:
		if id, ok := value[identityContextKey]; ok {
			return strings.TrimSpace(fmt.Sprint(id))
		}
	}
	return ""
}

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

func isKratosIdentityID(value string) bool {
	if value == "" {
		return false
	}

	if _, err := uuid.Parse(value); err != nil {
		return false
	}

	return true
}
