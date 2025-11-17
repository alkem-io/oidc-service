package challenge

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	hydraAdmin "github.com/ory/hydra-client-go/v2"
	"github.com/stretchr/testify/require"
)

func TestNewServiceRequiresHydraClient(t *testing.T) {
	_, err := NewService(Options{
		Identity: identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
			return nil, nil
		}},
		Alkemio: alkemioResolverStub{},
	})
	require.EqualError(t, err, "hydra client is required")
}

func TestNewServiceRequiresIdentityFetcher(t *testing.T) {
	_, err := NewService(Options{Hydra: &hydraClientMock{}, Alkemio: alkemioResolverStub{}})
	require.EqualError(t, err, "identity fetcher is required")
}

func TestNewServiceRequiresAlkemioResolver(t *testing.T) {
	_, err := NewService(Options{Hydra: &hydraClientMock{}, Identity: identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		return nil, nil
	}}})
	require.EqualError(t, err, "alkemio resolver is required")
}

func TestResolveLoginSkipUsesHydraSubject(t *testing.T) {
	login := hydraAdmin.NewOAuth2LoginRequestWithDefaults()
	login.SetChallenge("login-challenge")
	login.SetSkip(true)
	login.SetSubject("123e4567-e89b-12d3-a456-426614174000")

	mock := &hydraClientMock{
		getLogin: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
			require.Equal(t, "login-challenge", challengeID)
			return login, &http.Response{StatusCode: http.StatusOK}, nil
		},
		acceptLogin: func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2LoginRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			require.Equal(t, "login-challenge", challengeID)
			require.NotNil(t, body)
			require.Equal(t, "123e4567-e89b-12d3-a456-426614174000", body.Subject)
			require.True(t, body.GetRemember())
			require.Equal(t, int64(time.Hour.Seconds()), body.GetRememberFor())
			return hydraAdmin.NewOAuth2RedirectTo("https://redirect.example"), &http.Response{StatusCode: http.StatusOK}, nil
		},
	}

	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		t.Fatalf("identity fetch should not be called for skip flow")
		return nil, nil
	}}

	svc, err := NewService(Options{
		Hydra:       mock,
		Identity:    idFetcher,
		Alkemio:     alkemioResolverStub{},
		RememberFor: time.Hour,
	})
	require.NoError(t, err)

	resolution, err := svc.ResolveLogin(context.Background(), "login-challenge")
	require.NoError(t, err)
	require.NotNil(t, resolution)
	require.Equal(t, "https://redirect.example", resolution.RedirectURL)
}

func TestResolveLoginSkipHydraSubjectInvalidUsesHint(t *testing.T) {
	login := hydraAdmin.NewOAuth2LoginRequestWithDefaults()
	login.SetChallenge("login-challenge")
	login.SetSkip(true)
	login.SetSubject("admin@alkem.io")

	var capturedContext interface{}

	mock := &hydraClientMock{
		getLogin: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
			return login, &http.Response{StatusCode: http.StatusOK}, nil
		},
		acceptLogin: func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2LoginRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			require.Equal(t, "login-challenge", challengeID)
			require.Equal(t, "123e4567-e89b-12d3-a456-426614174001", body.Subject)
			require.True(t, body.GetRemember())
			require.Equal(t, int64(time.Hour.Seconds()), body.GetRememberFor())
			capturedContext = body.GetContext()
			return hydraAdmin.NewOAuth2RedirectTo("https://redirect.example"), &http.Response{StatusCode: http.StatusOK}, nil
		},
	}

	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		t.Fatalf("identity fetch should not be called for skip flow")
		return nil, nil
	}}

	svc, err := NewService(Options{
		Hydra:       mock,
		Identity:    idFetcher,
		Alkemio:     alkemioResolverStub{},
		RememberFor: time.Hour,
	})
	require.NoError(t, err)

	provider := IdentityHintFunc(func(ctx context.Context) (string, error) {
		return "123e4567-e89b-12d3-a456-426614174001", nil
	})

	ctx := WithIdentityHintProvider(context.Background(), provider)
	resolution, err := svc.ResolveLogin(ctx, "login-challenge")
	require.NoError(t, err)
	require.NotNil(t, resolution)
	require.Equal(t, "https://redirect.example", resolution.RedirectURL)

	ctxMap, ok := capturedContext.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "123e4567-e89b-12d3-a456-426614174001", ctxMap[identityContextKey])
}

