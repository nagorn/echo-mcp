package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// OutcomeResponseReceived means the application webhook endpoint returned an HTTP response.
	OutcomeResponseReceived = "response_received"
	// OutcomeTransportError means delivery failed before an HTTP response was received.
	OutcomeTransportError = "transport_error"

	defaultTimeout = 5 * time.Second
)

// Event is one webhook-style event requested through the control plane.
type Event struct {
	EventID      string
	EndpointName string
	Body         map[string]any
}

// Delivery is the result of one webhook-style delivery attempt.
type Delivery struct {
	EventID      string
	EndpointName string
	Method       string
	Outcome      string
	StatusCode   int
	Error        string
}

// Sender sends one webhook-style HTTP event to a configured application endpoint.
type Sender struct {
	endpoints *Endpoints
	client    *http.Client
}

// NewSender creates a webhook sender for configured application endpoints.
func NewSender(endpoints *Endpoints, client *http.Client) *Sender {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	return &Sender{
		endpoints: endpoints,
		client:    client,
	}
}

// Send sends one HTTP POST webhook-style event to a configured application endpoint.
func (s *Sender) Send(ctx context.Context, event Event) (Delivery, error) {
	if s == nil || s.endpoints == nil {
		return Delivery{}, fmt.Errorf("no application webhook endpoint is configured")
	}
	if strings.TrimSpace(event.EventID) == "" {
		return Delivery{}, fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(event.EndpointName) == "" {
		return Delivery{}, fmt.Errorf("endpoint_name is required")
	}
	if event.Body == nil {
		return Delivery{}, fmt.Errorf("request.body is required")
	}

	endpoint, ok := s.endpoints.Resolve(event.EndpointName)
	if !ok {
		return Delivery{}, fmt.Errorf("unknown application webhook endpoint %q", event.EndpointName)
	}

	payload, err := json.Marshal(event.Body)
	if err != nil {
		return Delivery{}, fmt.Errorf("encode request.body as JSON: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.Address, bytes.NewReader(payload))
	if err != nil {
		return Delivery{}, fmt.Errorf("create webhook delivery request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	delivery := Delivery{
		EventID:      event.EventID,
		EndpointName: endpoint.Name,
		Method:       http.MethodPost,
	}

	response, err := s.client.Do(request)
	if err != nil {
		delivery.Outcome = OutcomeTransportError
		delivery.Error = err.Error()
		return delivery, nil
	}

	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		delivery.Outcome = OutcomeTransportError
		delivery.Error = err.Error()
		closeErr := response.Body.Close()
		if closeErr != nil {
			delivery.Error = delivery.Error + "; " + closeErr.Error()
		}
		return delivery, nil
	}
	if err := response.Body.Close(); err != nil {
		delivery.Outcome = OutcomeTransportError
		delivery.Error = err.Error()
		return delivery, nil
	}

	delivery.Outcome = OutcomeResponseReceived
	delivery.StatusCode = response.StatusCode
	return delivery, nil
}
