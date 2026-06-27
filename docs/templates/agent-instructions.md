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
`ECHO_MCP_HTTP_ADDR`.

Webhook-style events must be sent only to developer-configured application
webhook endpoints selected by endpoint name.

## Required Agent Workflow

1. Inspect Echo MCP initialize instructions, tool descriptions, prompts, and
   resources.
2. Read the test objective.
3. Choose `manual_mock`, `hybrid_validation`, or `contract_first` based on the
   project need and available contracts.
4. Identify the external dependency behavior that must be simulated.
5. Configure Echo MCP through MCP tools.
6. Run the E2E test through the application, not by bypassing the application.
7. Inspect Echo MCP observations before concluding that the dependency
   interaction behaved as expected.
8. Verify the application's behavior using the project's normal assertions.
9. Reset Echo MCP between tests or scenarios.

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

- `configure_behavior`
- `get_observations`
- `reset`
- `send_webhook_event`

Current guidance prompts are:

- `echo_mcp_getting_started`
- `echo_mcp_choose_workflow`
- `echo_mcp_manual_mock_workflow`
- `echo_mcp_contract_validation_workflow`

Current guidance resources are:

- `echo://guides/getting-started`
- `echo://guides/workflows`
- `echo://guides/manual-mock`
- `echo://guides/contract-validation`

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
available, configure behavior that conforms to the contract and treat contract
rejection as a failed test setup. Do not expect Echo MCP to import, generate, or
own external API contracts.

For webhook-style events:

- use configured `endpoint_name` values only
- do not provide arbitrary outbound URLs
- do not configure webhook endpoints pointing to arbitrary third-party systems
- expect one immediate delivery attempt
- do not assume retries, scheduling, signatures, delivery persistence, AsyncAPI,
  event buses, or inbound public webhook receiving

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
- OpenAPI import or generation
- full OpenAPI-first runtime
- `echo-mcp.yaml` project manifest
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
