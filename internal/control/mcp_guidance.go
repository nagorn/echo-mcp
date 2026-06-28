package control

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverInstructions = `Echo MCP is a controllable API simulation server.
MCP is the control plane for configuring behavior and reading observations; REST data plane and webhook HTTP calls are used by the application under test.
Manual mock behavior is useful for quick exploration, but manual mocks are not contract-validated.
If provider contract fidelity matters, use the runtime contract-backed workflow when available and inspect validation_capabilities.
Validation mode strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.
Recommended first steps:
1. inspect available tools
2. inspect guidance prompts/resources if available
3. choose manual_mock, hybrid_validation, or contract_first based on project needs
4. obtain or locate the local OpenAPI contract when contract fidelity matters
5. call load_openapi_contract
6. call get_contract_status
7. call configure_behavior
8. run the application test through the REST data plane
9. use reset between scenarios without unloading the active contract
10. use unload_openapi_contract only when switching contract contexts

Echo MCP performs partial response validation for supported OpenAPI 3.0 JSON capabilities; it does not generate behavior from OpenAPI and is not fully OpenAPI-first by itself.`

const configureBehaviorDescription = `Configure one REST data-plane response rule.

What it does: stores one method/path/status/body rule used by the REST data plane, either as manual mock behavior or partially contract-validated behavior.
When to use: use for quick manual mock simulation, exploration, or after load_openapi_contract in a contract-backed workflow.
When not to use: do not use this alone to prove provider contract fidelity or to send webhook events.
Contract note: manual mock is still allowed. If contract fidelity matters, call load_openapi_contract first. If a contract is active, configured behavior is validated unless explicitly skipped, using partial response validation for supported OpenAPI 3.0 JSON capabilities. Skipping validation requires a reason.
Validation mode strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.
Common next steps: run the application test normally, then call get_observations for data-plane evidence.`

const loadOpenAPIContractDescription = `Load one local OpenAPI contract into Echo MCP runtime state.

What it does: reads a local file only in MVP, activates one generic OpenAPI contract, and enables partial contract-backed validation for configured REST response behavior.
When to use: use after starting Echo MCP and before configure_behavior when provider contract fidelity matters.
When not to use: does not fetch remote URLs, does not upload or mutate contract files, does not add provider-specific logic, and does not make Echo MCP fully OpenAPI-first by itself.
Contract note: generic OpenAPI, not Stripe-specific. Loading validates configured response behavior only for supported capabilities; Echo MCP does not generate mock behavior from the contract.
Path note: paths must resolve under the configured contract root. Set ECHO_MCP_CONTRACT_ROOT to choose the root; when unset, Echo MCP uses the process working directory.
Validation mode strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.
Common next steps: call get_contract_status, then call configure_behavior with contract-valid responses.`

const getContractStatusDescription = `Read-only inspection of the active OpenAPI contract.

What it does: reports whether an active OpenAPI contract is loaded, including contract id, source display path, OpenAPI version, operation count, component schema count, loaded timestamp, and validation mode.
When to use: use after startup loading or load_openapi_contract to confirm contract-backed validation state.
When not to use: do not use this to load, unload, or mutate contracts.
Contract note: this is read-only and idempotent.
Validation mode strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.
Common next steps: call configure_behavior when active, or load_openapi_contract when inactive.`

const unloadOpenAPIContractDescription = `Unload the active OpenAPI contract from Echo MCP runtime state.

What it does: clears active contract state only; it does not delete files and does not mutate source documents.
When to use: use only when switching contract contexts or returning to manual mock mode.
When not to use: do not unload while configured behavior should remain associated with the active contract.
Contract note: unload enforces a safe-state check. If behavior is active, reset first or pass force: true.
Common next steps: call configure_behavior for manual mock mode or load another contract.`

const resetDescription = `Clear Echo MCP control-plane state.

What it does: clears configured behavior, REST observations, and webhook delivery observations.
When to use: use before the next scenario or after a test to return Echo MCP to an empty in-memory state.
When not to use: do not use while another scenario still needs the current observations as evidence.
Contract note: reset keeps the active OpenAPI contract loaded so scenario runners can reuse contract-backed validation across scenarios.
Common next steps: configure the next manual mock behavior or webhook scenario.`

