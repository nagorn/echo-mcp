# AI-Assisted Installation

This document is written for AI coding assistants.

Its purpose is to help an AI install, configure, verify, and use Echo MCP
without requiring human intervention. Developers may also use it as a checklist.

Echo MCP is source-available under the Elastic License 2.0. It is not OSI open
source.

Echo MCP is free to use for internal development and testing under the license
terms. Commercial or enterprise licensing may be offered separately later.

## Expected Result

After installation:

- Echo MCP binary is available.
- Echo MCP is registered as an MCP stdio server.
- REST data plane is reachable.
- The AI can discover the available MCP tools:
  - `configure_behavior`
  - `reset`
  - `get_observations`
  - `send_webhook_event`
- The smoke test succeeds.
- The Application Under Test remains unaware of MCP and Echo MCP.

## Discovery

Before installing Echo MCP into another project, determine whether the developer
actually wants it for that project.

Recommended discovery order:

1. User instruction in the current task.
2. Project-local AI instruction files such as `AGENTS.md`, `CLAUDE.md`, Cursor
   rules, GitHub Copilot instructions, or equivalent.
3. Project documentation.
4. Existing E2E test examples.
5. Available MCP tools.

If Echo MCP is not requested or configured, do not assume it is part of the
project's test workflow.

## Installation Steps

Prefer GitHub Release binaries unless the user explicitly asks to build from
source or the current platform is unsupported.

### 1. Detect OS And Architecture

Detect the host OS and CPU architecture, then choose the matching v0.1.0 asset.

| Platform | Asset |
| --- | --- |
| macOS Apple Silicon | `echo-mcp_darwin_arm64.tar.gz` |
| macOS Intel | `echo-mcp_darwin_amd64.tar.gz` |
| Linux amd64 | `echo-mcp_linux_amd64.tar.gz` |
| Linux arm64 | `echo-mcp_linux_arm64.tar.gz` |
| Windows amd64 | `echo-mcp_windows_amd64.zip` |

Release page:

```text
https://github.com/nagorn/echo-mcp/releases/tag/v0.1.0
```

### 2. Download And Verify The Release Binary

Create a project-local tool directory:

```bash
mkdir -p .codex/bin
```

Download the selected archive and `checksums.txt`. Example for macOS Apple
Silicon:

```bash
curl -L -o /tmp/echo-mcp_darwin_arm64.tar.gz \
  https://github.com/nagorn/echo-mcp/releases/download/v0.1.0/echo-mcp_darwin_arm64.tar.gz
curl -L -o /tmp/echo-mcp-checksums.txt \
  https://github.com/nagorn/echo-mcp/releases/download/v0.1.0/checksums.txt
```

Verify SHA-256 before extraction:

```bash
(cd /tmp && grep 'echo-mcp_darwin_arm64.tar.gz' echo-mcp-checksums.txt | shasum -a 256 -c -)
```

Extract the binary:

```bash
tar -xzf /tmp/echo-mcp_darwin_arm64.tar.gz -C .codex/bin
chmod +x .codex/bin/echo-mcp
```

For Windows, extract `echo-mcp.exe` from the `.zip` archive and use that
executable path in MCP configuration.

The binary should live at a project-local path such as:

```text
/absolute/path/to/project/.codex/bin/echo-mcp
```

### 3. Choose the Dependency Process Model

Use one Echo MCP process per simulated external dependency.

For one dependency, use one MCP server registration and one REST data-plane
address.

For multiple dependencies, use multiple Echo MCP process registrations:

```text
payment dependency  -> ECHO_MCP_HTTP_ADDR=127.0.0.1:18080
fraud dependency    -> ECHO_MCP_HTTP_ADDR=127.0.0.1:18081
shipping dependency -> ECHO_MCP_HTTP_ADDR=127.0.0.1:18082
```

Do not configure one Echo MCP process to represent multiple independent
dependencies. That is future work and would require a future ADR.

### 4. Register the MCP Server

Register Echo MCP as a stdio MCP server in the project's MCP host.

Generic shape:

