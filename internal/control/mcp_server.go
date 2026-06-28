package control

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"echo-mcp/internal/contract"
	"echo-mcp/internal/state"
	"echo-mcp/internal/webhook"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	outcomeHTTPResponse = "http_response"
)

// NewMCPServer creates the MVP MCP control-plane server.
func NewMCPServer(plane Plane) *mcp.Server {
	if plane == nil {
		plane = New(nil)
	}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "echo-mcp", Version: "v0.3.0"},
		&mcp.ServerOptions{Instructions: serverInstructions},
	)
	registerGuidancePrompts(server)
	registerGuidanceResources(server)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "configure_behavior",
		Description: configureBehaviorDescription,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Configure REST Behavior",
			ReadOnlyHint:    false,
			DestructiveHint: boolPtr(false),
			IdempotentHint:  false,
			OpenWorldHint:   boolPtr(false),
		},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input configureBehaviorInput) (*mcp.CallToolResult, configureBehaviorOutput, error) {
		if err := validateConfigureBehavior(input); err != nil {
			return nil, configureBehaviorOutput{}, err
		}

		validationWarnings, err := plane.ConfigureResponseRuleWithValidation(state.ResponseRule{
			ID:          input.BehaviorID,
			Method:      input.Match.Method,
			Path:        input.Match.Path,
			StatusCode:  input.Outcome.StatusCode,
			ContentType: input.Outcome.ContentType,
			Body:        input.Outcome.Body,
		}, configureBehaviorValidationOverride(input.Validation))
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, configureBehaviorOutputFromError(err, plane.ContractStatus()), nil
		}

		warnings := append([]string{}, validationWarnings...)
		warnings = append(warnings, configureBehaviorWarnings(plane)...)

		output := configureBehaviorOutput{
			Configured:           true,
			BehaviorID:           input.BehaviorID,
			Warnings:             warnings,
			Guidance:             configureBehaviorGuidance(plane),
			SuggestedNextActions: []string{"Run the application test normally.", "Call get_observations to inspect data-plane evidence."},
		}
		applyValidationDisclosure(&output, plane.ContractStatus())
		return nil, output, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "load_openapi_contract",
		Description: loadOpenAPIContractDescription,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Load OpenAPI Contract",
			ReadOnlyHint:    false,
			DestructiveHint: boolPtr(false),
			IdempotentHint:  false,
			OpenWorldHint:   boolPtr(false),
		},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input loadOpenAPIContractInput) (*mcp.CallToolResult, loadOpenAPIContractOutput, error) {
		result, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{
			Path:           input.Path,
			ContractID:     input.ContractID,
			ValidationMode: ValidationMode(input.ValidationMode),
			Force:          input.Force,
		})
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, loadOpenAPIContractOutputFromError(err), nil
		}

		return nil, loadOpenAPIContractOutput{
			Loaded:                    result.Loaded,
			ContractID:                result.ContractID,
			SourcePath:                result.SourcePath,
			OpenAPIVersion:            result.OpenAPIVersion,
			OperationsCount:           result.OperationsCount,
			SchemasCount:              result.SchemasCount,
			ValidationMode:            string(result.ValidationMode),
			ValidationScope:           result.ValidationScope,
			ValidationCapabilities:    validationCapabilitiesOutputFromDomain(result.ValidationCapabilities),
			ValidationModeDescription: result.ValidationModeDescription,
			UnsupportedFeatures:       result.UnsupportedFeatures,
			Warnings:                  emptyStringSliceIfNil(result.Warnings),
			SuggestedNextActions:      result.SuggestedNextActions,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_contract_status",
		Description: getContractStatusDescription,
		Annotations: &mcp.ToolAnnotations{
			Title:          "Get Contract Status",
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  boolPtr(false),
		},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getContractStatusInput) (*mcp.CallToolResult, getContractStatusOutput, error) {
		status := plane.ContractStatus()
		return nil, contractStatusOutput(status), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "unload_openapi_contract",
		Description: unloadOpenAPIContractDescription,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Unload OpenAPI Contract",
			ReadOnlyHint:    false,
			DestructiveHint: boolPtr(false),
			IdempotentHint:  false,
			OpenWorldHint:   boolPtr(false),
		},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input unloadOpenAPIContractInput) (*mcp.CallToolResult, unloadOpenAPIContractOutput, error) {
		result, err := plane.UnloadOpenAPIContract(UnloadOpenAPIContractCommand{Force: input.Force})
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, unloadOpenAPIContractOutputFromError(err), nil
		}

		return nil, unloadOpenAPIContractOutput{
			Unloaded:             result.Unloaded,
			PreviousContractID:   result.PreviousContractID,
			SuggestedNextActions: result.SuggestedNextActions,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "reset",
		Description: resetDescription,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Reset Echo MCP State",
			ReadOnlyHint:    false,
			DestructiveHint: boolPtr(true),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
		},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input resetInput) (*mcp.CallToolResult, resetOutput, error) {
		if err := plane.Reset(); err != nil {
			return nil, resetOutput{}, err
		}

		status := plane.ContractStatus()
		return nil, resetOutput{
			Reset:                true,
			Cleared:              []string{"behavior", "observations", "webhook_deliveries"},
			ContractActive:       status.Active,
			ContractID:           status.ContractID,
			SuggestedNextActions: []string{"Configure the next behavior or webhook scenario."},
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "send_webhook_event",
		Description: sendWebhookEventDescription,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Send Webhook Event",
			ReadOnlyHint:    false,
			DestructiveHint: boolPtr(false),
			IdempotentHint:  false,
			OpenWorldHint:   boolPtr(false),
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input sendWebhookEventInput) (*mcp.CallToolResult, sendWebhookEventOutput, error) {
		if err := validateSendWebhookEvent(input); err != nil {
			return nil, sendWebhookEventOutput{}, err
		}

		delivery, err := plane.SendWebhookEvent(ctx, webhook.Event{
			EventID:      input.EventID,
			EndpointName: input.EndpointName,
			Body:         input.Request.Body,
		})
		if err != nil {
			return nil, sendWebhookEventOutput{}, err
		}

		return nil, sendWebhookEventOutput{
			Attempted:    true,
			EventID:      delivery.EventID,
			EndpointName: delivery.EndpointName,
			Delivery: webhookDeliveryOutput{
				Outcome:    delivery.Outcome,
				StatusCode: delivery.StatusCode,
				Error:      delivery.Error,
			},
			Warnings:             webhookDeliveryWarnings(delivery.Outcome),
			SuggestedNextActions: []string{"Assert application behavior normally.", "Call get_observations to inspect webhook delivery evidence."},
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_observations",
		Description: getObservationsDescription,
		Annotations: &mcp.ToolAnnotations{
			Title:          "Get Observations",
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  boolPtr(false),
		},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getObservationsInput) (*mcp.CallToolResult, getObservationsOutput, error) {
		observations := plane.Observations()
		output := getObservationsOutput{
			Observations:      make([]observationOutput, 0, len(observations)),
			WebhookDeliveries: make([]webhookDeliveryObservationOutput, 0, len(plane.WebhookDeliveryObservations())),
			Guidance:          observationGuidance(len(observations), len(plane.WebhookDeliveryObservations())),
		}

		for _, observation := range observations {
			output.Observations = append(output.Observations, observationOutput{
				Request: requestObservation{
					Method: observation.RequestMethod,
					Path:   observation.RequestPath,
				},
				Selection: selectionObservation{
					MatchedBehaviorID: observation.MatchedRuleID,
					MatchedOn:         []string{"method", "path"},
				},
				Outcome: outcomeObservation{
					Type:       outcomeHTTPResponse,
					StatusCode: observation.OutcomeStatusCode,
				},
			})
		}

		for _, delivery := range plane.WebhookDeliveryObservations() {
			output.WebhookDeliveries = append(output.WebhookDeliveries, webhookDeliveryObservationOutput{
				EventID:      delivery.EventID,
				EndpointName: delivery.EndpointName,
				Method:       delivery.Method,
				Outcome:      delivery.Outcome,
				StatusCode:   delivery.StatusCode,
				Error:        delivery.Error,
			})
		}

		return nil, output, nil
	})

	return server
}

