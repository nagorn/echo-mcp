# MCP Control-Plane Reference

Echo MCP exposes a small MCP control-plane surface. Applications must not call
these tools, prompts, or resources. The Application Under Test uses only the
REST data plane and its normal webhook endpoints.

## Agent Guidance Surfaces

Echo MCP exposes MCP-standard guidance surfaces. They are advisory and help MCP
clients choose a workflow before silently defaulting to manual mock behavior.

- `initialize` instructions explain what Echo MCP is, the control-plane/data-plane
  boundary, manual mock caveats, partial contract validation, and recommended
  first steps.
- `tools/list` descriptions explain what each tool does, when to use it, when
  not to use it, common next steps, and whether the behavior is manual mock or
  contract-related.
- Tool annotations provide MCP-standard hints such as read-only, destructive,
  idempotent, and open-world behavior where supported by the client.
- `prompts/list` exposes guidance prompts for AI coding agents.
- `resources/list` exposes concise markdown guides.

Agent behavior may vary by MCP client. Some clients may not automatically read
prompts or resources unless instructed to do so.

## Prompts

Echo MCP exposes these prompts:

- `echo_mcp_getting_started`: inspect tools, prompts, and resources; choose a
  workflow; use `configure_behavior` for REST behavior; inspect observations;
  reset between scenarios.
- `echo_mcp_choose_workflow`: choose between `manual_mock`, `hybrid_validation`,
  and `contract_first`.
- `echo_mcp_manual_mock_workflow`: configure exact manual behavior, document the
  non-contract-validated caveat, run the application test, inspect observations,
  and reset.
- `echo_mcp_contract_validation_workflow`: use `load_openapi_contract`,
  `get_contract_status`, `configure_behavior`, app tests, and `reset` when an
  OpenAPI contract exists and provider fidelity matters.

## Resources

Echo MCP exposes these `text/markdown` resources:

- `echo://guides/getting-started`
- `echo://guides/workflows`
- `echo://guides/manual-mock`
- `echo://guides/contract-validation`

These resources are guidance only. They do not create a project manifest, change
runtime behavior, or make Echo MCP OpenAPI-first.

## Tools

Echo MCP exposes seven tools:

- `load_openapi_contract`
- `get_contract_status`
- `unload_openapi_contract`
- `configure_behavior`
- `send_webhook_event`
- `get_observations`
- `reset`

## Runtime Contract Workflow

Recommended contract-backed workflow:

1. Obtain or locate a local OpenAPI 3.0 JSON contract.
2. Start Echo MCP.
3. Call `load_openapi_contract`.
4. Call `get_contract_status` and inspect validation capability disclosure.
5. Call `configure_behavior`.
6. Let Echo MCP validate the configured response against supported capabilities.
7. Run app tests against the REST data plane.
8. Use `reset` between scenarios; the contract remains active.
9. Use `unload_openapi_contract` only when switching contract contexts.

Strict mode means strict enforcement of the validation capabilities currently
supported by Echo MCP. It is not full OpenAPI validation.

## Validation Capability Disclosure

When a contract is active, load/status/configure outputs include:

```json
{
  "validation_scope": "partial",
  "validation_capabilities": {
    "method_path": true,
    "response_status": true,
    "response_content_type": true,
    "response_body": true,
    "inline_json_response_schema": true,
    "request_body": false,
    "request_query": false,
    "request_headers": false,
    "path_parameter_schema": false,
    "ref_resolution": true,
    "local_ref_resolution": true,
    "remote_ref_resolution": false,
    "arrays": true,
    "enum": true,
    "allOf": false,
    "oneOf": false,
    "anyOf": false,
    "nullable": true,
    "additional_properties": true,
    "additional_properties_schema": false,
    "openapi_3_1": false,
    "yaml": false,
    "remote_fetch": false
  },
  "validation_mode_description": "strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation."
}
```

Supported response schema validation covers common OpenAPI 3.0 JSON object and
primitive response schemas, required properties, enum, nullable, local internal
refs, nested local refs, arrays of supported item schemas, and omitted or
boolean `additionalProperties`.

Unsupported features are diagnosed as unsupported instead of being reported as
response-body mismatches.

`schemas_count` is the number of component schemas discovered in the loaded
OpenAPI document. It is not the number of schemas fully supported by Echo MCP's
validator.

## `load_openapi_contract`

Loads one local OpenAPI 3.0 JSON contract through the MCP control plane. This is
state-changing and non-destructive. It does not mutate source files, fetch remote
URLs, introduce provider-specific behavior, or make Echo MCP fully
OpenAPI-first.

Input:

```json
{
  "path": "stripe-openapi.spec3.json",
  "contract_id": "stripe",
  "validation_mode": "strict"
}
```

Fields:

- `path`: required local filesystem path. Relative paths resolve under
  `ECHO_MCP_CONTRACT_ROOT`; absolute paths must resolve inside that root.
- `contract_id`: optional caller-provided active contract identifier.
- `validation_mode`: optional, one of `strict`, `warn`, or `off`; defaults to
  `strict`.
- `force`: optional. Allows replacing the active contract even when behavior is
  configured. Use carefully because existing behavior would otherwise be
  silently associated with a different contract.

Success output:

```json
{
  "loaded": true,
  "contract_id": "stripe",
  "source_path": "stripe-openapi.spec3.json",
  "openapi_version": "3.0.0",
  "operations_count": 123,
  "schemas_count": 456,
  "validation_mode": "strict",
  "validation_scope": "partial",
  "validation_capabilities": {
    "method_path": true,
    "response_status": true,
    "response_content_type": true,
    "response_body": true,
    "local_ref_resolution": true,
    "remote_ref_resolution": false,
    "request_body": false
  },
  "validation_mode_description": "strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.",
  "unsupported_features": {
    "oneOf": 3
  },
  "warnings": [
    "Contract contains oneOf schemas (3 occurrence(s)). Echo MCP currently does not validate oneOf composition in this MVP."
  ],
  "suggested_next_actions": [
    "Call get_contract_status.",
    "Call configure_behavior with contract-valid responses."
  ]
}
```

`source_path` is displayed relative to the contract root when possible, including
when the caller supplied an absolute in-root path. Denied paths are rejected
without leaking private outside-root path details.

Common structured error shape:

```json
{
  "error": "contract path is outside the allowed contract root",
  "code": "contract_path_not_allowed",
  "diagnostics": [
    "OpenAPI contract paths must resolve under the configured contract root."
  ]
}
```

Other diagnostics include unreadable file, invalid JSON, unsupported OpenAPI
version, missing paths, schema compilation failure, unresolved refs, cyclic refs,
and ambiguous route where applicable. Diagnostics avoid including secrets or
large file contents.

## `get_contract_status`

Returns active contract status. This tool is read-only and idempotent.

Active output:

```json
{
  "active": true,
  "contract_id": "stripe",
  "source_path": "stripe-openapi.spec3.json",
  "openapi_version": "3.0.0",
  "operations_count": 123,
  "schemas_count": 456,
  "loaded_at": "2026-06-28T10:00:00Z",
  "validation_mode": "strict",
  "validation_scope": "partial",
  "validation_capabilities": {
    "method_path": true,
    "response_status": true,
    "response_content_type": true,
    "response_body": true,
    "local_ref_resolution": true,
    "remote_ref_resolution": false,
    "request_body": false
  },
  "validation_mode_description": "strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.",
  "contract_root_configured": true,
  "contract_root_source": "ECHO_MCP_CONTRACT_ROOT"
}
```

Inactive output:

```json
{
  "active": false,
  "message": "No OpenAPI contract is currently loaded.",
  "contract_root_configured": false,
  "contract_root_source": "working_directory"
}
```

## `unload_openapi_contract`

Clears the active OpenAPI contract. It does not delete files and does not mutate
source documents.

Safe-state rule: unload is rejected when active behavior exists unless the
caller passes `force: true`. Echo MCP does not silently invalidate or reassign
existing configured behavior.

Input:

```json
{
  "force": true
}
```

Success output:

```json
{
  "unloaded": true,
  "previous_contract_id": "stripe",
  "suggested_next_actions": [
    "Call configure_behavior for manual mock mode or load another contract."
  ]
}
```

## `configure_behavior`

Configures one REST data-plane response rule. Successful calls replace the
currently configured behavior rule.

When no contract is active, this behaves as manual mock behavior and remains
backward compatible. When a contract is active, Echo MCP validates the configured
response against the active contract unless validation is explicitly skipped.

Input:

```json
{
  "behavior_id": "hello-ok",
  "match": {
    "method": "GET",
    "path": "/hello"
  },
  "outcome": {
    "type": "http_response",
    "status_code": 200,
    "content_type": "application/json",
    "body": "{\"message\":\"hello\"}"
  },
  "validation": {
    "mode": "strict",
    "reason": "normal contract-valid behavior"
  }
}
```

Fields:

- `behavior_id`: required non-empty string used in observations.
- `match.method`: required HTTP method.
- `match.path`: required HTTP path.
- `outcome.type`: required, currently only `http_response`.
- `outcome.status_code`: required HTTP status code.
- `outcome.content_type`: optional response content type.
- `outcome.body`: required response body string. Use an empty string for no
  body.
- `validation.mode`: optional per-behavior override, one of `strict`, `warn`, or
  `off`.
- `validation.reason`: required when `validation.mode` is `off` while a contract
  is active.

