# Scope

## What Echo MCP Is

Echo MCP is an AI-controlled simulator for external dependencies used during
development and end-to-end testing.

Echo MCP is intended to:

- stand in for services outside the application boundary
- represent one simulated external dependency per Echo MCP process in the
  current implementation
- expose normal data-plane interfaces to the application under test
- let AI agents and other MCP clients configure simulated behavior at runtime
  through MCP
- validate configured REST behavior against developer-provided OpenAPI 3.0.x
  JSON contracts when such a contract is configured
- support repeatable test scenarios that involve success, failure, and selected
  dependency behavior
- send immediate webhook-style HTTP events to registered application webhook
  endpoints when configured for a test
- help validate application behavior when external dependencies are unavailable,
  unreliable, expensive, rate-limited, or difficult to control

## What Echo MCP Is Not

Echo MCP is not intended to be:

- a production payment gateway, identity provider, CRM, shipping provider, bank,
  or webhook provider
- an embedded test library inside the application under test
- a static response replay server only
- a replacement for application unit tests
- a source of truth for business rules owned by external services
- a monitoring or observability product for live third-party integrations
- a built-in simulator for public APIs such as Stripe, GitHub, or Slack
- an OpenAPI-driven mock generator
- a multi-dependency simulator inside one process
- a committed full service-virtualization platform before real use cases validate
  that direction

## Current Scope Boundary

The current boundary is external-dependency simulation for AI-driven
end-to-end testing.

Echo MCP currently remains focused on local external-dependency simulation for
AI-driven tests. Broader service virtualization, richer contract testing,
traffic replay, or chaos testing are not current commitments.

## Out of Scope Until Explicitly Revisited

- production traffic handling
- public hosted simulator service
- SDKs for specific application languages
- GitHub Actions integration
- traffic recording and replay
- complete OpenAPI implementation
- OpenAPI-driven generation
- multi-dependency support inside one Echo MCP process
- AsyncAPI implementation
- webhook retries, scheduling, signatures, delivery persistence, or event buses
- multi-agent testing workflows
- long-running scenario scripting language
