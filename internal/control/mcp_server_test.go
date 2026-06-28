package control

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"echo-mcp/internal/contract"
	"echo-mcp/internal/httpserver"
	"echo-mcp/internal/state"
	"echo-mcp/internal/webhook"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const paymentIntentDeclinedBody = `{"error":{"type":"card_error","code":"card_declined","decline_code":"generic_decline","message":"Your card was declined.","payment_intent":{"id":"pi_123","object":"payment_intent","status":"requires_payment_method"}}}`

func TestMCPInitializeInstructionsExposeAgentGuidance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(New(state.New())))
	initializeResult := clientSession.InitializeResult()
	if initializeResult.ServerInfo == nil {
		t.Fatal("InitializeResult().ServerInfo = nil")
	}
	if initializeResult.ServerInfo.Name != "echo-mcp" {
		t.Fatalf("serverInfo.name = %q, want echo-mcp", initializeResult.ServerInfo.Name)
	}
	if initializeResult.ServerInfo.Version != "v0.3.0" {
		t.Fatalf("serverInfo.version = %q, want v0.3.0", initializeResult.ServerInfo.Version)
	}
	instructions := initializeResult.Instructions

	for _, want := range []string{
		"controllable API simulation server",
		"control plane",
		"REST data plane",
		"Manual mock behavior",
		"not contract-validated",
		"strict means strict enforcement of the validation capabilities currently supported by Echo MCP",
		"not full OpenAPI validation",
		"inspect available tools",
		"guidance prompts/resources",
		"manual_mock, hybrid_validation, or contract_first",
	} {
		if !strings.Contains(instructions, want) {
			t.Fatalf("initialize instructions missing %q:\n%s", want, instructions)
		}
	}
}

func TestMCPListToolsExposeAgentGuidance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(New(state.New())))
	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	tools := toolsByName(result.Tools)
	for _, name := range []string{"configure_behavior", "load_openapi_contract", "get_contract_status", "unload_openapi_contract", "reset", "send_webhook_event", "get_observations"} {
		if _, ok := tools[name]; !ok {
			t.Fatalf("ListTools() missing %q; tools = %+v", name, toolNames(result.Tools))
		}
	}

	configure := requireTool(t, tools, "configure_behavior")
	for _, want := range []string{
		"manual mock behavior",
		"When to use",
		"When not to use",
		"call load_openapi_contract first",
		"validated unless explicitly skipped",
		"Skipping validation requires a reason",
		"strict means strict enforcement of the validation capabilities currently supported by Echo MCP",
		"not full OpenAPI validation",
		"get_observations",
	} {
		if !strings.Contains(configure.Description, want) {
			t.Fatalf("configure_behavior description missing %q:\n%s", want, configure.Description)
		}
	}
	assertToolAnnotation(t, configure, "Configure REST Behavior", false, ptrBool(false), false, ptrBool(false))

	loadContract := requireTool(t, tools, "load_openapi_contract")
	for _, want := range []string{
		"local file only",
		"generic OpenAPI",
		"does not fetch remote URLs",
		"does not make Echo MCP fully OpenAPI-first",
		"contract-backed validation",
		"strict means strict enforcement of the validation capabilities currently supported by Echo MCP",
	} {
		if !strings.Contains(loadContract.Description, want) {
			t.Fatalf("load_openapi_contract description missing %q:\n%s", want, loadContract.Description)
		}
	}
	assertToolAnnotation(t, loadContract, "Load OpenAPI Contract", false, ptrBool(false), false, ptrBool(false))

	contractStatus := requireTool(t, tools, "get_contract_status")
	if !strings.Contains(strings.ToLower(contractStatus.Description), "read-only") || !strings.Contains(contractStatus.Description, "active OpenAPI contract") {
		t.Fatalf("get_contract_status description is not agent-facing:\n%s", contractStatus.Description)
	}
	if !strings.Contains(contractStatus.Description, "not full OpenAPI validation") {
		t.Fatalf("get_contract_status description does not disclose partial validation:\n%s", contractStatus.Description)
	}
	assertToolAnnotation(t, contractStatus, "Get Contract Status", true, nil, true, ptrBool(false))

	unloadContract := requireTool(t, tools, "unload_openapi_contract")
	if !strings.Contains(unloadContract.Description, "does not delete files") || !strings.Contains(unloadContract.Description, "force") {
		t.Fatalf("unload_openapi_contract description is not safe-state oriented:\n%s", unloadContract.Description)
	}
	assertToolAnnotation(t, unloadContract, "Unload OpenAPI Contract", false, ptrBool(false), false, ptrBool(false))

	reset := requireTool(t, tools, "reset")
	if !strings.Contains(reset.Description, "When to use") || !strings.Contains(reset.Description, "clears configured behavior") {
		t.Fatalf("reset description is not agent-facing:\n%s", reset.Description)
	}
	assertToolAnnotation(t, reset, "Reset Echo MCP State", false, ptrBool(true), true, ptrBool(false))

	sendWebhook := requireTool(t, tools, "send_webhook_event")
	if !strings.Contains(sendWebhook.Description, "When not to use") || !strings.Contains(sendWebhook.Description, "not for arbitrary outbound URLs") {
		t.Fatalf("send_webhook_event description is not agent-facing:\n%s", sendWebhook.Description)
	}
	assertToolAnnotation(t, sendWebhook, "Send Webhook Event", false, ptrBool(false), false, ptrBool(false))

	observations := requireTool(t, tools, "get_observations")
	observationDescription := strings.ToLower(observations.Description)
	if !strings.Contains(observationDescription, "read-only") || !strings.Contains(observations.Description, "test evidence") {
		t.Fatalf("get_observations description is not agent-facing:\n%s", observations.Description)
	}
	assertToolAnnotation(t, observations, "Get Observations", true, nil, true, ptrBool(false))
}