func TestResolveLoginFetchesIdentityAndSetsContext(t *testing.T) {
	login := hydraAdmin.NewOAuth2LoginRequestWithDefaults()
	login.SetChallenge("login-challenge")
	login.SetSubject("identity-id")

	var capturedContext interface{}
	var capturedSubject string

	mock := &hydraClientMock{
		getLogin: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
			return login, &http.Response{StatusCode: http.StatusOK}, nil
		},
		acceptLogin: func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2LoginRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			capturedContext = body.GetContext()
			capturedSubject = body.Subject
			return hydraAdmin.NewOAuth2RedirectTo("https://redirect.example"), &http.Response{StatusCode: http.StatusOK}, nil
		},
	}

	traits := map[string]any{"display_name": "Example User"}
	idFetcher := identityFetcherStub{fetch: func(ctx context.Context, identityID string) (*IdentityProfile, error) {
		require.Equal(t, "identity-id", identityID)
		return &IdentityProfile{
			ID:           identityID,
			Email:        "user@example.com",
			DisplayName:  "Example User",
			MatrixUserID: "@user:example.com",
			Traits:       traits,
		}, nil
	}}

	svc, err := NewService(Options{
		Hydra:    mock,
		Identity: idFetcher,
		Alkemio:  alkemioResolverStub{},
	})
	require.NoError(t, err)

	resolution, err := svc.ResolveLogin(context.Background(), "login-challenge")
	require.NoError(t, err)
	require.Equal(t, "https://redirect.example", resolution.RedirectURL)

	require.Equal(t, "identity-id", capturedSubject)

	ctxMap, ok := capturedContext.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "identity-id", ctxMap[identityContextKey])
	require.Equal(t, "user@example.com", ctxMap["email"])
	require.Equal(t, "Example User", ctxMap["display_name"])
	require.Equal(t, "@user:example.com", ctxMap["matrix_user_id"])
	require.Equal(t, traits, ctxMap["traits"])
}

func TestResolveLoginUsesIdentityHintProvider(t *testing.T) {
	login := hydraAdmin.NewOAuth2LoginRequestWithDefaults()
	login.SetChallenge("login-challenge")

	var capturedSubject string
	var providerCalls int

	mock := &hydraClientMock{
		getLogin: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
			return login, &http.Response{StatusCode: http.StatusOK}, nil
		},
		acceptLogin: func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2LoginRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			capturedSubject = body.Subject
			return hydraAdmin.NewOAuth2RedirectTo("https://redirect.example"), &http.Response{StatusCode: http.StatusOK}, nil
		},
	}

	idFetcher := identityFetcherStub{fetch: func(ctx context.Context, identityID string) (*IdentityProfile, error) {
		require.Equal(t, "identity-hint", identityID)
		return &IdentityProfile{
			ID:          identityID,
			Email:       "user@example.com",
			DisplayName: "Example User",
		}, nil
	}}

	svc, err := NewService(Options{Hydra: mock, Identity: idFetcher, Alkemio: alkemioResolverStub{}})
	require.NoError(t, err)

	provider := IdentityHintFunc(func(ctx context.Context) (string, error) {
		providerCalls++
		return "identity-hint", nil
	})

	ctx := WithIdentityHintProvider(context.Background(), provider)
	resolution, err := svc.ResolveLogin(ctx, "login-challenge")
	require.NoError(t, err)
	require.NotNil(t, resolution)
	require.Equal(t, "https://redirect.example", resolution.RedirectURL)
	require.Equal(t, "identity-hint", capturedSubject)
	require.Equal(t, 1, providerCalls)
}

