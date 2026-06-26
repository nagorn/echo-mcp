# ADR-0003: MCP SDK and Transport

Status: APPROVED
Date: 2026-06-25
Owner: Nagorn Smanote

## Context

ADR-0001 establishes MCP as Echo MCP's control channel and normal service
interfaces as the data channel. ADR-0002 establishes Go as the MVP implementation
platform and leaves exact MCP tool names, schemas, protocol transport details,
and third-party MCP library choice undecided.

The initial MVP tool surface defined the following MCP tools:

- `configure_behavior`
- `reset`
- `get_observations`

The next implementation step needs a concrete MCP SDK/library and initial MCP
transport so Echo MCP can expose the approved control-plane tools without
introducing non-MVP surfaces.

The official Model Context Protocol documentation lists Go as a Tier 1 SDK and
documents support for creating MCP servers with local and remote transports. The
official Go SDK documentation identifies `github.com/modelcontextprotocol/go-sdk`
as the Go SDK and documents stdio and streamable HTTP transports.

## Decision

For the MVP external MCP interface, Echo MCP will use:

- MCP SDK/library: the official Go SDK,
  `github.com/modelcontextprotocol/go-sdk`.
- Protocol: Model Context Protocol (MCP).
- MVP transport: stdio.
- MCP server role: Echo MCP exposes MCP tools to AI agents.
- MCP tool surface: the minimal MVP tool surface.

The initial implementation should use only the SDK packages required to expose
the MVP tools over stdio. It must not enable OAuth/auth middleware, protected
resource metadata, streamable HTTP, custom transports, resources, prompts,
sampling, elicitation, UI, CLI, admin APIs, persistence, metrics, audit logs,
recording, replay, webhook delivery, OpenAPI/AsyncAPI awareness, or production
deployment architecture.

The REST-style HTTP data plane remains separate from MCP. The application under
test continues to use normal HTTP request/response interfaces and must not need
MCP awareness or Echo MCP-specific request metadata.

The selected stdio transport is an MVP implementation decision only and does not
modify the architectural separation defined by ADR-0001.

Later ADRs and tasks extended the tool surface with contract-constrained REST
validation and webhook-style event delivery. Those additions do not change this
SDK and transport decision.

## Reasoning

Using the official Go SDK aligns the MVP with the protocol's maintained Go
implementation while keeping Echo MCP in the Go platform selected by ADR-0002.
It avoids hand-rolling JSON-RPC, tool registration, lifecycle handling, and
transport plumbing before the MVP validates the control-plane/data-plane
boundary.

Stdio is the narrowest initial MCP transport for a local AI-controlled simulator.
It allows an AI agent or MCP host to launch Echo MCP as a local MCP server
without adding a remote control-plane HTTP endpoint. That keeps authentication,
authorization, remote exposure, session management, and production deployment
concerns out of the MVP.

Streamable HTTP remains useful for a later remote or shared-hosting scenario,
but selecting it now would add another externally reachable HTTP surface beyond
the existing REST-style data plane and would make auth and deployment questions
harder to keep out of scope.

## Consequences

- Implementation can replace the local provisional adapter with real MCP tool
  registration while preserving the same internal control-plane boundary.
- Echo MCP will have two logical channels in one local binary:
  - MCP over stdio for AI-agent control.
  - REST-style HTTP for application data-plane interactions.
- The data-plane HTTP server must not write simulator control data or MCP
  messages.
- Runtime state remains in memory only.
- The MVP does not expose a remote MCP endpoint.
- The MVP does not add an Echo MCP CLI; stdio is a transport for an MCP host, not
  a user-facing command surface.
- If stdio proves insufficient for a concrete MVP client integration, the
  limitation should be recorded and the transport decision revisited before
  implementing streamable HTTP or a custom transport.

## Non-Decisions

This ADR does not decide:

- exact Go SDK version pin or update policy
- exact generated MCP JSON schemas or SDK-specific registration code
- exact MCP protocol version negotiation behavior
- final MCP error envelope or machine-readable error names
- streamable HTTP MCP support
- custom MCP transports
- authentication, authorization, user management, or multi-tenancy
- production deployment, remote hosting, or public simulator access
- persistence, retention, metrics, audit logging, import/export, recording, or
  replay
- OpenAPI/AsyncAPI awareness
- webhook-style event delivery
- resources, prompts, sampling, elicitation, or MCP Apps/UI features

If any non-decision becomes necessary for the MVP, it must be raised as a
separate unresolved decision rather than silently added during implementation.

## Related Documents

- [ADR-0001: Separate Control Channel from Data Channel](ADR-0001-separate-control-channel-from-data-channel.md)
- [ADR-0002: MVP Implementation Platform](ADR-0002-mvp-implementation-platform.md)
- [MCP Tool Reference](../reference/mcp-tools.md)

## Reference Inputs

- [Model Context Protocol SDKs](https://modelcontextprotocol.io/docs/sdk)
- [MCP Go SDK](https://go.sdk.modelcontextprotocol.io/)
- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)
