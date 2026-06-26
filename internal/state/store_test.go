package state

import (
	"net/http"
	"testing"
)

func TestStoreStartsAtGenerationZero(t *testing.T) {
	store := New()

	if got := store.Generation(); got != 0 {
		t.Fatalf("Generation() = %d, want 0", got)
	}
}

func TestStoreResetAdvancesGeneration(t *testing.T) {
	store := New()

	store.Reset()

	if got := store.Generation(); got != 1 {
		t.Fatalf("Generation() = %d, want 1", got)
	}
}

func TestStoreMatchesConfiguredResponseRule(t *testing.T) {
	store := New()
	rule := ResponseRule{
		ID:         "rule-payment-ok",
		Method:     http.MethodGet,
		Path:       "/payments/123",
		StatusCode: http.StatusAccepted,
		Body:       `{"status":"accepted"}`,
	}

	store.ConfigureResponseRule(rule)

	matched, ok := store.MatchResponseRule(http.MethodGet, "/payments/123")
	if !ok {
		t.Fatal("MatchResponseRule() ok = false, want true")
	}
	if matched != rule {
		t.Fatalf("matched rule = %+v, want %+v", matched, rule)
	}
}

func TestStoreDoesNotMatchDifferentRequest(t *testing.T) {
	store := New()
	store.ConfigureResponseRule(ResponseRule{
		ID:         "rule-payment-ok",
		Method:     http.MethodGet,
		Path:       "/payments/123",
		StatusCode: http.StatusAccepted,
		Body:       `{"status":"accepted"}`,
	})

	if _, ok := store.MatchResponseRule(http.MethodGet, "/payments/999"); ok {
		t.Fatal("MatchResponseRule() ok = true, want false")
	}
}

func TestStoreRecordsObservation(t *testing.T) {
	store := New()

	store.RecordObservation(Observation{
		RequestPath:       "/payments/123",
		MatchedRuleID:     "rule-payment-ok",
		OutcomeStatusCode: http.StatusAccepted,
	})

	observations := store.Observations()
	if len(observations) != 1 {
		t.Fatalf("len(Observations()) = %d, want 1", len(observations))
	}
	if observations[0].MatchedRuleID != "rule-payment-ok" {
		t.Fatalf("MatchedRuleID = %q, want %q", observations[0].MatchedRuleID, "rule-payment-ok")
	}
}
