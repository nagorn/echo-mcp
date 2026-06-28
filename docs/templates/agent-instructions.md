# Echo MCP Agent Instructions Template

This template is intended to be copied into project-local AI instruction files
or equivalent project-specific AI instruction mechanisms when the project uses
Echo MCP for external dependency simulation.

Suitable locations include:

- `AGENTS.md`
- `CLAUDE.md`
- Cursor Rules
- GitHub Copilot Instructions
- other project-specific AI instruction mechanisms

Echo MCP does not require any specific AI assistant. These instructions describe
how any AI coding agent should use Echo MCP when the project has configured it.

```markdown
# Echo MCP Usage

This project uses Echo MCP for external dependency simulation during
end-to-end tests.

## Architecture Boundary

Echo MCP has two planes:

- MCP control plane for AI agents and other MCP clients.
- REST data plane for the Application Under Test.

Use Echo MCP MCP tools to configure simulated external dependency behavior. The
application must continue using normal REST HTTP and must not know Echo MCP is
involved.

MVP usage: one Echo MCP process represents one simulated external dependency.
When a test needs several external dependencies, use one Echo MCP process and
one MCP server registration per dependency, each with its own
`ECHO_MCP_HTTP_ADDR` and optional `ECHO_MCP_CONTRACT_ROOT`.

Webhook-style events must be sent only to developer-configured application
webhook endpoints selected by endpoint name.

## Required Agent Workflow

1. Inspect Echo MCP initialize instructions, tool descriptions, prompts, and
   resources.
2. Read the test objective.
3. Choose `manual_mock`, `hybrid_validation`, or `contract_first` based on the
   project need and available contracts.
4. Identify the external dependency behavior that must be simulated.
5. If provider fidelity matters, load a local OpenAPI 3.0 JSON contract with
   `load_openapi_contract` and confirm it with `get_contract_status`.
6. Configure Echo MCP through MCP tools.
7. Run the E2E test through the application, not by bypassing the application.
8. Inspect Echo MCP observations before concluding that the dependency
   interaction behaved as expected.
9. Verify the application's behavior using the project's normal assertions.
10. Reset Echo MCP between tests or scenarios.

## Workflow Selection

Before configuring Echo MCP, identify the workflow intent:

- `manual_mock`: use for exploration, prototyping, failure simulation, or when
  the provider contract is unavailable. Manual mocks are not
  provider-contract validated.
- `hybrid_validation`: use when a contract exists but manual scenarios remain
  useful or validation coverage is partial.
- `contract_first`: use when provider fidelity matters and an official or
  internal OpenAPI 3.0 JSON contract is available. In Echo MCP v0.3.0 this means
  partial response validation for supported capabilities, not full OpenAPI
  validation.

If the developer only says "use Echo MCP to mock provider X", do not assume
contract-backed validation. The fastest working path may be handwritten
provider-like types and manual scenario fixtures. That can be acceptable for
`manual_mock`, but it is not the same as contract-backed validation.

When provider contract fidelity matters:

- locate the OpenAPI contract or ask the developer for it before writing
  provider schemas
- use `load_openapi_contract`
- use `get_contract_status`
- inspect `validation_scope`, `validation_capabilities`, and
  `validation_mode_description`
- do not handwrite provider response schemas when the contract can validate the
  configured response
- if validation is unavailable or unsupported for this scenario, document the
  downgrade to `hybrid_validation` or `manual_mock`
- ask the developer before silently downgrading contract fidelity

## Rules for Application Code

Keep application code production-like.

Do not:

- add Echo MCP-specific branches
- add MCP awareness to the application
- add simulator-specific request headers, request fields, or metadata
- make the application call Echo MCP control-plane tools
- make application behavior depend on Echo MCP observations
- call Echo MCP's REST data plane directly as the test subject
- add Echo MCP-specific webhook branches

The application may be configured to use Echo MCP as the external dependency
base URL in the test environment. That configuration must preserve the same code
paths the application uses for the real dependency.

Application webhook handlers should remain normal production-like webhook
handlers.

## Rules for Echo MCP Behavior

Use supported Echo MCP MCP tools only.

Current tools are:

- `load_openapi_contract`
- `get_contract_status`
- `unload_openapi_contract`
- `configure_behavior`
- `get_observations`
- `reset`
- `send_webhook_event`

Configure concrete behavior. Echo MCP is a deterministic execution engine, not
an AI runtime. Do not expect Echo MCP to infer, generate, expand, or repair
behavior from high-level test intent.

Do not assume one Echo MCP process can hold multiple dependency contracts or
independent behavior sets. The current implementation supports one OpenAPI
contract, one registered webhook endpoint, and one active REST behavior rule per
process.

If the application receives HTTP `501 Not Implemented` from Echo MCP's REST data
plane, treat it as missing simulator setup. Inspect available observations and
test/application logs, configure the missing behavior through MCP, and rerun the
test. Do not treat Echo MCP's unmatched-request `501` as a simulated provider
response.

Manual mock behavior is useful for quick exploration, but it is not
provider-contract validated unless OpenAPI-backed validation is active. If
fidelity matters, prefer `hybrid_validation` or `contract_first` when available,
or ask the developer whether a contract exists.

Respect known API contracts. If a developer-provided contract constraint is
available, configure behavior that conforms to the supported contract subset and
treat contract rejection as a failed test setup. Do not expect Echo MCP to
import, generate, fetch, or own external API contracts.

Strict mode means strict enforcement of the validation capabilities currently
supported by Echo MCP. It is not full OpenAPI validation.

For intentional negative tests while a contract is active, use:

```json
{
  "validation": {
    "mode": "off",
    "reason": "intentional malformed response test"
  }
}
```

For webhook-style events:

- use configured `endpoint_name` values only
- do not provide arbitrary outbound URLs
- do not configure webhook endpoints pointing to arbitrary third-party systems
- expect one immediate delivery attempt
- do not assume retries, scheduling, signatures, delivery persistence, AsyncAPI,
  event buses, or inbound public webhook receiving

## Contract Loading Boundary

If `ECHO_MCP_CONTRACT_ROOT` is set, OpenAPI contract paths must resolve under
that root. If it is unset, the process working directory is the root. Relative
paths resolve against the root. Absolute paths are allowed only when they resolve
inside the root. Traversal and symlink escapes are rejected where detectable.

Startup `ECHO_MCP_OPENAPI_FILE` and runtime `load_openapi_contract` use the same
boundary.

## Verification Expectations

Before concluding that an E2E test passed:

- confirm the application test passed through the normal application test path
- inspect Echo MCP observations
- confirm the expected request was received
- confirm the expected behavior matched
- confirm the expected outcome was produced
- confirm expected webhook delivery observations when webhook events are part of
  the test
- treat unexpected HTTP `501` from Echo MCP as missing simulator setup, not a
  provider-like response
- reset Echo MCP before the next scenario

## Unsupported Capabilities

Do not invent unsupported Echo MCP capabilities.

Unless this project explicitly documents otherwise, assume Echo MCP does not
provide:

- UI or dashboard
- CLI workflow
- authentication or authorization
- user management
- persistence
- full OpenAPI-first runtime
- `echo-mcp.yaml` project manifest
- OpenAPI 3.1 or YAML support
- remote/file refs
- `allOf`, `oneOf`, or `anyOf` semantics
- request body, query, header, or path parameter validation
- automatic scenario generation
- provider-specific simulators
- built-in public API contracts
- multi-dependency support inside one Echo MCP process
- recording or replay
- webhook retries, scheduling, signatures, or delivery persistence
- AsyncAPI or event bus behavior
- inbound public webhook receiving
- non-REST protocol simulation
- metrics or audit logs
- production deployment behavior
```
