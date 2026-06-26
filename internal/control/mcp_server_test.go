package control

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"echo-mcp/internal/contract"
	"echo-mcp/internal/httpserver"
	"echo-mcp/internal/state"
	"echo-mcp/internal/webhook"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const paymentIntentDeclinedBody = `{"error":{"type":"card_error","code":"card_declined","decline_code":"generic_decline","message":"Your card was declined.","payment_intent":{"id":"pi_123","object":"payment_intent","status":"requires_payment_method"}}}`

func TestMCPConfigureBehaviorDrivesRESTDataPlaneAndReportsObservations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	store := state.New()
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(New(store)))

	configureResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "rule-payment-ok",
			"match": map[string]any{
				"method": http.MethodGet,
				"path":   "/payments/123",
			},
			"outcome": map[string]any{
				"type":        "http_response",
				"status_code": http.StatusAccepted,
				"body":        `{"status":"accepted"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, configureResult)
	configured := decodeStructuredContent[configureBehaviorOutput](t, configureResult)
	if !configured.Configured {
		t.Fatal("configured = false, want true")
	}
	if configured.BehaviorID != "rule-payment-ok" {
		t.Fatalf("behavior_id = %q, want %q", configured.BehaviorID, "rule-payment-ok")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	request := httptest.NewRequest(http.MethodGet, "/payments/123", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if got := response.Body.String(); got != `{"status":"accepted"}` {
		t.Fatalf("body = %q, want %q", got, `{"status":"accepted"}`)
	}

	observationsResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_observations",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_observations CallTool() error = %v", err)
	}
	assertToolSuccess(t, observationsResult)
	observations := decodeStructuredContent[getObservationsOutput](t, observationsResult)
	if len(observations.Observations) != 1 {
		t.Fatalf("len(observations) = %d, want 1", len(observations.Observations))
	}

	observation := observations.Observations[0]
	if observation.Request.Method != http.MethodGet {
		t.Fatalf("request.method = %q, want %q", observation.Request.Method, http.MethodGet)
	}
	if observation.Request.Path != "/payments/123" {
		t.Fatalf("request.path = %q, want %q", observation.Request.Path, "/payments/123")
	}
	if observation.Selection.MatchedBehaviorID != "rule-payment-ok" {
		t.Fatalf("matched_behavior_id = %q, want %q", observation.Selection.MatchedBehaviorID, "rule-payment-ok")
	}
	if want := []string{"method", "path"}; !equalStrings(observation.Selection.MatchedOn, want) {
		t.Fatalf("matched_on = %+v, want %+v", observation.Selection.MatchedOn, want)
	}
	if observation.Outcome.Type != "http_response" {
		t.Fatalf("outcome.type = %q, want %q", observation.Outcome.Type, "http_response")
	}
	if observation.Outcome.StatusCode != http.StatusAccepted {
		t.Fatalf("outcome.status_code = %d, want %d", observation.Outcome.StatusCode, http.StatusAccepted)
	}
}

func TestMCPResetClearsConfiguredBehaviorAndObservations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	store := state.New()
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(New(store)))

	_, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "rule-payment-ok",
			"match": map[string]any{
				"method": http.MethodGet,
				"path":   "/payments/123",
			},
			"outcome": map[string]any{
				"type":        "http_response",
				"status_code": http.StatusAccepted,
				"body":        `{"status":"accepted"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	server.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/payments/123", nil))

	resetResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "reset",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("reset CallTool() error = %v", err)
	}
	assertToolSuccess(t, resetResult)
	reset := decodeStructuredContent[resetOutput](t, resetResult)
	if !reset.Reset {
		t.Fatal("reset = false, want true")
	}
	if want := []string{"behavior", "observations", "webhook_deliveries"}; !equalStrings(reset.Cleared, want) {
		t.Fatalf("cleared = %+v, want %+v", reset.Cleared, want)
	}

	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/payments/123", nil))
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("status after reset = %d, want %d", response.Code, http.StatusNotImplemented)
	}

	observationsResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_observations",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_observations CallTool() error = %v", err)
	}
	assertToolSuccess(t, observationsResult)
	observations := decodeStructuredContent[getObservationsOutput](t, observationsResult)
	if len(observations.Observations) != 0 {
		t.Fatalf("len(observations) = %d, want 0", len(observations.Observations))
	}
}

