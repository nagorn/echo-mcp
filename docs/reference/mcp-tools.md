# MCP Tool Reference

Echo MCP exposes a small MCP control-plane tool surface.

Applications must not call these tools. The Application Under Test uses only the
REST data plane and its normal webhook endpoints.

## `configure_behavior`

Configures one in-memory REST-style HTTP response behavior.

Successful calls replace the currently configured behavior rule.

Input:

```json
{
  "behavior_id": "hello-ok",
  "match": {
    "method": "GET",
    "path": "/hello"
  },
  "outcome": {
    "type": "http_response",
    "status_code": 200,
    "content_type": "application/json",
    "body": "{\"message\":\"hello\"}"
  }
}
```

Fields:

- `behavior_id`: required non-empty string used in observations.
- `match.method`: required HTTP method.
- `match.path`: required HTTP path.
- `outcome.type`: required, currently only `http_response`.
- `outcome.status_code`: required HTTP status code.
- `outcome.content_type`: optional response content type.
- `outcome.body`: required response body string. Use an empty string for no
  body.

Output:

```json
{
  "configured": true,
  "behavior_id": "hello-ok"
}
```

Invalid behavior is rejected and must not replace the currently configured
behavior. If `ECHO_MCP_OPENAPI_FILE` is configured, Echo MCP validates the
behavior against the provided OpenAPI 3.0.x JSON contract before accepting it.

## `send_webhook_event`

Sends one immediate webhook-style HTTP `POST` to a registered application
webhook endpoint.

Input:

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

Fields:

- `event_id`: required non-empty string used in delivery observations.
- `endpoint_name`: required registered endpoint name.
- `request.body`: required JSON object sent to the application webhook endpoint.

The MVP supports only JSON request bodies. It does not support configurable
headers, signatures, retries, scheduling, content negotiation, or raw outbound
URLs supplied by MCP clients.

Output:

```json
{
  "attempted": true,
  "event_id": "evt_payment_failed_001",
  "endpoint_name": "payment-events",
  "delivery": {
    "outcome": "response_received",
    "status_code": 204
  }
}
```

Transport failures are delivery outcomes and are recorded in observations.

## `get_observations`

Returns currently available observation information.

Output shape:

```json
{
  "observations": [
    {
      "request": {
        "method": "GET",
        "path": "/hello"
      },
      "selection": {
        "matched_behavior_id": "hello-ok",
        "matched_on": ["method", "path"]
      },
      "outcome": {
        "type": "http_response",
        "status_code": 200
      }
    }
  ],
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

`observations` describes REST data-plane requests. `webhook_deliveries`
describes webhook delivery attempts.

If no observations are available, collections are empty.

## `reset`

Clears configured behavior and currently available observation information.

Input:

```json
{}
```

Output:

```json
{
  "reset": true,
  "cleared": ["behavior", "observations", "webhook_deliveries"]
}
```

Use `reset` between test scenarios.
