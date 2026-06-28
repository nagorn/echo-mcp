# Changelog

All notable changes to Echo MCP are documented in this file.

## [0.3.0] - 2026-06-28

### Added

- Runtime OpenAPI contract loading through MCP control-plane tools.
- `load_openapi_contract`, `get_contract_status`, and `unload_openapi_contract`.
- Partial contract-backed response validation for supported OpenAPI 3.0 JSON features.
- Local internal `$ref` resolution for response schemas.
- Contract root boundary via `ECHO_MCP_CONTRACT_ROOT`.
- Safe contract source path display relative to the contract root.
- Validation capability disclosure through `validation_scope`, `validation_capabilities`, and `validation_mode_description`.
- Reproducible OpenAPI compatibility smoke script.

### Changed

- `configure_behavior` validates configured responses against the active contract when one is loaded.
- `reset` keeps the active OpenAPI contract loaded while clearing behavior and observations.
- Startup OpenAPI loading uses the same contract manager and contract-root boundary as runtime loading.

### Compatibility

- Existing manual mock behavior remains supported when no contract is active.
- REST data-plane responses are not modified by contract guidance or validation warnings.
- Validation is partial and limited to supported OpenAPI 3.0 JSON response features.
- Strict mode means strict enforcement of supported validation capabilities only.

### Not Included

- No `echo-mcp.yaml`.
- No full OpenAPI-first runtime.
- No OpenAPI 3.1 or YAML support.
- No remote/file refs.
- No `allOf`, `oneOf`, or `anyOf` semantics.
- No request body/query/header/path parameter validation.
- No automatic scenario generation.
- No provider-specific Stripe simulator.

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
