# Configuration Reference

Echo MCP is configured with environment variables.

The current MVP does not include CLI flags, config files, persistence, auth,
admin APIs, or production deployment configuration. It does not introduce
`echo-mcp.yaml`.

## Environment Variables

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `ECHO_MCP_HTTP_ADDR` | No | `:8080` | REST data-plane listen address. Use localhost-bound addresses such as `127.0.0.1:18080` where possible. |
| `ECHO_MCP_CONTRACT_ROOT` | No | process working directory | Directory boundary for runtime and startup OpenAPI contract loading. |
| `ECHO_MCP_OPENAPI_FILE` | No | unset | Path to one developer-provided OpenAPI 3.0 JSON file loaded at startup as a validation constraint. The path must resolve under the contract root. |
| `ECHO_MCP_WEBHOOK_ENDPOINT_NAME` | No | unset | Name of one registered application webhook endpoint. Required with `ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS` when webhook delivery is enabled. |
| `ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS` | No | unset | Address of the registered application webhook endpoint. Required with `ECHO_MCP_WEBHOOK_ENDPOINT_NAME` when webhook delivery is enabled. |

## Codex MCP Server Name

For Codex, use `echo_mcp` as the recommended MCP server name:

```toml
[mcp_servers.echo_mcp]
command = "/absolute/path/to/echo-mcp"
args = []
env = { ECHO_MCP_HTTP_ADDR = "127.0.0.1:18080", ECHO_MCP_CONTRACT_ROOT = "/absolute/path/to/project/contracts" }
```

Some MCP clients display tools under a namespace derived from the server name,
such as `echo_mcp`.

## REST Data Plane

If `ECHO_MCP_HTTP_ADDR` is unset, Echo MCP listens on:

```text
:8080
```

For local tests, prefer explicit localhost-bound addresses:

```bash
ECHO_MCP_HTTP_ADDR=127.0.0.1:18080 ./bin/echo-mcp
```

## Contract Root

`ECHO_MCP_CONTRACT_ROOT` defines the only filesystem tree from which Echo MCP may
load OpenAPI contracts.

Default behavior:

- If `ECHO_MCP_CONTRACT_ROOT` is unset, Echo MCP uses the process working
  directory as the contract root.
- Relative `load_openapi_contract.path` values and relative
  `ECHO_MCP_OPENAPI_FILE` values resolve against the contract root.
- Absolute paths are allowed only when they resolve inside the contract root.
- Paths outside the root are rejected.
- Traversal such as `../secrets.json` is rejected when it escapes the root.
- Symlink escapes are rejected where the platform can resolve symlinks.

Rejected path diagnostics are structured and avoid leaking private absolute
paths for denied locations.

Example:

```bash
ECHO_MCP_CONTRACT_ROOT=/absolute/path/to/project/contracts ./bin/echo-mcp
```

With that root, the MCP client can load:

```json
{
  "path": "stripe-openapi.spec3.json",
  "contract_id": "stripe",
  "validation_mode": "strict"
}
```

## OpenAPI Contract Loading

Echo MCP can load one local OpenAPI 3.0 JSON contract at a time.

Runtime loading is done through the MCP tool `load_openapi_contract`. Startup
loading is still available with `ECHO_MCP_OPENAPI_FILE`:

```bash
ECHO_MCP_CONTRACT_ROOT=./contracts \
ECHO_MCP_OPENAPI_FILE=payment.openapi.json \
./bin/echo-mcp
```

Startup and runtime loading share the same contract manager and the same
contract-root boundary.

The contract is a validation constraint only. Echo MCP does not generate
behavior, fetch remote specs, import specs, repair specs, or validate business
rules outside the supported contract subset.

Validation is partial response validation for supported OpenAPI 3.0 JSON
features. Strict mode means strict enforcement of supported validation
capabilities only; it is not full OpenAPI validation.

Unsupported in v0.3.0:

- OpenAPI 3.1
- YAML
- remote refs or file refs
- `allOf`, `oneOf`, or `anyOf` semantics
- discriminators
- request body validation
- query, header, or path parameter validation
- automatic scenario generation
- provider-specific simulators

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
ECHO_MCP_HTTP_ADDR=127.0.0.1:18080 ECHO_MCP_CONTRACT_ROOT=./contracts/payment ./bin/echo-mcp
ECHO_MCP_HTTP_ADDR=127.0.0.1:18081 ECHO_MCP_CONTRACT_ROOT=./contracts/fraud ./bin/echo-mcp
```

Each process needs its own MCP server registration.
