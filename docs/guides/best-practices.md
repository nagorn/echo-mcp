# Echo MCP Best Practices

This guide helps developers choose the right Echo MCP workflow before asking an
AI coding agent to build or test an external API integration.

Echo MCP can guide agents through MCP initialize instructions, tool
descriptions, prompts, resources, and structured tool results. Agent behavior
may still vary by client. Developers should state the workflow intent clearly in
the initial prompt.

## Choose A Workflow First

### `manual_mock`

Use `manual_mock` when:

- exploring an integration
- prototyping application behavior
- simulating failures
- the API contract is unavailable
- contract fidelity is not the primary goal

Manual mocks are hand-authored behavior. They are useful, fast, and supported,
but they are not provider-contract validated.

### `hybrid_validation`

Use `hybrid_validation` when:

- a contract exists
- manual scenarios are still useful
- partial response validation is enough for the current risk
- migration from manual mocks is desired

Hybrid workflows let teams keep practical scenario coverage while using
contract-backed checks where Echo MCP supports them.

### `contract_first`

Use `contract_first` when:

- external provider fidelity matters
- CI should catch response schema drift for supported OpenAPI features
- an official or internal OpenAPI 3.0 JSON contract is available

The OpenAPI contract should be the source of truth for paths, response statuses,
content types, and response schemas. Do not ask the agent to handwrite provider
response schemas when the contract is available.

In v0.3.0, contract-first means partial response validation for supported
OpenAPI 3.0 JSON features. It does not mean full OpenAPI validation, request
validation, automatic scenario generation, or a provider-specific simulator.

## Prompt Warning

If you only say "use Echo MCP to mock provider X", an AI agent may choose the
fastest working path: handwritten provider-like types and manual scenario
fixtures. That can be acceptable for `manual_mock` workflows, but it is not the
same as contract-backed validation.

If provider contract fidelity matters, say so explicitly in the initial prompt.
Also tell the agent what contract file or source to use, or tell it to ask for
the contract before writing provider schemas.

## Prompt Templates

### Manual Mock Prompt

```text
Use Echo MCP in manual_mock mode for this integration test.

Goal:
- Simulate the minimum provider behavior needed for this scenario.
- Keep the application code production-like.
- Configure Echo MCP through MCP tools and run the application test normally.

Constraints:
- Manual mock behavior is acceptable for this task.
- Do not claim this proves provider contract fidelity.
- Document any provider-like request or response shape that is hand-authored.
- Inspect Echo MCP observations and reset between scenarios.
```

### Hybrid Validation Prompt

```text
Use Echo MCP in hybrid_validation mode for this integration test.

Before configuring behavior:
- Look for an official or internal OpenAPI 3.0 JSON contract in the project.
- Start Echo MCP and call load_openapi_contract when a local contract is available.
- Call get_contract_status and inspect validation_scope and validation_capabilities.
- Manual scenarios are allowed, but document which parts are manual and which
  parts are contract-backed.

If validation is unavailable or unsupported for this scenario:
- Document the downgrade to manual behavior.
- Keep the scenario narrow and do not duplicate large provider schemas unless
  necessary for the test.
- Ask before reducing contract fidelity further.
```

### Contract-First Prompt

```text
Use Echo MCP in contract_first mode for this external API integration.

Provider contract fidelity matters for this task.

Before writing provider request or response schemas:
- Locate the official or internal OpenAPI 3.0 JSON contract, or ask me for it if
  it is not present.
- Start Echo MCP with ECHO_MCP_CONTRACT_ROOT set to the contract directory when possible.
- Call load_openapi_contract.
- Call get_contract_status and inspect validation_scope, validation_capabilities,
  and validation_mode_description.
- Do not handwrite provider response schemas when the contract can validate the
  configured response.
- Treat strict mode as strict for Echo MCP's supported validation capabilities,
  not as full OpenAPI validation.

Echo MCP setup:
- Configure behavior through MCP tools.
- If validation rejects because a feature is unsupported, report the unsupported
  feature instead of claiming the response body is invalid.
- If intentionally testing malformed provider behavior, use validation.mode=off
  with a non-empty reason.
- Ask me before silently downgrading contract fidelity.

Test behavior:
- Keep application code production-like.
- Run the application test normally.
- Inspect Echo MCP observations and reset between scenarios.
```

## Caveats

- Echo MCP does not automatically fetch or import provider contracts.
- Echo MCP does not make manual mocks provider-contract validated.
- Echo MCP does not require or read `echo-mcp.yaml`.
- Echo MCP is not a full OpenAPI-first runtime.
- Validation is partial response validation for supported OpenAPI 3.0 JSON
  features.
- Request body, query, header, and path parameter validation are not
  implemented.
- OpenAPI 3.1, YAML, remote/file refs, `allOf`, `oneOf`, and `anyOf` are not
  implemented.
- Manual mocks remain useful when speed, exploration, or failure simulation is
  the priority.
