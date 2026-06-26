// Package webhook sends webhook-style HTTP events to configured application
// webhook endpoints.
package webhook

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// EnvEndpointName is the startup environment variable for the registered endpoint name.
	EnvEndpointName = "ECHO_MCP_WEBHOOK_ENDPOINT_NAME"
	// EnvEndpointAddress is the startup environment variable for the registered endpoint address.
	EnvEndpointAddress = "ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS"
)

// Endpoint is one developer-configured application webhook endpoint.
type Endpoint struct {
	Name    string
	Address string
}

// Endpoints resolves registered application webhook endpoints by name.
type Endpoints struct {
	endpoint Endpoint
}

// NewEndpoints creates a resolver for one configured application webhook endpoint.
func NewEndpoints(endpoint Endpoint) *Endpoints {
	return &Endpoints{endpoint: endpoint}
}

// LoadEndpointFromEnvironment loads one configured application webhook endpoint.
func LoadEndpointFromEnvironment(getenv func(string) string) (*Endpoints, error) {
	name := strings.TrimSpace(getenv(EnvEndpointName))
	address := strings.TrimSpace(getenv(EnvEndpointAddress))

	if name == "" && address == "" {
		return nil, nil
	}
	if name == "" || address == "" {
		return nil, fmt.Errorf("%s and %s must both be set to register a webhook endpoint", EnvEndpointName, EnvEndpointAddress)
	}
	if err := validateEndpointAddress(address); err != nil {
		return nil, err
	}

	return NewEndpoints(Endpoint{Name: name, Address: address}), nil
}

// Resolve returns the registered endpoint when the name matches.
func (e *Endpoints) Resolve(name string) (Endpoint, bool) {
	if e == nil {
		return Endpoint{}, false
	}
	if e.endpoint.Name != name {
		return Endpoint{}, false
	}

	return e.endpoint, true
}

func validateEndpointAddress(address string) error {
	parsed, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("parse configured application webhook endpoint: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("configured application webhook endpoint must use http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("configured application webhook endpoint must include a host")
	}

	return nil
}
