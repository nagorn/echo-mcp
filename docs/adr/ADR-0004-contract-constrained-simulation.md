# ADR-0004: Contract-Constrained Simulation

Status: APPROVED
Date: 2026-06-25
Owner: Nagorn Smanote

## Context

Echo MCP allows an AI agent to configure simulated external dependency behavior
through the MCP control plane. The application under test then exercises that
behavior through the normal data plane.

This preserves ADR-0001's control-plane/data-plane separation, but it creates a
risk: an AI agent could configure behavior that makes an application test pass
while violating the real external dependency's API contract.

For REST-style dependencies, developers commonly have access to OpenAPI
specifications. Echo MCP can use an OpenAPI specification as a constraint on
AI-configured behavior without turning Echo MCP into an OpenAPI-driven mock
generator or full service-virtualization platform.

Echo MCP is not an AI runtime. Echo MCP is a deterministic execution engine. The
AI agent is responsible for deciding the test scenario and constructing the
complete simulated behavior. Echo MCP is responsible for validating and
executing that configured behavior.

## Decision

Echo MCP should support contract-constrained REST simulation for developer-
provided OpenAPI specifications.

The OpenAPI specification is a constraint, not a source of automatic scenario
generation:

- Developers provide the OpenAPI specification for a simulated REST dependency.
- AI agents choose the test scenario, construct concrete simulated behavior, and
  configure that behavior through MCP.
- Echo MCP validates configured behavior against the provided contract.
- If behavior violates the contract, Echo MCP rejects it immediately.
- Echo MCP must not silently accept invalid AI-configured behavior.
- Echo MCP does not infer, generate, expand, or repair simulated responses from
  OpenAPI documents.

Design principle: Echo MCP validates behavior. It does not generate behavior.

The first contract-constrained implementation should remain compatible with the
current one-rule in-memory behavior model. Contract validation constrains the
one configured rule; it does not require multiple rules, scenario scripting,
persistence, recording/replay, or service virtualization.

This ADR does not modify ADR-0001. MCP remains the control channel, the
application under test still uses normal REST-style HTTP, and the application
must not know Echo MCP is involved.

## Scope

### Responsibility Split

The AI agent is responsible for test intent and behavior construction:

- understand the testing intent
- read external API documentation when available
- construct a concrete simulated request match and response outcome
- send the complete behavior configuration to Echo MCP through MCP

Echo MCP is responsible for deterministic validation and execution:

- validate the configured behavior against the provided contract
- reject behavior that violates the contract
- store accepted behavior in runtime state
- execute accepted behavior deterministically through the data plane

Echo MCP must not infer business semantics or use the OpenAPI specification to
construct responses on behalf of the AI agent.

### OpenAPI File Ownership and Location

OpenAPI files should live outside Echo MCP's runtime state, under developer
control. Typical locations include the application repository, a test fixture
directory, or a checked-in dependency-contract directory.

Echo MCP should treat those files as read-only contract inputs. Echo MCP should
not become the source of truth for external dependency contracts.

### Loading Model

For the first implementation, contract files should be loaded before behavior is
configured, preferably during local simulator startup or initialization.

The MCP control plane should not be the first mechanism for uploading or
mutating OpenAPI files. Loading contracts through MCP would allow the same AI
agent that configures behavior to alter the constraint, which weakens the
purpose of contract-constrained simulation.

The exact startup configuration mechanism remains undecided.

### MVP Validation Requirement

The smallest useful validation should check that an AI-configured REST behavior
is compatible with the provided OpenAPI operation and response shape.

For MVP contract-constrained simulation, validation should include:

- request method and path correspond to an operation in the OpenAPI document
- configured response status is allowed by that operation, either explicitly or
  through an applicable default response
- configured response body is compatible with the selected response schema when
  a response schema is available
- configured response content type is compatible with the selected response when
  content types are represented by the behavior model

If the current behavior model cannot represent a contract feature, Echo MCP
should not pretend to validate that feature. It should either ignore that feature
with explicit documentation or reject behavior that depends on it.