func TestResolveLoginMissingHintReturnsSessionRequired(t *testing.T) {
	login := hydraAdmin.NewOAuth2LoginRequestWithDefaults()
	login.SetChallenge("login-challenge")

	mock := &hydraClientMock{
		getLogin: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
			return login, &http.Response{StatusCode: http.StatusOK}, nil
		},
	}

	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		t.Fatalf("identity fetch should not be called when hint missing")
		return nil, nil
	}}

	svc, err := NewService(Options{Hydra: mock, Identity: idFetcher, Alkemio: alkemioResolverStub{}})
	require.NoError(t, err)

	_, err = svc.ResolveLogin(context.Background(), "login-challenge")
	require.Error(t, err)

	var challengeErr Error
	require.ErrorAs(t, err, &challengeErr)
	require.Equal(t, http.StatusUnauthorized, challengeErr.StatusCode())
	require.Equal(t, "session_required", challengeErr.Code())
}

func TestResolveLoginInvalidHintReturnsSessionInvalid(t *testing.T) {
	login := hydraAdmin.NewOAuth2LoginRequestWithDefaults()
	login.SetChallenge("login-challenge")

	mock := &hydraClientMock{
		getLogin: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
			return login, &http.Response{StatusCode: http.StatusOK}, nil
		},
	}

	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		t.Fatalf("identity fetch should not be called when hint invalid")
		return nil, nil
	}}

	svc, err := NewService(Options{Hydra: mock, Identity: idFetcher, Alkemio: alkemioResolverStub{}})
	require.NoError(t, err)

	provider := IdentityHintFunc(func(ctx context.Context) (string, error) {
		return "", ErrIdentitySessionInvalid
	})

	ctx := WithIdentityHintProvider(context.Background(), provider)
	_, err = svc.ResolveLogin(ctx, "login-challenge")
	require.Error(t, err)

	var challengeErr Error
	require.ErrorAs(t, err, &challengeErr)
	require.Equal(t, http.StatusUnauthorized, challengeErr.StatusCode())
	require.Equal(t, "session_invalid", challengeErr.Code())
}

func TestResolveLoginMissingTraitsReturnsDomainError(t *testing.T) {
	login := hydraAdmin.NewOAuth2LoginRequestWithDefaults()
	login.SetChallenge("login-challenge")
	login.SetSubject("identity-id")

	mock := &hydraClientMock{
		getLogin: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
			return login, &http.Response{StatusCode: http.StatusOK}, nil
		},
	}

	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		return nil, &MissingTraitsError{Traits: []string{"traits.email"}}
	}}

	svc, err := NewService(Options{Hydra: mock, Identity: idFetcher, Alkemio: alkemioResolverStub{}})
	require.NoError(t, err)

	_, err = svc.ResolveLogin(context.Background(), "login-challenge")
	require.Error(t, err)

	var challengeErr Error
	require.ErrorAs(t, err, &challengeErr)
	require.Equal(t, http.StatusBadRequest, challengeErr.StatusCode())
	require.Equal(t, "missing_traits", challengeErr.Code())
	require.Equal(t, []string{"traits.email"}, challengeErr.MissingTraits())
}

func TestResolveLoginHydra404ReturnsInvalidChallenge(t *testing.T) {
	mock := &hydraClientMock{
		getLogin: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
			return nil, &http.Response{StatusCode: http.StatusNotFound}, errors.New("not found")
		},
	}

	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		t.Fatalf("identity fetch should not be called on hydra error")
		return nil, nil
	}}

	svc, err := NewService(Options{Hydra: mock, Identity: idFetcher, Alkemio: alkemioResolverStub{}})
	require.NoError(t, err)

	_, err = svc.ResolveLogin(context.Background(), "missing")
	require.Error(t, err)

	var challengeErr Error
	require.ErrorAs(t, err, &challengeErr)
	require.Equal(t, http.StatusNotFound, challengeErr.StatusCode())
	require.Equal(t, "invalid_challenge", challengeErr.Code())
}