const sendWebhookEventDescription = `Send one webhook-style event to a configured application webhook endpoint.

What it does: sends an immediate HTTP POST with the supplied JSON body to a configured endpoint name.
When to use: use when testing application webhook handling through Echo MCP's control plane.
When not to use: not for arbitrary outbound URLs, not for configuring REST manual mock behavior, and not for provider contract validation.
Contract note: webhook delivery is not OpenAPI contract validation.
Common next steps: assert application behavior normally, then call get_observations for delivery evidence.`

const getObservationsDescription = `Read-only inspection of Echo MCP data-plane evidence.

What it does: returns REST request observations and webhook delivery observations currently held in memory.
When to use: use after the application test or webhook delivery to inspect what Echo MCP actually saw or sent.
When not to use: do not treat observations as application assertions by themselves, and do not use this to configure behavior.
Contract note: observations are test evidence; they do not make manual mocks contract-validated.
Common next steps: compare observations with application assertions and reset before the next scenario.`

const manualMockWarning = "Manual mock behavior is active. This behavior is not contract-validated. If provider contract fidelity matters, consider OpenAPI-backed validation or hybrid validation."

const (
	resourceGettingStarted     = "echo://guides/getting-started"
	resourceWorkflows          = "echo://guides/workflows"
	resourceManualMock         = "echo://guides/manual-mock"
	resourceContractValidation = "echo://guides/contract-validation"
)

type promptDefinition struct {
	name        string
	title       string
	description string
	text        string
}

var guidancePrompts = []promptDefinition{
	{
		name:        "echo_mcp_getting_started",
		title:       "Echo MCP Getting Started",
		description: "Initial workflow guidance for AI coding agents using Echo MCP.",
		text: `Use Echo MCP as a deterministic API simulation server.

1. Inspect tools, prompts, and resources before configuring behavior.
2. Decide whether this task needs manual_mock, hybrid_validation, or contract_first.
3. If contract fidelity matters, locate the local OpenAPI contract and call load_openapi_contract.
4. Call get_contract_status to confirm the active contract.
5. Use configure_behavior for REST behavior; active contracts perform partial response validation for supported capabilities unless explicitly skipped with a reason.
6. Run the application test through the REST data plane.
7. Use get_observations as evidence and reset between scenarios.

Validation mode strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.
Manual mock behavior is useful for quick exploration, but it is not contract-validated.`,
	},
	{
		name:        "echo_mcp_choose_workflow",
		title:       "Choose Echo MCP Workflow",
		description: "Helps an agent choose manual_mock, hybrid_validation, or contract_first.",
		text: `Choose the Echo MCP workflow before configuring behavior.

manual_mock: fastest path for hand-authored method/path/status/body behavior. It is useful for exploration and is not contract-validated.
hybrid_validation: migration path where manual behavior is allowed, but validation/reporting should be used where available.
contract_first: preferred when an OpenAPI contract exists and provider contract fidelity matters.

Validation mode strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.
If a contract-backed workflow is available, use load_openapi_contract, get_contract_status, configure_behavior, reset, and unload_openapi_contract only when switching contract contexts. Do not duplicate API schemas in scenario files. Ask the developer whether a contract is available when fidelity matters.`,
	},
	{
		name:        "echo_mcp_manual_mock_workflow",
		title:       "Manual Mock Workflow",
		description: "Guidance for quick, hand-authored REST mock behavior.",
		text: `Manual mock workflow:

1. Configure exactly the REST behavior needed for the current scenario.
2. Record that the behavior is manual and not contract-validated.
3. Run the application test normally against the REST data plane.
4. Call get_observations and compare evidence with application assertions.
5. Reset before the next scenario.

Manual mocks may drift from provider contracts.`,
	},
	{
		name:        "echo_mcp_contract_validation_workflow",
		title:       "Contract Validation Workflow",
		description: "Guidance for contract-backed or hybrid validation workflows.",
		text: `Contract validation workflow:

Use contract_first when an OpenAPI contract exists and provider contract fidelity matters.
Use hybrid_validation when migrating from manual mocks toward validation.

Recommended runtime workflow:
1. Obtain or locate the OpenAPI contract.
2. Start Echo MCP.
3. Call load_openapi_contract.
4. Call get_contract_status.
5. Call configure_behavior.
6. Let Echo MCP apply partial response validation for supported OpenAPI 3.0 JSON capabilities.
7. Use reset between scenarios.
8. Use unload_openapi_contract only when switching contract contexts.

Validation mode strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.
Echo MCP may validate configured response behavior for supported capabilities when OpenAPI-backed validation is active, but it does not implement request validation or generate behavior from OpenAPI by itself in this release. Keep behavior deterministic and avoid duplicating API schemas when a contract-backed workflow is available.`,
	},
}