type configureBehaviorInput struct {
	BehaviorID string                      `json:"behavior_id" jsonschema:"behavior rule identifier used in observations"`
	Match      configureBehaviorMatch      `json:"match" jsonschema:"data-plane request match criteria"`
	Outcome    configureBehaviorOutcome    `json:"outcome" jsonschema:"simulated outcome for the matching request"`
	Validation configureBehaviorValidation `json:"validation,omitempty" jsonschema:"optional per-behavior contract validation override"`
}

type configureBehaviorMatch struct {
	Method string `json:"method" jsonschema:"HTTP method for the incoming data-plane request"`
	Path   string `json:"path" jsonschema:"HTTP path for the incoming data-plane request"`
}

type configureBehaviorOutcome struct {
	Type        string `json:"type" jsonschema:"outcome type; only http_response is supported in the first slice"`
	StatusCode  int    `json:"status_code" jsonschema:"HTTP response status code"`
	ContentType string `json:"content_type,omitempty" jsonschema:"HTTP response content type"`
	Body        string `json:"body" jsonschema:"HTTP response body"`
}

type configureBehaviorValidation struct {
	Mode   string `json:"mode,omitempty" jsonschema:"validation mode override; strict, warn, or off"`
	Reason string `json:"reason,omitempty" jsonschema:"required when mode is off while a contract is active"`
}

