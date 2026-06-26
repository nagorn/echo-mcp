package contract

import (
	"net/http"
	"path/filepath"
	"testing"

	"echo-mcp/internal/state"
)

const paymentIntentDeclinedBody = `{"error":{"type":"card_error","code":"card_declined","decline_code":"generic_decline","message":"Your card was declined.","payment_intent":{"id":"pi_123","object":"payment_intent","status":"requires_payment_method"}}}`

func TestLoadOpenAPIFileAcceptsOpenAPI30JSON(t *testing.T) {
	validator, err := LoadOpenAPIFile(filepath.Join("testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}
	if validator == nil {
		t.Fatal("validator = nil")
	}
}

func TestOpenAPIValidatorAcceptsMatchingPaymentIntentFailure(t *testing.T) {
	validator, err := LoadOpenAPIFile(filepath.Join("testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}

	err = validator.ValidateResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusPaymentRequired,
		ContentType: "application/json",
		Body:        paymentIntentDeclinedBody,
	})
	if err != nil {
		t.Fatalf("ValidateResponseRule() error = %v", err)
	}
}

func TestOpenAPIValidatorRejectsUnsupportedResponseStatus(t *testing.T) {
	validator, err := LoadOpenAPIFile(filepath.Join("testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}

	err = validator.ValidateResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        paymentIntentDeclinedBody,
	})
	if err == nil {
		t.Fatal("ValidateResponseRule() error = nil, want response status rejection")
	}
}

func TestOpenAPIValidatorRejectsMissingRequiredResponseField(t *testing.T) {
	validator, err := LoadOpenAPIFile(filepath.Join("testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}

	err = validator.ValidateResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusPaymentRequired,
		ContentType: "application/json",
		Body:        `{"error":{"type":"card_error","code":"card_declined"}}`,
	})
	if err == nil {
		t.Fatal("ValidateResponseRule() error = nil, want missing required field rejection")
	}
}

func TestOpenAPIValidatorRejectsMultipleJSONValues(t *testing.T) {
	validator, err := LoadOpenAPIFile(filepath.Join("testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}

	err = validator.ValidateResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusPaymentRequired,
		ContentType: "application/json",
		Body:        paymentIntentDeclinedBody + ` {}`,
	})
	if err == nil {
		t.Fatal("ValidateResponseRule() error = nil, want multiple JSON values rejection")
	}
}