func TestMCPConfigureBehaviorReturnsManualMockGuidanceOnlyInControlPlane(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	store := state.New()
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(New(store)))

	configureResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "rule-payment-ok",
			"match": map[string]any{
				"method": http.MethodGet,
				"path":   "/payments/123",
			},
			"outcome": map[string]any{
				"type":        "http_response",
				"status_code": http.StatusAccepted,
				"body":        `{"status":"accepted"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, configureResult)

	guidance := decodeStructuredMap(t, configureResult)
	assertStringSliceContains(t, guidance, "warnings", "Manual mock behavior is active. This behavior is not contract-validated. If provider contract fidelity matters, consider OpenAPI-backed validation or hybrid validation.")
	assertStringSliceContains(t, guidance, "suggested_next_actions", "Run the application test normally.")
	assertStringSliceContains(t, guidance, "suggested_next_actions", "Call get_observations to inspect data-plane evidence.")
	if _, ok := guidance["validation_scope"]; ok {
		t.Fatalf("manual configure output unexpectedly included validation_scope: %+v", guidance)
	}
	if _, ok := guidance["validation_capabilities"]; ok {
		t.Fatalf("manual configure output unexpectedly included validation_capabilities: %+v", guidance)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/payments/123", nil))

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if got := response.Body.String(); got != `{"status":"accepted"}` {
		t.Fatalf("body = %q, want configured body only", got)
	}
	if got := response.Header().Get("X-Echo-MCP-Warning"); got != "" {
		t.Fatalf("X-Echo-MCP-Warning = %q, want no REST data-plane warning header", got)
	}
}

func TestMCPConfigureBehaviorOmitsManualMockWarningWhenContractValidationIsActive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	validator, err := contract.LoadOpenAPIFile(filepath.Join("..", "contract", "testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithValidator(state.New(), validator)))

	configureResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "stripe-like-paymentintent-card-declined",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "application/json",
				"body":         paymentIntentDeclinedBody,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, configureResult)

	guidance := decodeStructuredMap(t, configureResult)
	assertStringSliceNotContains(t, guidance, "warnings", "Manual mock behavior is active.")
	assertStringSliceContains(t, guidance, "guidance", "Contract validation is active for configured REST behavior.")
}

func TestMCPPromptsExposeAgentWorkflowGuidance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(New(state.New())))
	result, err := clientSession.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}

	prompts := promptsByName(result.Prompts)
	for _, name := range []string{
		"echo_mcp_getting_started",
		"echo_mcp_choose_workflow",
		"echo_mcp_manual_mock_workflow",
		"echo_mcp_contract_validation_workflow",
	} {
		if _, ok := prompts[name]; !ok {
			t.Fatalf("ListPrompts() missing %q; prompts = %+v", name, promptNames(result.Prompts))
		}
	}

	prompt, err := clientSession.GetPrompt(ctx, &mcp.GetPromptParams{Name: "echo_mcp_choose_workflow"})
	if err != nil {
		t.Fatalf("GetPrompt(echo_mcp_choose_workflow) error = %v", err)
	}
	text := promptText(t, prompt)
	for _, want := range []string{
		"manual_mock",
		"hybrid_validation",
		"contract_first",
		"not contract-validated",
		"not full OpenAPI validation",
		"Do not duplicate API schemas",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("choose workflow prompt missing %q:\n%s", want, text)
		}
	}
}

func TestMCPResourcesExposeAgentGuides(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(New(state.New())))
	result, err := clientSession.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	resources := resourcesByURI(result.Resources)
	for _, uri := range []string{
		"echo://guides/getting-started",
		"echo://guides/workflows",
		"echo://guides/manual-mock",
		"echo://guides/contract-validation",
	} {
		if _, ok := resources[uri]; !ok {
			t.Fatalf("ListResources() missing %q; resources = %+v", uri, resourceURIs(result.Resources))
		}
	}

	guide, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "echo://guides/workflows"})
	if err != nil {
		t.Fatalf("ReadResource(echo://guides/workflows) error = %v", err)
	}
	text := resourceText(t, guide)
	for _, want := range []string{
		"guided",
		"manual_mock",
		"contract_first",
		"hybrid_validation",
		"not full OpenAPI validation",
		"Echo MCP does not generate behavior",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("workflows resource missing %q:\n%s", want, text)
		}
	}
}

func TestMCPConfigureBehaviorDrivesRESTDataPlaneAndReportsObservations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	store := state.New()
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(New(store)))

	configureResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "rule-payment-ok",
			"match": map[string]any{
				"method": http.MethodGet,
				"path":   "/payments/123",
			},
			"outcome": map[string]any{
				"type":        "http_response",
				"status_code": http.StatusAccepted,
				"body":        `{"status":"accepted"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, configureResult)
	configured := decodeStructuredContent[configureBehaviorOutput](t, configureResult)
	if !configured.Configured {
		t.Fatal("configured = false, want true")
	}
	if configured.BehaviorID != "rule-payment-ok" {
		t.Fatalf("behavior_id = %q, want %q", configured.BehaviorID, "rule-payment-ok")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	request := httptest.NewRequest(http.MethodGet, "/payments/123", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if got := response.Body.String(); got != `{"status":"accepted"}` {
		t.Fatalf("body = %q, want %q", got, `{"status":"accepted"}`)
	}

	observationsResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_observations",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_observations CallTool() error = %v", err)
	}
	assertToolSuccess(t, observationsResult)
	observations := decodeStructuredContent[getObservationsOutput](t, observationsResult)
	if len(observations.Observations) != 1 {
		t.Fatalf("len(observations) = %d, want 1", len(observations.Observations))
	}

	observation := observations.Observations[0]
	if observation.Request.Method != http.MethodGet {
		t.Fatalf("request.method = %q, want %q", observation.Request.Method, http.MethodGet)
	}
	if observation.Request.Path != "/payments/123" {
		t.Fatalf("request.path = %q, want %q", observation.Request.Path, "/payments/123")
	}
	if observation.Selection.MatchedBehaviorID != "rule-payment-ok" {
		t.Fatalf("matched_behavior_id = %q, want %q", observation.Selection.MatchedBehaviorID, "rule-payment-ok")
	}
	if want := []string{"method", "path"}; !equalStrings(observation.Selection.MatchedOn, want) {
		t.Fatalf("matched_on = %+v, want %+v", observation.Selection.MatchedOn, want)
	}
	if observation.Outcome.Type != "http_response" {
		t.Fatalf("outcome.type = %q, want %q", observation.Outcome.Type, "http_response")
	}
	if observation.Outcome.StatusCode != http.StatusAccepted {
		t.Fatalf("outcome.status_code = %d, want %d", observation.Outcome.StatusCode, http.StatusAccepted)
	}
}

