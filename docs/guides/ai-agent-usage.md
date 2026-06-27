# AI Agent Usage Guide

This guide explains how an AI coding agent should discover and use Echo MCP
during a developer's end-to-end testing workflow.

Echo MCP has two separate planes:

- MCP control plane for AI agents and other MCP clients.
- REST data plane for the Application Under Test.

An AI agent, or another MCP client, configures simulated external dependency
behavior through MCP tools. The Application Under Test continues to use normal
REST HTTP, as if it were talking to the real external dependency.

## Discovery

Before using Echo MCP, an AI agent should determine whether the current project
is configured to use it.

Recommended discovery order:

1. Project-local AI instruction files, such as `AGENTS.md`, `CLAUDE.md`,
   Cursor rules, GitHub Copilot instructions, or equivalent project-specific
   guidance.
2. Project documentation.
3. Available MCP servers and tools.
4. Echo MCP initialize instructions, tool descriptions, prompts, and resources
   when Echo MCP is available.
5. Existing E2E test examples.

If Echo MCP is not configured for the project, do not assume its presence. The
AI agent should gracefully continue using the project's normal testing workflow.
Echo MCP v0.2.0 does not introduce or require an `echo-mcp.yaml` project
manifest.

## Core Rule

Preserve the control-plane/data-plane boundary.

An AI agent using Echo MCP should:

- configure external dependency behavior through Echo MCP MCP tools
- treat one Echo MCP process as one simulated external dependency
- select configured webhook endpoint names when sending webhook-style events
- run the application's end-to-end test normally
- inspect Echo MCP observations through MCP
- reset Echo MCP between test scenarios
- keep configured behavior compatible with known API contracts when contract
  constraints are available

The AI agent must not:

- modify application code to make it aware of Echo MCP
- add Echo MCP-specific branches, headers, request fields, or test hooks to the
  application
- bypass the application by calling Echo MCP's REST data plane as the test
  subject
- provide arbitrary outbound URLs for webhook delivery
- treat Echo MCP as an AI runtime
- expect Echo MCP to infer, generate, or expand behavior from test intent
- invent unsupported Echo MCP capabilities

Echo MCP is a deterministic execution engine. The AI agent decides the test
scenario, constructs the concrete simulated behavior, and sends the complete
behavior configuration through MCP. Echo MCP stores accepted request/response
behavior and executes it deterministically when the application sends a matching
REST request. Echo MCP also sends immediate single-attempt webhook-style events
to developer-configured application webhook endpoints selected by endpoint name.

## Recommended Workflow

1. Inspect Echo MCP initialize instructions, tool descriptions, prompts, and
   resources.
2. Read the test objective.
3. Choose `manual_mock`, `hybrid_validation`, or `contract_first` based on the
   project need and available contracts.
4. Identify which external dependency behavior must be simulated.
5. Configure Echo MCP via MCP.
6. Run the application E2E test normally.
7. Inspect Echo MCP observations.
8. Verify application behavior.
9. Reset Echo MCP before the next scenario.

Use `manual_mock` for quick hand-authored behavior. Manual mock behavior is not
provider-contract validated unless OpenAPI-backed validation is active. Use
`hybrid_validation` or `contract_first` when a developer-provided OpenAPI
contract exists and provider fidelity matters.

## Step Details

### 1. Read the Test Objective

Understand the application behavior under test before configuring Echo MCP.
Identify the user-visible outcome, application state change, or integration
behavior that the test must prove.

### 2. Identify the External Dependency Behavior

Determine which external dependency interaction the application should perform
and what response the dependency should return.

If a test involves multiple external dependencies, use one Echo MCP process per
dependency. Each process should have its own `ECHO_MCP_HTTP_ADDR`, optional
contract configuration, and MCP server registration. Do not assume one Echo MCP
process can hold multiple dependency contracts or independent behavior sets.

For REST-style request/response tests, configure one active behavior rule:

- HTTP method
- request path
- HTTP response status
- optional response content type
- response body
- behavior identifier for later observation

