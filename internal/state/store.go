package state

import "sync"

// ResponseRule is a single in-memory HTTP response behavior rule.
type ResponseRule struct {
	ID          string
	Method      string
	Path        string
	StatusCode  int
	ContentType string
	Body        string
}

// Observation is verification information for a data-plane interaction.
type Observation struct {
	RequestMethod     string
	RequestPath       string
	MatchedRuleID     string
	OutcomeStatusCode int
}

// WebhookDeliveryObservation is verification information for a webhook delivery attempt.
type WebhookDeliveryObservation struct {
	EventID      string
	EndpointName string
	Method       string
	Outcome      string
	StatusCode   int
	Error        string
}

// Store is the in-memory runtime state placeholder for the MVP.
type Store struct {
	mu                sync.RWMutex
	generation        uint64
	responseRule      *ResponseRule
	observations      []Observation
	webhookDeliveries []WebhookDeliveryObservation
}

// New creates an empty in-memory runtime state store.
func New() *Store {
	return &Store{}
}

// Reset clears runtime behavior placeholders by advancing the state generation.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.responseRule = nil
	s.observations = nil
	s.webhookDeliveries = nil
	s.generation++
}

// Generation returns the current in-memory state generation.
func (s *Store) Generation() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.generation
}

// ConfigureResponseRule stores the single MVP response rule in memory.
func (s *Store) ConfigureResponseRule(rule ResponseRule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.responseRule = &rule
}

// HasResponseRule reports whether a behavior rule is currently configured.
func (s *Store) HasResponseRule() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.responseRule != nil
}

// MatchResponseRule returns the configured response rule when method and path match.
func (s *Store) MatchResponseRule(method string, path string) (ResponseRule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.responseRule == nil {
		return ResponseRule{}, false
	}
	if s.responseRule.Method != method || s.responseRule.Path != path {
		return ResponseRule{}, false
	}

	return *s.responseRule, true
}

// RecordObservation appends verification information in memory.
func (s *Store) RecordObservation(observation Observation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.observations = append(s.observations, observation)
}

// Observations returns a copy of in-memory verification information.
func (s *Store) Observations() []Observation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	observations := make([]Observation, len(s.observations))
	copy(observations, s.observations)
	return observations
}

// RecordWebhookDeliveryObservation appends webhook delivery verification information in memory.
func (s *Store) RecordWebhookDeliveryObservation(observation WebhookDeliveryObservation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.webhookDeliveries = append(s.webhookDeliveries, observation)
}

// WebhookDeliveryObservations returns a copy of in-memory webhook delivery information.
func (s *Store) WebhookDeliveryObservations() []WebhookDeliveryObservation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	observations := make([]WebhookDeliveryObservation, len(s.webhookDeliveries))
	copy(observations, s.webhookDeliveries)
	return observations
}