func TestMCPResetClearsConfiguredBehaviorAndObservations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	store := state.New()
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(New(store)))

	_, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "rule-payment-ok",
			"match": map[string]any{
				"method": http.MethodGet,
				"path":   "/payments/123",
			},
			"outcome": map[string]any{
				"type":        "http_response",
				"status_code": http.StatusAccepted,
				"body":        `{"status":"accepted"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	server.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/payments/123", nil))

	resetResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "reset",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("reset CallTool() error = %v", err)
	}
	assertToolSuccess(t, resetResult)
	reset := decodeStructuredContent[resetOutput](t, resetResult)
	if !reset.Reset {
		t.Fatal("reset = false, want true")
	}
	if want := []string{"behavior", "observations", "webhook_deliveries"}; !equalStrings(reset.Cleared, want) {
		t.Fatalf("cleared = %+v, want %+v", reset.Cleared, want)
	}

	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/payments/123", nil))
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("status after reset = %d, want %d", response.Code, http.StatusNotImplemented)
	}

	observationsResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_observations",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_observations CallTool() error = %v", err)
	}
	assertToolSuccess(t, observationsResult)
	observations := decodeStructuredContent[getObservationsOutput](t, observationsResult)
	if len(observations.Observations) != 0 {
		t.Fatalf("len(observations) = %d, want 0", len(observations.Observations))
	}
}

func TestMCPRuntimeOpenAPIContractLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	store := state.New()
	clientSession, sourcePath := connectMCPTestClientWithPaymentIntentContractRoot(t, ctx, store)

	statusResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_contract_status",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_contract_status CallTool() error = %v", err)
	}
	assertToolSuccess(t, statusResult)
	inactive := decodeStructuredContent[getContractStatusOutput](t, statusResult)
	if inactive.Active {
		t.Fatal("status active = true before load, want false")
	}
	if inactive.Message != "No OpenAPI contract is currently loaded." {
		t.Fatalf("inactive message = %q", inactive.Message)
	}
	if !inactive.ContractRootConfigured {
		t.Fatal("inactive contract_root_configured = false, want true")
	}
	if inactive.ContractRootSource != ContractRootSourceEnv {
		t.Fatalf("inactive contract_root_source = %q, want %q", inactive.ContractRootSource, ContractRootSourceEnv)
	}

	loadResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "load_openapi_contract",
		Arguments: map[string]any{
			"path":            sourcePath,
			"contract_id":     "stripe",
			"validation_mode": "strict",
		},
	})
	if err != nil {
		t.Fatalf("load_openapi_contract CallTool() error = %v", err)
	}
	assertToolSuccess(t, loadResult)
	loaded := decodeStructuredContent[loadOpenAPIContractOutput](t, loadResult)
	if !loaded.Loaded {
		t.Fatal("loaded = false, want true")
	}
	if loaded.ContractID != "stripe" {
		t.Fatalf("contract_id = %q, want stripe", loaded.ContractID)
	}
	if loaded.SourcePath != sourcePath {
		t.Fatalf("source_path = %q, want %q", loaded.SourcePath, sourcePath)
	}
	if loaded.OpenAPIVersion != "3.0.3" {
		t.Fatalf("openapi_version = %q, want 3.0.3", loaded.OpenAPIVersion)
	}
	if loaded.OperationsCount != 1 {
		t.Fatalf("operations_count = %d, want 1", loaded.OperationsCount)
	}
	if loaded.SchemasCount != 0 {
		t.Fatalf("schemas_count = %d, want 0 component schemas", loaded.SchemasCount)
	}
	if loaded.ValidationMode != "strict" {
		t.Fatalf("validation_mode = %q, want strict", loaded.ValidationMode)
	}
	assertPartialValidationDisclosure(t, loaded.ValidationScope, loaded.ValidationCapabilities, loaded.ValidationModeDescription)
	assertStringSliceContains(t, decodeStructuredMap(t, loadResult), "suggested_next_actions", "Call get_contract_status.")

	statusResult, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_contract_status",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_contract_status after load CallTool() error = %v", err)
	}
	assertToolSuccess(t, statusResult)
	active := decodeStructuredContent[getContractStatusOutput](t, statusResult)
	if !active.Active {
		t.Fatal("status active = false after load, want true")
	}
	if active.ContractID != "stripe" {
		t.Fatalf("status contract_id = %q, want stripe", active.ContractID)
	}
	if active.LoadedAt == "" {
		t.Fatal("loaded_at is empty, want timestamp")
	}
	if !active.ContractRootConfigured {
		t.Fatal("active contract_root_configured = false, want true")
	}
	if active.ContractRootSource != ContractRootSourceEnv {
		t.Fatalf("active contract_root_source = %q, want %q", active.ContractRootSource, ContractRootSourceEnv)
	}
	assertPartialValidationDisclosure(t, active.ValidationScope, active.ValidationCapabilities, active.ValidationModeDescription)

	configureResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "stripe-like-paymentintent-card-declined",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "application/json",
				"body":         paymentIntentDeclinedBody,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, configureResult)
	configured := decodeStructuredContent[configureBehaviorOutput](t, configureResult)
	assertPartialValidationDisclosure(t, configured.ValidationScope, configured.ValidationCapabilities, configured.ValidationModeDescription)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/payment_intents/pi_123/confirm", nil))
	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusPaymentRequired)
	}
	if got := response.Body.String(); got != paymentIntentDeclinedBody {
		t.Fatalf("body = %q, want configured body", got)
	}

	resetResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "reset",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("reset CallTool() error = %v", err)
	}
	assertToolSuccess(t, resetResult)
	reset := decodeStructuredContent[resetOutput](t, resetResult)
	if !reset.ContractActive {
		t.Fatal("reset contract_active = false, want true")
	}
	if reset.ContractID != "stripe" {
		t.Fatalf("reset contract_id = %q, want stripe", reset.ContractID)
	}

	statusResult, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_contract_status",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_contract_status after reset CallTool() error = %v", err)
	}
	assertToolSuccess(t, statusResult)
	activeAfterReset := decodeStructuredContent[getContractStatusOutput](t, statusResult)
	if !activeAfterReset.Active {
		t.Fatal("status active = false after reset, want true")
	}

	unloadResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "unload_openapi_contract",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("unload_openapi_contract CallTool() error = %v", err)
	}
	assertToolSuccess(t, unloadResult)
	unloaded := decodeStructuredContent[unloadOpenAPIContractOutput](t, unloadResult)
	if !unloaded.Unloaded {
		t.Fatal("unloaded = false, want true")
	}
	if unloaded.PreviousContractID != "stripe" {
		t.Fatalf("previous_contract_id = %q, want stripe", unloaded.PreviousContractID)
	}

	statusResult, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_contract_status",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_contract_status after unload CallTool() error = %v", err)
	}
	assertToolSuccess(t, statusResult)
	statusAfterUnload := decodeStructuredContent[getContractStatusOutput](t, statusResult)
	if statusAfterUnload.Active {
		t.Fatal("status active = true after unload, want false")
	}
}

func TestMCPRuntimeOpenAPIContractLoadReturnsStructuredDiagnostics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	clientSession, _ := connectMCPTestClientWithContractRoot(t, ctx, state.New(), t.TempDir())

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "load_openapi_contract",
		Arguments: map[string]any{
			"path": "missing-openapi.json",
		},
	})
	if err != nil {
		t.Fatalf("load_openapi_contract CallTool() protocol error = %v", err)
	}
	toolError := assertStructuredToolError(t, result)
	if toolError.Code != "unreadable_file" {
		t.Fatalf("error code = %q, want unreadable_file", toolError.Code)
	}
	if len(toolError.Diagnostics) == 0 {
		t.Fatal("diagnostics empty, want diagnostic details")
	}
}

func TestMCPLoadOpenAPIContractRejectsPathOutsideContractRoot(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	root := t.TempDir()
	clientSession, _ := connectMCPTestClientWithContractRoot(t, ctx, state.New(), root)
	outsidePath := filepath.Join(t.TempDir(), "outside-openapi.json")
	if err := os.WriteFile(outsidePath, []byte(`{"openapi":"3.0.3","paths":{}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "load_openapi_contract",
		Arguments: map[string]any{
			"path": outsidePath,
		},
	})
	if err != nil {
		t.Fatalf("load_openapi_contract CallTool() protocol error = %v", err)
	}
	toolError := assertStructuredToolError(t, result)
	if toolError.Code != "contract_path_not_allowed" {
		t.Fatalf("error code = %q, want contract_path_not_allowed", toolError.Code)
	}
	if toolError.Error != "contract path is outside the allowed contract root" {
		t.Fatalf("error = %q, want outside-root message", toolError.Error)
	}
	if len(toolError.Diagnostics) != 1 || toolError.Diagnostics[0] != "OpenAPI contract paths must resolve under the configured contract root." {
		t.Fatalf("diagnostics = %+v, want safe outside-root diagnostic", toolError.Diagnostics)
	}
	diagnostics := strings.Join(toolError.Diagnostics, "\n")
	if strings.Contains(diagnostics, outsidePath) || strings.Contains(diagnostics, root) {
		t.Fatalf("diagnostics leak local paths: %+v", toolError.Diagnostics)
	}
}

