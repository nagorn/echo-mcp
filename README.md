# Echo MCP

Echo MCP is a source-available local simulator for external dependencies in
AI-assisted end-to-end tests.

Modern AI coding assistants can write application code quickly, but end-to-end
tests still depend on external systems that are slow, expensive, rate-limited,
unavailable, or hard to put into the exact failure state a test needs. That
usually means happy paths get exercised while decline paths, malformed
responses, retry behavior, webhook handling, and upstream failures stay thin.

Echo MCP gives an AI agent, or any MCP client, a deterministic way to configure
external dependency behavior through MCP while the Application Under Test keeps
using normal REST HTTP and normal webhook endpoints. The application does not
need to know Echo MCP exists.

## Who Is Echo MCP For?

- AI-assisted software development.
- Teams integrating with external REST APIs.
- Developers who want deterministic end-to-end tests.
- Engineers validating failure handling before external systems exist.
- Test harnesses that need controlled external dependency behavior without
  changing application code.

## Who Is Echo MCP Not For?

- Production API gateway.
- Reverse proxy.
- Public webhook relay.
- General-purpose service virtualization platform.
- Built-in simulator for Stripe, GitHub, Slack, or any other public API.
- Hosted service for untrusted users or public internet traffic.

## The Solution

Echo MCP has two planes:

- **Control Plane:** MCP over stdio for AI agents and other MCP clients.
- **Data Plane:** normal REST HTTP and webhook-style HTTP delivery for the
  Application Under Test.

The MCP client sends complete concrete behavior to Echo MCP. Echo MCP stores
accepted behavior in memory and executes it deterministically when the
application sends a matching request.

When a developer provides an OpenAPI 3.0.x JSON contract with
`ECHO_MCP_OPENAPI_FILE`, Echo MCP validates configured REST behavior against that
contract before accepting it. OpenAPI is a validation constraint only. Echo MCP
does not generate, fetch, import, repair, or complete OpenAPI specs, and it does
not generate behavior from OpenAPI.

## Core Philosophy

- AI decides the test scenario.
- The MCP client sends concrete behavior.
- Echo MCP validates behavior when constraints are available.
- Echo MCP executes accepted behavior deterministically.
- Applications remain production-like.
- The Control Plane stays separate from the Data Plane.

## Architecture

```mermaid
flowchart LR
    Agent["AI agent or MCP client"] -->|"MCP control plane (stdio)"| Echo["Echo MCP"]
    App["Application Under Test"] -->|"REST HTTP request"| Echo
    Echo -->|"Configured REST response"| App
    Echo -->|"Webhook-style HTTP POST"| Webhook["Application webhook endpoint"]
    Contract["Developer-provided OpenAPI 3.0.x JSON"] -.->|"Validation constraint"| Echo
```

The application must not contain MCP awareness, Echo MCP-specific branches,
simulator headers, or observation reads.

## Features

Current implemented capabilities:

- MCP control plane over stdio.
- `configure_behavior` for one in-memory REST response rule.
- REST data plane on `:8080` by default, configurable with
  `ECHO_MCP_HTTP_ADDR`.
- Configured HTTP status, response body, and optional response content type.
- HTTP `501 Not Implemented` for unmatched REST requests, which indicates
  missing simulator setup rather than a simulated provider response.
- `get_observations` for received requests, matched behavior, match criteria,
  produced outcomes, and webhook delivery attempts.
- `reset` for clearing configured behavior and observations.
- Optional OpenAPI 3.0.x JSON response validation with `ECHO_MCP_OPENAPI_FILE`.
- `send_webhook_event` for one immediate HTTP `POST` to one configured
  application webhook endpoint.
- Local Go binary, Dockerfile, and `make` commands for local development.

Current limits:

- One Echo MCP process represents one simulated external dependency.
- One process supports one OpenAPI contract, one registered webhook endpoint,
  and one active REST behavior rule.
- Multiple external dependencies require multiple Echo MCP processes with
  separate `ECHO_MCP_HTTP_ADDR` values and MCP server registrations.
- No UI, CLI, authentication, authorization, persistence, admin API, metrics,
  audit log, recording, replay, SaaS, or production deployment architecture.
- No OpenAPI import, OpenAPI generation, OpenAPI 3.1.x, YAML, external `$ref`,
  built-in public API contracts, or vendor-specific simulator behavior.