type configureBehaviorOutput struct {
	Configured                bool                          `json:"configured"`
	BehaviorID                string                        `json:"behavior_id"`
	Warnings                  []string                      `json:"warnings,omitempty"`
	Guidance                  []string                      `json:"guidance,omitempty"`
	SuggestedNextActions      []string                      `json:"suggested_next_actions,omitempty"`
	ValidationScope           string                        `json:"validation_scope,omitempty"`
	ValidationCapabilities    *validationCapabilitiesOutput `json:"validation_capabilities,omitempty"`
	ValidationModeDescription string                        `json:"validation_mode_description,omitempty"`
	Error                     string                        `json:"error,omitempty"`
	Code                      string                        `json:"code,omitempty"`
	Diagnostics               []string                      `json:"diagnostics,omitempty"`
}

type loadOpenAPIContractInput struct {
	Path           string `json:"path" jsonschema:"local filesystem path to an OpenAPI 3.0.x JSON contract"`
	ContractID     string `json:"contract_id,omitempty" jsonschema:"optional caller-provided active contract identifier"`
	ValidationMode string `json:"validation_mode,omitempty" jsonschema:"strict, warn, or off; defaults to strict"`
	Force          bool   `json:"force,omitempty" jsonschema:"allow replacing the active contract even when behavior is configured"`
}

type loadOpenAPIContractOutput struct {
	Loaded                    bool                          `json:"loaded"`
	ContractID                string                        `json:"contract_id,omitempty"`
	SourcePath                string                        `json:"source_path,omitempty"`
	OpenAPIVersion            string                        `json:"openapi_version,omitempty"`
	OperationsCount           int                           `json:"operations_count,omitempty"`
	SchemasCount              int                           `json:"schemas_count,omitempty"`
	ValidationMode            string                        `json:"validation_mode,omitempty"`
	ValidationScope           string                        `json:"validation_scope,omitempty"`
	ValidationCapabilities    *validationCapabilitiesOutput `json:"validation_capabilities,omitempty"`
	ValidationModeDescription string                        `json:"validation_mode_description,omitempty"`
	UnsupportedFeatures       map[string]int                `json:"unsupported_features,omitempty"`
	Warnings                  []string                      `json:"warnings"`
	SuggestedNextActions      []string                      `json:"suggested_next_actions"`
	Error                     string                        `json:"error,omitempty"`
	Code                      string                        `json:"code,omitempty"`
	Diagnostics               []string                      `json:"diagnostics,omitempty"`
}

type getContractStatusInput struct{}

type getContractStatusOutput struct {
	Active                    bool                          `json:"active"`
	Message                   string                        `json:"message,omitempty"`
	ContractID                string                        `json:"contract_id,omitempty"`
	SourcePath                string                        `json:"source_path,omitempty"`
	OpenAPIVersion            string                        `json:"openapi_version,omitempty"`
	OperationsCount           int                           `json:"operations_count,omitempty"`
	SchemasCount              int                           `json:"schemas_count,omitempty"`
	LoadedAt                  string                        `json:"loaded_at,omitempty"`
	ValidationMode            string                        `json:"validation_mode,omitempty"`
	ValidationScope           string                        `json:"validation_scope,omitempty"`
	ValidationCapabilities    *validationCapabilitiesOutput `json:"validation_capabilities,omitempty"`
	ValidationModeDescription string                        `json:"validation_mode_description,omitempty"`
	UnsupportedFeatures       map[string]int                `json:"unsupported_features,omitempty"`
	ContractRootConfigured    bool                          `json:"contract_root_configured"`
	ContractRootSource        string                        `json:"contract_root_source,omitempty"`
}