func TestMCPLoadOpenAPIContractSanitizesAbsoluteInRootSourcePath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	root := t.TempDir()
	relativePath := filepath.Join("contracts", "payment-intent-openapi.json")
	copyMCPContractFixture(t, root, relativePath)
	clientSession, _ := connectMCPTestClientWithContractRoot(t, ctx, state.New(), root)

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "load_openapi_contract",
		Arguments: map[string]any{
			"path":        filepath.Join(root, relativePath),
			"contract_id": "absolute-in-root",
		},
	})
	if err != nil {
		t.Fatalf("load_openapi_contract CallTool() error = %v", err)
	}
	assertToolSuccess(t, result)
	loaded := decodeStructuredContent[loadOpenAPIContractOutput](t, result)
	if loaded.SourcePath != relativePath {
		t.Fatalf("source_path = %q, want relative display path %q", loaded.SourcePath, relativePath)
	}
	if strings.Contains(loaded.SourcePath, root) {
		t.Fatalf("source_path leaks contract root: %q", loaded.SourcePath)
	}

	statusResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_contract_status",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_contract_status CallTool() error = %v", err)
	}
	assertToolSuccess(t, statusResult)
	status := decodeStructuredContent[getContractStatusOutput](t, statusResult)
	if status.SourcePath != relativePath {
		t.Fatalf("status source_path = %q, want relative display path %q", status.SourcePath, relativePath)
	}
	if strings.Contains(status.SourcePath, root) {
		t.Fatalf("status source_path leaks contract root: %q", status.SourcePath)
	}
}

func TestMCPLoadOpenAPIContractWarnsForUnsupportedSchemaFeatures(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	root := t.TempDir()
	clientSession, _ := connectMCPTestClientWithContractRoot(t, ctx, state.New(), root)
	sourcePath := writeUnsupportedFeatureOpenAPI(t, root)

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "load_openapi_contract",
		Arguments: map[string]any{
			"path":        sourcePath,
			"contract_id": "stripe-ref-like",
		},
	})
	if err != nil {
		t.Fatalf("load_openapi_contract CallTool() error = %v", err)
	}
	assertToolSuccess(t, result)
	output := decodeStructuredContent[loadOpenAPIContractOutput](t, result)
	assertPartialValidationDisclosure(t, output.ValidationScope, output.ValidationCapabilities, output.ValidationModeDescription)
	assertStringSliceContainsSubstring(t, decodeStructuredMap(t, result), "warnings", "contains oneOf schemas")
	if output.UnsupportedFeatures["oneOf"] == 0 {
		t.Fatalf("unsupported_features[oneOf] = 0, want positive count: %+v", output.UnsupportedFeatures)
	}
	if output.UnsupportedFeatures["$ref"] != 0 {
		t.Fatalf("unsupported_features[$ref] = %d, want 0 for supported local refs", output.UnsupportedFeatures["$ref"])
	}
	if output.UnsupportedFeatures["arrays"] != 0 {
		t.Fatalf("unsupported_features[arrays] = %d, want 0 for supported arrays", output.UnsupportedFeatures["arrays"])
	}
}

func TestMCPUnloadOpenAPIContractRejectsActiveBehaviorUnlessForced(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	clientSession, sourcePath := connectMCPTestClientWithPaymentIntentContractRoot(t, ctx, state.New())
	loadStrictContract(t, ctx, clientSession, sourcePath)
	configureValidPaymentIntentBehavior(t, ctx, clientSession)

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "unload_openapi_contract",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("unload_openapi_contract CallTool() protocol error = %v", err)
	}
	toolError := assertStructuredToolError(t, result)
	if toolError.Code != "reset_required" {
		t.Fatalf("error code = %q, want reset_required", toolError.Code)
	}

	forced, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "unload_openapi_contract",
		Arguments: map[string]any{
			"force": true,
		},
	})
	if err != nil {
		t.Fatalf("forced unload_openapi_contract CallTool() error = %v", err)
	}
	assertToolSuccess(t, forced)
}

func TestMCPLoadOpenAPIContractRejectsContractSwitchWhenBehaviorIsActive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	clientSession, sourcePath := connectMCPTestClientWithPaymentIntentContractRoot(t, ctx, state.New())
	loadStrictContract(t, ctx, clientSession, sourcePath)
	configureValidPaymentIntentBehavior(t, ctx, clientSession)

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "load_openapi_contract",
		Arguments: map[string]any{
			"path":        sourcePath,
			"contract_id": "stripe-v2",
		},
	})
	if err != nil {
		t.Fatalf("load_openapi_contract CallTool() protocol error = %v", err)
	}
	toolError := assertStructuredToolError(t, result)
	if toolError.Code != "reset_required" {
		t.Fatalf("error code = %q, want reset_required", toolError.Code)
	}
}

