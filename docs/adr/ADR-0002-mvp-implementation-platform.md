# ADR-0002: MVP Implementation Platform

Status: APPROVED
Date: 2026-06-25
Owner: Nagorn Smanote

## Context

ADR-0001 separates Echo MCP's control channel from its data channel.

The MVP now needs an implementation platform so coding can begin without
accidentally introducing additional product surfaces, persistence concerns, or
production architecture. The MVP remains limited to:

- an MCP control plane for AI agents
- a REST-style HTTP request/response data plane for the application under test
- in-memory runtime behavior suitable for repeatable local end-to-end testing

The planning documents intentionally exclude UI, CLI, authentication,
authorization, user management, persistence, import/export, recording, replay,
full service virtualization, OpenAPI/AsyncAPI awareness, webhook-style event
delivery, non-REST protocols, multi-tenancy, audit logs, metrics, and production
deployment concerns from the MVP.

Later ADRs promoted narrow contract-constrained REST validation and
webhook-style event delivery slices without changing this implementation
platform decision.

## Decision

Echo MCP's MVP implementation platform will be:

- Programming language: Go.
- HTTP server approach: Go standard library `net/http`.
- Router: Go standard library `http.ServeMux`, unless a concrete MVP limitation
  is found during implementation.
- Logging: Go standard library `log/slog`.
- MCP integration approach: an in-process Go MCP control-plane integration in
  the same local binary as the HTTP data plane.
- Runtime state model: in-memory only.
- Testing approach: `go test`.
- Build/run approach: local Go binary plus Docker image for local containerized
  execution.

The initial implementation must not add:

- Clean Architecture, DDD, CQRS, database repositories, plugin systems, or
  framework-driven layering
- admin APIs, UI layers, dashboards, CLIs, or management surfaces
- authentication, authorization, user management, or multi-tenancy
- persistent storage, import/export, recording, replay, metrics, audit logs, or
  production deployment architecture

## Reasoning

Go fits the MVP because it supports a small single-binary service with
concurrency, HTTP serving, structured logging, tests, and container builds using
the standard toolchain.

Using `net/http`, `http.ServeMux`, and `log/slog` keeps the initial platform
small and reduces dependencies while the project validates the core control-plane
and data-plane boundary. A larger web framework, router, layered architecture, or
plugin system is not justified before the MVP exposes a concrete limitation.

Keeping runtime state in memory matches the MVP's repeatable local-test focus and
avoids prematurely deciding persistence, retention, migration, tenancy, or
operational concerns.

Running the MCP control plane and HTTP data plane in the same local binary keeps
the MVP runnable as a local simulator while preserving ADR-0001's logical
separation between control and data channels.

## Consequences

- The MVP can start with a small Go module, local binary, Dockerfile, and Go test
  suite.
- The application under test still interacts only with normal REST-style HTTP
  request/response interfaces.
- AI agents still interact only through MCP control-plane tools.
- Runtime configuration, behavior rules, observations, and resets are lost when
  the process exits.
- Tests should verify behavior through Go tests and local HTTP/MCP interactions,
  not through UI, CLI, database, or production deployment surfaces.
- If `http.ServeMux` cannot satisfy a concrete MVP routing need, the project
  should record the limitation and revisit the router choice.

## Non-Decisions

This ADR does not decide:

- exact MCP tool names, schemas, or protocol transport details
- exact REST endpoint paths, request bodies, response bodies, or matching
  language
- whether to use a specific third-party MCP library
- persistent storage, database, repository, retention, or migration strategy
- authentication, authorization, user management, multi-tenancy, audit, metrics,
  or production operations
- webhook-style event delivery
- OpenAPI, AsyncAPI, or other contract-aware generation
- service virtualization, traffic recording, traffic replay, or chaos
  engineering beyond targeted MVP request/response behavior

If any of these becomes necessary to complete the MVP, it must be raised as a
separate unresolved decision rather than silently added to the implementation
platform.

## Related Documents

- [ADR-0001: Separate Control Channel from Data Channel](ADR-0001-separate-control-channel-from-data-channel.md)
- [Project Assumptions and Vision](../project/project-assumptions-and-vision.md)
- [Project Scope](../project/scope.md)
- [Terminology](../project/terminology.md)
- [Core Abstractions](../project/core-abstractions.md)
- [First User Stories](../project/first-user-stories.md)
- [MVP Success Criteria](../project/mvp-success-criteria.md)
