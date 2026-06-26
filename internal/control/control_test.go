package control

import (
	"net/http"
	"testing"

	"echo-mcp/internal/state"
)

func TestLocalPlaneIdentifiesMCPControlPlane(t *testing.T) {
	var plane Plane = New(state.New())

	if got := plane.Protocol(); got != "mcp" {
		t.Fatalf("Protocol() = %q, want %q", got, "mcp")
	}
}

func TestLocalPlaneConfiguresResponseRuleInMemory(t *testing.T) {
	store := state.New()
	plane := New(store)

	rule := state.ResponseRule{
		ID:         "rule-payment-ok",
		Method:     http.MethodGet,
		Path:       "/payments/123",
		StatusCode: http.StatusAccepted,
		Body:       `{"status":"accepted"}`,
	}

	if err := plane.ConfigureResponseRule(rule); err != nil {
		t.Fatalf("ConfigureResponseRule() error = %v", err)
	}

	matched, ok := store.MatchResponseRule(http.MethodGet, "/payments/123")
	if !ok {
		t.Fatal("MatchResponseRule() ok = false, want true")
	}
	if matched != rule {
		t.Fatalf("matched rule = %+v, want %+v", matched, rule)
	}
}