type unloadOpenAPIContractInput struct {
	Force bool `json:"force,omitempty" jsonschema:"force unload even when configured behavior is active"`
}

type unloadOpenAPIContractOutput struct {
	Unloaded             bool     `json:"unloaded"`
	PreviousContractID   string   `json:"previous_contract_id,omitempty"`
	SuggestedNextActions []string `json:"suggested_next_actions,omitempty"`
	Error                string   `json:"error,omitempty"`
	Code                 string   `json:"code,omitempty"`
	Diagnostics          []string `json:"diagnostics,omitempty"`
}

type toolErrorOutput struct {
	Error                     string                        `json:"error"`
	Code                      string                        `json:"code"`
	Diagnostics               []string                      `json:"diagnostics,omitempty"`
	ValidationScope           string                        `json:"validation_scope,omitempty"`
	ValidationCapabilities    *validationCapabilitiesOutput `json:"validation_capabilities,omitempty"`
	ValidationModeDescription string                        `json:"validation_mode_description,omitempty"`
}

type validationCapabilitiesOutput struct {
	MethodPath                 bool `json:"method_path"`
	ResponseStatus             bool `json:"response_status"`
	ResponseContentType        bool `json:"response_content_type"`
	ResponseBody               bool `json:"response_body"`
	InlineJSONResponseSchema   bool `json:"inline_json_response_schema"`
	RequestBody                bool `json:"request_body"`
	RequestQuery               bool `json:"request_query"`
	RequestHeaders             bool `json:"request_headers"`
	PathParameterSchema        bool `json:"path_parameter_schema"`
	RefResolution              bool `json:"ref_resolution"`
	LocalRefResolution         bool `json:"local_ref_resolution"`
	RemoteRefResolution        bool `json:"remote_ref_resolution"`
	Arrays                     bool `json:"arrays"`
	Enum                       bool `json:"enum"`
	AllOf                      bool `json:"allOf"`
	OneOf                      bool `json:"oneOf"`
	AnyOf                      bool `json:"anyOf"`
	Nullable                   bool `json:"nullable"`
	AdditionalProperties       bool `json:"additional_properties"`
	AdditionalPropertiesSchema bool `json:"additional_properties_schema"`
	OpenAPI31                  bool `json:"openapi_3_1"`
	YAML                       bool `json:"yaml"`
	RemoteFetch                bool `json:"remote_fetch"`
}

type resetInput struct{}

type resetOutput struct {
	Reset                bool     `json:"reset"`
	Cleared              []string `json:"cleared"`
	ContractActive       bool     `json:"contract_active"`
	ContractID           string   `json:"contract_id,omitempty"`
	SuggestedNextActions []string `json:"suggested_next_actions,omitempty"`
}

type sendWebhookEventInput struct {
	EventID      string                  `json:"event_id" jsonschema:"webhook event identifier used in observations"`
	EndpointName string                  `json:"endpoint_name" jsonschema:"configured application webhook endpoint name"`
	Request      sendWebhookEventRequest `json:"request" jsonschema:"webhook request payload"`
}

type sendWebhookEventRequest struct {
	Body map[string]any `json:"body" jsonschema:"JSON request body sent to the application webhook endpoint"`
}

type sendWebhookEventOutput struct {
	Attempted            bool                  `json:"attempted"`
	EventID              string                `json:"event_id"`
	EndpointName         string                `json:"endpoint_name"`
	Delivery             webhookDeliveryOutput `json:"delivery"`
	Warnings             []string              `json:"warnings,omitempty"`
	SuggestedNextActions []string              `json:"suggested_next_actions,omitempty"`
}

