# Installation and Release Guidance

This guide covers local installation, release packaging expectations, and
verification checks for Echo MCP. It does not add product surfaces or production
deployment architecture.

Echo MCP is source-available under the Elastic License 2.0. It is not OSI open
source.

Echo MCP is free to use for internal development and testing under the license
terms. Commercial or enterprise licensing may be offered separately later.

## Requirements

- An MCP host that can run a local stdio MCP server.
- `curl` or another HTTPS download tool.
- `shasum`, `sha256sum`, or equivalent SHA-256 verification tool.
- Go `1.26.2` or compatible Go `1.26.x` toolchain only when building from
  source.
- `make` only when building from source.
- Docker only if you want to build the local container image.

## Recommended Install From GitHub Releases

For normal users and AI-assisted installation, prefer release binaries:

```text
GitHub Releases binary -> verify checksum -> extract -> register as MCP stdio server
```

Release page:

```text
https://github.com/nagorn/echo-mcp/releases/tag/v0.1.0
```

Asset mapping:

| Platform | Asset |
| --- | --- |
| macOS Apple Silicon | `echo-mcp_darwin_arm64.tar.gz` |
| macOS Intel | `echo-mcp_darwin_amd64.tar.gz` |
| Linux amd64 | `echo-mcp_linux_amd64.tar.gz` |
| Linux arm64 | `echo-mcp_linux_arm64.tar.gz` |
| Windows amd64 | `echo-mcp_windows_amd64.zip` |

Example for macOS Apple Silicon:

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

For Windows, extract `echo-mcp.exe` from the `.zip` archive.

## Source Build Fallback

Build from source when developing Echo MCP, running on an unsupported platform,
or intentionally inspecting or modifying the source before execution.

Clone the public repository:

```bash
git clone https://github.com/nagorn/echo-mcp.git
cd echo-mcp
```

Verify and build:

```bash
make test
make build
```

The source-built binary is written to:

```text
./bin/echo-mcp
```

## MCP Stdio Configuration

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

If you built from source instead, point the command at the source-built
`bin/echo-mcp` path.

## Runtime Configuration

### REST Data Plane

By default, the REST data plane listens on:

```text
:8080
```

Set a different listen address with:

```bash
ECHO_MCP_HTTP_ADDR=127.0.0.1:18080 ./bin/echo-mcp
```

Bind to localhost where possible. Do not expose Echo MCP to untrusted networks.

### OpenAPI Contract Constraint

Optional contract validation can be enabled with one developer-provided OpenAPI
3.0.x JSON file:

```bash
ECHO_MCP_OPENAPI_FILE=./contracts/payment.openapi.json ./bin/echo-mcp
```

The file should live with the application test fixtures or another
developer-controlled contract directory. Echo MCP does not fetch, import,
generate, repair, or complete OpenAPI specifications. The contract is a
validation constraint only.

OpenAPI 3.1.x, YAML, and external `$ref` resolution are not part of the current
MVP.

### Webhook Endpoint

Optional webhook-style event delivery can be enabled with one registered
application webhook endpoint:

```bash
ECHO_MCP_WEBHOOK_ENDPOINT_NAME=payment-events \
ECHO_MCP_WEBHOOK_ENDPOINT_ADDRESS=http://127.0.0.1:3000/webhooks/payments \
./bin/echo-mcp
```

MCP clients select the configured endpoint by `endpoint_name`. They must not
provide raw outbound URLs.

## Dependency Process Model

MVP usage: one Echo MCP process represents one simulated external dependency.

The current implementation supports one OpenAPI contract, one registered
webhook endpoint, and one active REST behavior rule per process.

For an application that integrates with several external dependencies, run one
Echo MCP process per dependency. Give each process its own:

- `ECHO_MCP_HTTP_ADDR`
- MCP server registration in the project's MCP host
- optional `ECHO_MCP_OPENAPI_FILE`
- optional webhook endpoint configuration

Example:

```bash
ECHO_MCP_HTTP_ADDR=127.0.0.1:18080 \
ECHO_MCP_OPENAPI_FILE=./contracts/payment.openapi.json \
./bin/echo-mcp

ECHO_MCP_HTTP_ADDR=127.0.0.1:18081 \
ECHO_MCP_OPENAPI_FILE=./contracts/fraud.openapi.json \
./bin/echo-mcp
```

Multi-dependency support inside one Echo MCP process is future work and would
require a future ADR.

## Smoke Test

After registering Echo MCP with an MCP host:

1. Ask the MCP client to call `configure_behavior` for `GET /hello`.
2. Send a REST request to the configured data-plane address.
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

## Security Note

Echo MCP is intended for local development, local AI-assisted testing, and
controlled CI/test environments only.

Do not expose Echo MCP directly to the public internet. The MVP intentionally
does not include authentication or authorization. The MCP control plane can
configure simulator behavior, and webhook-style event delivery can send
outbound HTTP requests to configured application webhook endpoints. If exposed
incorrectly, Echo MCP could be abused as an outbound request relay or DDoS
helper.

Bind or expose Echo MCP to localhost by default where possible. If running
outside a local machine, use firewalling, container/network isolation, or
private CI networking. Do not configure webhook endpoints pointing to arbitrary
third-party systems. Only configure webhook endpoints owned by the
application/test environment.

## Release Strategy

Recommended first public version:

```text
v0.1.0
```

Use semantic versioning after the first public release. Before `v1.0.0`, minor
versions may include breaking changes when the release notes call them out.

## Release Artifact Layout

Do not generate binaries until the maintainer explicitly approves a release.

Recommended future GitHub release artifacts:

```text
echo-mcp_<version>_darwin_arm64.tar.gz
echo-mcp_<version>_darwin_amd64.tar.gz
echo-mcp_<version>_linux_arm64.tar.gz
echo-mcp_<version>_linux_amd64.tar.gz
echo-mcp_<version>_windows_amd64.zip
echo-mcp_<version>_checksums.txt
```

Recommended supported operating systems and CPU architectures for the first
binary release:

- macOS arm64
- macOS amd64
- Linux arm64
- Linux amd64
- Windows amd64

The source repository should remain buildable with the Go toolchain even before
binary artifacts are published.

## Checksums

Generate SHA-256 checksums for release artifacts:

```bash
shasum -a 256 echo-mcp_<version>_* > echo-mcp_<version>_checksums.txt
```

If GoReleaser is adopted, let GoReleaser generate the checksum file and include
it in the GitHub release.

## GoReleaser Recommendation

GoReleaser is recommended for repeatable multi-platform release builds, archives,
and checksums. It is not required for local development and should not be added
until release automation is explicitly approved.

## Release Checks

Before publishing a release, verify:

- documentation describes only implemented capabilities
- ADRs match the release surface
- excluded features remain documented as limitations or non-goals
- `go test ./...` passes
- `go test -race ./...` passes
- `go vet ./...` passes
- `go build -o bin/echo-mcp ./cmd/echo-mcp` passes
- `make test` passes
- `make build` passes
- `git diff --check` passes
- no generated release artifact is published without separate approval

Docker image builds can be checked with:

```bash
make docker-build
```

If Docker is unavailable locally, record that as an environment limitation
rather than treating it as an implementation failure.
