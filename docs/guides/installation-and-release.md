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
https://github.com/nagorn/echo-mcp/releases/tag/v0.3.0
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
  https://github.com/nagorn/echo-mcp/releases/download/v0.3.0/echo-mcp_darwin_arm64.tar.gz
curl -L -o /tmp/echo-mcp-checksums.txt \
  https://github.com/nagorn/echo-mcp/releases/download/v0.3.0/checksums.txt
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
  ECHO_MCP_CONTRACT_ROOT=/absolute/path/to/project/contracts
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

### OpenAPI Contract Root

`ECHO_MCP_CONTRACT_ROOT` bounds local contract loading. If unset, Echo MCP uses
the process working directory as the contract root.

Relative paths resolve against the contract root. Absolute paths are accepted
only if they resolve inside the contract root. Paths outside the root, traversal
escapes, and symlink escapes where detected are rejected.

Runtime loading and startup loading share this boundary.

### OpenAPI Contract Constraint

Preferred runtime workflow:

1. Start Echo MCP.
2. Call `load_openapi_contract` with a local OpenAPI 3.0 JSON contract under the
   contract root.
3. Call `get_contract_status`.
4. Configure behavior through `configure_behavior`.
5. Run app tests against the REST data plane.
6. Use `reset` between scenarios; the contract remains active.
7. Use `unload_openapi_contract` when switching contract contexts.

Startup loading remains supported:

```bash
ECHO_MCP_CONTRACT_ROOT=./contracts \
ECHO_MCP_OPENAPI_FILE=payment.openapi.json \
./bin/echo-mcp
```

The contract is a validation constraint only. Echo MCP does not fetch, import,
generate, repair, or complete OpenAPI specifications.

Validation is partial response validation for supported OpenAPI 3.0 JSON
features. Strict mode means strict enforcement of supported validation
capabilities only; it is not full OpenAPI validation.

Not included in v0.3.0: OpenAPI 3.1, YAML, remote/file refs,
`allOf`/`oneOf`/`anyOf`, request body validation, query/header/path parameter
validation, automatic scenario generation, and provider-specific simulators.

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
- optional `ECHO_MCP_CONTRACT_ROOT`
- optional startup `ECHO_MCP_OPENAPI_FILE`
- optional webhook endpoint configuration

## Smoke Test

After registering Echo MCP with an MCP host:

1. Ask the MCP client to call `configure_behavior` for `GET /hello`.
2. Send a REST request to the configured data-plane address.
3. Ask the MCP client to call `get_observations`.
4. Confirm the observation includes the received request, matched behavior, and
   HTTP outcome.
5. Call `reset`.

For contract-backed smoke tests, also call `load_openapi_contract`, confirm
`get_contract_status`, configure one valid strict response, configure one
invalid strict response and confirm MCP rejection, then verify
`validation.mode = "off"` with a reason accepts intentional fault behavior.

## Security Note

Echo MCP is intended for local development, local AI-assisted testing, and
controlled CI/test environments only.

Do not expose Echo MCP directly to the public internet. The MVP intentionally
does not include authentication or authorization. The MCP control plane can
configure simulator behavior, load local contracts under the contract root, and
webhook-style event delivery can send outbound HTTP requests to configured
application webhook endpoints. If exposed incorrectly, Echo MCP could be abused.

Bind or expose Echo MCP to localhost by default where possible. If running
outside a local machine, use firewalling, container/network isolation, or
private CI networking. Do not configure webhook endpoints pointing to arbitrary
third-party systems. Only configure webhook endpoints owned by the
application/test environment.

## Release Strategy

Current public release target:

```text
v0.3.0
```

Use semantic versioning after the first public release. Before `v1.0.0`, minor
versions may include breaking changes when the release notes call them out.

## Release Artifact Layout

Do not publish or upload generated binaries until the maintainer explicitly
approves the release.

Current GitHub release artifact names:

```text
echo-mcp_darwin_arm64.tar.gz
echo-mcp_darwin_amd64.tar.gz
echo-mcp_linux_arm64.tar.gz
echo-mcp_linux_amd64.tar.gz
echo-mcp_windows_amd64.zip
checksums.txt
```

Supported operating systems and CPU architectures for the current binary
release matrix:

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
shasum -a 256 echo-mcp_* > checksums.txt
```

Final release assets should be rebuilt after the release commit and tag, before
uploading them to GitHub Releases.
