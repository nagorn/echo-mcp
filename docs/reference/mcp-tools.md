# MCP Control-Plane Reference

Echo MCP exposes a small MCP control-plane surface. Applications must not call
these tools, prompts, or resources. The Application Under Test uses only the
REST data plane and its normal webhook endpoints.

## Agent Guidance Surfaces

v0.2.0 adds MCP Standard agent guidance surfaces. They are advisory and help MCP
clients choose a workflow before silently defaulting to manual mock behavior.

- `initialize` instructions explain what Echo MCP is, the control-plane/data-plane
  boundary, manual mock caveats, and recommended first steps.
- `tools/list` descriptions explain what each tool does, when to use it, when
  not to use it, common next steps, and whether the behavior is manual mock or
  contract-related.
- Tool annotations provide MCP-standard hints such as read-only, destructive,
  idempotent, and open-world behavior where supported by the client.
- `prompts/list` exposes guidance prompts for AI coding agents.
- `resources/list` exposes concise markdown guides.

Agent behavior may vary by MCP client. Some clients may not automatically read
prompts or resources unless instructed to do so.

## Prompts

Echo MCP exposes these prompts:

- `echo_mcp_getting_started`: inspect tools, prompts, and resources; choose a
  workflow; use `configure_behavior` only for REST manual mock behavior; inspect
  observations; reset between scenarios.
- `echo_mcp_choose_workflow`: choose between `manual_mock`, `hybrid_validation`,
  and `contract_first`.
- `echo_mcp_manual_mock_workflow`: configure exact manual behavior, document the
  non-contract-validated caveat, run the application test, inspect observations,
  and reset.
- `echo_mcp_contract_validation_workflow`: prefer contract-backed or hybrid
  validation when an OpenAPI contract exists and provider fidelity matters.

## Resources

Echo MCP exposes these `text/markdown` resources:

- `echo://guides/getting-started`
- `echo://guides/workflows`
- `echo://guides/manual-mock`
- `echo://guides/contract-validation`

These resources are guidance only. They do not create a project manifest, change
runtime behavior, or make Echo MCP OpenAPI-first.

## Tools

Echo MCP exposes four tools: `configure_behavior`, `send_webhook_event`,
`get_observations`, and `reset`.

## `configure_behavior`

Configures one REST data-plane response rule as manual mock behavior. Successful
calls replace the currently configured behavior rule.

Use it for quick simulation, exploration, or a documented `manual_mock` workflow.
Do not use it alone to prove provider contract fidelity. If contract fidelity
matters, look for OpenAPI-backed validation or ask the developer whether a
contract is available.

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

Output includes the original backward-compatible fields plus additive guidance
fields when applicable:

```json
{
  "configured": true,
  "behavior_id": "hello-ok",
  "warnings": [
    "Manual mock behavior is active. This behavior is not contract-validated. If provider contract fidelity matters, consider OpenAPI-backed validation or hybrid validation."
  ],
  "guidance": [
    "Manual mock behavior is active for configured REST behavior."
  ],
  "suggested_next_actions": [
    "Run the application test normally.",
    "Call get_observations to inspect data-plane evidence."
  ]
}
```

Strict MCP clients should tolerate additional structured result fields. The
guidance fields are returned only through the MCP control plane. Echo MCP does
not mutate REST data-plane response bodies or headers to carry this warning.

Invalid behavior is rejected and must not replace the currently configured
behavior. If `ECHO_MCP_OPENAPI_FILE` is configured, Echo MCP validates the
behavior against the provided OpenAPI 3.0.x JSON contract before accepting it.
This validation constraint does not import, generate, or own provider contracts.

## `send_webhook_event`

Sends one immediate webhook-style HTTP `POST` to a registered application
webhook endpoint. This is not provider contract validation and is not for
arbitrary outbound URLs.

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
  },
  "suggested_next_actions": [
    "Assert application behavior normally.",
    "Call get_observations to inspect webhook delivery evidence."
  ]
}
```

Transport failures are delivery outcomes and are recorded in observations.

## `get_observations`

Returns currently available observation information. This tool is read-only and
provides test evidence; it does not configure behavior and does not make manual
mocks contract-validated.

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
  ],
  "guidance": [
    "Use observations as Echo MCP evidence and keep application behavior assertions in the application test."
  ]
}
```

`observations` describes REST data-plane requests. `webhook_deliveries`
describes webhook delivery attempts. If no observations are available,
collections are empty.

## `reset`

Clears configured behavior and currently available observation information. Use
`reset` between test scenarios.

Input:

```json
{}
```

Output:

```json
{
  "reset": true,
  "cleared": ["behavior", "observations", "webhook_deliveries"],
  "suggested_next_actions": [
    "Configure the next behavior or webhook scenario."
  ]
}
```

## Not Included In v0.2.0

v0.2.0 does not introduce `echo-mcp.yaml`, a project manifest, a full
OpenAPI-first runtime, provider-specific simulators, or a public contract
repository. Manual mocks remain supported but do not guarantee provider contract
fidelity.
