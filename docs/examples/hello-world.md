# Echo MCP Hello World

This example is the canonical first workflow for Echo MCP.

It demonstrates the architectural boundary from ADR-0001:

- An AI agent controls Echo MCP through MCP.
- The application under test uses normal REST-style HTTP.
- The application does not know about MCP or Echo MCP-specific control behavior.

The example uses one in-memory behavior rule. It does not require UI, CLI,
authentication, persistence, admin APIs, metrics, audit logs, OpenAPI/AsyncAPI,
webhook delivery, recording, replay, or production deployment architecture.

## Workflow Summary

1. AI invokes `configure_behavior` through the MCP control plane.
2. Echo MCP stores one behavior rule in memory.
3. The application sends a normal REST request.
4. Echo MCP returns the configured HTTP response.
5. AI invokes `get_observations` through the MCP control plane.
6. Echo MCP returns observation information that explains the received request,
   matched behavior, and resulting outcome.

## Step 1: AI Configures Behavior

The AI agent calls the MCP tool `configure_behavior`.

Tool call:

```json
{
  "tool": "configure_behavior",
  "arguments": {
    "behavior_id": "hello-world-payment-ok",
    "match": {
      "method": "GET",
      "path": "/payments/123"
    },
    "outcome": {
      "type": "http_response",
      "status_code": 202,
      "body": "{\"status\":\"accepted\"}"
    }
  }
}
```

Expected tool result:

```json
{
  "configured": true,
  "behavior_id": "hello-world-payment-ok"
}
```

Echo MCP now has one in-memory behavior:

- Match `GET /payments/123`.
- Return HTTP `202`.
- Return body `{"status":"accepted"}`.
- Identify the rule as `hello-world-payment-ok` in observation output.

## Step 2: Application Sends Normal REST Request

The application under test sends a normal HTTP request through the data plane.
There is no MCP metadata, simulator-specific header, or Echo MCP control
protocol in the request.

Example request:

```http
GET /payments/123 HTTP/1.1
Host: localhost:8080
```

Equivalent local probe:

```bash
curl -i http://localhost:8080/payments/123
```

If the application sends this request before Step 1 configures the behavior, or
if it sends a different method or path, Echo MCP returns HTTP `501 Not
Implemented`. That response means no matching simulator behavior exists; it is
not a simulated external provider response.

## Step 3: Echo MCP Returns Configured Response

Because the request method and path match the configured behavior, Echo MCP
returns the configured HTTP response through the data plane.

Expected response:

```http
HTTP/1.1 202 Accepted

{"status":"accepted"}
```

From the application's perspective, this is a normal response from an external
dependency. The application does not need MCP awareness.

## Step 4: AI Retrieves Observations

The AI agent calls the MCP tool `get_observations`.

Tool call:

```json
{
  "tool": "get_observations",
  "arguments": {}
}
```

Expected tool result:

```json
{
  "observations": [
    {
      "request": {
        "method": "GET",
        "path": "/payments/123"
      },
      "selection": {
        "matched_behavior_id": "hello-world-payment-ok",
        "matched_on": ["method", "path"]
      },
      "outcome": {
        "type": "http_response",
        "status_code": 202
      }
    }
  ],
  "webhook_deliveries": []
}
```

The observation explains:

- Echo MCP received `GET /payments/123`.
- The request matched behavior `hello-world-payment-ok`.
- The behavior was selected by matching `method` and `path`.
- Echo MCP produced an `http_response` outcome with status code `202`.

Provider-like responses are returned only when explicitly configured through the
MCP control plane. An unexpected HTTP `501` should cause the AI agent to inspect
available observations and configure the missing behavior before rerunning the
test.

## What This Proves

This Hello World proves the minimal Echo MCP loop:

- Control-plane configuration happens through MCP.
- Runtime behavior is held in memory.
- Data-plane interaction is normal REST-style HTTP.
- The application under test remains unaware of MCP.
- Observation information is available through MCP for AI-driven verification.

The automated equivalent of this workflow is covered by
`TestMCPConfigureBehaviorDrivesRESTDataPlaneAndReportsObservations` in
`internal/control/mcp_server_test.go`.
