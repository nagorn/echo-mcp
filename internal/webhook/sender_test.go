package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestLoadEndpointFromEnvironmentRegistersOneEndpoint(t *testing.T) {
	endpoints, err := LoadEndpointFromEnvironment(func(key string) string {
		switch key {
		case EnvEndpointName:
			return "payment-events"
		case EnvEndpointAddress:
			return "http://127.0.0.1:18080/webhooks/payments"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("LoadEndpointFromEnvironment() error = %v", err)
	}

	endpoint, ok := endpoints.Resolve("payment-events")
	if !ok {
		t.Fatal("Resolve(payment-events) ok = false, want true")
	}
	if endpoint.Address != "http://127.0.0.1:18080/webhooks/payments" {
		t.Fatalf("endpoint.Address = %q", endpoint.Address)
	}
}

func TestLoadEndpointFromEnvironmentRejectsPartialConfiguration(t *testing.T) {
	_, err := LoadEndpointFromEnvironment(func(key string) string {
		if key == EnvEndpointName {
			return "payment-events"
		}
		return ""
	})
	if err == nil {
		t.Fatal("LoadEndpointFromEnvironment() error = nil, want partial configuration rejection")
	}
}

func TestSenderPostsJSONToConfiguredEndpoint(t *testing.T) {
	received := make(chan webhookRequest, 1)
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Decode(body) error = %v", err)
		}
		if err := r.Body.Close(); err != nil {
			t.Errorf("Close(body) error = %v", err)
		}
		received <- webhookRequest{
			Method:      r.Method,
			ContentType: r.Header.Get("Content-Type"),
			Body:        body,
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}

	endpoints := NewEndpoints(Endpoint{Name: "payment-events", Address: "http://application.test/webhooks/payments"})
	sender := NewSender(endpoints, client)

	delivery, err := sender.Send(context.Background(), Event{
		EventID:      "evt_payment_failed_001",
		EndpointName: "payment-events",
		Body: map[string]any{
			"type": "payment.failed",
			"data": map[string]any{
				"object": map[string]any{"id": "pay_123"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if delivery.Outcome != OutcomeResponseReceived {
		t.Fatalf("delivery.Outcome = %q, want %q", delivery.Outcome, OutcomeResponseReceived)
	}
	if delivery.StatusCode != http.StatusNoContent {
		t.Fatalf("delivery.StatusCode = %d, want %d", delivery.StatusCode, http.StatusNoContent)
	}

	request := <-received
	if request.Method != http.MethodPost {
		t.Fatalf("request.Method = %q, want %q", request.Method, http.MethodPost)
	}
	if request.ContentType != "application/json" {
		t.Fatalf("request.ContentType = %q, want %q", request.ContentType, "application/json")
	}
	if got := request.Body["type"]; got != "payment.failed" {
		t.Fatalf("request body type = %v", got)
	}
}

func TestSenderReturnsTransportErrorOutcome(t *testing.T) {
	endpoints := NewEndpoints(Endpoint{Name: "payment-events", Address: "http://127.0.0.1:1/webhooks/payments"})
	sender := NewSender(endpoints, &http.Client{})

	delivery, err := sender.Send(context.Background(), Event{
		EventID:      "evt_payment_failed_001",
		EndpointName: "payment-events",
		Body:         map[string]any{"type": "payment.failed"},
	})
	if err != nil {
		t.Fatalf("Send() error = %v, want transport error as delivery outcome", err)
	}
	if delivery.Outcome != OutcomeTransportError {
		t.Fatalf("delivery.Outcome = %q, want %q", delivery.Outcome, OutcomeTransportError)
	}
	if delivery.Error == "" {
		t.Fatal("delivery.Error is empty, want transport error information")
	}
}

type webhookRequest struct {
	Method      string
	ContentType string
	Body        map[string]any
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
