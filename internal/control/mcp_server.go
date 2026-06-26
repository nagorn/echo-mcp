package control

import (
	"context"
	"fmt"
	"strings"

	"echo-mcp/internal/state"
	"echo-mcp/internal/webhook"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	outcomeHTTPResponse = "http_response"
)

// NewMCPServer creates the MVP MCP control-plane server.
func NewMCPServer(plane Plane) *mcp.Server {
	if plane == nil {
		plane = New(nil)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "echo-mcp", Version: "v0.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "configure_behavior",
		Description: "Configure one in-memory REST-style HTTP response behavior.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input configureBehaviorInput) (*mcp.CallToolResult, configureBehaviorOutput, error) {
		if err := validateConfigureBehavior(input); err != nil {
			return nil, configureBehaviorOutput{}, err
		}

		if err := plane.ConfigureResponseRule(state.ResponseRule{
			ID:          input.BehaviorID,
			Method:      input.Match.Method,
			Path:        input.Match.Path,
			StatusCode:  input.Outcome.StatusCode,
			ContentType: input.Outcome.ContentType,
			Body:        input.Outcome.Body,
		}); err != nil {
			return nil, configureBehaviorOutput{}, err
		}

		return nil, configureBehaviorOutput{
			Configured: true,
			BehaviorID: input.BehaviorID,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "reset",
		Description: "Clear configured behavior and observation state.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input resetInput) (*mcp.CallToolResult, resetOutput, error) {
		if err := plane.Reset(); err != nil {
			return nil, resetOutput{}, err
		}

		return nil, resetOutput{
			Reset:   true,
			Cleared: []string{"behavior", "observations", "webhook_deliveries"},
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "send_webhook_event",
		Description: "Send one webhook-style event to a configured application webhook endpoint.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input sendWebhookEventInput) (*mcp.CallToolResult, sendWebhookEventOutput, error) {
		if err := validateSendWebhookEvent(input); err != nil {
			return nil, sendWebhookEventOutput{}, err
		}

		delivery, err := plane.SendWebhookEvent(ctx, webhook.Event{
			EventID:      input.EventID,
			EndpointName: input.EndpointName,
			Body:         input.Request.Body,
		})
		if err != nil {
			return nil, sendWebhookEventOutput{}, err
		}

		return nil, sendWebhookEventOutput{
			Attempted:    true,
			EventID:      delivery.EventID,
			EndpointName: delivery.EndpointName,
			Delivery: webhookDeliveryOutput{
				Outcome:    delivery.Outcome,
				StatusCode: delivery.StatusCode,
				Error:      delivery.Error,
			},
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_observations",
		Description: "Return currently available data-plane observation information.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getObservationsInput) (*mcp.CallToolResult, getObservationsOutput, error) {
		observations := plane.Observations()
		output := getObservationsOutput{
			Observations:      make([]observationOutput, 0, len(observations)),
			WebhookDeliveries: make([]webhookDeliveryObservationOutput, 0, len(plane.WebhookDeliveryObservations())),
		}

		for _, observation := range observations {
			output.Observations = append(output.Observations, observationOutput{
				Request: requestObservation{
					Method: observation.RequestMethod,
					Path:   observation.RequestPath,
				},
				Selection: selectionObservation{
					MatchedBehaviorID: observation.MatchedRuleID,
					MatchedOn:         []string{"method", "path"},
				},
				Outcome: outcomeObservation{
					Type:       outcomeHTTPResponse,
					StatusCode: observation.OutcomeStatusCode,
				},
			})
		}

		for _, delivery := range plane.WebhookDeliveryObservations() {
			output.WebhookDeliveries = append(output.WebhookDeliveries, webhookDeliveryObservationOutput{
				EventID:      delivery.EventID,
				EndpointName: delivery.EndpointName,
				Method:       delivery.Method,
				Outcome:      delivery.Outcome,
				StatusCode:   delivery.StatusCode,
				Error:        delivery.Error,
			})
		}

		return nil, output, nil
	})

	return server
}

type configureBehaviorInput struct {
	BehaviorID string                   `json:"behavior_id" jsonschema:"behavior rule identifier used in observations"`
	Match      configureBehaviorMatch   `json:"match" jsonschema:"data-plane request match criteria"`
	Outcome    configureBehaviorOutcome `json:"outcome" jsonschema:"simulated outcome for the matching request"`
}

type configureBehaviorMatch struct {
	Method string `json:"method" jsonschema:"HTTP method for the incoming data-plane request"`
	Path   string `json:"path" jsonschema:"HTTP path for the incoming data-plane request"`
}

type configureBehaviorOutcome struct {
	Type        string `json:"type" jsonschema:"outcome type; only http_response is supported in the first slice"`
	StatusCode  int    `json:"status_code" jsonschema:"HTTP response status code"`
	ContentType string `json:"content_type,omitempty" jsonschema:"HTTP response content type"`
	Body        string `json:"body" jsonschema:"HTTP response body"`
}

type configureBehaviorOutput struct {
	Configured bool   `json:"configured"`
	BehaviorID string `json:"behavior_id"`
}

type resetInput struct{}

type resetOutput struct {
	Reset   bool     `json:"reset"`
	Cleared []string `json:"cleared"`
}

type sendWebhookEventInput struct {
	EventID      string                  `json:"event_id" jsonschema:"webhook event identifier used in observations"`
	EndpointName string                  `json:"endpoint_name" jsonschema:"configured application webhook endpoint name"`
	Request      sendWebhookEventRequest `json:"request" jsonschema:"webhook request payload"`
}

type sendWebhookEventRequest struct {
	Body map[string]any `json:"body" jsonschema:"JSON request body sent to the application webhook endpoint"`
}

type sendWebhookEventOutput struct {
	Attempted    bool                  `json:"attempted"`
	EventID      string                `json:"event_id"`
	EndpointName string                `json:"endpoint_name"`
	Delivery     webhookDeliveryOutput `json:"delivery"`
}

type webhookDeliveryOutput struct {
	Outcome    string `json:"outcome"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

type getObservationsInput struct{}

type getObservationsOutput struct {
	Observations      []observationOutput                `json:"observations"`
	WebhookDeliveries []webhookDeliveryObservationOutput `json:"webhook_deliveries"`
}

type observationOutput struct {
	Request   requestObservation   `json:"request"`
	Selection selectionObservation `json:"selection"`
	Outcome   outcomeObservation   `json:"outcome"`
}

type requestObservation struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type selectionObservation struct {
	MatchedBehaviorID string   `json:"matched_behavior_id"`
	MatchedOn         []string `json:"matched_on"`
}

type outcomeObservation struct {
	Type       string `json:"type"`
	StatusCode int    `json:"status_code"`
}

type webhookDeliveryObservationOutput struct {
	EventID      string `json:"event_id"`
	EndpointName string `json:"endpoint_name"`
	Method       string `json:"method"`
	Outcome      string `json:"outcome"`
	StatusCode   int    `json:"status_code,omitempty"`
	Error        string `json:"error,omitempty"`
}

func validateConfigureBehavior(input configureBehaviorInput) error {
	if strings.TrimSpace(input.BehaviorID) == "" {
		return fmt.Errorf("behavior_id is required")
	}
	if strings.TrimSpace(input.Match.Method) == "" {
		return fmt.Errorf("match.method is required")
	}
	if strings.TrimSpace(input.Match.Path) == "" {
		return fmt.Errorf("match.path is required")
	}
	if input.Outcome.Type != outcomeHTTPResponse {
		return fmt.Errorf("outcome.type must be %q", outcomeHTTPResponse)
	}
	if input.Outcome.StatusCode < 100 || input.Outcome.StatusCode > 999 {
		return fmt.Errorf("outcome.status_code must be a valid HTTP status code")
	}

	return nil
}

func validateSendWebhookEvent(input sendWebhookEventInput) error {
	if strings.TrimSpace(input.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(input.EndpointName) == "" {
		return fmt.Errorf("endpoint_name is required")
	}
	if input.Request.Body == nil {
		return fmt.Errorf("request.body is required")
	}

	return nil
}
