# Stripe-Like PaymentIntent Scenario

This example shows a realistic Echo MCP workflow using Stripe PaymentIntents as a
reference domain.

Stripe publishes public OpenAPI specifications and documents a REST-style
PaymentIntent confirmation endpoint. This Echo MCP example is manually derived
from that public API shape. Echo MCP does not include Stripe-specific behavior,
built-in Stripe contracts, Stripe request validation, or Stripe-specific
response generation.

Echo MCP is not a Stripe simulator. Stripe OpenAPI can be used as a real-provider
compatibility smoke probe, but the validator is provider-neutral and the MCP
client still supplies complete concrete behavior.

## References

- [Stripe OpenAPI repository](https://github.com/stripe/openapi)
- [Stripe PaymentIntent confirm API reference](https://docs.stripe.com/api/payment_intents/confirm)
- [Stripe decline codes](https://docs.stripe.com/declines/codes)

## Scenario

The application under test attempts to confirm a payment intent against a
simulated Stripe-like external dependency.

The AI agent wants to test how the application handles a payment confirmation
failure where the payment intent returns to `requires_payment_method` and carries
a card decline error.

The example uses one behavior rule:

- Data-plane request: `POST /v1/payment_intents/pi_123/confirm`
- Configured outcome: HTTP `402`
- Contract target content type: `application/json`
- Response body: Stripe-like error envelope JSON

This remains inside the current MVP scope:

- one in-memory behavior
- one REST-style HTTP request/response interaction
- MCP control plane for AI-driven configuration and verification
- optional partial validation against a developer-provided OpenAPI 3.0 JSON
  contract
- no authentication
- no persistence
- no OpenAPI-driven response generation
- no request body validation
- no Stripe-specific logic in Echo MCP

OpenAPI, when loaded, is a validation constraint only. The AI agent still
constructs the complete concrete behavior; Echo MCP validates accepted behavior
for supported response capabilities and executes it deterministically.

Strict mode means strict enforcement of supported validation capabilities only.
It is not full OpenAPI validation.

## Optional Contract-Backed Setup

If a local Stripe OpenAPI JSON file is available under the contract root, an MCP
client can load it before configuring behavior:

```json
{
  "path": "stripe-openapi.spec3.json",
  "contract_id": "stripe",
  "validation_mode": "strict"
}
```

Then call `get_contract_status` and inspect `validation_scope`,
`validation_capabilities`, and `validation_mode_description`. Echo MCP supports
local internal `$ref` response schemas and common object/primitive response
schema features, but it does not support OpenAPI 3.1, YAML, remote/file refs,
`allOf`, `oneOf`, `anyOf`, request body validation, query/header/path parameter
validation, automatic scenario generation, or provider-specific simulation.

For intentional malformed Stripe-like responses, pass `validation.mode = "off"`
with a non-empty reason.

## Workflow Summary

1. AI optionally invokes `load_openapi_contract` and `get_contract_status`.
2. AI invokes `configure_behavior` through the MCP control plane.
3. Echo MCP stores one in-memory behavior rule.
4. The application sends a normal REST-style HTTP request.
5. Echo MCP returns the configured Stripe-like JSON response.
6. AI invokes `get_observations` through the MCP control plane.
7. Echo MCP returns observation information that explains the received request,
   matched behavior, match criteria, and resulting outcome.
8. AI invokes `reset` before the next scenario; the active contract remains
   loaded.

## Step 1: AI Configures the Payment Failure

The AI agent calls the MCP tool `configure_behavior`.

Tool call:

```json
{
  "tool": "configure_behavior",
  "arguments": {
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
}
```

Expected tool result includes:

```json
{
  "configured": true,
  "behavior_id": "stripe-like-paymentintent-card-declined"
}
```

When a contract is active, the result also includes `validation_scope`,
`validation_capabilities`, and `validation_mode_description`.

Echo MCP now has one in-memory behavior:

- Match `POST /v1/payment_intents/pi_123/confirm`.
- Return HTTP `402`.
- Return `Content-Type: application/json`.
- Return a Stripe-like error envelope JSON body.
- Identify the rule as `stripe-like-paymentintent-card-declined` in observation
  output.

## Step 2: Application Sends a Normal REST Request

The application under test sends the request it would normally send to an
external REST dependency.

Example request:

```http
POST /v1/payment_intents/pi_123/confirm HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded

payment_method=pm_card_visa
```

Equivalent local probe:

```bash
curl -i \
  -X POST \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'payment_method=pm_card_visa' \
  http://localhost:8080/v1/payment_intents/pi_123/confirm
```

The application request does not include MCP metadata, Echo MCP control data, or
simulator-specific request fields.

If the application sends this request before Step 1 configures the behavior, or
if it sends a different method or path, Echo MCP returns HTTP `501 Not
Implemented`. That response means no matching simulator behavior exists; it is a
simulator setup failure, not a Stripe-like provider response.

## Step 3: Echo MCP Returns the Configured Response

Because the method and path match the configured behavior, Echo MCP returns the
configured HTTP response through the data plane.

Expected response:

```http
HTTP/1.1 402 Payment Required
Content-Type: application/json

{
  "error": {
    "type": "card_error",
    "code": "card_declined",
    "decline_code": "generic_decline",
    "message": "Your card was declined.",
    "payment_intent": {
      "id": "pi_123",
      "object": "payment_intent",
      "status": "requires_payment_method"
    }
  }
}
```

From the application's perspective, this is a normal REST response from an
external dependency. Echo MCP does not need the application to know that MCP is
involved.

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
        "method": "POST",
        "path": "/v1/payment_intents/pi_123/confirm"
      },
      "selection": {
        "matched_behavior_id": "stripe-like-paymentintent-card-declined",
        "matched_on": ["method", "path"]
      },
      "outcome": {
        "type": "http_response",
        "status_code": 402
      }
    }
  ],
  "webhook_deliveries": []
}
```

The Stripe-like HTTP `402` response appears only because the AI agent explicitly
configured it. An unexpected HTTP `501` from Echo MCP should cause the AI agent
to inspect available observations and configure the missing behavior before
rerunning the test.

## What This Proves

This scenario proves that Echo MCP can simulate a realistic external REST
dependency workflow without adding dependency-specific behavior.

The important architectural boundary remains unchanged:

- AI control happens through MCP.
- Application traffic uses normal REST-style HTTP.
- Runtime behavior is in memory.
- Observations are retrieved through MCP.
- The application under test remains unaware of MCP and Echo MCP.

## Current MVP Limits

This example intentionally does not demonstrate:

- Stripe-specific OpenAPI import, generation, or built-in contract awareness
- OpenAPI-driven response generation
- full OpenAPI-first runtime
- Stripe-specific request validation
- request body, query, header, or path parameter validation
- OpenAPI 3.1 or YAML
- remote/file refs
- `allOf`, `oneOf`, or `anyOf` semantics
- Stripe authentication
- multiple behavior rules
- persistence
- recording or replay
- metrics or audit logs
- production deployment behavior