func TestResolveConsentBuildsSessionClaims(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest("consent-challenge")
	consent.SetSubject("user@example.com")
	consent.SetRequestedScope([]string{"openid", "profile"})
	consent.SetContext(map[string]any{identityContextKey: "identity-id"})

	var capturedSession hydraAdmin.AcceptOAuth2ConsentRequestSession
	var capturedContext interface{}

	mock := &hydraClientMock{
		getConsent: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			require.Equal(t, "consent-challenge", challengeID)
			return consent, &http.Response{StatusCode: http.StatusOK}, nil
		},
		acceptConsent: func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			capturedSession = body.GetSession()
			capturedContext = body.GetContext()
			require.Equal(t, []string{"openid", "profile"}, body.GetGrantScope())
			return hydraAdmin.NewOAuth2RedirectTo("https://redirect.example"), &http.Response{StatusCode: http.StatusOK}, nil
		},
	}

	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		return &IdentityProfile{
			ID:           "identity-id",
			Email:        "user@example.com",
			DisplayName:  "Example User",
			MatrixUserID: "@user:example.com",
			Traits:       map[string]any{"role": "member"},
		}, nil
	}}

	svc, err := NewService(Options{Hydra: mock, Identity: idFetcher, Alkemio: alkemioResolverStub{}})
	require.NoError(t, err)

	resolution, err := svc.ResolveConsent(context.Background(), "consent-challenge")
	require.NoError(t, err)
	require.Equal(t, "https://redirect.example", resolution.RedirectURL)

	idTokenClaims, ok := capturedSession.GetIdToken().(map[string]any)
	require.True(t, ok)
	require.Equal(t, "user@example.com", idTokenClaims["email"])
	require.Equal(t, "Example User", idTokenClaims["display_name"])
	require.Equal(t, "@user:example.com", idTokenClaims["matrix_user_id"])

	accessTokenClaims, ok := capturedSession.GetAccessToken().(map[string]any)
	require.True(t, ok)
	require.Equal(t, "identity-id", accessTokenClaims[identityContextKey])
	require.Equal(t, map[string]any{"role": "member"}, accessTokenClaims["traits"])

	ctxMap, ok := capturedContext.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "identity-id", ctxMap[identityContextKey])
	require.Equal(t, "@user:example.com", ctxMap["matrix_user_id"])
}

func TestResolveConsentMissingIdentityReferenceReturnsError(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest("consent-challenge")

	mock := &hydraClientMock{
		getConsent: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK}, nil
		},
	}

	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		t.Fatalf("identity fetch should not be called when identity id missing")
		return nil, nil
	}}

	svc, err := NewService(Options{Hydra: mock, Identity: idFetcher, Alkemio: alkemioResolverStub{}})
	require.NoError(t, err)

	_, err = svc.ResolveConsent(context.Background(), "consent-challenge")
	require.Error(t, err)

	var challengeErr Error
	require.ErrorAs(t, err, &challengeErr)
	require.Equal(t, http.StatusInternalServerError, challengeErr.StatusCode())
	require.Equal(t, "hydra_failure", challengeErr.Code())
}

func TestResolveConsentHydra404ReturnsInvalidChallenge(t *testing.T) {
	mock := &hydraClientMock{
		getConsent: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return nil, &http.Response{StatusCode: http.StatusNotFound}, errors.New("not found")
		},
	}

	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		t.Fatalf("identity fetch should not be called on hydra 404")
		return nil, nil
	}}

	svc, err := NewService(Options{Hydra: mock, Identity: idFetcher, Alkemio: alkemioResolverStub{}})
	require.NoError(t, err)

	_, err = svc.ResolveConsent(context.Background(), "missing")
	require.Error(t, err)

	var challengeErr Error
	require.ErrorAs(t, err, &challengeErr)
	require.Equal(t, http.StatusNotFound, challengeErr.StatusCode())
	require.Equal(t, "invalid_challenge", challengeErr.Code())
}

func TestReadinessReturnsConfiguredState(t *testing.T) {
	mock := &hydraClientMock{}
	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		return nil, errors.New("not implemented")
	}}

	svc, err := NewService(Options{
		Hydra:            mock,
		Identity:         idFetcher,
		Alkemio:          alkemioResolverStub{},
		Readiness:        ReadinessState{Version: "1.2.3"},
		HydraProbe:       readinessProbeStub{version: "hydra-2.0"},
		KratosProbe:      readinessProbeStub{version: "kratos-1.0"},
		ReadinessTimeout: time.Second,
	})
	require.NoError(t, err)

	state := svc.Readiness(context.Background())
	require.Equal(t, "ready", state.Status)
	require.Equal(t, "ok", state.Hydra)
	require.Equal(t, "ok", state.Kratos)
	require.Equal(t, "hydra-2.0", state.Version)
}