func TestMCPConfigureBehaviorValidatesAgainstOpenAPIContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	validator, err := contract.LoadOpenAPIFile(filepath.Join("..", "contract", "testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}
	store := state.New()
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithValidator(store, validator)))

	configureResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "stripe-like-paymentintent-card-declined",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "application/json",
				"body":         paymentIntentDeclinedBody,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, configureResult)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/payment_intents/pi_123/confirm", nil))

	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusPaymentRequired)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}
}

func TestMCPConfigureBehaviorRejectsContractViolationWithoutReplacingCurrentRule(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	validator, err := contract.LoadOpenAPIFile(filepath.Join("..", "contract", "testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}
	store := state.New()
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithValidator(store, validator)))

	validResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "stripe-like-paymentintent-card-declined",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "application/json",
				"body":         paymentIntentDeclinedBody,
			},
		},
	})
	if err != nil {
		t.Fatalf("valid configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, validResult)

	invalidResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "stripe-like-paymentintent-invalid",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusOK,
				"content_type": "application/json",
				"body":         `{"ok":true}`,
			},
		},
	})
	if err == nil && (invalidResult == nil || !invalidResult.IsError) {
		t.Fatal("invalid configure_behavior succeeded, want MCP error")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpserver.New(store, logger)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/payment_intents/pi_123/confirm", nil))

	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("status after rejected rule = %d, want previous status %d", response.Code, http.StatusPaymentRequired)
	}
	if got := response.Body.String(); got != paymentIntentDeclinedBody {
		t.Fatalf("body after rejected rule = %q, want previous body %q", got, paymentIntentDeclinedBody)
	}
}

func TestMCPConfigureBehaviorStrictModeRejectsContractViolations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	tests := []struct {
		name     string
		match    map[string]any
		outcome  map[string]any
		wantCode string
	}{
		{
			name: "unknown path",
			match: map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/unknown",
			},
			outcome: map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "application/json",
				"body":         paymentIntentDeclinedBody,
			},
			wantCode: "contract_validation_failed",
		},
		{
			name: "invalid status",
			match: map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			outcome: map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusOK,
				"content_type": "application/json",
				"body":         `{"ok":true}`,
			},
			wantCode: "contract_validation_failed",
		},
		{
			name: "invalid content type",
			match: map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			outcome: map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "text/plain",
				"body":         paymentIntentDeclinedBody,
			},
			wantCode: "contract_validation_failed",
		},
		{
			name: "invalid response body",
			match: map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			outcome: map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "application/json",
				"body":         `{"error":{"type":"card_error"}}`,
			},
			wantCode: "contract_validation_failed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			clientSession, sourcePath := connectMCPTestClientWithPaymentIntentContractRoot(t, ctx, state.New())
			loadStrictContract(t, ctx, clientSession, sourcePath)

			result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name: "configure_behavior",
				Arguments: map[string]any{
					"behavior_id": "contract-violation",
					"match":       tt.match,
					"outcome":     tt.outcome,
				},
			})
			if err != nil {
				t.Fatalf("configure_behavior CallTool() protocol error = %v", err)
			}
			toolError := assertStructuredToolError(t, result)
			if toolError.Code != tt.wantCode {
				t.Fatalf("error code = %q, want %q", toolError.Code, tt.wantCode)
			}
		})
	}
}

func TestMCPConfigureBehaviorRejectsUnsupportedFeatureAfterLocalRefResolution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	root := t.TempDir()
	clientSession, _ := connectMCPTestClientWithContractRoot(t, ctx, state.New(), root)
	sourcePath := writeUnsupportedFeatureOpenAPI(t, root)

	loadResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "load_openapi_contract",
		Arguments: map[string]any{
			"path":        sourcePath,
			"contract_id": "stripe-ref-like",
		},
	})
	if err != nil {
		t.Fatalf("load_openapi_contract CallTool() error = %v", err)
	}
	assertToolSuccess(t, loadResult)

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "stripe-paymentintent-create-success",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusOK,
				"content_type": "application/json",
				"body":         `{"id":"pi_123","object":"payment_intent","status":"requires_payment_method"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() protocol error = %v", err)
	}
	toolError := assertStructuredToolError(t, result)
	if toolError.Code != "unsupported_contract_feature" {
		t.Fatalf("error code = %q, want unsupported_contract_feature", toolError.Code)
	}
	assertPartialValidationDisclosure(t, toolError.ValidationScope, toolError.ValidationCapabilities, toolError.ValidationModeDescription)
	diagnostics := strings.Join(toolError.Diagnostics, "\n")
	for _, want := range []string{
		`schema path "$" contains unsupported oneOf`,
		"oneOf validation is unsupported in this MVP",
		"behavior was not validated",
		"use validation.mode=off with reason",
		"use a reduced/inline schema fixture",
		"wait for oneOf support",
	} {
		if !strings.Contains(diagnostics, want) {
			t.Fatalf("diagnostics missing %q:\n%s", want, diagnostics)
		}
	}
	if strings.Contains(toolError.Error, "violates") {
		t.Fatalf("error implies behavior invalid rather than unsupported schema: %q", toolError.Error)
	}
}

func TestMCPConfigureBehaviorWarnModeAcceptsContractViolationWithWarning(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	clientSession, sourcePath := connectMCPTestClientWithPaymentIntentContractRoot(t, ctx, state.New())
	loadResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "load_openapi_contract",
		Arguments: map[string]any{
			"path":            sourcePath,
			"contract_id":     "stripe",
			"validation_mode": "warn",
		},
	})
	if err != nil {
		t.Fatalf("load_openapi_contract CallTool() error = %v", err)
	}
	assertToolSuccess(t, loadResult)

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "intentional-contract-warning",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/unknown",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusOK,
				"content_type": "application/json",
				"body":         `{"ok":true}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, result)
	output := decodeStructuredMap(t, result)
	assertStringSliceContainsSubstring(t, output, "warnings", "Contract validation warning")
}

