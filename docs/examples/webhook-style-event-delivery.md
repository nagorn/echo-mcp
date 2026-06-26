# Webhook-Style Event Delivery

This example shows the canonical Echo MCP workflow for sending one
webhook-style event to an Application Under Test.

Webhook delivery preserves the Echo MCP control-plane/data-plane boundary:

- An MCP client controls Echo MCP through MCP.
- Echo MCP sends normal HTTP to a developer-configured application webhook
  endpoint.
- The Application Under Test receives the event through its normal webhook
  handler.
- The application does not know Echo MCP or MCP is involved.

Webhook endpoint addresses are configured by the developer or test harness.
AI agents select configured endpoint names; they do not provide arbitrary
outbound URLs.

## Current Capability

The current webhook slice supports:

- one registered application webhook endpoint
- endpoint selection by `endpoint_name`
- immediate single-attempt HTTP `POST`
- `Content-Type: application/json`
- JSON request body under `request.body`
- in-memory webhook delivery observations

It does not add retries, scheduling, signatures, persistence, AsyncAPI, event
buses, inbound public webhook receiving, UI, CLI, authentication,
authorization, metrics, audit logs, or production deployment architecture.

## Scenario

The Application Under Test exposes a normal webhook endpoint:

```text
POST http://127.0.0.1:18080/webhooks/payments
```

The developer wants an AI agent to trigger a payment-related event without the
application knowing a simulator is involved.

## Step 1: Developer Starts Echo MCP With a Registered Endpoint

The developer or test harness starts Echo MCP with one registered application
webhook endpoint:

```bash
ECHO_MCP_WEBHOOK_ENDPOINT_NAME=payment-events \
ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS=http://127.0.0.1:18080/webhooks/payments \
./bin/echo-mcp
```

The endpoint name is the only value the AI agent uses through MCP. The endpoint
address remains developer-controlled startup configuration.

## Step 2: AI Calls `send_webhook_event`

The AI agent calls the MCP tool `send_webhook_event`.

Tool call:

```json
{
  "tool": "send_webhook_event",
  "arguments": {
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
}
```

The tool call does not include a URL, host, port, scheme, custom headers,
signature settings, retry settings, or scheduling metadata.

## Step 3: Echo MCP Sends a Normal HTTP Request

Echo MCP resolves `payment-events` to the configured application webhook
endpoint and sends one immediate HTTP request:

```http
POST /webhooks/payments HTTP/1.1
Host: 127.0.0.1:18080
Content-Type: application/json

{
  "type": "payment.failed",
  "data": {
    "object": {
      "id": "pay_123"
    }
  }
}
```

Echo MCP sends exactly one delivery attempt for the MCP call. Response status
codes from the application are delivery outcomes, not retries.

## Step 4: Application Receives the Event Normally

The Application Under Test receives the request through its ordinary webhook
handler. It should use the same application path it would use for a real
external dependency webhook.

The application must not:

- call MCP
- inspect Echo MCP observations
- include Echo MCP-specific request branches
- depend on simulator metadata

## Step 5: AI Retrieves Observations

The AI agent calls `get_observations` through MCP.

Tool call:

```json
{
  "tool": "get_observations",
  "arguments": {}
}
```

Expected observation shape:

```json
{
  "observations": [],
  "webhook_deliveries": [
    {
      "event_id": "evt_payment_failed_001",
      "endpoint_name": "payment-events",
      "method": "POST",
      "outcome": "response_received",
      "status_code": 204
    }
  ]
}
```

If delivery fails before the application returns an HTTP response, Echo MCP
records a transport-error delivery outcome:

```json
{
  "observations": [],
  "webhook_deliveries": [
    {
      "event_id": "evt_payment_failed_001",
      "endpoint_name": "payment-events",
      "method": "POST",
      "outcome": "transport_error",
      "error": "connection refused"
    }
  ]
}
```

## What This Proves

This example proves the complete webhook-style control-plane/data-plane loop:

1. Developer configures the application webhook endpoint.
2. AI selects the configured endpoint by name through MCP.
3. Echo MCP sends normal HTTP to the Application Under Test.
4. The application receives the event through its normal webhook handler.
5. AI inspects webhook delivery observations through MCP.

The AI controls test conditions. Echo MCP executes deterministic delivery. The
application stays production-like.