func TestReadinessDegradedWhenProbeFails(t *testing.T) {
	mock := &hydraClientMock{}
	idFetcher := identityFetcherStub{fetch: func(context.Context, string) (*IdentityProfile, error) {
		return nil, errors.New("not implemented")
	}}

	svc, err := NewService(Options{
		Hydra:       mock,
		Identity:    idFetcher,
		Alkemio:     alkemioResolverStub{},
		HydraProbe:  readinessProbeStub{readyErr: context.DeadlineExceeded},
		KratosProbe: readinessProbeStub{},
	})
	require.NoError(t, err)

	state := svc.Readiness(context.Background())
	require.Equal(t, "degraded", state.Status)
	require.Equal(t, "timeout", state.Hydra)
	require.Equal(t, "ok", state.Kratos)
	require.Equal(t, "", state.Version)
}

func TestCloneTraitsDeepCopy(t *testing.T) {
	original := map[string]any{
		"level": "root",
		"map":   map[string]any{"inner": "value"},
		"slice": []any{"one", map[string]any{"inner": "slice"}},
	}

	cloned := cloneTraits(original)
	require.NotNil(t, cloned)

	clonedMap := cloned["map"].(map[string]any)
	clonedMap["inner"] = "changed"

	clonedSlice := cloned["slice"].([]any)
	clonedSlice[0] = "two"
	clonedSlice[1].(map[string]any)["inner"] = "mutated"

	require.Equal(t, "value", original["map"].(map[string]any)["inner"])
	require.Equal(t, "one", original["slice"].([]any)[0])
	require.Equal(t, "slice", original["slice"].([]any)[1].(map[string]any)["inner"])
}

type hydraClientMock struct {
	getLogin      func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error)
	acceptLogin   func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2LoginRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error)
	getConsent    func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error)
	acceptConsent func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error)
}

type alkemioResolverStub struct {
	resolve func(ctx context.Context, authenticationID string) (string, error)
}

func (s alkemioResolverStub) Resolve(ctx context.Context, authenticationID string) (string, error) {
	if s.resolve != nil {
		return s.resolve(ctx, authenticationID)
	}
	return "alkemio-user", nil
}

func (m *hydraClientMock) GetLoginRequest(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
	if m.getLogin == nil {
		return nil, nil, errors.New("getLogin not stubbed")
	}
	return m.getLogin(ctx, challengeID)
}

func (m *hydraClientMock) AcceptLoginRequest(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2LoginRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
	if m.acceptLogin == nil {
		return nil, nil, errors.New("acceptLogin not stubbed")
	}
	return m.acceptLogin(ctx, challengeID, body)
}

func (m *hydraClientMock) GetConsentRequest(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
	if m.getConsent == nil {
		return nil, nil, errors.New("getConsent not stubbed")
	}
	return m.getConsent(ctx, challengeID)
}

func (m *hydraClientMock) AcceptConsentRequest(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
	if m.acceptConsent == nil {
		return nil, nil, errors.New("acceptConsent not stubbed")
	}
	return m.acceptConsent(ctx, challengeID, body)
}

type identityFetcherStub struct {
	fetch func(ctx context.Context, identityID string) (*IdentityProfile, error)
}

func (i identityFetcherStub) Fetch(ctx context.Context, identityID string) (*IdentityProfile, error) {
	if i.fetch == nil {
		return nil, errors.New("identity fetch not stubbed")
	}
	return i.fetch(ctx, identityID)
}

type readinessProbeStub struct {
	readyErr   error
	version    string
	versionErr error
}

func (p readinessProbeStub) Ready(context.Context) error {
	return p.readyErr
}

func (p readinessProbeStub) Version(context.Context) (string, error) {
	if p.versionErr != nil {
		return "", p.versionErr
	}
	return p.version, nil
}