func TestMCPConfigureBehaviorValidatesAgainstOpenAPIContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	validator, err := contract.LoadOpenAPIFile(filepath.Join("..", "contract", "testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}
	store := state.New()
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithValidator(store, validator)))

	configureResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "stripe-like-paymentintent-card-declined",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "application/json",
				"body":         paymentIntentDeclinedBody,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, configureResult)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/payment_intents/pi_123/confirm", nil))

	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusPaymentRequired)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}
}

func TestMCPConfigureBehaviorRejectsContractViolationWithoutReplacingCurrentRule(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	validator, err := contract.LoadOpenAPIFile(filepath.Join("..", "contract", "testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}
	store := state.New()
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithValidator(store, validator)))

	validResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "stripe-like-paymentintent-card-declined",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "application/json",
				"body":         paymentIntentDeclinedBody,
			},
		},
	})
	if err != nil {
		t.Fatalf("valid configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, validResult)

	invalidResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "stripe-like-paymentintent-invalid",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusOK,
				"content_type": "application/json",
				"body":         `{"ok":true}`,
			},
		},
	})
	if err == nil && (invalidResult == nil || !invalidResult.IsError) {
		t.Fatal("invalid configure_behavior succeeded, want MCP error")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/payment_intents/pi_123/confirm", nil))

	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("status after rejected rule = %d, want previous status %d", response.Code, http.StatusPaymentRequired)
	}
	if got := response.Body.String(); got != paymentIntentDeclinedBody {
		t.Fatalf("body after rejected rule = %q, want previous body %q", got, paymentIntentDeclinedBody)
	}
}

func TestMCPSendWebhookEventDeliversToConfiguredEndpointAndReportsObservation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	received := make(chan webhookRequest, 1)
	webhookClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
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

	store := state.New()
	endpoints := webhook.NewEndpoints(webhook.Endpoint{Name: "payment-events", Address: "http://application.test/webhooks/payments"})
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithWebhookSender(store, nil, webhook.NewSender(endpoints, webhookClient))))

	sendResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "send_webhook_event",
		Arguments: map[string]any{
			"event_id":      "evt_payment_failed_001",
			"endpoint_name": "payment-events",
			"request": map[string]any{
				"body": map[string]any{
					"type": "payment.failed",
					"data": map[string]any{
						"object": map[string]any{"id": "pay_123"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("send_webhook_event CallTool() error = %v", err)
	}
	assertToolSuccess(t, sendResult)
	delivery := decodeStructuredContent[sendWebhookEventOutput](t, sendResult)
	if !delivery.Attempted {
		t.Fatal("attempted = false, want true")
	}
	if delivery.EventID != "evt_payment_failed_001" {
		t.Fatalf("event_id = %q", delivery.EventID)
	}
	if delivery.EndpointName != "payment-events" {
		t.Fatalf("endpoint_name = %q", delivery.EndpointName)
	}
	if delivery.Delivery.Outcome != "response_received" {
		t.Fatalf("delivery.outcome = %q, want response_received", delivery.Delivery.Outcome)
	}
	if delivery.Delivery.StatusCode != http.StatusNoContent {
		t.Fatalf("delivery.status_code = %d, want %d", delivery.Delivery.StatusCode, http.StatusNoContent)
	}

	request := <-received
	if request.Method != http.MethodPost {
		t.Fatalf("request.Method = %q, want %q", request.Method, http.MethodPost)
	}
	if request.ContentType != "application/json" {
		t.Fatalf("request.ContentType = %q, want application/json", request.ContentType)
	}

	observationsResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_observations",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_observations CallTool() error = %v", err)
	}
	assertToolSuccess(t, observationsResult)
	observations := decodeStructuredContent[getObservationsOutput](t, observationsResult)
	if len(observations.WebhookDeliveries) != 1 {
		t.Fatalf("len(webhook_deliveries) = %d, want 1", len(observations.WebhookDeliveries))
	}
	webhookDelivery := observations.WebhookDeliveries[0]
	if webhookDelivery.EventID != "evt_payment_failed_001" {
		t.Fatalf("webhook event_id = %q", webhookDelivery.EventID)
	}
	if webhookDelivery.EndpointName != "payment-events" {
		t.Fatalf("webhook endpoint_name = %q", webhookDelivery.EndpointName)
	}
	if webhookDelivery.Method != http.MethodPost {
		t.Fatalf("webhook method = %q", webhookDelivery.Method)
	}
	if webhookDelivery.Outcome != "response_received" {
		t.Fatalf("webhook outcome = %q", webhookDelivery.Outcome)
	}
	if webhookDelivery.StatusCode != http.StatusNoContent {
		t.Fatalf("webhook status_code = %d", webhookDelivery.StatusCode)
	}
}

func TestMCPSendWebhookEventRejectsUnknownEndpointWithoutDelivery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	received := make(chan struct{}, 1)
	webhookClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		received <- struct{}{}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}

	store := state.New()
	endpoints := webhook.NewEndpoints(webhook.Endpoint{Name: "payment-events", Address: "http://application.test/webhooks/payments"})
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithWebhookSender(store, nil, webhook.NewSender(endpoints, webhookClient))))

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "send_webhook_event",
		Arguments: map[string]any{
			"event_id":         "evt_payment_failed_001",
			"endpoint_name":    "unknown-events",
			"endpoint_address": "http://application.test/webhooks/payments",
			"request": map[string]any{
				"body": map[string]any{"type": "payment.failed"},
			},
		},
	})
	if err == nil && (result == nil || !result.IsError) {
		t.Fatal("send_webhook_event succeeded, want unknown endpoint rejection")
	}

	select {
	case <-received:
		t.Fatal("application webhook endpoint received a request for unknown endpoint")
	default:
	}
}

