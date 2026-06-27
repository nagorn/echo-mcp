# Changelog

All notable changes to Echo MCP are documented in this file.

## [0.2.0] - 2026-06-27

### Added

- MCP initialize instructions for agent guidance.
- Workflow-aware tool descriptions.
- Tool annotations for safer client interpretation.
- Guidance prompts for getting started, workflow choice, manual mock workflow,
  and contract validation workflow.
- Guidance resources for getting started, workflows, manual mocks, and contract
  validation.
- Structured `configure_behavior` warnings, guidance, and suggested next
  actions.

### Changed

- `configure_behavior` results now include additive guidance fields while
  preserving existing `configured` and `behavior_id` fields.
- Manual mock behavior remains supported and unchanged on the REST data plane.

### Compatibility

- REST data-plane response bodies are not mutated by agent guidance.
- Existing manual mock behavior remains backward compatible.
- Strict MCP clients should tolerate additional structured result fields.

### Not Included

- No `echo-mcp.yaml` project manifest.
- No full OpenAPI-first runtime.
- No provider-specific Stripe simulator.
- No public contract repository.

## [0.1.0] - 2026-06-26

### Added

- First public source-available release.
- MCP stdio control plane.
- REST HTTP data plane.
- OpenAPI 3.0.x JSON contract validation as a constraint for configured
  behavior.
- Webhook-style event delivery to developer-configured endpoints.
- In-memory deterministic behavior.
- AI-first installation and usage documentation.
