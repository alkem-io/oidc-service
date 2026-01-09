package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	kratosClient "github.com/ory/client-go"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/middleware"
)

// maxWebhookBodySize limits request body size to prevent large payload attacks.
const maxWebhookBodySize = 4 * 1024 // 4KB

// AlkemioResolver resolves Alkemio identity mappings from authentication IDs.
type AlkemioResolver interface {
	// Resolve looks up the Alkemio identity mapping for the given Kratos authentication ID.
	Resolve(ctx context.Context, authenticationID string) (*alkemio.IdentityMapping, error)
}

// KratosAdmin provides access to Kratos Admin API for identity operations.
type KratosAdmin interface {
	// PatchIdentity updates identity metadata using JSON Patch.
	PatchIdentity(ctx context.Context, identityID string, patches []kratosClient.JsonPatch) error
}

// Handler processes Kratos webhook requests.
type Handler struct {
	resolver AlkemioResolver
	kratos   KratosAdmin
	logger   *zap.Logger
}

// HandlerConfig holds the dependencies for creating a Handler.
type HandlerConfig struct {
	Resolver AlkemioResolver
	Kratos   KratosAdmin
	Logger   *zap.Logger
}

// NewHandler creates a new webhook handler.
func NewHandler(cfg HandlerConfig) (*Handler, error) {
	if cfg.Resolver == nil {
		return nil, &ConfigError{Field: "Resolver", Message: "resolver is required"}
	}

	if cfg.Kratos == nil {
		return nil, &ConfigError{Field: "Kratos", Message: "kratos admin client is required"}
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Handler{
		resolver: cfg.Resolver,
		kratos:   cfg.Kratos,
		logger:   logger,
	}, nil
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}

// loggerFromCtx returns the request-scoped logger from context, falling back to the static logger.
func (h *Handler) loggerFromCtx(ctx context.Context) *zap.Logger {
	if logger := middleware.Logger(ctx); logger != nil {
		return logger
	}
	return h.logger
}

// PostLogin handles POST /webhooks/kratos/post-login requests.
// It resolves Alkemio claims and patches the identity via Kratos Admin API.
func (h *Handler) PostLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := h.loggerFromCtx(ctx)

	identityID, mapping, err := h.resolveIdentity(ctx, r)
	if err != nil {
		h.handleError(ctx, w, err)
		return
	}

	// Patch identity metadata via Kratos Admin API.
	// Use "add" (not "replace") because it creates the path if missing and
	// replaces if it exists, handling both new and existing identities.
	patches := []kratosClient.JsonPatch{
		{
			Op:   "add",
			Path: "/metadata_public",
			Value: map[string]interface{}{
				"alkemio_actor_id": mapping.UserID,
				"alkemio_agent_id": mapping.AgentID,
			},
		},
	}

	if err := h.kratos.PatchIdentity(ctx, identityID, patches); err != nil {
		logger.Error("failed to patch identity metadata",
			zap.String("identity_id", maskID(identityID)),
			zap.Error(err),
		)
		h.writeError(ctx, w, http.StatusInternalServerError, "patch_failed", "failed to update identity metadata")
		return
	}

	logger.Info("patched identity metadata",
		zap.String("identity_id", maskID(identityID)),
		zap.String("actor_id", maskID(mapping.UserID)),
		zap.String("agent_id", maskID(mapping.AgentID)),
	)

	// Return empty success response (Kratos doesn't parse this for login)
	h.writeJSON(ctx, w, http.StatusOK, map[string]string{"status": "ok"})
}

// resolveIdentity parses the request and resolves Alkemio identity.
func (h *Handler) resolveIdentity(ctx context.Context, r *http.Request) (string, *alkemio.IdentityMapping, error) {
	logger := h.loggerFromCtx(ctx)

	var req Request
	if err := json.NewDecoder(io.LimitReader(r.Body, maxWebhookBodySize)).Decode(&req); err != nil {
		logger.Warn("failed to decode webhook request", zap.Error(err))
		return "", nil, &webhookError{
			status:  http.StatusBadRequest,
			code:    "bad_request",
			message: "invalid JSON payload",
		}
	}

	identityID := strings.TrimSpace(req.IdentityID)
	if identityID == "" {
		logger.Warn("webhook request missing identity_id")
		return "", nil, &webhookError{
			status:  http.StatusBadRequest,
			code:    "bad_request",
			message: "missing identity_id",
		}
	}

	logger.Debug("processing webhook request",
		zap.String("identity_id", maskID(identityID)),
	)

	mapping, err := h.resolver.Resolve(ctx, identityID)
	if err != nil {
		logger.Error("failed to resolve alkemio identity",
			zap.String("identity_id", maskID(identityID)),
			zap.String("error_type", classifyError(err)),
			zap.Error(err),
		)
		// Distinguish "not found" from other resolution errors for clearer diagnostics
		if errors.Is(err, alkemio.ErrNotFound) {
			return "", nil, &webhookError{
				status:  http.StatusNotFound,
				code:    "identity_not_found",
				message: "Alkemio identity mapping not found",
			}
		}
		return "", nil, &webhookError{
			status:  http.StatusInternalServerError,
			code:    "resolution_failed",
			message: "failed to resolve Alkemio identity",
		}
	}

	if err := h.validateMapping(mapping); err != nil {
		logger.Error("resolved mapping has invalid UUIDs",
			zap.String("identity_id", maskID(identityID)),
			zap.Error(err),
		)
		return "", nil, &webhookError{
			status:  http.StatusInternalServerError,
			code:    "resolution_failed",
			message: "resolved identity has invalid format",
		}
	}

	return identityID, mapping, nil
}

func (h *Handler) validateMapping(mapping *alkemio.IdentityMapping) error {
	if mapping == nil {
		return &ValidationError{Field: "mapping", Message: "mapping is nil"}
	}

	if _, err := uuid.Parse(mapping.UserID); err != nil {
		return &ValidationError{Field: "actor_id", Message: "invalid UUID format"}
	}

	if _, err := uuid.Parse(mapping.AgentID); err != nil {
		return &ValidationError{Field: "agent_id", Message: "invalid UUID format"}
	}

	return nil
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// webhookError represents an error that should be returned to the client.
type webhookError struct {
	status  int
	code    string
	message string
}

// Error implements the error interface, returning a formatted error message.
func (e *webhookError) Error() string {
	return fmt.Sprintf("%s: %s", e.code, e.message)
}

func (h *Handler) handleError(ctx context.Context, w http.ResponseWriter, err error) {
	var we *webhookError
	if errors.As(err, &we) {
		h.writeError(ctx, w, we.status, we.code, we.message)
		return
	}
	h.writeError(ctx, w, http.StatusInternalServerError, "internal_error", "unexpected error")
}

func (h *Handler) writeJSON(ctx context.Context, w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.loggerFromCtx(ctx).Error("failed to encode response", zap.Error(err))
	}
}

func (h *Handler) writeError(ctx context.Context, w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(ctx, w, status, ErrorResponse{
		Error:   code,
		Message: message,
	})
}

func maskID(id string) string {
	if len(id) <= 8 {
		return "***"
	}
	return id[:8] + "..."
}

func classifyError(err error) string {
	if err == nil {
		return "none"
	}
	if errors.Is(err, alkemio.ErrNotFound) {
		return "not_found"
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "timeout"
	}
	return "error"
}
