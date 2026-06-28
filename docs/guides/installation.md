# Installation

The recommended installation path is:

```text
GitHub Releases binary -> verify checksum -> extract -> register as MCP stdio server
```

Use source builds only as a fallback for development, unsupported platforms, or
when you specifically want to inspect or modify the source before running Echo
MCP.

Echo MCP is source-available under the Elastic License 2.0. It is not OSI open
source.

Echo MCP is intended for local development, local AI-assisted testing, and
controlled CI/test environments. Do not expose the HTTP data plane or MCP
control plane to untrusted networks.

## Release

Current release:

```text
v0.3.0
```

Release page:

```text
https://github.com/nagorn/echo-mcp/releases/tag/v0.3.0
```

## Choose The Asset

| Platform | Asset |
| --- | --- |
| macOS Apple Silicon | `echo-mcp_darwin_arm64.tar.gz` |
| macOS Intel | `echo-mcp_darwin_amd64.tar.gz` |
| Linux amd64 | `echo-mcp_linux_amd64.tar.gz` |
| Linux arm64 | `echo-mcp_linux_arm64.tar.gz` |
| Windows amd64 | `echo-mcp_windows_amd64.zip` |

## Install A Release Binary

Create a project-local tool directory:

```bash
mkdir -p .codex/bin
```

Download the correct archive and `checksums.txt` from the release page. Example
for macOS Apple Silicon:

```bash
curl -L -o /tmp/echo-mcp_darwin_arm64.tar.gz \
  https://github.com/nagorn/echo-mcp/releases/download/v0.3.0/echo-mcp_darwin_arm64.tar.gz
curl -L -o /tmp/echo-mcp-checksums.txt \
  https://github.com/nagorn/echo-mcp/releases/download/v0.3.0/checksums.txt
```

Verify the SHA-256 checksum:

```bash
(cd /tmp && grep 'echo-mcp_darwin_arm64.tar.gz' echo-mcp-checksums.txt | shasum -a 256 -c -)
```

Extract the binary:

```bash
tar -xzf /tmp/echo-mcp_darwin_arm64.tar.gz -C .codex/bin
chmod +x .codex/bin/echo-mcp
```

For Windows, extract `echo-mcp.exe` from the `.zip` archive and configure the
MCP server command to point at that executable.

## Register As MCP Stdio Server

Register Echo MCP in your MCP host as a stdio server.

Codex-compatible configuration shape:

```toml
[mcp_servers.echo_mcp]
command = "/absolute/path/to/project/.codex/bin/echo-mcp"
args = []
env = { ECHO_MCP_HTTP_ADDR = "127.0.0.1:18080", ECHO_MCP_CONTRACT_ROOT = "/absolute/path/to/project/contracts" }
```

The recommended MCP server name is `echo_mcp`; some clients may display tool
names under that namespace. Client-specific formats differ. Echo MCP does not
require a particular MCP host; the host only needs to start the binary over
stdio and pass environment variables.

## Optional Runtime Configuration

Use `ECHO_MCP_CONTRACT_ROOT` to bound OpenAPI contract loading:

```text
ECHO_MCP_CONTRACT_ROOT=/absolute/path/to/project/contracts
```

If unset, Echo MCP uses the process working directory as the contract root.
Runtime `load_openapi_contract` paths and startup `ECHO_MCP_OPENAPI_FILE` paths
must resolve under this root.

Startup-load one developer-provided OpenAPI 3.0 JSON contract:

```text
ECHO_MCP_OPENAPI_FILE=provider.openapi.json
```

Runtime loading through MCP is usually preferable for AI agents and test
harnesses that start Echo MCP first, then load a contract before configuring
behavior.

Register one application webhook endpoint:

```text
ECHO_MCP_WEBHOOK_ENDPOINT_NAME=payment-events
ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS=http://127.0.0.1:3000/webhooks/payments
```

Webhook endpoint addresses must be controlled by the developer or test harness.
AI agents select configured endpoint names; they must not provide arbitrary
outbound URLs.

## Smoke Verification

After MCP registration:

1. Confirm the MCP client can see Echo MCP initialize instructions.
2. Confirm `tools/list` includes `load_openapi_contract`,
   `get_contract_status`, `unload_openapi_contract`, `configure_behavior`,
   `get_observations`, `reset`, and `send_webhook_event`.
3. Confirm guidance surfaces describe partial validation, not full OpenAPI
   validation.
4. Ask the MCP client to call `configure_behavior` for `GET /hello`.
5. Confirm the MCP result includes `configured`, `behavior_id`, and additive
   guidance fields such as `warnings`, `guidance`, or `suggested_next_actions`.
6. Send a REST request to `http://127.0.0.1:18080/hello`.
7. Ask the MCP client to call `get_observations`.
8. Confirm the observation includes the received request, matched behavior, and
   HTTP outcome.
9. Call `reset`.

Example behavior:

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

Example diagnostic request:

```bash
curl -i http://127.0.0.1:18080/hello
```

In real end-to-end tests, the Application Under Test should make the request
through its normal external dependency configuration.

Agent guidance and validation warnings are returned only through MCP
control-plane surfaces. They do not change REST data-plane response bodies.

## Contract-Backed Smoke Verification

For contract-backed behavior:

1. Put a local OpenAPI 3.0 JSON contract under `ECHO_MCP_CONTRACT_ROOT`.
2. Start Echo MCP.
3. Call `load_openapi_contract`.
4. Call `get_contract_status` and inspect `validation_scope`,
   `validation_capabilities`, and `validation_mode_description`.
5. Call `configure_behavior` with a contract-valid response.
6. Confirm invalid strict responses are rejected through MCP.
7. Use `validation.mode = "off"` with a reason for intentional fault tests.
8. Call `reset` and confirm the contract remains active.
9. Call `unload_openapi_contract` when switching contract contexts.

Validation is partial response validation for supported OpenAPI 3.0 JSON
features. Strict means strict for supported capabilities only.

## Source Build Fallback

Build from source when you are developing Echo MCP, running on an unsupported
platform, or deliberately inspecting/modifying the source before execution.

```bash
git clone https://github.com/nagorn/echo-mcp.git
cd echo-mcp
make test
make build
```

The local binary is written to:

```text
./bin/echo-mcp
```
