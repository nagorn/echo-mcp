package control

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"echo-mcp/internal/httpserver"
	"echo-mcp/internal/state"
)

func TestToolAdapterConfiguresResponseRuleThroughControlPlane(t *testing.T) {
	store := state.New()
	plane := New(store)
	adapter := NewToolAdapter(plane)

	result, err := adapter.ConfigureResponseRule(ConfigureResponseRuleCommand{
		RuleID:     "rule-payment-ok",
		Method:     http.MethodGet,
		Path:       "/payments/123",
		StatusCode: http.StatusAccepted,
		Body:       `{"status":"accepted"}`,
	})
	if err != nil {
		t.Fatalf("ConfigureResponseRule() error = %v", err)
	}
	if !result.Configured {
		t.Fatal("Configured = false, want true")
	}
	if result.RuleID != "rule-payment-ok" {
		t.Fatalf("RuleID = %q, want %q", result.RuleID, "rule-payment-ok")
	}

	matched, ok := store.MatchResponseRule(http.MethodGet, "/payments/123")
	if !ok {
		t.Fatal("MatchResponseRule() ok = false, want true")
	}
	if matched.ID != "rule-payment-ok" {
		t.Fatalf("matched rule ID = %q, want %q", matched.ID, "rule-payment-ok")
	}
	if matched.StatusCode != http.StatusAccepted {
		t.Fatalf("matched status = %d, want %d", matched.StatusCode, http.StatusAccepted)
	}
}

func TestToolAdapterConfiguredRuleIsConsumedByRESTDataPlane(t *testing.T) {
	store := state.New()
	plane := New(store)
	adapter := NewToolAdapter(plane)

	_, err := adapter.ConfigureResponseRule(ConfigureResponseRuleCommand{
		RuleID:     "rule-payment-ok",
		Method:     http.MethodGet,
		Path:       "/payments/123",
		StatusCode: http.StatusCreated,
		Body:       `{"status":"created"}`,
	})
	if err != nil {
		t.Fatalf("ConfigureResponseRule() error = %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	request := httptest.NewRequest(http.MethodGet, "/payments/123", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if got := response.Body.String(); got != `{"status":"created"}` {
		t.Fatalf("body = %q, want %q", got, `{"status":"created"}`)
	}

	observations := store.Observations()
	if len(observations) != 1 {
		t.Fatalf("len(Observations()) = %d, want 1", len(observations))
	}
	observation := observations[0]
	if observation.RequestMethod != http.MethodGet {
		t.Fatalf("RequestMethod = %q, want %q", observation.RequestMethod, http.MethodGet)
	}
	if observation.RequestPath != "/payments/123" {
		t.Fatalf("RequestPath = %q, want %q", observation.RequestPath, "/payments/123")
	}
	if observation.MatchedRuleID != "rule-payment-ok" {
		t.Fatalf("MatchedRuleID = %q, want %q", observation.MatchedRuleID, "rule-payment-ok")
	}
	if observation.OutcomeStatusCode != http.StatusCreated {
		t.Fatalf("OutcomeStatusCode = %d, want %d", observation.OutcomeStatusCode, http.StatusCreated)
	}
}
