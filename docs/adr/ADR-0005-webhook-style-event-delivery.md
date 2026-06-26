# ADR-0005: Webhook-Style Event Delivery

Status: APPROVED
Date: 2026-06-25
Owner: Nagorn Smanote

## Context

ADR-0001 separates Echo MCP's control channel from its data channel. AI agents
configure Echo MCP through MCP, while the application under test uses normal
service interfaces and should not know Echo MCP is involved.

Some external dependencies do not only respond to application requests. They
also send asynchronous callbacks or webhook-style HTTP events to the
application. Examples include payment status updates, identity verification
events, shipping updates, and CRM notifications.

At the time this ADR was proposed, the implementation proved the REST-style
request/response loop and did not deliver outbound webhook-style events.

## Decision

Echo MCP should support outbound webhook-style HTTP event delivery to the
application under test as a data-plane capability.

Webhook-style event delivery must preserve ADR-0001:

- AI agents configure webhook-style event behavior through the MCP control
  plane.
- Echo MCP sends the event through normal HTTP to a registered webhook endpoint
  exposed by the application under test.
- The application receives the event as a normal inbound webhook request.
- The application must not know Echo MCP is involved.

Webhook delivery is a data-plane interaction initiated by Echo MCP. It is not an
MCP message to the application and it is not an inbound public webhook receiver.
Webhook targets are configured by the developer or application test setup; the
AI agent should not freely choose arbitrary outbound destinations.

## Milestone Placement

Webhook-style event delivery should be a post-MVP milestone, not part of the
first MVP request/response milestone.

The current MVP remains focused on REST-style request/response behavior:

- AI configures one behavior through MCP.
- Application sends a normal REST request.
- Echo MCP returns the configured response.
- AI retrieves observations through MCP.

Webhook delivery should be introduced only after that loop remains stable and a
separate task or milestone explicitly promotes webhook delivery into active
implementation scope.

Implementation note: a later implementation slice promoted and completed the
first webhook delivery capability.

## Smallest First Webhook Scenario

The smallest first webhook scenario should be one immediate HTTP `POST` event to
a registered application webhook endpoint.

The AI-configured event should include:

- registered webhook endpoint selected from developer configuration
- HTTP method, initially `POST`
- JSON body
- optional minimal headers if approved by the implementation task

When triggered, Echo MCP sends one HTTP request to the configured application
endpoint and records observation information for that delivery attempt.

This first scenario should not include retries, signing, scheduling, delivery
persistence, event buses, AsyncAPI, or inbound webhook receiving.

## Observation Model

Echo MCP should make webhook delivery observations available through the MCP
control plane.

Observation information for a sent webhook event should include:

- configured application webhook endpoint
- HTTP method
- delivery attempt outcome
- response status code when the application responds
- transport error information when delivery fails before a response
- the configured event identifier or behavior identifier that caused the
  delivery

Observation information remains in memory unless a separate persistence decision
is approved later.

## Non-Goals

This ADR does not add:

- inbound public webhook receiving
- event buses or message brokers
- AsyncAPI support
- retries
- signing or signature verification
- scheduling or delayed delivery
- delivery persistence
- UI, dashboard, CLI, or admin API
- authentication, authorization, user management, or multi-tenancy
- metrics or audit logs
- traffic recording or replay
- service virtualization
- production deployment architecture

## Consequences

- Echo MCP can model dependencies that initiate callbacks to the application
  under test.
- The data plane expands beyond request/response behavior to include outbound
  HTTP event delivery.
- The application still uses normal HTTP and remains unaware of MCP.
- MCP remains the control plane for configuring and inspecting webhook-style
  behavior.
- Implementation must prevent webhook delivery from becoming a general event bus
  or production webhook relay.
- Observation shape must distinguish received application requests from webhook
  events sent by Echo MCP.

## Answers to Key Questions

### Is webhook delivery part of the same MVP or a post-MVP milestone?

Webhook delivery should be post-MVP. It should not be added to the current first
request/response MVP unless explicitly promoted later.

### What is the smallest first webhook scenario?

One AI-configured immediate HTTP `POST` with a JSON body to an application
webhook target configured by the developer. Echo MCP sends the request once and
records the delivery attempt.

### How does Echo MCP record observations for sent webhook events?

Echo MCP should record in-memory observation information that identifies the
configured event, configured application webhook endpoint, HTTP method, delivery
outcome, response status when available, and transport error when no response is
received.

## Intentionally Undecided

- exact MCP tool names and schemas for configuring or triggering webhook events
- whether webhook events are configured then triggered, or configured and sent in
  a single MCP call
- exact observation schema for webhook deliveries
- whether headers are configurable in the first webhook scenario
- request timeout behavior for delivery attempts
- TLS and local-network restrictions
- retries and retry policy
- signing and signature verification
- scheduling and delayed delivery
- delivery persistence or durable history
- AsyncAPI support
- correlation between request/response interactions and later webhook events
- whether webhook delivery should share the same behavior-rule abstraction or
  use a separate event abstraction

## Related Documents

- [ADR-0001: Separate Control Channel from Data Channel](ADR-0001-separate-control-channel-from-data-channel.md)
- [ADR-0002: MVP Implementation Platform](ADR-0002-mvp-implementation-platform.md)
- [ADR-0003: MCP SDK and Transport](ADR-0003-mcp-sdk-and-transport.md)
- [Scope](../concepts/scope.md)
- [Core Abstractions](../concepts/core-abstractions.md)
