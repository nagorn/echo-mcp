# Stripe-Like PaymentIntent Scenario

This example shows a more realistic Echo MCP workflow using Stripe PaymentIntents
as a reference domain.

Stripe publishes public OpenAPI specifications and documents a REST-style
PaymentIntent confirmation endpoint. This Echo MCP example is manually derived
from that public API shape only. Echo MCP does not include Stripe-specific
behavior, built-in Stripe contracts, Stripe request validation, or
Stripe-specific response generation.

Echo MCP is not a Stripe simulator. In this scenario, Echo MCP acts as a manually
configured external REST dependency that happens to use a Stripe-like endpoint
and JSON response body.

The value of the example is that an AI agent can configure realistic external
dependency behavior without the application knowing a simulator is involved.

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
- optional validation against a developer-provided OpenAPI 3.0.x JSON contract
- no authentication
- no persistence
- no OpenAPI import, generation, or Stripe-specific contract awareness
- no Stripe-specific logic in Echo MCP

OpenAPI, when configured, is a validation constraint only. The AI agent still
constructs the complete concrete behavior; Echo MCP validates accepted behavior
and executes it deterministically.

## Workflow Summary

1. AI invokes `configure_behavior` through the MCP control plane.
2. Echo MCP stores one in-memory behavior rule.
3. The application sends a normal REST-style HTTP request.
4. Echo MCP returns the configured Stripe-like JSON response.
5. AI invokes `get_observations` through the MCP control plane.
6. Echo MCP returns observation information that explains the received request,
   matched behavior, match criteria, and resulting outcome.

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

Expected tool result:

```json
{
  "configured": true,
  "behavior_id": "stripe-like-paymentintent-card-declined"
}
```

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
Implemented`. That response means no matching simulator behavior exists; it is
a simulator setup failure, not a Stripe-like provider response.

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

The body uses Stripe's documented error envelope shape: an outer `error` object,
`type` set to `card_error`, `code` set to `card_declined`, `decline_code` set to
`generic_decline`, and a nested `payment_intent` object whose status is
`requires_payment_method`.

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

The observation explains:

- Echo MCP received `POST /v1/payment_intents/pi_123/confirm`.
- The request matched behavior `stripe-like-paymentintent-card-declined`.
- The behavior was selected by matching `method` and `path`.
- Echo MCP produced an `http_response` outcome with status code `402`.

The Stripe-like HTTP `402` response appears only because the AI agent explicitly
configured it. An unexpected HTTP `501` from Echo MCP should cause the AI agent
to inspect available observations and configure the missing behavior before
rerunning the test.

## What This Proves

This scenario proves that Echo MCP can manually simulate a realistic external
REST dependency workflow without adding dependency-specific behavior.

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
- Stripe-specific request validation
- Stripe authentication
- multiple behavior rules
- persistence
- recording or replay
- metrics or audit logs
- production deployment behavior
