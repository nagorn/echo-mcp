package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"echo-mcp/internal/state"
)

func TestServerReturnsNotImplementedWhenNoBehaviorIsConfigured(t *testing.T) {
	store := state.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := New(store, logger)

	request := httptest.NewRequest(http.MethodGet, "/payments/123", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotImplemented)
	}
	body := response.Body.String()
	if !strings.Contains(body, "behavior matching") || !strings.Contains(body, "not implemented") {
		t.Fatalf("body = %q, want clear unmatched behavior message", body)
	}
}

func TestServerReturnsConfiguredResponseForMatchingRequest(t *testing.T) {
	store := state.New()
	store.ConfigureResponseRule(state.ResponseRule{
		ID:         "rule-payment-ok",
		Method:     http.MethodGet,
		Path:       "/payments/123",
		StatusCode: http.StatusAccepted,
		Body:       `{"status":"accepted"}`,
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := New(store, logger)

	request := httptest.NewRequest(http.MethodGet, "/payments/123", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if got := response.Body.String(); got != `{"status":"accepted"}` {
		t.Fatalf("body = %q, want %q", got, `{"status":"accepted"}`)
	}

	observations := store.Observations()
	if len(observations) != 1 {
		t.Fatalf("len(Observations()) = %d, want 1", len(observations))
	}
	observation := observations[0]
	if observation.RequestPath != "/payments/123" {
		t.Fatalf("RequestPath = %q, want %q", observation.RequestPath, "/payments/123")
	}
	if observation.MatchedRuleID != "rule-payment-ok" {
		t.Fatalf("MatchedRuleID = %q, want %q", observation.MatchedRuleID, "rule-payment-ok")
	}
	if observation.OutcomeStatusCode != http.StatusAccepted {
		t.Fatalf("OutcomeStatusCode = %d, want %d", observation.OutcomeStatusCode, http.StatusAccepted)
	}
}

func TestServerReturnsConfiguredContentTypeForMatchingRequest(t *testing.T) {
	store := state.New()
	store.ConfigureResponseRule(state.ResponseRule{
		ID:          "rule-payment-ok",
		Method:      http.MethodGet,
		Path:        "/payments/123",
		StatusCode:  http.StatusAccepted,
		ContentType: "application/json",
		Body:        `{"status":"accepted"}`,
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := New(store, logger)

	request := httptest.NewRequest(http.MethodGet, "/payments/123", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}
}

func TestServerDoesNotUseConfiguredResponseForNonMatchingRequest(t *testing.T) {
	store := state.New()
	store.ConfigureResponseRule(state.ResponseRule{
		ID:         "rule-payment-ok",
		Method:     http.MethodGet,
		Path:       "/payments/123",
		StatusCode: http.StatusAccepted,
		Body:       `{"status":"accepted"}`,
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := New(store, logger)

	request := httptest.NewRequest(http.MethodGet, "/payments/999", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code == http.StatusAccepted {
		t.Fatalf("status = %d, want non-matching request not to receive configured response", response.Code)
	}
}