type resourceDefinition struct {
	uri         string
	name        string
	title       string
	description string
	text        string
}

var guidanceResources = []resourceDefinition{
	{
		uri:         resourceGettingStarted,
		name:        "getting-started",
		title:       "Echo MCP Getting Started",
		description: "Concise agent-readable startup guidance.",
		text: `# Echo MCP Getting Started

Echo MCP is a controllable API simulation server. MCP is the control plane; REST and webhook HTTP calls are the data plane.

First inspect available tools, prompts, and resources. Then choose manual_mock, hybrid_validation, or contract_first based on the project need. For partial contract-backed response validation, call load_openapi_contract, confirm with get_contract_status, configure behavior, run tests, and reset between scenarios. Document manual behavior when it is not contract-validated.`,
	},
	{
		uri:         resourceWorkflows,
		name:        "workflows",
		title:       "Echo MCP Workflows",
		description: "Guidance for choosing Echo MCP workflows.",
		text: `# Echo MCP Workflows

guided: recommended default for AI agents. Inspect available MCP guidance and avoid silently assuming manual mocks.

manual_mock: fastest path for exploration. Manual behavior may drift and is not contract-validated.

contract_first: preferred when an OpenAPI contract exists and provider contract fidelity matters.

hybrid_validation: migration mode where manual behavior is allowed while validation/reporting is used where available.

Runtime contract-backed workflow: start Echo MCP, load a local OpenAPI contract, confirm status and validation_capabilities, configure behavior, apply partial response validation for supported capabilities, run app tests, and reset between scenarios.

Validation mode strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.
Echo MCP does not generate behavior from test intent or OpenAPI in this workflow.`,
	},
	{
		uri:         resourceManualMock,
		name:        "manual-mock",
		title:       "Manual Mock Guidance",
		description: "Manual mock behavior caveats and next steps.",
		text: `# Manual Mock Guidance

Use configure_behavior for a hand-authored REST method/path/status/body rule. This is useful for quick simulation and exploration.

Manual mock behavior is not contract-validated. If provider contract fidelity matters, call load_openapi_contract first and confirm with get_contract_status.

After configuring behavior, run the application test, call get_observations, and reset before the next scenario.`,
	},
	{
		uri:         resourceContractValidation,
		name:        "contract-validation",
		title:       "Contract Validation Guidance",
		description: "Guidance for OpenAPI-backed or hybrid validation.",
		text: `# Contract Validation Guidance

When an OpenAPI contract exists and fidelity matters, prefer contract_first or hybrid_validation over manual_mock.

Contract-backed validation should be used for supported method/path, response status, response content type, and response body capabilities where available. It is partial response validation, not full provider fidelity.

Use load_openapi_contract for local files only. Echo MCP does not fetch remote URLs, mutate source contracts, or add provider-specific behavior. Reset keeps the active contract loaded between scenarios.
Contract paths must resolve under the configured contract root. Set ECHO_MCP_CONTRACT_ROOT to choose the root; when unset, Echo MCP uses the process working directory.

Validation mode strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation.
This release does not implement a full OpenAPI-first runtime. Echo MCP remains deterministic and does not generate behavior.`,
	},
}

func registerGuidancePrompts(server *mcp.Server) {
	for _, definition := range guidancePrompts {
		definition := definition
		server.AddPrompt(&mcp.Prompt{
			Name:        definition.name,
			Title:       definition.title,
			Description: definition.description,
		}, func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{
				Description: definition.description,
				Messages: []*mcp.PromptMessage{
					{
						Role:    "user",
						Content: &mcp.TextContent{Text: definition.text},
					},
				},
			}, nil
		})
	}
}

func registerGuidanceResources(server *mcp.Server) {
	for _, definition := range guidanceResources {
		definition := definition
		server.AddResource(&mcp.Resource{
			URI:         definition.uri,
			Name:        definition.name,
			Title:       definition.title,
			Description: definition.description,
			MIMEType:    "text/markdown",
		}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			if req.Params.URI != definition.uri {
				return nil, mcp.ResourceNotFoundError(req.Params.URI)
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      definition.uri,
						MIMEType: "text/markdown",
						Text:     definition.text,
					},
				},
			}, nil
		})
	}
}

func boolPtr(value bool) *bool {
	return &value
}