- No webhook retries, scheduling, signatures, delivery persistence, AsyncAPI,
  event bus, or inbound public webhook receiver.

## Security

Echo MCP is intentionally designed for local development and controlled test
environments.

Do not expose the HTTP data plane or MCP control plane to untrusted networks.
The MVP does not include authentication or authorization. The MCP control plane
can configure simulator behavior, and webhook-style event delivery can send
outbound HTTP requests to configured application webhook endpoints. If exposed
incorrectly, Echo MCP could be abused as an outbound request relay or DDoS
helper.

Bind or expose Echo MCP to localhost by default where possible. If running
outside a local machine, use firewalling, container/network isolation, or
private CI networking. Do not configure webhook endpoints pointing to arbitrary
third-party systems. Only configure webhook endpoints owned by the
application/test environment.

See [SECURITY.md](SECURITY.md) for release security guidance.

## License

Echo MCP is source-available under the [Elastic License 2.0](LICENSE). It is not
OSI open source.

Echo MCP is free to use for internal development and testing under the license
terms. Commercial or enterprise licensing may be offered separately later.

## Tell Your AI

Using ChatGPT, Codex, Claude, Cursor, or another MCP-capable coding assistant?

Instead of manually wiring every step, tell your AI:

```text
Install Echo MCP and configure it for this project.
```

Then point the AI to
[AI-Assisted Installation](docs/guides/ai-assisted-installation.md). The guide
is written as a deterministic checklist for AI coding assistants.

## Quick Start

Recommended installation path:

1. Download the correct binary archive from
   [Echo MCP v0.1.0](https://github.com/nagorn/echo-mcp/releases/tag/v0.1.0).
2. Download `checksums.txt`.
3. Verify the archive SHA-256 checksum.
4. Extract the `echo-mcp` binary.
5. Register that binary as an MCP stdio server.

Asset mapping:

| Platform | Asset |
| --- | --- |
| macOS Apple Silicon | `echo-mcp_darwin_arm64.tar.gz` |
| macOS Intel | `echo-mcp_darwin_amd64.tar.gz` |
| Linux amd64 | `echo-mcp_linux_amd64.tar.gz` |
| Linux arm64 | `echo-mcp_linux_arm64.tar.gz` |
| Windows amd64 | `echo-mcp_windows_amd64.zip` |

For example, on macOS Apple Silicon:

```bash
mkdir -p .codex/bin
curl -L -o /tmp/echo-mcp_darwin_arm64.tar.gz \
  https://github.com/nagorn/echo-mcp/releases/download/v0.1.0/echo-mcp_darwin_arm64.tar.gz
curl -L -o /tmp/echo-mcp-checksums.txt \
  https://github.com/nagorn/echo-mcp/releases/download/v0.1.0/checksums.txt
(cd /tmp && grep 'echo-mcp_darwin_arm64.tar.gz' echo-mcp-checksums.txt | shasum -a 256 -c -)
tar -xzf /tmp/echo-mcp_darwin_arm64.tar.gz -C .codex/bin
chmod +x .codex/bin/echo-mcp
```

Register Echo MCP as an MCP stdio server in your MCP host:

```text
command: /absolute/path/to/project/.codex/bin/echo-mcp
args: []
env:
  ECHO_MCP_HTTP_ADDR=127.0.0.1:18080
```

Client-specific MCP configuration formats vary. Echo MCP only requires that the
host starts the binary over stdio and passes any needed environment variables.

Building from source is the fallback path for development, unsupported
platforms, or users who specifically want to inspect or modify the source before
running Echo MCP.

Ask the MCP client to call `configure_behavior`:

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

Send a normal REST request through the data plane:

```bash
curl -i http://127.0.0.1:18080/hello
```

Expected response:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"message":"hello from Echo MCP"}
```

Ask the MCP client to call `get_observations`. It should report the received
request, the matched behavior `hello-ok`, the match criteria, and the produced
HTTP outcome.

Call `reset` before the next scenario.

This direct `curl` request is a wiring smoke test. In real end-to-end tests, the
Application Under Test should make the REST request through its normal external
dependency configuration.

## Multiple External Dependencies

MVP usage: one Echo MCP process represents one simulated external dependency.

For an application that talks to several external dependencies, run one Echo MCP
process per dependency and give each process its own `ECHO_MCP_HTTP_ADDR` and
MCP server registration.

Example:

```bash
ECHO_MCP_HTTP_ADDR=127.0.0.1:18080 \
ECHO_MCP_OPENAPI_FILE=./contracts/payment.openapi.json \
./bin/echo-mcp