type webhookDeliveryOutput struct {
	Outcome    string `json:"outcome"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

type getObservationsInput struct{}

type getObservationsOutput struct {
	Observations      []observationOutput                `json:"observations"`
	WebhookDeliveries []webhookDeliveryObservationOutput `json:"webhook_deliveries"`
	Guidance          []string                           `json:"guidance,omitempty"`
}

type observationOutput struct {
	Request   requestObservation   `json:"request"`
	Selection selectionObservation `json:"selection"`
	Outcome   outcomeObservation   `json:"outcome"`
}

type requestObservation struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type selectionObservation struct {
	MatchedBehaviorID string   `json:"matched_behavior_id"`
	MatchedOn         []string `json:"matched_on"`
}

type outcomeObservation struct {
	Type       string `json:"type"`
	StatusCode int    `json:"status_code"`
}

type webhookDeliveryObservationOutput struct {
	EventID      string `json:"event_id"`
	EndpointName string `json:"endpoint_name"`
	Method       string `json:"method"`
	Outcome      string `json:"outcome"`
	StatusCode   int    `json:"status_code,omitempty"`
	Error        string `json:"error,omitempty"`
}

func validateConfigureBehavior(input configureBehaviorInput) error {
	if strings.TrimSpace(input.BehaviorID) == "" {
		return fmt.Errorf("behavior_id is required")
	}
	if strings.TrimSpace(input.Match.Method) == "" {
		return fmt.Errorf("match.method is required")
	}
	if strings.TrimSpace(input.Match.Path) == "" {
		return fmt.Errorf("match.path is required")
	}
	if input.Outcome.Type != outcomeHTTPResponse {
		return fmt.Errorf("outcome.type must be %q", outcomeHTTPResponse)
	}
	if input.Outcome.StatusCode < 100 || input.Outcome.StatusCode > 999 {
		return fmt.Errorf("outcome.status_code must be a valid HTTP status code")
	}

	return nil
}

func configureBehaviorValidationOverride(input configureBehaviorValidation) BehaviorValidationOverride {
	mode := strings.TrimSpace(input.Mode)
	return BehaviorValidationOverride{
		Mode:    ValidationMode(mode),
		Reason:  strings.TrimSpace(input.Reason),
		ModeSet: mode != "",
	}
}

func contractStatusOutput(status ContractStatus) getContractStatusOutput {
	if !status.Active {
		return getContractStatusOutput{
			Active:                 false,
			Message:                status.Message,
			ContractRootConfigured: status.ContractRootConfigured,
			ContractRootSource:     status.ContractRootSource,
		}
	}

	loadedAt := ""
	if !status.LoadedAt.IsZero() {
		loadedAt = status.LoadedAt.Format(time.RFC3339Nano)
	}
	return getContractStatusOutput{
		Active:                    true,
		ContractID:                status.ContractID,
		SourcePath:                status.SourcePath,
		OpenAPIVersion:            status.OpenAPIVersion,
		OperationsCount:           status.OperationsCount,
		SchemasCount:              status.SchemasCount,
		LoadedAt:                  loadedAt,
		ValidationMode:            string(status.ValidationMode),
		ValidationScope:           status.ValidationScope,
		ValidationCapabilities:    validationCapabilitiesOutputFromDomain(status.ValidationCapabilities),
		ValidationModeDescription: status.ValidationModeDescription,
		UnsupportedFeatures:       status.UnsupportedFeatures,
		ContractRootConfigured:    status.ContractRootConfigured,
		ContractRootSource:        status.ContractRootSource,
	}
}

func applyValidationDisclosure(output *configureBehaviorOutput, status ContractStatus) {
	if output == nil || !status.Active {
		return
	}
	output.ValidationScope = status.ValidationScope
	output.ValidationCapabilities = validationCapabilitiesOutputFromDomain(status.ValidationCapabilities)
	output.ValidationModeDescription = status.ValidationModeDescription
}

func validationCapabilitiesOutputFromDomain(capabilities ValidationCapabilities) *validationCapabilitiesOutput {
	return &validationCapabilitiesOutput{
		MethodPath:                 capabilities.MethodPath,
		ResponseStatus:             capabilities.ResponseStatus,
		ResponseContentType:        capabilities.ResponseContentType,
		ResponseBody:               capabilities.ResponseBody,
		InlineJSONResponseSchema:   capabilities.InlineJSONResponseSchema,
		RequestBody:                capabilities.RequestBody,
		RequestQuery:               capabilities.RequestQuery,
		RequestHeaders:             capabilities.RequestHeaders,
		PathParameterSchema:        capabilities.PathParameterSchema,
		RefResolution:              capabilities.RefResolution,
		LocalRefResolution:         capabilities.LocalRefResolution,
		RemoteRefResolution:        capabilities.RemoteRefResolution,
		Arrays:                     capabilities.Arrays,
		Enum:                       capabilities.Enum,
		AllOf:                      capabilities.AllOf,
		OneOf:                      capabilities.OneOf,
		AnyOf:                      capabilities.AnyOf,
		Nullable:                   capabilities.Nullable,
		AdditionalProperties:       capabilities.AdditionalProperties,
		AdditionalPropertiesSchema: capabilities.AdditionalPropertiesSchema,
		OpenAPI31:                  capabilities.OpenAPI31,
		YAML:                       capabilities.YAML,
		RemoteFetch:                capabilities.RemoteFetch,
	}
}

func toolErrorFromError(err error) toolErrorOutput {
	if err == nil {
		return toolErrorOutput{}
	}

	var operationErr *OperationError
	if errors.As(err, &operationErr) {
		return toolErrorOutput{
			Error:       operationErr.Message,
			Code:        operationErr.Code,
			Diagnostics: operationErr.Diagnostics,
		}
	}

	var loadErr *contract.LoadError
	if errors.As(err, &loadErr) {
		return toolErrorOutput{
			Error:       loadErr.Message,
			Code:        loadErr.Code,
			Diagnostics: loadErr.Diagnostics,
		}
	}

	return toolErrorOutput{
		Error:       err.Error(),
		Code:        "operation_failed",
		Diagnostics: []string{err.Error()},
	}
}

func emptyStringSliceIfNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func configureBehaviorOutputFromError(err error, status ContractStatus) configureBehaviorOutput {
	toolError := toolErrorFromError(err)
	output := configureBehaviorOutput{
		Error:       toolError.Error,
		Code:        toolError.Code,
		Diagnostics: toolError.Diagnostics,
	}
	applyValidationDisclosure(&output, status)
	return output
}

func loadOpenAPIContractOutputFromError(err error) loadOpenAPIContractOutput {
	toolError := toolErrorFromError(err)
	return loadOpenAPIContractOutput{
		Error:       toolError.Error,
		Code:        toolError.Code,
		Diagnostics: toolError.Diagnostics,
	}
}

func unloadOpenAPIContractOutputFromError(err error) unloadOpenAPIContractOutput {
	toolError := toolErrorFromError(err)
	return unloadOpenAPIContractOutput{
		Error:       toolError.Error,
		Code:        toolError.Code,
		Diagnostics: toolError.Diagnostics,
	}
}

func validateSendWebhookEvent(input sendWebhookEventInput) error {
	if strings.TrimSpace(input.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(input.EndpointName) == "" {
		return fmt.Errorf("endpoint_name is required")
	}
	if input.Request.Body == nil {
		return fmt.Errorf("request.body is required")
	}

	return nil
}

type contractValidationReporter interface {
	ContractValidationActive() bool
}

func contractValidationActive(plane Plane) bool {
	reporter, ok := plane.(contractValidationReporter)
	return ok && reporter.ContractValidationActive()
}

func contractLoaded(plane Plane) bool {
	return plane.ContractStatus().Active
}

func configureBehaviorWarnings(plane Plane) []string {
	if contractLoaded(plane) {
		return nil
	}
	return []string{manualMockWarning}
}

func configureBehaviorGuidance(plane Plane) []string {
	if contractValidationActive(plane) {
		return []string{"Contract validation is active for configured REST behavior."}
	}
	if status := plane.ContractStatus(); status.Active {
		return []string{"An OpenAPI contract is active, but contract validation mode is off for configured REST behavior."}
	}
	return []string{"Manual mock behavior is active for configured REST behavior."}
}

func webhookDeliveryWarnings(outcome string) []string {
	if outcome == webhook.OutcomeTransportError {
		return []string{"Webhook delivery transport error. Inspect the configured application webhook endpoint and call get_observations for evidence."}
	}
	return nil
}

func observationGuidance(restObservationCount int, webhookDeliveryCount int) []string {
	if restObservationCount == 0 && webhookDeliveryCount == 0 {
		return []string{"No data-plane observations are available. Run the application test or send a webhook event before expecting observations."}
	}
	return []string{"Use observations as Echo MCP evidence and keep application behavior assertions in the application test."}
}