With no active contract, output includes backward-compatible fields plus
additive guidance:

```json
{
  "configured": true,
  "behavior_id": "hello-ok",
  "warnings": [
    "Manual mock behavior is active. This behavior is not contract-validated. If provider contract fidelity matters, consider OpenAPI-backed validation or hybrid validation."
  ],
  "guidance": [
    "Manual mock behavior is active for configured REST behavior."
  ],
  "suggested_next_actions": [
    "Run the application test normally.",
    "Call get_observations to inspect data-plane evidence."
  ]
}
```

With an active contract, output includes validation disclosure:

```json
{
  "configured": true,
  "behavior_id": "paymentintent-ok",
  "validation_scope": "partial",
  "validation_capabilities": {
    "method_path": true,
    "response_status": true,
    "response_content_type": true,
    "response_body": true,
    "local_ref_resolution": true,
    "request_body": false
  },
  "validation_mode_description": "strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.",
  "suggested_next_actions": [
    "Run the application test normally.",
    "Call get_observations to inspect data-plane evidence."
  ]
}
```

Intentional fault tests can skip validation explicitly:

```json
{
  "behavior_id": "malformed-provider-response",
  "match": {
    "method": "GET",
    "path": "/resource"
  },
  "outcome": {
    "type": "http_response",
    "status_code": 200,
    "content_type": "application/json",
    "body": "{not valid json"
  },
  "validation": {
    "mode": "off",
    "reason": "intentional malformed response test"
  }
}
```

Warning:

```json
{
  "warnings": [
    "Contract validation was skipped for this behavior: intentional malformed response test."
  ]
}
```

Strict validation failures are returned through the MCP control plane and do not
change REST data-plane response bodies:

```json
{
  "error": "Configured behavior violates the active OpenAPI contract",
  "code": "contract_validation_failed",
  "diagnostics": [
    "response status 418 is not defined for POST /v1/payment_intents"
  ],
  "validation_scope": "partial",
  "validation_capabilities": {
    "method_path": true,
    "response_status": true,
    "response_body": true
  },
  "validation_mode_description": "strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation."
}
```

## `send_webhook_event`

Sends one immediate webhook-style HTTP `POST` to a registered application
webhook endpoint. This is not provider contract validation and is not for
arbitrary outbound URLs.

Input:

```json
{
  "event_id": "evt_payment_failed_001",
  "endpoint_name": "payment-events",
  "request": {
    "body": {
      "type": "payment.failed",
      "data": {
        "object": {
          "id": "pay_123"
        }
      }
    }
  }
}
```

The MVP supports only JSON request bodies. It does not support configurable
headers, signatures, retries, scheduling, content negotiation, or raw outbound
URLs supplied by MCP clients.

## `get_observations`

Returns currently available observation information. This tool is read-only and
provides test evidence; it does not configure behavior and does not make manual
mocks contract-validated.

Output shape:

```json
{
  "observations": [
    {
      "request": {
        "method": "GET",
        "path": "/hello"
      },
      "selection": {
        "matched_behavior_id": "hello-ok",
        "matched_on": ["method", "path"]
      },
      "outcome": {
        "type": "http_response",
        "status_code": 200
      }
    }
  ],
  "webhook_deliveries": [],
  "guidance": [
    "Use observations as Echo MCP evidence and keep application behavior assertions in the application test."
  ]
}
```

## `reset`

Clears configured behavior and currently available observation information. Use
`reset` between test scenarios. Reset keeps the active OpenAPI contract loaded.

Input:

```json
{}
```

Output:

```json
{
  "reset": true,
  "cleared": ["behavior", "observations", "webhook_deliveries"],
  "contract_active": true,
  "contract_id": "stripe",
  "suggested_next_actions": [
    "Configure the next behavior or webhook scenario."
  ]
}
```

## Compatibility Corpus

The repository includes a small synthetic OpenAPI corpus under
`internal/contract/testdata/openapi-corpus/` for inline schemas, local refs,
nested refs, arrays of refs, required fields, enum, nullable, unresolved refs,
cyclic refs, and unsupported composition keywords.

`scripts/smoke-openapi-compatibility.sh` can also probe real provider contracts,
including Stripe OpenAPI and GitHub REST OpenAPI, without making normal unit
tests depend on network access. Stripe and GitHub are compatibility probes, not
implementation assumptions.

## Not Included In v0.3.0

v0.3.0 does not introduce `echo-mcp.yaml`, a project manifest, a full
OpenAPI-first runtime, provider-specific simulators, remote fetching, OpenAPI
3.1, YAML, remote/file refs, `allOf`/`oneOf`/`anyOf` semantics, request
body/query/header/path parameter validation, automatic scenario generation, or a
public contract repository. Manual mocks remain supported but do not guarantee
provider contract fidelity.