ECHO_MCP_HTTP_ADDR=127.0.0.1:18081 \
ECHO_MCP_OPENAPI_FILE=./contracts/fraud.openapi.json \
./bin/echo-mcp
```

Point the application's payment base URL at `http://127.0.0.1:18080` and its
fraud base URL at `http://127.0.0.1:18081`.

Multi-dependency support inside one Echo MCP process is future work and would
require a future ADR.

## Hello World

Start with [Echo MCP Hello World](docs/examples/hello-world.md).

It shows the canonical workflow:

1. Configure one behavior through MCP.
2. Send a normal REST request from the application.
3. Receive the configured HTTP response.
4. Inspect observations through MCP.

If the application calls the REST data plane before a matching behavior is
configured, Echo MCP returns HTTP `501 Not Implemented`. Provider-like responses
are returned only when explicitly configured.

## Real-World Example

See the
[Stripe-like PaymentIntent scenario](docs/examples/stripe-paymentintent-scenario.md)
for a realistic payment-confirmation failure.

The example is manually derived from Stripe's public API shape. Echo MCP is not a
Stripe simulator and does not include Stripe-specific behavior.

## Webhook Example

See
[Webhook-Style Event Delivery](docs/examples/webhook-style-event-delivery.md)
for the current webhook slice.

Webhook endpoint addresses are configured by the developer or test harness with
`ECHO_MCP_WEBHOOK_ENDPOINT_NAME` and `ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS`. MCP
clients call `send_webhook_event` using `endpoint_name`; they do not provide raw
outbound URLs.

## Documentation

| Area | Document |
| --- | --- |
| Architecture decisions | [ADRs](docs/adr/) |
| Concepts | [Scope](docs/concepts/scope.md), [Terminology](docs/concepts/terminology.md), [Core Abstractions](docs/concepts/core-abstractions.md) |
| Configuration | [Configuration Reference](docs/reference/configuration.md) |
| MCP tools | [MCP Tool Reference](docs/reference/mcp-tools.md) |
| Unmatched requests | [Unmatched REST Requests](docs/reference/unmatched-rest-requests.md) |
| Developer workflow | [Developer Usage Guide](docs/guides/developer-usage.md) |
| AI workflow | [AI Agent Usage Guide](docs/guides/ai-agent-usage.md) |
| Installation | [Installation Guide](docs/guides/installation.md) |
| AI installation | [AI-Assisted Installation](docs/guides/ai-assisted-installation.md) |
| Installation and release | [Installation and Release Guidance](docs/guides/installation-and-release.md) |
| Examples | [Hello World](docs/examples/hello-world.md), [Stripe-like PaymentIntent](docs/examples/stripe-paymentintent-scenario.md), [Webhook-style Event Delivery](docs/examples/webhook-style-event-delivery.md) |
| Agent instructions | [Copy-pasteable agent template](docs/templates/agent-instructions.md) |

## Roadmap

Current implemented capabilities:

- One in-memory REST response behavior.
- Configurable REST data-plane listen address with `ECHO_MCP_HTTP_ADDR`.
- Deterministic HTTP `501` response for unmatched REST requests.
- Optional OpenAPI 3.0.x JSON validation constraint.
- Immediate single-attempt webhook-style event delivery to one configured
  application webhook endpoint.
- In-memory observations and reset.

Current public release:

- `v0.1.0`

Active next work:

- Documentation and example hardening based on real project usage.

## Contributing

Echo MCP is maintained primarily by the author. Feedback and focused fixes are
welcome, but substantial feature work should start with an issue or design
discussion first.

Before proposing changes, read [CONTRIBUTING.md](CONTRIBUTING.md), run:

```bash
make test
make build
git diff --check
```

---

🛠 **Built on the Workbench**

This repository is one of many small tools from **Nagorn's Lab**.

Every tool here exists because some repeated task became annoying enough to fix.

🏠 https://nagorn.io
