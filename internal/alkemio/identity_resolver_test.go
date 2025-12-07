package alkemio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	validKratosID     = "a30b6fbc-04a3-4ff7-8db0-94589f5cb918"
	validUserID       = "ec1c8293-8b91-4de0-9033-9c7de3b5a963"
	validAgentID      = "6f4ed2d2-0ad0-4b83-8d43-8d9b9b4970b3"
	altValidUserID    = "02d7c369-99d9-4a33-83ba-6df2a4bfb7c8"
	altValidAgentID   = "3547bbca-54de-4d64-926d-e5dd51e8e37d"
	invalidUUIDText   = "not-a-uuid"
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
	errFmtNewResolver = "new resolver: %v"
	errFmtUnexpected  = "unexpected error: %v"
)

func TestResolveSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["authenticationId"] != validKratosID {
			t.Fatalf("unexpected authentication id %s", payload["authenticationId"])
		}
		_ = r.Body.Close()
		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(map[string]string{"userId": validUserID, "agentId": validAgentID})
	}))
	defer server.Close()

	resolver, err := NewIdentityResolver(Config{
		BaseURL:     server.URL,
		ResolvePath: "/rest/internal/identity/resolve",
		Timeout:     time.Second,
		MaxRetries:  3,
	})
	if err != nil {
		t.Fatalf(errFmtNewResolver, err)
	}

	mapping, err := resolver.Resolve(context.Background(), validKratosID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if mapping.UserID != validUserID {
		t.Fatalf("unexpected user id %s", mapping.UserID)
	}
	if mapping.AgentID != validAgentID {
		t.Fatalf("unexpected agent id %s", mapping.AgentID)
	}
}

func TestResolveNotFound(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resolver, err := NewIdentityResolver(Config{BaseURL: server.URL, ResolvePath: "/", Timeout: time.Second, MaxRetries: 3})
	if err != nil {
		t.Fatalf(errFmtNewResolver, err)
	}

	_, err = resolver.Resolve(context.Background(), validKratosID)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected single attempt, got %d", attempts)
	}
}

func TestResolveRetriesOnServerError(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(map[string]string{"userId": altValidUserID, "agentId": altValidAgentID})
	}))
	defer server.Close()

	resolver, err := NewIdentityResolver(Config{BaseURL: server.URL, ResolvePath: "/", Timeout: 2 * time.Second, MaxRetries: 3})
	if err != nil {
		t.Fatalf(errFmtNewResolver, err)
	}

	mapping, err := resolver.Resolve(context.Background(), validKratosID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if mapping.UserID != altValidUserID {
		t.Fatalf("unexpected user id %s", mapping.UserID)
	}
	if mapping.AgentID != altValidAgentID {
		t.Fatalf("unexpected agent id %s", mapping.AgentID)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestResolveFailsAfterExhaustingRetries(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	resolver, err := NewIdentityResolver(Config{BaseURL: server.URL, ResolvePath: "/", Timeout: time.Second, MaxRetries: 2})
	if err != nil {
		t.Fatalf(errFmtNewResolver, err)
	}

	_, err = resolver.Resolve(context.Background(), validKratosID)
	if err == nil {
		t.Fatalf("expected failure after retries")
	}
}

func TestResolveHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resolver, err := NewIdentityResolver(Config{BaseURL: server.URL, ResolvePath: "/", Timeout: 500 * time.Millisecond, MaxRetries: 5})
	if err != nil {
		t.Fatalf(errFmtNewResolver, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err = resolver.Resolve(ctx, validKratosID)
	if err == nil {
		t.Fatalf("expected context error")
	}
}

func TestNewIdentityResolverRequiresBaseURL(t *testing.T) {
	if _, err := NewIdentityResolver(Config{}); err == nil {
		t.Fatalf("expected error for empty base url")
	}
}

func TestResolveTimesOut(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	resolver, err := NewIdentityResolver(Config{
		BaseURL:     server.URL,
		ResolvePath: "/",
		Timeout:     20 * time.Millisecond,
		MaxRetries:  1,
	})
	if err != nil {
		t.Fatalf(errFmtNewResolver, err)
	}

	_, err = resolver.Resolve(context.Background(), validKratosID)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestResolveRejectsInvalidAuthenticationID(t *testing.T) {
	resolver, err := NewIdentityResolver(Config{
		BaseURL:     "http://example.com",
		ResolvePath: "/",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf(errFmtNewResolver, err)
	}

	_, err = resolver.Resolve(context.Background(), invalidUUIDText)
	if err == nil {
		t.Fatalf("expected error for invalid uuid")
	}
	if !strings.Contains(err.Error(), "valid uuid") {
		t.Fatalf(errFmtUnexpected, err)
	}
}

func TestResolveFailsWhenResponseUserIDIsNotUUID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(map[string]string{"userId": invalidUUIDText, "agentId": validAgentID})
	}))
	defer server.Close()

	resolver, err := NewIdentityResolver(Config{
		BaseURL:     server.URL,
		ResolvePath: "/",
		Timeout:     time.Second,
		MaxRetries:  1,
	})
	if err != nil {
		t.Fatalf(errFmtNewResolver, err)
	}

	_, err = resolver.Resolve(context.Background(), validKratosID)
	if err == nil {
		t.Fatalf("expected error for invalid userId")
	}
	if !strings.Contains(err.Error(), "userId must be a valid uuid") {
		t.Fatalf(errFmtUnexpected, err)
	}
}

func TestResolveFailsWhenResponseAgentIDIsInvalid(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(map[string]string{"userId": validUserID, "agentId": invalidUUIDText})
	}))
	defer server.Close()

	resolver, err := NewIdentityResolver(Config{
		BaseURL:     server.URL,
		ResolvePath: "/",
		Timeout:     time.Second,
		MaxRetries:  1,
	})
	if err != nil {
		t.Fatalf(errFmtNewResolver, err)
	}

	_, err = resolver.Resolve(context.Background(), validKratosID)
	if err == nil {
		t.Fatalf("expected error for invalid agentId")
	}
	if !strings.Contains(err.Error(), "agentId must be a valid uuid") {
		t.Fatalf(errFmtUnexpected, err)
	}
}

func TestResolveFailsWhenAgentIDMissing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(map[string]string{"userId": validUserID})
	}))
	defer server.Close()

	resolver, err := NewIdentityResolver(Config{
		BaseURL:     server.URL,
		ResolvePath: "/",
		Timeout:     time.Second,
		MaxRetries:  1,
	})
	if err != nil {
		t.Fatalf(errFmtNewResolver, err)
	}

	_, err = resolver.Resolve(context.Background(), validKratosID)
	if err == nil {
		t.Fatalf("expected error for missing agentId")
	}
	if !strings.Contains(err.Error(), "missing agentId") {
		t.Fatalf(errFmtUnexpected, err)
	}
}