func TestMCPConfigureBehaviorOffModeRequiresReasonAndAcceptsIntentionalFault(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	clientSession, sourcePath := connectMCPTestClientWithPaymentIntentContractRoot(t, ctx, state.New())
	loadStrictContract(t, ctx, clientSession, sourcePath)

	missingReason, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "invalid-without-reason",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "text/plain",
				"body":         `not-json`,
			},
			"validation": map[string]any{
				"mode": "off",
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior missing reason CallTool() protocol error = %v", err)
	}
	toolError := assertStructuredToolError(t, missingReason)
	if toolError.Code != "validation_reason_required" {
		t.Fatalf("error code = %q, want validation_reason_required", toolError.Code)
	}

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "intentional-malformed-response",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "text/plain",
				"body":         `not-json`,
			},
			"validation": map[string]any{
				"mode":   "off",
				"reason": "intentional malformed response test",
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior off mode CallTool() error = %v", err)
	}
	assertToolSuccess(t, result)
	output := decodeStructuredMap(t, result)
	assertStringSliceContains(t, output, "warnings", "Contract validation was skipped for this behavior: intentional malformed response test.")
}

func TestMCPSendWebhookEventDeliversToConfiguredEndpointAndReportsObservation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	received := make(chan webhookRequest, 1)
	webhookClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Decode(body) error = %v", err)
		}
		if err := r.Body.Close(); err != nil {
			t.Errorf("Close(body) error = %v", err)
		}
		received <- webhookRequest{
			Method:      r.Method,
			ContentType: r.Header.Get("Content-Type"),
			Body:        body,
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}

	store := state.New()
	endpoints := webhook.NewEndpoints(webhook.Endpoint{Name: "payment-events", Address: "http://application.test/webhooks/payments"})
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithWebhookSender(store, nil, webhook.NewSender(endpoints, webhookClient))))

	sendResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "send_webhook_event",
		Arguments: map[string]any{
			"event_id":      "evt_payment_failed_001",
			"endpoint_name": "payment-events",
			"request": map[string]any{
				"body": map[string]any{
					"type": "payment.failed",
					"data": map[string]any{
						"object": map[string]any{"id": "pay_123"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("send_webhook_event CallTool() error = %v", err)
	}
	assertToolSuccess(t, sendResult)
	delivery := decodeStructuredContent[sendWebhookEventOutput](t, sendResult)
	if !delivery.Attempted {
		t.Fatal("attempted = false, want true")
	}
	if delivery.EventID != "evt_payment_failed_001" {
		t.Fatalf("event_id = %q", delivery.EventID)
	}
	if delivery.EndpointName != "payment-events" {
		t.Fatalf("endpoint_name = %q", delivery.EndpointName)
	}
	if delivery.Delivery.Outcome != "response_received" {
		t.Fatalf("delivery.outcome = %q, want response_received", delivery.Delivery.Outcome)
	}
	if delivery.Delivery.StatusCode != http.StatusNoContent {
		t.Fatalf("delivery.status_code = %d, want %d", delivery.Delivery.StatusCode, http.StatusNoContent)
	}

	request := <-received
	if request.Method != http.MethodPost {
		t.Fatalf("request.Method = %q, want %q", request.Method, http.MethodPost)
	}
	if request.ContentType != "application/json" {
		t.Fatalf("request.ContentType = %q, want application/json", request.ContentType)
	}

	observationsResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_observations",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_observations CallTool() error = %v", err)
	}
	assertToolSuccess(t, observationsResult)
	observations := decodeStructuredContent[getObservationsOutput](t, observationsResult)
	if len(observations.WebhookDeliveries) != 1 {
		t.Fatalf("len(webhook_deliveries) = %d, want 1", len(observations.WebhookDeliveries))
	}
	webhookDelivery := observations.WebhookDeliveries[0]
	if webhookDelivery.EventID != "evt_payment_failed_001" {
		t.Fatalf("webhook event_id = %q", webhookDelivery.EventID)
	}
	if webhookDelivery.EndpointName != "payment-events" {
		t.Fatalf("webhook endpoint_name = %q", webhookDelivery.EndpointName)
	}
	if webhookDelivery.Method != http.MethodPost {
		t.Fatalf("webhook method = %q", webhookDelivery.Method)
	}
	if webhookDelivery.Outcome != "response_received" {
		t.Fatalf("webhook outcome = %q", webhookDelivery.Outcome)
	}
	if webhookDelivery.StatusCode != http.StatusNoContent {
		t.Fatalf("webhook status_code = %d", webhookDelivery.StatusCode)
	}
}

func TestMCPSendWebhookEventRejectsUnknownEndpointWithoutDelivery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	received := make(chan struct{}, 1)
	webhookClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		received <- struct{}{}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}

	store := state.New()
	endpoints := webhook.NewEndpoints(webhook.Endpoint{Name: "payment-events", Address: "http://application.test/webhooks/payments"})
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithWebhookSender(store, nil, webhook.NewSender(endpoints, webhookClient))))

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "send_webhook_event",
		Arguments: map[string]any{
			"event_id":         "evt_payment_failed_001",
			"endpoint_name":    "unknown-events",
			"endpoint_address": "http://application.test/webhooks/payments",
			"request": map[string]any{
				"body": map[string]any{"type": "payment.failed"},
			},
		},
	})
	if err == nil && (result == nil || !result.IsError) {
		t.Fatal("send_webhook_event succeeded, want unknown endpoint rejection")
	}

	select {
	case <-received:
		t.Fatal("application webhook endpoint received a request for unknown endpoint")
	default:
	}
}

func TestMCPResetClearsWebhookDeliveryObservations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	webhookClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}

	store := state.New()
	endpoints := webhook.NewEndpoints(webhook.Endpoint{Name: "payment-events", Address: "http://application.test/webhooks/payments"})
	clientSession := connectMCPTestClient(t, ctx, NewMCPServer(NewWithWebhookSender(store, nil, webhook.NewSender(endpoints, webhookClient))))

	_, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "send_webhook_event",
		Arguments: map[string]any{
			"event_id":      "evt_payment_failed_001",
			"endpoint_name": "payment-events",
			"request": map[string]any{
				"body": map[string]any{"type": "payment.failed"},
			},
		},
	})
	if err != nil {
		t.Fatalf("send_webhook_event CallTool() error = %v", err)
	}

	resetResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "reset",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("reset CallTool() error = %v", err)
	}
	assertToolSuccess(t, resetResult)

	observationsResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_observations",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("get_observations CallTool() error = %v", err)
	}
	assertToolSuccess(t, observationsResult)
	observations := decodeStructuredContent[getObservationsOutput](t, observationsResult)
	if len(observations.WebhookDeliveries) != 0 {
		t.Fatalf("len(webhook_deliveries) = %d, want 0", len(observations.WebhookDeliveries))
	}
}