For webhook-style event tests, select a configured application webhook endpoint
by `endpoint_name` and provide a JSON `request.body`.

Do not add extra simulator behavior unless Echo MCP explicitly supports it.

### 3. Configure Echo MCP via MCP

Use the MCP control plane to configure behavior. The current MCP
control-plane tool surface is:

- `configure_behavior`
- `reset`
- `send_webhook_event`
- `get_observations`

The current guidance prompts are:

- `echo_mcp_getting_started`
- `echo_mcp_choose_workflow`
- `echo_mcp_manual_mock_workflow`
- `echo_mcp_contract_validation_workflow`

The current guidance resources are:

- `echo://guides/getting-started`
- `echo://guides/workflows`
- `echo://guides/manual-mock`
- `echo://guides/contract-validation`

The application must not call these tools. They are for MCP control-plane
clients only.

If contract constraints are available for the simulated dependency, configure a
complete concrete behavior that conforms to the contract. When a project has
enabled contract-constrained simulation, Echo MCP validates behavior against
available constraints; it does not generate behavior from the contract.

For webhook-style event delivery, use only configured `endpoint_name` values.
Do not provide raw URLs, hosts, ports, schemes, custom headers, signatures,
retry settings, or scheduling metadata.

### 4. Run the Application E2E Test Normally

Run the application test through the application entry point, browser flow, API,
worker, or normal test harness.

The application should send ordinary REST HTTP traffic to Echo MCP only because
the developer or test environment has pointed the dependency base URL at Echo
MCP. The application should not include MCP awareness or Echo MCP-specific
logic.

A direct REST probe against Echo MCP can help diagnose simulator wiring, but it
is not a substitute for the application E2E test.

If the Application Under Test receives HTTP `501 Not Implemented` from Echo
MCP's REST data plane, treat it as missing simulator setup. It means no
configured behavior matched the request. Inspect available observations and
test/application logs, configure the missing behavior through the MCP control
plane, and rerun the test.

Do not treat Echo MCP's unmatched-request `501` as a simulated external provider
response. Provider-like responses such as HTTP `404`, `500`, or `503` should be
returned only when explicitly configured through `configure_behavior`.

### 5. Inspect Echo MCP Observations

After the application runs, call `get_observations` through MCP.

Use observations to verify:

- which request Echo MCP received
- which behavior matched
- which match criteria were applied
- which outcome Echo MCP produced
- which webhook delivery attempts Echo MCP made
- which configured endpoint name was selected
- whether the webhook delivery received an HTTP response or failed before a
  response

Observation information for unmatched REST requests may be improved later. For
now, use HTTP `501` as the deterministic signal that the requested behavior was
not configured or did not match.

Observation data is for test verification. It should not become part of
application behavior.

### 6. Verify Application Behavior

Check the application's externally visible behavior and internal state using the
project's normal test assertions.

A passing Echo MCP observation is not enough by itself. The test should also
prove that the application handled the simulated dependency behavior correctly.

### 7. Reset Before the Next Scenario

Call `reset` through MCP before starting the next scenario.

Resetting clears the configured behavior and currently available observations so
the next scenario starts from known empty runtime state.

## Stripe PaymentIntent Example

Scenario: test how an application handles a failed Stripe-like PaymentIntent
confirmation.

The AI agent configures one behavior through `configure_behavior`:

```json
{
  "behavior_id": "stripe-like-paymentintent-card-declined",
  "match": {
    "method": "POST",
    "path": "/v1/payment_intents/pi_123/confirm"
  },
  "outcome": {
    "type": "http_response",
    "status_code": 402,
    "content_type": "application/json",
    "body": "{\"error\":{\"type\":\"card_error\",\"code\":\"card_declined\",\"decline_code\":\"generic_decline\",\"message\":\"Your card was declined.\",\"payment_intent\":{\"id\":\"pi_123\",\"object\":\"payment_intent\",\"status\":\"requires_payment_method\"}}}"
  }
}
```

