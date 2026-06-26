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
controlled CI/test environments. Do not expose the HTTP data plane to untrusted
networks.

## Release

Current release:

```text
v0.1.0
```

Release page:

```text
https://github.com/nagorn/echo-mcp/releases/tag/v0.1.0
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
  https://github.com/nagorn/echo-mcp/releases/download/v0.1.0/echo-mcp_darwin_arm64.tar.gz
curl -L -o /tmp/echo-mcp-checksums.txt \
  https://github.com/nagorn/echo-mcp/releases/download/v0.1.0/checksums.txt
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

Generic configuration shape:

```text
command: /absolute/path/to/project/.codex/bin/echo-mcp
args: []
env:
  ECHO_MCP_HTTP_ADDR=127.0.0.1:18080
```

Client-specific formats differ. Echo MCP does not require a particular MCP host;
the host only needs to start the binary over stdio and pass environment
variables.

## Optional Runtime Configuration

Use a developer-provided OpenAPI 3.0.x JSON contract as a validation constraint:

```text
ECHO_MCP_OPENAPI_FILE=/absolute/path/to/project/contracts/payment.openapi.json
```

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

1. Ask the MCP client to call `configure_behavior` for `GET /hello`.
2. Send a REST request to `http://127.0.0.1:18080/hello`.
3. Ask the MCP client to call `get_observations`.
4. Confirm the observation includes the received request, matched behavior, and
   HTTP outcome.
5. Call `reset`.

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