func connectMCPTestClient(t *testing.T, ctx context.Context, server *mcp.Server) *mcp.ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server Connect() error = %v", err)
	}
	t.Cleanup(func() {
		if err := serverSession.Close(); err != nil {
			t.Fatalf("server session Close() error = %v", err)
		}
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "echo-mcp-test-client", Version: "v0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect() error = %v", err)
	}
	t.Cleanup(func() {
		if err := clientSession.Close(); err != nil {
			t.Fatalf("client session Close() error = %v", err)
		}
	})

	return clientSession
}

func connectMCPTestClientWithPaymentIntentContractRoot(t *testing.T, ctx context.Context, store *state.Store) (*mcp.ClientSession, string) {
	t.Helper()
	root := t.TempDir()
	sourcePath := filepath.Join("contracts", "payment-intent-openapi.json")
	copyMCPContractFixture(t, root, sourcePath)
	return connectMCPTestClientWithContractRoot(t, ctx, store, root)
}

func connectMCPTestClientWithContractRoot(t *testing.T, ctx context.Context, store *state.Store, root string) (*mcp.ClientSession, string) {
	t.Helper()
	manager, err := NewContractManagerWithContractRoot(root, ContractRootSourceEnv)
	if err != nil {
		t.Fatalf("NewContractManagerWithContractRoot() error = %v", err)
	}
	return connectMCPTestClient(t, ctx, NewMCPServer(NewWithContractManager(store, manager, nil))), filepath.Join("contracts", "payment-intent-openapi.json")
}

func copyMCPContractFixture(t *testing.T, root string, relativePath string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "contract", "testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("ReadFile() fixture error = %v", err)
	}
	fullPath := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(fullPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func assertToolSuccess(t *testing.T, result *mcp.CallToolResult) {
	t.Helper()

	if result == nil {
		t.Fatal("tool result = nil")
	}
	if result.IsError {
		t.Fatalf("tool result IsError = true; content = %+v", result.Content)
	}
}

func assertStructuredToolError(t *testing.T, result *mcp.CallToolResult) toolErrorOutput {
	t.Helper()

	if result == nil {
		t.Fatal("tool result = nil")
	}
	if !result.IsError {
		t.Fatalf("tool result IsError = false, want true; content = %+v", result.Content)
	}
	return decodeStructuredContent[toolErrorOutput](t, result)
}

func assertPartialValidationDisclosure(t *testing.T, scope string, capabilities *validationCapabilitiesOutput, modeDescription string) {
	t.Helper()

	if scope != "partial" {
		t.Fatalf("validation_scope = %q, want partial", scope)
	}
	if capabilities == nil {
		t.Fatal("validation_capabilities = nil, want partial capability map")
	}
	if !capabilities.MethodPath {
		t.Fatal("validation_capabilities.method_path = false, want true")
	}
	if !capabilities.ResponseStatus {
		t.Fatal("validation_capabilities.response_status = false, want true")
	}
	if !capabilities.ResponseContentType {
		t.Fatal("validation_capabilities.response_content_type = false, want true")
	}
	if !capabilities.ResponseBody {
		t.Fatal("validation_capabilities.response_body = false, want true")
	}
	if !capabilities.InlineJSONResponseSchema {
		t.Fatal("validation_capabilities.inline_json_response_schema = false, want true")
	}
	if !capabilities.RefResolution {
		t.Fatal("validation_capabilities.ref_resolution = false, want true")
	}
	if !capabilities.LocalRefResolution {
		t.Fatal("validation_capabilities.local_ref_resolution = false, want true")
	}
	if !capabilities.Arrays {
		t.Fatal("validation_capabilities.arrays = false, want true")
	}
	if !capabilities.Enum {
		t.Fatal("validation_capabilities.enum = false, want true")
	}
	if !capabilities.Nullable {
		t.Fatal("validation_capabilities.nullable = false, want true")
	}
	if !capabilities.AdditionalProperties {
		t.Fatal("validation_capabilities.additional_properties = false, want true")
	}
	for name, value := range map[string]bool{
		"request_body":                 capabilities.RequestBody,
		"request_query":                capabilities.RequestQuery,
		"request_headers":              capabilities.RequestHeaders,
		"path_parameter_schema":        capabilities.PathParameterSchema,
		"remote_ref_resolution":        capabilities.RemoteRefResolution,
		"allOf":                        capabilities.AllOf,
		"oneOf":                        capabilities.OneOf,
		"anyOf":                        capabilities.AnyOf,
		"additional_properties_schema": capabilities.AdditionalPropertiesSchema,
		"openapi_3_1":                  capabilities.OpenAPI31,
		"yaml":                         capabilities.YAML,
		"remote_fetch":                 capabilities.RemoteFetch,
	} {
		if value {
			t.Fatalf("validation_capabilities.%s = true, want false", name)
		}
	}
	if !strings.Contains(modeDescription, "strict means strict enforcement of the validation capabilities currently supported by Echo MCP") {
		t.Fatalf("validation_mode_description missing strict subset wording: %q", modeDescription)
	}
	if !strings.Contains(modeDescription, "not full OpenAPI validation") {
		t.Fatalf("validation_mode_description missing full-validation caveat: %q", modeDescription)
	}
}

func writeUnsupportedFeatureOpenAPI(t *testing.T, root string) string {
	t.Helper()

	sourcePath := filepath.Join(root, "unsupported-feature-openapi.json")
	document := `{
  "openapi": "3.0.3",
  "paths": {
    "/v1/payment_intents": {
      "post": {
        "responses": {
          "200": {
            "description": "PaymentIntent",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/payment_intent"
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "payment_intent": {
        "oneOf": [
          {
            "type": "object",
            "properties": {
              "id": {"type": "string"},
              "object": {"type": "string"},
              "status": {"type": "string"},
              "charges": {
                "type": "array",
                "items": {"type": "object"}
              }
            }
          }
        ]
      }
    }
  }
}`
	if err := os.WriteFile(sourcePath, []byte(document), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return sourcePath
}

func loadStrictContract(t *testing.T, ctx context.Context, clientSession *mcp.ClientSession, sourcePath string) string {
	t.Helper()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "load_openapi_contract",
		Arguments: map[string]any{
			"path":            sourcePath,
			"contract_id":     "stripe",
			"validation_mode": "strict",
		},
	})
	if err != nil {
		t.Fatalf("load_openapi_contract CallTool() error = %v", err)
	}
	assertToolSuccess(t, result)
	return sourcePath
}

