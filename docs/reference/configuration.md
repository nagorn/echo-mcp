# Configuration Reference

Echo MCP is configured with environment variables.

The current MVP does not include CLI flags, config files, persistence, auth,
admin APIs, or production deployment configuration.

## Environment Variables

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `ECHO_MCP_HTTP_ADDR` | No | `:8080` | REST data-plane listen address. Use localhost-bound addresses such as `127.0.0.1:18080` where possible. |
| `ECHO_MCP_OPENAPI_FILE` | No | unset | Path to one developer-provided OpenAPI 3.0.x JSON file used as a validation constraint for configured REST behavior. |
| `ECHO_MCP_WEBHOOK_ENDPOINT_NAME` | No | unset | Name of one registered application webhook endpoint. Required with `ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS` when webhook delivery is enabled. |
| `ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS` | No | unset | Address of the registered application webhook endpoint. Required with `ECHO_MCP_WEBHOOK_ENDPOINT_NAME` when webhook delivery is enabled. |

## REST Data Plane

If `ECHO_MCP_HTTP_ADDR` is unset, Echo MCP listens on:

```text
:8080
```

For local tests, prefer explicit localhost-bound addresses:

```bash
ECHO_MCP_HTTP_ADDR=127.0.0.1:18080 ./bin/echo-mcp
```

## OpenAPI Contract

`ECHO_MCP_OPENAPI_FILE` points to one OpenAPI 3.0.x JSON file.

Example:

```bash
ECHO_MCP_OPENAPI_FILE=./contracts/payment.openapi.json ./bin/echo-mcp
```

The contract is a validation constraint only. Echo MCP does not generate
behavior, fetch specs, import specs, repair specs, or validate business rules
outside the contract.

OpenAPI 3.1.x, YAML, and external `$ref` resolution are future work.

## Webhook Endpoint

Webhook-style event delivery requires both endpoint variables:

```bash
ECHO_MCP_WEBHOOK_ENDPOINT_NAME=payment-events \
ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS=http://127.0.0.1:3000/webhooks/payments \
./bin/echo-mcp
```

MCP clients call `send_webhook_event` with `endpoint_name`. They do not provide
raw outbound URLs.

Only configure webhook endpoints owned by the application/test environment.

## Multiple Dependencies

Use one Echo MCP process per simulated external dependency.

Example:

```bash
ECHO_MCP_HTTP_ADDR=127.0.0.1:18080 ./bin/echo-mcp
ECHO_MCP_HTTP_ADDR=127.0.0.1:18081 ./bin/echo-mcp
```

Each process needs its own MCP server registration.
