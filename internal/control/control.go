package control

import (
	"context"
	"fmt"

	"echo-mcp/internal/state"
	"echo-mcp/internal/webhook"
)

// Plane is the placeholder boundary for the MCP control plane.
type Plane interface {
	Protocol() string
	ConfigureResponseRule(state.ResponseRule) error
	SendWebhookEvent(context.Context, webhook.Event) (state.WebhookDeliveryObservation, error)
	Reset() error
	Observations() []state.Observation
	WebhookDeliveryObservations() []state.WebhookDeliveryObservation
}

// ResponseRuleValidator constrains response rules before they are stored.
type ResponseRuleValidator interface {
	ValidateResponseRule(state.ResponseRule) error
}

// WebhookSender sends webhook-style events to configured application endpoints.
type WebhookSender interface {
	Send(context.Context, webhook.Event) (webhook.Delivery, error)
}

// LocalPlane is the in-process MCP control-plane integration for the MVP.
type LocalPlane struct {
	store         *state.Store
	validator     ResponseRuleValidator
	webhookSender WebhookSender
}

// New creates an in-process control-plane boundary backed by in-memory state.
func New(store *state.Store) *LocalPlane {
	return NewWithValidator(store, nil)
}

// NewWithValidator creates a control-plane boundary with optional rule validation.
func NewWithValidator(store *state.Store, validator ResponseRuleValidator) *LocalPlane {
	return NewWithWebhookSender(store, validator, nil)
}

// NewWithWebhookSender creates a control-plane boundary with optional webhook delivery.
func NewWithWebhookSender(store *state.Store, validator ResponseRuleValidator, webhookSender WebhookSender) *LocalPlane {
	if store == nil {
		store = state.New()
	}

	return &LocalPlane{
		store:         store,
		validator:     validator,
		webhookSender: webhookSender,
	}
}

// Protocol identifies the control-plane protocol represented by this boundary.
func (*LocalPlane) Protocol() string {
	return "mcp"
}

// ConfigureResponseRule stores one HTTP response behavior rule in memory.
func (p *LocalPlane) ConfigureResponseRule(rule state.ResponseRule) error {
	if p.validator != nil {
		if err := p.validator.ValidateResponseRule(rule); err != nil {
			return err
		}
	}

	p.store.ConfigureResponseRule(rule)
	return nil
}

// SendWebhookEvent sends one webhook-style event and records the delivery attempt.
func (p *LocalPlane) SendWebhookEvent(ctx context.Context, event webhook.Event) (state.WebhookDeliveryObservation, error) {
	if p.webhookSender == nil {
		return state.WebhookDeliveryObservation{}, fmt.Errorf("no application webhook endpoint is configured")
	}

	delivery, err := p.webhookSender.Send(ctx, event)
	if err != nil {
		return state.WebhookDeliveryObservation{}, err
	}

	observation := state.WebhookDeliveryObservation{
		EventID:      delivery.EventID,
		EndpointName: delivery.EndpointName,
		Method:       delivery.Method,
		Outcome:      delivery.Outcome,
		StatusCode:   delivery.StatusCode,
		Error:        delivery.Error,
	}
	p.store.RecordWebhookDeliveryObservation(observation)

	return observation, nil
}

// Reset clears configured behavior and observation state in memory.
func (p *LocalPlane) Reset() error {
	p.store.Reset()
	return nil
}

// Observations returns currently available verification information.
func (p *LocalPlane) Observations() []state.Observation {
	return p.store.Observations()
}

// WebhookDeliveryObservations returns currently available webhook delivery information.
func (p *LocalPlane) WebhookDeliveryObservations() []state.WebhookDeliveryObservation {
	return p.store.WebhookDeliveryObservations()
}