func configureValidPaymentIntentBehavior(t *testing.T, ctx context.Context, clientSession *mcp.ClientSession) {
	t.Helper()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "configure_behavior",
		Arguments: map[string]any{
			"behavior_id": "stripe-like-paymentintent-card-declined",
			"match": map[string]any{
				"method": http.MethodPost,
				"path":   "/v1/payment_intents/pi_123/confirm",
			},
			"outcome": map[string]any{
				"type":         "http_response",
				"status_code":  http.StatusPaymentRequired,
				"content_type": "application/json",
				"body":         paymentIntentDeclinedBody,
			},
		},
	})
	if err != nil {
		t.Fatalf("configure_behavior CallTool() error = %v", err)
	}
	assertToolSuccess(t, result)
}

func decodeStructuredContent[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()

	if result.StructuredContent == nil {
		t.Fatal("StructuredContent = nil")
	}
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("Marshal(StructuredContent) error = %v", err)
	}
	var decoded T
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal(StructuredContent) error = %v", err)
	}
	return decoded
}

func decodeStructuredMap(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()

	if result.StructuredContent == nil {
		t.Fatal("StructuredContent = nil")
	}
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("Marshal(StructuredContent) error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal(StructuredContent) error = %v", err)
	}
	return decoded
}

func assertStringSliceContains(t *testing.T, structured map[string]any, key string, want string) {
	t.Helper()

	values := stringSliceField(t, structured, key)
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("%s = %+v, want value %q", key, values, want)
}

func assertStringSliceContainsSubstring(t *testing.T, structured map[string]any, key string, want string) {
	t.Helper()

	values := stringSliceField(t, structured, key)
	for _, value := range values {
		if strings.Contains(value, want) {
			return
		}
	}
	t.Fatalf("%s = %+v, want substring %q", key, values, want)
}

func assertStringSliceNotContains(t *testing.T, structured map[string]any, key string, unwanted string) {
	t.Helper()

	if _, ok := structured[key]; !ok {
		return
	}
	values := stringSliceField(t, structured, key)
	for _, value := range values {
		if strings.Contains(value, unwanted) {
			t.Fatalf("%s = %+v, contains unwanted value %q", key, values, unwanted)
		}
	}
}

func stringSliceField(t *testing.T, structured map[string]any, key string) []string {
	t.Helper()

	raw, ok := structured[key]
	if !ok {
		t.Fatalf("StructuredContent missing %q: %+v", key, structured)
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("StructuredContent[%q] = %T, want []any", key, raw)
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		value, ok := item.(string)
		if !ok {
			t.Fatalf("StructuredContent[%q] contains %T, want string", key, item)
		}
		values = append(values, value)
	}
	return values
}

func toolsByName(tools []*mcp.Tool) map[string]*mcp.Tool {
	byName := make(map[string]*mcp.Tool, len(tools))
	for _, tool := range tools {
		byName[tool.Name] = tool
	}
	return byName
}

func requireTool(t *testing.T, tools map[string]*mcp.Tool, name string) *mcp.Tool {
	t.Helper()

	tool, ok := tools[name]
	if !ok {
		t.Fatalf("tool %q missing", name)
	}
	return tool
}

func toolNames(tools []*mcp.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}

func assertToolAnnotation(t *testing.T, tool *mcp.Tool, title string, readOnly bool, destructive *bool, idempotent bool, openWorld *bool) {
	t.Helper()

	if tool.Annotations == nil {
		t.Fatalf("%s annotations = nil", tool.Name)
	}
	if tool.Annotations.Title != title {
		t.Fatalf("%s annotations.title = %q, want %q", tool.Name, tool.Annotations.Title, title)
	}
	if tool.Annotations.ReadOnlyHint != readOnly {
		t.Fatalf("%s annotations.readOnlyHint = %v, want %v", tool.Name, tool.Annotations.ReadOnlyHint, readOnly)
	}
	assertOptionalBool(t, tool.Name, "destructiveHint", tool.Annotations.DestructiveHint, destructive)
	if tool.Annotations.IdempotentHint != idempotent {
		t.Fatalf("%s annotations.idempotentHint = %v, want %v", tool.Name, tool.Annotations.IdempotentHint, idempotent)
	}
	assertOptionalBool(t, tool.Name, "openWorldHint", tool.Annotations.OpenWorldHint, openWorld)
}

func assertOptionalBool(t *testing.T, toolName string, field string, got *bool, want *bool) {
	t.Helper()

	if want == nil {
		if got != nil {
			t.Fatalf("%s annotations.%s = %v, want nil", toolName, field, *got)
		}
		return
	}
	if got == nil {
		t.Fatalf("%s annotations.%s = nil, want %v", toolName, field, *want)
	}
	if *got != *want {
		t.Fatalf("%s annotations.%s = %v, want %v", toolName, field, *got, *want)
	}
}

func ptrBool(value bool) *bool {
	return &value
}

func promptsByName(prompts []*mcp.Prompt) map[string]*mcp.Prompt {
	byName := make(map[string]*mcp.Prompt, len(prompts))
	for _, prompt := range prompts {
		byName[prompt.Name] = prompt
	}
	return byName
}

func promptNames(prompts []*mcp.Prompt) []string {
	names := make([]string, 0, len(prompts))
	for _, prompt := range prompts {
		names = append(names, prompt.Name)
	}
	return names
}

func promptText(t *testing.T, result *mcp.GetPromptResult) string {
	t.Helper()

	if result == nil {
		t.Fatal("GetPromptResult = nil")
	}
	var builder strings.Builder
	for _, message := range result.Messages {
		content, ok := message.Content.(*mcp.TextContent)
		if !ok {
			t.Fatalf("prompt content = %T, want *mcp.TextContent", message.Content)
		}
		builder.WriteString(content.Text)
		builder.WriteString("\n")
	}
	return builder.String()
}

func resourcesByURI(resources []*mcp.Resource) map[string]*mcp.Resource {
	byURI := make(map[string]*mcp.Resource, len(resources))
	for _, resource := range resources {
		byURI[resource.URI] = resource
	}
	return byURI
}

func resourceURIs(resources []*mcp.Resource) []string {
	uris := make([]string, 0, len(resources))
	for _, resource := range resources {
		uris = append(uris, resource.URI)
	}
	return uris
}

func resourceText(t *testing.T, result *mcp.ReadResourceResult) string {
	t.Helper()

	if result == nil {
		t.Fatal("ReadResourceResult = nil")
	}
	var builder strings.Builder
	for _, content := range result.Contents {
		if content.MIMEType != "text/markdown" {
			t.Fatalf("resource MIMEType = %q, want text/markdown", content.MIMEType)
		}
		builder.WriteString(content.Text)
		builder.WriteString("\n")
	}
	return builder.String()
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

type webhookRequest struct {
	Method      string
	ContentType string
	Body        map[string]any
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