The Application Under Test sends a normal REST request:

```http
POST /v1/payment_intents/pi_123/confirm HTTP/1.1
Content-Type: application/x-www-form-urlencoded

payment_method=pm_card_visa
```

Echo MCP returns the configured HTTP `402` JSON response. The AI agent then
uses the MCP control plane to call `get_observations` and verifies that Echo MCP
received the expected request, selected
`stripe-like-paymentintent-card-declined`, matched on method and path, and
produced the `http_response` outcome with status `402`.

The AI agent then verifies application behavior through the normal application
test: for example, the checkout remains unpaid, the user sees the expected
decline state, and no local payment record is marked complete.

This example is manually configured from a public Stripe-like API shape. Echo
MCP is not a Stripe simulator, does not include built-in Stripe contracts, and
does not generate Stripe behavior. If a developer-supplied OpenAPI 3.0.x JSON
contract is configured, Echo MCP can validate the concrete behavior as a
constraint before accepting it.

## Webhook Event Example

Scenario: test how an application handles a webhook-style event delivered to its
normal webhook endpoint.

The developer or test harness starts Echo MCP with one registered application
webhook endpoint:

```bash
ECHO_MCP_WEBHOOK_ENDPOINT_NAME=payment-events \
ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS=http://127.0.0.1:3000/webhooks/payments \
./bin/echo-mcp
```

The AI agent calls `send_webhook_event` with the configured endpoint name:

```json
{
  "event_id": "evt_payment_failed_001",
  "endpoint_name": "payment-events",
  "request": {
    "body": {
      "type": "payment.failed",
      "data": {
        "object": {
          "id": "pay_123"
        }
      }
    }
  }
}
```

Echo MCP sends one HTTP `POST` with `Content-Type: application/json` to the
configured application webhook endpoint. The AI agent then calls
`get_observations` and verifies the `webhook_deliveries` entry.

## Agent Guidance Compatibility

The v0.2.0 guidance surfaces are MCP-standard and advisory. Agent behavior may
vary by MCP client, and some clients may not automatically read prompts or
resources. Structured guidance fields in tool results are additive; strict MCP
clients should tolerate additional structured output fields.

Control-plane guidance does not change REST data-plane response bodies.

## Current Limits for AI Agents and MCP Clients

Do not assume Echo MCP supports capabilities that have not been explicitly added.
Current MVP limits include:

- REST-style request/response behavior for application-initiated requests
- one in-memory REST behavior rule
- one simulated external dependency per Echo MCP process
- configurable REST data-plane listen address with `ECHO_MCP_HTTP_ADDR`
- unmatched REST data-plane requests return HTTP `501 Not Implemented`
- optional validation against one developer-provided OpenAPI 3.0.x JSON
  contract
- immediate single-attempt webhook-style event delivery to one configured
  application webhook endpoint
- no UI
- no dashboard
- no CLI
- no authentication or authorization
- no user management
- no persistence
- no OpenAPI import or generation
- no full OpenAPI-first runtime
- no `echo-mcp.yaml` project manifest
- no OpenAPI 3.1.x, YAML OpenAPI, or external `$ref` support
- no built-in public API contracts
- no multi-dependency support inside one Echo MCP process
- no recording or replay
- no timeout outcome
- no multi-response retry sequence
- no webhook retries
- no webhook scheduling
- no webhook signatures
- no webhook delivery persistence
- no AsyncAPI support
- no event bus
- no inbound public webhook receiver
- no metrics or audit logs
- no production deployment architecture

When a scenario needs unsupported behavior, identify the gap instead of
silently modifying application code or inventing an Echo MCP capability.

## Design Philosophy

Echo MCP does not replace the application's external dependencies with AI.

Instead:

- AI decides the test scenario.
- Echo MCP validates and executes deterministic behavior.
- The application communicates through its normal REST interfaces.
- The application receives webhook-style events through its normal webhook
  endpoints.
- Observations provide evidence of what actually happened.

This separation allows AI reasoning and deterministic execution to remain
independent.
