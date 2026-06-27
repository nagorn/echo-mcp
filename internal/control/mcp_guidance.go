package control

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverInstructions = `Echo MCP is a controllable API simulation server.
MCP is the control plane for configuring behavior and reading observations; REST data plane and webhook HTTP calls are used by the application under test.
Manual mock behavior is useful for quick exploration, but manual mocks are not contract-validated.
If provider contract fidelity matters, prefer OpenAPI-backed validation or hybrid validation when available.
Recommended first steps:
1. inspect available tools
2. inspect guidance prompts/resources if available
3. choose manual_mock, hybrid_validation, or contract_first based on project needs
4. document when behavior is manual and not contract-validated`

const configureBehaviorDescription = `Configure one REST data-plane response rule as manual mock behavior.

What it does: stores one method/path/status/body rule used by the REST data plane.
When to use: use for quick simulation, exploration, or a documented manual_mock workflow.
When not to use: do not use this alone to prove provider contract fidelity or to send webhook events.
Contract note: manual mock behavior does not prove provider contract fidelity by itself; if contract fidelity matters, look for OpenAPI-backed validation or ask the developer whether a contract is available.
Common next steps: run the application test normally, then call get_observations for data-plane evidence.`

const resetDescription = `Clear Echo MCP control-plane state.

What it does: clears configured behavior, REST observations, and webhook delivery observations.
When to use: use before the next scenario or after a test to return Echo MCP to an empty in-memory state.
When not to use: do not use while another scenario still needs the current observations as evidence.
Contract note: reset is state cleanup; it is not contract-related behavior.
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
3. Use configure_behavior only for REST manual mock behavior.
4. Run the application test through the REST data plane.
5. Use get_observations as evidence and reset between scenarios.

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

If a contract-backed workflow is available, use it. Do not duplicate API schemas in scenario files. Ask the developer whether a contract is available when fidelity matters.`,
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

Echo MCP may validate configured behavior when OpenAPI-backed validation is active, but it does not generate behavior from OpenAPI by itself in this experiment. Keep behavior deterministic and avoid duplicating API schemas when a contract-backed workflow is available.`,
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

First inspect available tools, prompts, and resources. Then choose manual_mock, hybrid_validation, or contract_first based on the project need. Document manual behavior when it is not contract-validated.`,
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

Echo MCP does not generate behavior from test intent or OpenAPI in this experiment.`,
	},
	{
		uri:         resourceManualMock,
		name:        "manual-mock",
		title:       "Manual Mock Guidance",
		description: "Manual mock behavior caveats and next steps.",
		text: `# Manual Mock Guidance

Use configure_behavior for a hand-authored REST method/path/status/body rule. This is useful for quick simulation and exploration.

Manual mock behavior is not contract-validated. If provider contract fidelity matters, ask whether OpenAPI-backed validation or hybrid validation is available.

After configuring behavior, run the application test, call get_observations, and reset before the next scenario.`,
	},
	{
		uri:         resourceContractValidation,
		name:        "contract-validation",
		title:       "Contract Validation Guidance",
		description: "Guidance for OpenAPI-backed or hybrid validation.",
		text: `# Contract Validation Guidance

When an OpenAPI contract exists and fidelity matters, prefer contract_first or hybrid_validation over manual_mock.

Contract-backed validation should be the source of truth for provider methods, paths, schemas, and operation identity where available.

This experiment does not implement a full OpenAPI-first runtime. Echo MCP remains deterministic and does not generate behavior.`,
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
