package challenge

import (
	"context"
	"errors"
	"slices"
	"sync"

	"go.uber.org/zap"
)

// ErrRefreshTemporarilyUnavailable signals FR-006/006a: the refresh
// re-resolution was unable to surface `alkemio_actor_id` for a session whose
// scope requires it. Callers MUST propagate this to Hydra as
// `temporarily_unavailable` and MUST NOT rotate the refresh-token family.
var ErrRefreshTemporarilyUnavailable = errors.New("refresh: alkemio_actor_id still absent after re-resolution")

// KratosIdentityReader fetches the public `alkemio_actor_id` from a Kratos
// identity at refresh-token exchange time. Production wires the existing
// Kratos admin transport; tests substitute a deterministic stub.
type KratosIdentityReader interface {
	// ReadAlkemioActorID returns metadata_public.alkemio_actor_id on the
	// Kratos identity, "" + nil err when the claim is absent.
	ReadAlkemioActorID(ctx context.Context, identityID string) (string, error)
}

// AlkemioReStamper triggers a single identity re-stamp call against
// alkemio-server's /rest/internal/identity/resolve endpoint when Kratos is
// missing the claim (FR-006a). Production wires the alkemio transport; tests
// substitute a deterministic stub.
type AlkemioReStamper interface {
	// ReStamp invokes the resolve endpoint exactly once per refresh attempt.
	ReStamp(ctx context.Context, identityID string) error
}

// RefreshResolver re-resolves the Kratos identity at refresh-token exchange
// time. It reads `metadata_public.alkemio_actor_id` from Kratos; on claim
// absent + `alkemio` scope it POSTs /rest/internal/identity/resolve once
// against alkemio-server and re-reads. Still-absent or stamp-fail returns
// ErrRefreshTemporarilyUnavailable. FR-006 / FR-006a.
type RefreshResolver struct {
	kratos  KratosIdentityReader
	alkemio AlkemioReStamper
	logger  *zap.Logger
}

// RefreshResolverConfig wires production dependencies. Zero-value config
// yields an in-memory deterministic stub used by the contract test surface.
type RefreshResolverConfig struct {
	Logger  *zap.Logger
	Kratos  KratosIdentityReader
	Alkemio AlkemioReStamper
}

// NewRefreshResolver constructs the deterministic stub resolver suited to
// the contract test surface. Production callers should use
// NewRefreshResolverWithConfig with a real Kratos client + alkemio re-stamper.
func NewRefreshResolver() *RefreshResolver {
	return NewRefreshResolverWithConfig(RefreshResolverConfig{})
}

// NewRefreshResolverWithConfig constructs the resolver from cfg. When either
// transport is nil a shared deterministic stub is wired so the test surface
// can drive the magic-identity-id contract.
func NewRefreshResolverWithConfig(cfg RefreshResolverConfig) *RefreshResolver {
	kratos := cfg.Kratos
	alkemio := cfg.Alkemio
	if kratos == nil || alkemio == nil {
		shared := &sync.Map{}
		if kratos == nil {
			kratos = &stubKratosReader{restamped: shared}
		}
		if alkemio == nil {
			alkemio = &stubReStamper{restamped: shared}
		}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RefreshResolver{kratos: kratos, alkemio: alkemio, logger: logger}
}

// Resolve attempts to return the alkemio_actor_id for the given Kratos
// identity, calling the alkemio-server identity re-stamp endpoint once on
// cache miss. Returns "" + nil when scope omits `alkemio` (no-op path).
// Returns ErrRefreshTemporarilyUnavailable when the claim is still absent
// after re-resolution.
func (r *RefreshResolver) Resolve(
	ctx context.Context, identityID string, grantedScope []string,
) (string, error) {
	if !slices.Contains(grantedScope, "alkemio") {
		return "", nil
	}

	actorID, err := r.kratos.ReadAlkemioActorID(ctx, identityID)
	if err != nil {
		r.logRefreshIssue("kratos read failed before re-stamp", identityID, err)
		return "", ErrRefreshTemporarilyUnavailable
	}
	if actorID != "" {
		return actorID, nil
	}

	if err := r.alkemio.ReStamp(ctx, identityID); err != nil {
		r.logRefreshIssue("alkemio re-stamp failed", identityID, err)
		return "", ErrRefreshTemporarilyUnavailable
	}

	actorID, err = r.kratos.ReadAlkemioActorID(ctx, identityID)
	if err != nil {
		r.logRefreshIssue("kratos read failed after re-stamp", identityID, err)
		return "", ErrRefreshTemporarilyUnavailable
	}
	if actorID == "" {
		return "", ErrRefreshTemporarilyUnavailable
	}
	return actorID, nil
}

func (r *RefreshResolver) logRefreshIssue(msg, identityID string, err error) {
	if r == nil || r.logger == nil {
		return
	}
	r.logger.Warn(msg, zap.String("identity_id", maskIdentityID(identityID)), zap.Error(err))
}

// stubKratosReader returns deterministic responses keyed off the contract
// test's magic identity ids. Mirrors the pattern in challenge/stub.go for
// the login/consent surface.
type stubKratosReader struct {
	restamped *sync.Map
}

// ReadAlkemioActorID implements KratosIdentityReader for the stub surface.
func (s *stubKratosReader) ReadAlkemioActorID(_ context.Context, identityID string) (string, error) {
	switch identityID {
	case "kratos-happy":
		return "actor-happy", nil
	case "kratos-miss-then-present":
		if _, ok := s.restamped.Load(identityID); ok {
			return "actor-miss-then-present", nil
		}
		return "", nil
	case "kratos-permanent-miss":
		return "", nil
	default:
		return "", nil
	}
}

// stubReStamper records re-stamp calls in the shared sync.Map so the
// stubKratosReader can switch its response after the call. Exception:
// `kratos-permanent-miss` is recorded as a successful stamp but the reader
// continues to return "" so the caller surfaces ErrRefreshTemporarilyUnavailable.
type stubReStamper struct {
	restamped *sync.Map
}

// ReStamp implements AlkemioReStamper for the stub surface.
func (s *stubReStamper) ReStamp(_ context.Context, identityID string) error {
	if identityID == "kratos-permanent-miss" {
		return nil
	}
	s.restamped.Store(identityID, struct{}{})
	return nil
}