Contract validation is limited to what the provided contract can express. Echo
MCP validates structural contract compatibility; it does not guarantee business
semantics beyond the contract.

Examples of contract-expressible constraints include:

- response schemas
- required fields
- response status codes
- content types

Examples of business rules Echo MCP cannot validate unless they are explicitly
represented in the provided contract include:

- whether an amount is correct
- whether a merchant owns a transaction
- whether a decline reason is appropriate for a payment state
- whether a customer is eligible for a specific operation

### Rejection Behavior

Invalid AI-configured behavior should be rejected immediately during behavior
configuration. Rejection should happen before replacing the currently configured
behavior.

The error should be visible through MCP as a configuration failure. Exact error
schema, error codes, and human-readable diagnostics remain undecided.

## Non-Goals

This ADR does not add:

- OpenAPI-driven scenario generation
- OpenAPI-driven mock generation
- service virtualization
- traffic recording or replay
- persistence
- UI, dashboard, CLI, or admin API
- authentication, authorization, user management, or multi-tenancy
- metrics or audit logs
- webhook delivery
- AsyncAPI support
- production deployment architecture

## Consequences

- Echo MCP can reduce false confidence from AI-configured behavior that does not
  match a real dependency contract.
- Developers remain responsible for selecting and maintaining the external
  dependency contract.
- AI agents retain control over scenarios, but cannot configure behavior outside
  the contract when a contract is active.
- The current one-rule in-memory model can remain the first implementation
  target.
- Echo MCP gains a new contract-validation responsibility for REST dependencies.
- Implementation must carefully distinguish validation constraints from mock
  generation.
- Tests remain only as useful as the supplied contract. Contract validation
  cannot prove business correctness for semantics the contract does not express.

## Answers to Key Questions

### Where should OpenAPI spec files live?

OpenAPI files should live in developer-controlled project files, such as test
fixtures or dependency-contract directories. Echo MCP should read them as local
contract inputs, not own or persist them.

### Is the spec loaded at startup or configured through MCP?

The first implementation should load specs before behavior configuration,
preferably at startup or simulator initialization. Uploading or mutating specs
through MCP is not part of this proposal.

### What validation is required for MVP?

MVP validation should cover method/path operation existence, response status
compatibility, and response body schema compatibility when the relevant schema is
available. Content-type validation should be added when the behavior model can
represent response content type.

MVP validation should not attempt to infer or validate business semantics beyond
the provided contract.

### Should invalid AI-configured behavior be rejected immediately?

Yes. Invalid behavior should be rejected during configuration and should not
replace the current behavior.

### Does this change the current one-rule in-memory behavior model?

No. The first contract-constrained implementation can validate the existing
single in-memory rule. Multiple rules, priorities, and scenarios remain separate
future decisions.

## Intentionally Undecided

- exact OpenAPI parser or validation library
- exact startup configuration mechanism for contract file paths
- support for multiple simulated services with different contracts
- matching rules for path templates and path parameters
- request body, query parameter, and header validation depth
- response header and content-type behavior model
- schema-dialect handling and reference resolution
- exact MCP error shape for contract violations
- whether contract-constrained simulation belongs in the current MVP milestone
  or a follow-up milestone after the first request/response slice
- whether later work should support OpenAPI-driven scenario suggestions

Implementation note: the first implementation slice selected OpenAPI 3.0.x JSON
only for the first contract-constrained slice. OpenAPI 3.1.x JSON, YAML, and
external `$ref` resolution remain future work.

## Related Documents

- [ADR-0001: Separate Control Channel from Data Channel](ADR-0001-separate-control-channel-from-data-channel.md)
- [ADR-0002: MVP Implementation Platform](ADR-0002-mvp-implementation-platform.md)
- [ADR-0003: MCP SDK and Transport](ADR-0003-mcp-sdk-and-transport.md)
- [Scope](../concepts/scope.md)
- [Core Abstractions](../concepts/core-abstractions.md)
