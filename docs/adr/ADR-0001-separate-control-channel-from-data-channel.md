# ADR-0001: Separate Control Channel from Data Channel

Status: APPROVED
Date: 2026-06-25
Owner: Nagorn Smanote

## Context

Echo MCP is an AI-controlled external dependency simulator for end-to-end testing.
Applications under test need to interact with simulated external services through
normal service interfaces, while AI agents need a separate way to configure
runtime behavior during a test.

If these two concerns are mixed, the application under test may need simulator
specific behavior, test-only client logic, or knowledge that it is not talking to
a real dependency. That would weaken the value of end-to-end testing.

## Decision

Echo MCP separates the AI control plane from the application data plane.

- AI agents interact with Echo MCP through MCP as the control channel.
- Applications interact with Echo MCP through normal HTTP, WebSocket, or webhook
  interfaces as the data channel.
- The application under test should not know that Echo MCP is involved.

## Reasoning

This is the fundamental architectural boundary of the project and is unlikely to
change. The AI agent needs dynamic control over simulated dependency behavior,
but the application under test should experience Echo MCP as if it were a normal
external service dependency.

Keeping the control channel and data channel separate allows Echo MCP to support
AI-driven test orchestration without requiring application-specific test hooks in
the system under test.

## Consequences

- MCP is used for simulator control, not for normal application data exchange.
- Data-plane behavior must preserve the illusion that the application is talking
  to an external service.
- Runtime behavior changes should be expressible through the control channel.
- Application code should not need MCP awareness to participate in tests that use
  Echo MCP.
- Future design work must preserve this boundary unless a later ADR explicitly
  supersedes this decision.

## Non-Decisions

This ADR does not decide:

- implementation language or framework
- storage model
- deployment model
- exact MCP tool shape
- exact HTTP, WebSocket, or webhook API shape
- API contract format
- whether inbound webhooks are part of the first MVP
- whether OpenAPI, AsyncAPI, or another schema system will be used

## Related Documents

- [Project Assumptions and Vision](../project/project-assumptions-and-vision.md)
- [Project Scope](../project/scope.md)
- [Terminology](../project/terminology.md)
- [Core Abstractions](../project/core-abstractions.md)
- [First User Stories](../project/first-user-stories.md)
- [MVP Success Criteria](../project/mvp-success-criteria.md)