func TestMCPResetClearsWebhookDeliveryObservations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	webhookClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}

	store := state.New()
	endpoints := webhook.NewEndpoints(webhook.Endpoint{Name: "payment-events", Address: "http://application.test/webhooks/payments"})
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithWebhookSender(store, nil, webhook.NewSender(endpoints, webhookClient))))

	_, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "send_webhook_event",
		Arguments: map[string]any{
			"event_id":      "evt_payment_failed_001",
			"endpoint_name": "payment-events",
			"request": map[string]any{
				"body": map[string]any{"type": "payment.failed"},
			},
		},
	})
	if err != nil {
		t.Fatalf("send_webhook_event CallTool() error = %v", err)
	}

	resetResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "reset",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("reset CallTool() error = %v", err)
	}
	assertToolSuccess(t, resetResult)

	observationsResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_observations",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_observations CallTool() error = %v", err)
	}
	assertToolSuccess(t, observationsResult)
	observations := decodeStructuredContent[getObservationsOutput](t, observationsResult)
	if len(observations.WebhookDeliveries) != 0 {
		t.Fatalf("len(webhook_deliveries) = %d, want 0", len(observations.WebhookDeliveries))
	}
}

func connectMCPTestClient(t *testing.T, ctx context.Context, server *mcp.Server) *mcp.ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server Connect() error = %v", err)
	}
	t.Cleanup(func() {
		if err := serverSession.Close(); err != nil {
			t.Fatalf("server session Close() error = %v", err)
		}
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "echo-mcp-test-client", Version: "v0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect() error = %v", err)
	}
	t.Cleanup(func() {
		if err := clientSession.Close(); err != nil {
			t.Fatalf("client session Close() error = %v", err)
		}
	})

	return clientSession
}

func assertToolSuccess(t *testing.T, result *mcp.CallToolResult) {
	t.Helper()

	if result == nil {
		t.Fatal("tool result = nil")
	}
	if result.IsError {
		t.Fatalf("tool result IsError = true; content = %+v", result.Content)
	}
}

func decodeStructuredContent[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()

	if result.StructuredContent == nil {
		t.Fatal("StructuredContent = nil")
	}
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("Marshal(StructuredContent) error = %v", err)
	}
	var decoded T
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal(StructuredContent) error = %v", err)
	}
	return decoded
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
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
