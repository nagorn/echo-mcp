# Developer Usage Guide

This guide explains how developers should use Echo MCP in end-to-end tests.

Echo MCP is an AI-controlled external dependency simulator. It lets an AI agent
or another MCP client configure test-time dependency behavior through MCP while
the application under test continues to use normal HTTP.

## What Echo MCP Is Used For

Use Echo MCP to simulate external dependency behavior during local or automated
end-to-end tests.

Good fits include:

- forcing an external dependency to return a specific success response
- forcing a dependency failure response
- sending a webhook-style event to the application's normal webhook endpoint
- testing application retry behavior
- testing application error handling
- verifying which dependency interaction occurred during a test

Echo MCP is useful when the real dependency is unavailable, expensive,
rate-limited, difficult to put into the desired state, or risky to call during a
test.

## What Echo MCP Is Not Used For

Echo MCP is not:

- a production external service
- a replacement for unit tests
- a source of truth for another service's business rules
- a general-purpose contract testing platform
- a source of built-in public API contracts
- an OpenAPI importer or generator
- a traffic recorder or replay server
- a monitoring, metrics, or audit logging system
- a UI, dashboard, admin API, or CLI product surface

Echo MCP should not be used to hide test-only branches inside application code.

## Normal Workflow

1. Start the local Echo MCP process for the external dependency being simulated.
2. Point the application's external dependency base URL or webhook endpoint
   configuration to Echo MCP or to the application webhook receiver under test,
   depending on the scenario.
3. Use an AI agent, or another MCP client, to configure behavior through MCP.
4. Run the application test.
5. Use an AI agent, or another MCP client, to inspect observations through MCP.
6. Assert the application outcome and the dependency interaction that occurred.

The control plane and data plane remain separate throughout the workflow:

- The AI agent configures Echo MCP through MCP.
- The application sends normal REST-style HTTP requests.
- Echo MCP returns normal REST-style HTTP responses.
- Echo MCP sends webhook-style HTTP events only to configured application
  webhook endpoints.
- The AI agent retrieves observation information through MCP.

## Multiple External Dependencies

MVP usage: one Echo MCP process represents one simulated external dependency.

The current implementation supports one OpenAPI contract, one registered
webhook endpoint, and one active REST behavior rule per process. If the
Application Under Test integrates with several external dependencies at the same
time, run one Echo MCP process per dependency and give each process its own
`ECHO_MCP_HTTP_ADDR` and MCP server registration.

Example:

```bash
ECHO_MCP_HTTP_ADDR=127.0.0.1:18080 \
ECHO_MCP_OPENAPI_FILE=./contracts/payment.openapi.json \
./bin/echo-mcp

ECHO_MCP_HTTP_ADDR=127.0.0.1:18081 \
ECHO_MCP_OPENAPI_FILE=./contracts/fraud.openapi.json \
./bin/echo-mcp
```

Point each application dependency base URL at the matching Echo MCP data-plane
address. Multi-dependency support inside one Echo MCP process is future work
and would require a future ADR.

## Boundary Rule

The application under test must treat Echo MCP as a normal external REST
dependency.

The application must:

- use normal REST-style HTTP requests
- receive webhook-style events through its normal HTTP webhook endpoint
- configure only its external dependency base URL or equivalent runtime endpoint
- keep production-like request and response handling paths

The application must not:

- use MCP
- include Echo MCP-specific request metadata
- include Echo MCP-specific branches
- call simulator control tools
- depend on observation data

This boundary preserves the value of end-to-end testing. The application should
not know that a simulator is involved.

## Unmatched REST Requests

If the Application Under Test calls Echo MCP's REST data plane before any
matching behavior has been configured, Echo MCP returns HTTP `501 Not
Implemented`.

HTTP `501` from Echo MCP means simulator setup is missing for that request. It
is not a simulated external provider response. Provider-like responses such as
HTTP `404`, `500`, or `503` should be returned only when an MCP client
explicitly configures them as the behavior outcome.

When a test receives HTTP `501`, use an AI agent, or another MCP client, to
inspect available observations and configure the missing behavior through the
MCP control plane before rerunning the test.

Observation information for unmatched REST requests may be improved later; that
is not required for the current MVP policy.

## Example Use Cases

### Payment Declined

The AI agent configures Echo MCP to return a payment-provider-style HTTP `402`
response with a JSON error body. The application test verifies that the checkout
flow shows the correct failure state and does not mark the payment as complete.

### Retryable HTTP 429

The AI agent configures a dependency to return an HTTP `429` response. The
application test verifies that the application handles the retryable error
according to its current policy.

Echo MCP currently supports one active behavior rule, so multi-response retry
sequences remain future work.

### Malformed Upstream Response

The AI agent configures Echo MCP to return an HTTP success or failure response
with a malformed or unexpected body. The application test verifies that parsing
errors are handled without corrupting state.

### Duplicate Transaction Response

The AI agent configures a dependency to return a duplicate-transaction-style
response. The application test verifies idempotency handling and confirms that
the application does not create duplicate local records.

### Webhook-Style Event Delivery

The developer or test harness starts Echo MCP with a registered application
webhook endpoint:

```bash
ECHO_MCP_WEBHOOK_ENDPOINT_NAME=payment-events \
ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS=http://127.0.0.1:18080/webhooks/payments \
./bin/echo-mcp
```

The AI agent calls `send_webhook_event` with `endpoint_name` and `request.body`.
Echo MCP sends one immediate HTTP `POST` with `Content-Type: application/json`
to the configured application webhook endpoint. The application receives the
request through its normal webhook handler.

The AI agent then calls `get_observations` and verifies the
`webhook_deliveries` entry. AI agents select configured endpoint names; they do
not provide arbitrary outbound URLs.

## Current MVP Limitations

The current MVP is intentionally narrow:

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
- no CLI
- no authentication
- no authorization
- no user management
- no persistence
- no admin API
- no metrics
- no audit logs
- no OpenAPI import or generation
- no OpenAPI 3.1.x, YAML OpenAPI, or external `$ref` support
- no built-in public API contracts
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
- no non-REST protocols
- no production deployment architecture

These limits keep Echo MCP focused on proving the control-plane/data-plane
workflow before adding broader service simulation features.

## Recommended Developer Mindset

Use Echo MCP to simulate external dependency behavior, not to change how the
application is written.

Keep application code production-like:

- normal dependency base URL configuration
- normal HTTP clients
- normal request bodies
- normal response handling
- normal webhook handlers
- normal application assertions

Let the AI agent control test conditions through MCP. Let the application behave
as if it is talking to the real external dependency. Use observations to verify
what Echo MCP received, which behavior matched, and which outcome was produced.