```text
command: /absolute/path/to/project/.codex/bin/echo-mcp
args: []
env:
  ECHO_MCP_HTTP_ADDR=127.0.0.1:18080
```

If contract validation is needed, add:

```text
ECHO_MCP_OPENAPI_FILE=/absolute/path/to/project/contracts/payment.openapi.json
```

If webhook-style event delivery is needed, add:

```text
ECHO_MCP_WEBHOOK_ENDPOINT_NAME=payment-events
ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS=http://127.0.0.1:3000/webhooks/payments
```

The webhook endpoint address must be configured by the developer or test
harness. AI agents select configured endpoint names; they do not provide raw
outbound URLs.

### 5. Point the Application at Echo MCP

Configure only the application's normal external dependency base URL or runtime
endpoint setting.

Example:

```text
PAYMENT_API_BASE_URL=http://127.0.0.1:18080
```

Do not modify application code to add Echo MCP-specific branches, simulator
headers, MCP awareness, or observation reads.

### 6. Run the Smoke Test

Ask the MCP host to call `configure_behavior`:

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
    "body": "{\"message\":\"hello from Echo MCP\"}"
  }
}
```

Send a diagnostic REST request:

```bash
curl -i http://127.0.0.1:18080/hello
```

Expected response:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"message":"hello from Echo MCP"}
```

Ask the MCP host to call `get_observations`. Confirm it reports:

- request method `GET`
- request path `/hello`
- matched behavior `hello-ok`
- match criteria `method` and `path`
- outcome type `http_response`
- status code `200`

Call `reset` before the next scenario.

### 7. Use Echo MCP in an E2E Test

For a real test:

1. Read the test objective.
2. Identify the external dependency behavior to simulate.
3. Configure Echo MCP through `configure_behavior` or `send_webhook_event`.
4. Run the application E2E test through the application.
5. Inspect `get_observations`.
6. Verify application behavior with normal assertions.
7. Call `reset`.

Do not bypass the application by treating Echo MCP's REST data plane as the test
subject.

## Source Build Fallback

Build from source only when the user asks for it, the platform is unsupported by
the release assets, or the user wants to inspect or modify the source before
running Echo MCP.

```bash
git clone https://github.com/nagorn/echo-mcp.git
cd echo-mcp
make test
make build
```

The source-built binary is:

```text
/absolute/path/to/echo-mcp/bin/echo-mcp
```

## Stripe-Like Payment Decline Example

For a Stripe-like PaymentIntent confirmation failure, configure:

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

Then run the application checkout test normally. After the test, inspect
observations and verify that the application handled the decline correctly.

Echo MCP is not a Stripe simulator. This behavior is manually configured by the
MCP client based on the test scenario and any developer-supplied API contract.

## Unmatched REST Requests

If the Application Under Test calls Echo MCP before a matching behavior is
configured, Echo MCP returns HTTP `501 Not Implemented`.

Treat that as simulator setup failure, not provider behavior. Inspect
observations, configure the missing behavior, and rerun the test. Provider-like
responses such as `404`, `500`, or `503` should be returned only when explicitly
configured.

## Safety Rules for AI Agents

- Use MCP tools to configure Echo MCP.
- Keep application code production-like.
- Do not add Echo MCP-specific branches.
- Do not expose Echo MCP to untrusted networks.
- Do not configure webhook endpoints pointing to arbitrary third-party systems.
- Do not provide raw outbound URLs through MCP.
- Respect developer-supplied API contracts.
- Do not invent unsupported Echo MCP features.
- Reset Echo MCP between scenarios.

## References

- [README](../../README.md)
- [Developer Usage Guide](developer-usage.md)
- [AI Agent Usage Guide](ai-agent-usage.md)
- [MCP Tool Reference](../reference/mcp-tools.md)
- [Configuration Reference](../reference/configuration.md)
- [Hello World](../examples/hello-world.md)
- [Stripe-like PaymentIntent scenario](../examples/stripe-paymentintent-scenario.md)
- [Webhook-style event delivery](../examples/webhook-style-event-delivery.md)
