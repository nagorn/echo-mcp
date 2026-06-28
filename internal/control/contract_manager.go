package control

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"echo-mcp/internal/contract"
	"echo-mcp/internal/state"
)

// ValidationMode controls how an active contract constrains configured behavior.
type ValidationMode string

const (
	ValidationModeStrict ValidationMode = "strict"
	ValidationModeWarn   ValidationMode = "warn"
	ValidationModeOff    ValidationMode = "off"

	ValidationScopePartial        = "partial"
	ValidationModeDescriptionText = "strict means strict enforcement of the validation capabilities currently supported by Echo MCP. It is not full OpenAPI validation."
)

// ValidationCapabilities describes the OpenAPI validation subset Echo MCP enforces today.
type ValidationCapabilities struct {
	MethodPath                 bool
	ResponseStatus             bool
	ResponseContentType        bool
	ResponseBody               bool
	InlineJSONResponseSchema   bool
	RequestBody                bool
	RequestQuery               bool
	RequestHeaders             bool
	PathParameterSchema        bool
	RefResolution              bool
	LocalRefResolution         bool
	RemoteRefResolution        bool
	Arrays                     bool
	Enum                       bool
	AllOf                      bool
	OneOf                      bool
	AnyOf                      bool
	Nullable                   bool
	AdditionalProperties       bool
	AdditionalPropertiesSchema bool
	OpenAPI31                  bool
	YAML                       bool
	RemoteFetch                bool
}

// OperationError is a structured control-plane operation failure.
type OperationError struct {
	Code        string
	Message     string
	Diagnostics []string
}

func (e *OperationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

// LoadOpenAPIContractCommand loads one local OpenAPI contract into active state.
type LoadOpenAPIContractCommand struct {
	Path           string
	ContractID     string
	ValidationMode ValidationMode
	Force          bool
}

// LoadOpenAPIContractResult reports a successful runtime contract load.
type LoadOpenAPIContractResult struct {
	Loaded                    bool
	ContractID                string
	SourcePath                string
	OpenAPIVersion            string
	OperationsCount           int
	SchemasCount              int
	ValidationMode            ValidationMode
	ValidationScope           string
	ValidationCapabilities    ValidationCapabilities
	ValidationModeDescription string
	UnsupportedFeatures       map[string]int
	Warnings                  []string
	SuggestedNextActions      []string
}

// ContractStatus reports the active OpenAPI contract state.
type ContractStatus struct {
	Active                    bool
	Message                   string
	ContractID                string
	SourcePath                string
	OpenAPIVersion            string
	OperationsCount           int
	SchemasCount              int
	LoadedAt                  time.Time
	ValidationMode            ValidationMode
	ValidationScope           string
	ValidationCapabilities    ValidationCapabilities
	ValidationModeDescription string
	UnsupportedFeatures       map[string]int
	ContractRootConfigured    bool
	ContractRootSource        string
}

// UnloadOpenAPIContractCommand clears active contract state.
type UnloadOpenAPIContractCommand struct {
	Force bool
}

// UnloadOpenAPIContractResult reports a successful runtime contract unload.
type UnloadOpenAPIContractResult struct {
	Unloaded             bool
	PreviousContractID   string
	SuggestedNextActions []string
}

// BehaviorValidationOverride optionally overrides active contract validation.
type BehaviorValidationOverride struct {
	Mode    ValidationMode
	Reason  string
	ModeSet bool
}

type activeOpenAPIContract struct {
	contract       *contract.OpenAPIContract
	contractID     string
	loadedAt       time.Time
	validationMode ValidationMode
}

// ContractManager owns the single active OpenAPI contract for the process.
type ContractManager struct {
	mu           sync.RWMutex
	active       *activeOpenAPIContract
	contractRoot contractRoot
}

// NewContractManager creates an empty active contract manager.
func NewContractManager() *ContractManager {
	manager, err := NewContractManagerWithContractRoot("", "")
	if err != nil {
		return &ContractManager{
			contractRoot: contractRoot{
				path:       ".",
				configured: false,
				source:     ContractRootSourceWorkingDirectory,
			},
		}
	}
	return manager
}

// NewContractManagerWithContractRoot creates an empty active contract manager with a contract root boundary.
func NewContractManagerWithContractRoot(path string, source string) (*ContractManager, error) {
	root, err := newContractRoot(path, source)
	if err != nil {
		return nil, err
	}
	return &ContractManager{contractRoot: root}, nil
}

// NewContractManagerWithContract creates a manager with one startup-loaded contract.
func NewContractManagerWithContract(contractID string, validator *contract.OpenAPIContract, mode ValidationMode, loadedAt time.Time) (*ContractManager, error) {
	manager := NewContractManager()
	if validator == nil {
		return manager, nil
	}
	if loadedAt.IsZero() {
		loadedAt = time.Now().UTC()
	}
	normalizedMode, err := normalizeValidationMode(mode)
	if err != nil {
		return nil, err
	}
	manager.active = &activeOpenAPIContract{
		contract:       validator,
		contractID:     defaultContractID(contractID, validator.Metadata().SourcePath),
		loadedAt:       loadedAt,
		validationMode: normalizedMode,
	}
	return manager, nil
}

// LoadOpenAPIContract loads and activates a local OpenAPI contract.
func (m *ContractManager) LoadOpenAPIContract(command LoadOpenAPIContractCommand, behaviorActive bool) (LoadOpenAPIContractResult, error) {
	if m == nil {
		m = NewContractManager()
	}
	m.ensureContractRoot()
	if strings.TrimSpace(command.Path) == "" {
		return LoadOpenAPIContractResult{}, &OperationError{
			Code:        "missing_path",
			Message:     "path is required",
			Diagnostics: []string{"load_openapi_contract requires a local filesystem path."},
		}
	}
	if behaviorActive && !command.Force {
		return LoadOpenAPIContractResult{}, resetRequiredError("reset before loading a different OpenAPI contract or pass force: true")
	}
	mode, err := normalizeValidationMode(command.ValidationMode)
	if err != nil {
		return LoadOpenAPIContractResult{}, err
	}

	resolvedPath, err := m.contractRoot.resolve(command.Path)
	if err != nil {
		return LoadOpenAPIContractResult{}, err
	}

	validator, err := contract.LoadOpenAPIFileWithSourcePath(resolvedPath.readPath, resolvedPath.sourcePath)
	if err != nil {
		return LoadOpenAPIContractResult{}, err
	}
	metadata := validator.Metadata()
	contractID := defaultContractID(command.ContractID, resolvedPath.sourcePath)

	m.mu.Lock()
	m.active = &activeOpenAPIContract{
		contract:       validator,
		contractID:     contractID,
		loadedAt:       time.Now().UTC(),
		validationMode: mode,
	}
	m.mu.Unlock()

	return LoadOpenAPIContractResult{
		Loaded:                    true,
		ContractID:                contractID,
		SourcePath:                metadata.SourcePath,
		OpenAPIVersion:            metadata.OpenAPIVersion,
		OperationsCount:           metadata.OperationsCount,
		SchemasCount:              metadata.SchemasCount,
		ValidationMode:            mode,
		ValidationScope:           ValidationScopePartial,
		ValidationCapabilities:    CurrentValidationCapabilities(),
		ValidationModeDescription: ValidationModeDescriptionText,
		UnsupportedFeatures:       copyUnsupportedFeatures(metadata.UnsupportedFeatureCounts),
		Warnings:                  unsupportedFeatureWarnings(metadata.UnsupportedFeatureCounts),
		SuggestedNextActions: []string{
			"Call get_contract_status.",
			"Call configure_behavior with contract-valid responses.",
		},
	}, nil
}

// Status returns active contract metadata or an inactive status.
func (m *ContractManager) Status() ContractStatus {
	if m == nil {
		return inactiveContractStatus()
	}
	m.ensureContractRoot()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.active == nil || m.active.contract == nil {
		return m.inactiveContractStatus()
	}

	metadata := m.active.contract.Metadata()
	return ContractStatus{
		Active:                    true,
		ContractID:                m.active.contractID,
		SourcePath:                metadata.SourcePath,
		OpenAPIVersion:            metadata.OpenAPIVersion,
		OperationsCount:           metadata.OperationsCount,
		SchemasCount:              metadata.SchemasCount,
		LoadedAt:                  m.active.loadedAt,
		ValidationMode:            m.active.validationMode,
		ValidationScope:           ValidationScopePartial,
		ValidationCapabilities:    CurrentValidationCapabilities(),
		ValidationModeDescription: ValidationModeDescriptionText,
		UnsupportedFeatures:       copyUnsupportedFeatures(metadata.UnsupportedFeatureCounts),
		ContractRootConfigured:    m.contractRoot.configured,
		ContractRootSource:        m.contractRoot.source,
	}
}

// UnloadOpenAPIContract clears the active contract without mutating source files.
func (m *ContractManager) UnloadOpenAPIContract(command UnloadOpenAPIContractCommand, behaviorActive bool) (UnloadOpenAPIContractResult, error) {
	if m == nil {
		return UnloadOpenAPIContractResult{Unloaded: false}, nil
	}
	if behaviorActive && !command.Force {
		return UnloadOpenAPIContractResult{}, resetRequiredError("reset before unloading the active OpenAPI contract or pass force: true")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	previousID := ""
	if m.active != nil {
		previousID = m.active.contractID
	}
	m.active = nil

	return UnloadOpenAPIContractResult{
		Unloaded:           true,
		PreviousContractID: previousID,
		SuggestedNextActions: []string{
			"Call configure_behavior for manual mock mode or load another contract.",
		},
	}, nil
}

// ValidateResponseRule validates a configured behavior against active contract state.
func (m *ContractManager) ValidateResponseRule(rule state.ResponseRule, override BehaviorValidationOverride) ([]string, error) {
	if m == nil {
		return nil, nil
	}

	m.mu.RLock()
	active := m.active
	m.mu.RUnlock()
	if active == nil || active.contract == nil {
		return nil, nil
	}

	mode := active.validationMode
	if override.ModeSet {
		normalizedMode, err := normalizeValidationMode(override.Mode)
		if err != nil {
			return nil, err
		}
		mode = normalizedMode
	}

	if mode == ValidationModeOff {
		if override.ModeSet && strings.TrimSpace(override.Reason) == "" {
			return nil, &OperationError{
				Code:        "validation_reason_required",
				Message:     "validation.reason is required when validation.mode is off and a contract is active",
				Diagnostics: []string{"Provide a non-empty reason for intentional contract validation skip."},
			}
		}
		if override.ModeSet {
			return []string{fmt.Sprintf("Contract validation was skipped for this behavior: %s.", strings.TrimSpace(override.Reason))}, nil
		}
		return []string{"Contract validation is off for the active OpenAPI contract."}, nil
	}

	if err := active.contract.ValidateResponseRule(rule); err != nil {
		if mode == ValidationModeWarn {
			return []string{"Contract validation warning: " + err.Error()}, nil
		}
		var validationErr *contract.ValidationError
		if errors.As(err, &validationErr) {
			return nil, &OperationError{
				Code:        validationErr.Code,
				Message:     validationErr.Message,
				Diagnostics: validationErr.Diagnostics,
			}
		}
		return nil, &OperationError{
			Code:        "contract_validation_failed",
			Message:     "Configured behavior violates the active OpenAPI contract",
			Diagnostics: []string{err.Error()},
		}
	}

	return nil, nil
}

func (m *ContractManager) ensureContractRoot() {
	if m == nil || m.contractRoot.path != "" {
		return
	}
	root, err := newContractRoot("", "")
	if err != nil {
		m.contractRoot = contractRoot{
			path:       ".",
			configured: false,
			source:     ContractRootSourceWorkingDirectory,
		}
		return
	}
	m.contractRoot = root
}

func inactiveContractStatus() ContractStatus {
	return ContractStatus{
		Active:             false,
		Message:            "No OpenAPI contract is currently loaded.",
		ContractRootSource: ContractRootSourceWorkingDirectory,
	}
}

func (m *ContractManager) inactiveContractStatus() ContractStatus {
	status := inactiveContractStatus()
	status.ContractRootConfigured = m.contractRoot.configured
	status.ContractRootSource = m.contractRoot.source
	return status
}

// CurrentValidationCapabilities reports the fixed OpenAPI subset enforced by this MVP.
func CurrentValidationCapabilities() ValidationCapabilities {
	return ValidationCapabilities{
		MethodPath:                 true,
		ResponseStatus:             true,
		ResponseContentType:        true,
		ResponseBody:               true,
		InlineJSONResponseSchema:   true,
		RequestBody:                false,
		RequestQuery:               false,
		RequestHeaders:             false,
		PathParameterSchema:        false,
		RefResolution:              true,
		LocalRefResolution:         true,
		RemoteRefResolution:        false,
		Arrays:                     true,
		Enum:                       true,
		AllOf:                      false,
		OneOf:                      false,
		AnyOf:                      false,
		Nullable:                   true,
		AdditionalProperties:       true,
		AdditionalPropertiesSchema: false,
		OpenAPI31:                  false,
		YAML:                       false,
		RemoteFetch:                false,
	}
}

func unsupportedFeatureWarnings(features map[string]int) []string {
	if len(features) == 0 {
		return nil
	}

	warnings := []string{}
	if count := features["non_local_refs"]; count > 0 {
		warnings = append(warnings, fmt.Sprintf("Contract contains non-local $ref values (%d occurrence(s)). Echo MCP currently resolves only local internal OpenAPI JSON pointer refs.", count))
	}
	if count := features["allOf"]; count > 0 {
		warnings = append(warnings, fmt.Sprintf("Contract contains allOf schemas (%d occurrence(s)). Echo MCP currently does not validate allOf composition in this MVP.", count))
	}
	if count := features["oneOf"]; count > 0 {
		warnings = append(warnings, fmt.Sprintf("Contract contains oneOf schemas (%d occurrence(s)). Echo MCP currently does not validate oneOf composition in this MVP.", count))
	}
	if count := features["anyOf"]; count > 0 {
		warnings = append(warnings, fmt.Sprintf("Contract contains anyOf schemas (%d occurrence(s)). Echo MCP currently does not validate anyOf composition in this MVP.", count))
	}
	if count := features["additional_properties_schema"]; count > 0 {
		warnings = append(warnings, fmt.Sprintf("Contract contains schema-valued additionalProperties (%d occurrence(s)). Echo MCP currently supports omitted or boolean additionalProperties only.", count))
	}
	return warnings
}

func copyUnsupportedFeatures(features map[string]int) map[string]int {
	if len(features) == 0 {
		return map[string]int{}
	}
	copied := make(map[string]int, len(features))
	for feature, count := range features {
		copied[feature] = count
	}
	return copied
}

func normalizeValidationMode(mode ValidationMode) (ValidationMode, error) {
	switch ValidationMode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case "":
		return ValidationModeStrict, nil
	case ValidationModeStrict:
		return ValidationModeStrict, nil
	case ValidationModeWarn:
		return ValidationModeWarn, nil
	case ValidationModeOff:
		return ValidationModeOff, nil
	default:
		return "", &OperationError{
			Code:        "invalid_validation_mode",
			Message:     fmt.Sprintf("validation_mode must be one of %q, %q, or %q", ValidationModeStrict, ValidationModeWarn, ValidationModeOff),
			Diagnostics: []string{"Supported validation modes: strict, warn, off."},
		}
	}
}

func defaultContractID(contractID string, sourcePath string) string {
	if strings.TrimSpace(contractID) != "" {
		return strings.TrimSpace(contractID)
	}
	base := filepath.Base(sourcePath)
	extension := filepath.Ext(base)
	if extension != "" {
		base = strings.TrimSuffix(base, extension)
	}
	if base == "" || base == "." {
		return "openapi-contract"
	}
	return base
}

func resetRequiredError(message string) *OperationError {
	return &OperationError{
		Code:        "reset_required",
		Message:     message,
		Diagnostics: []string{"Configured behavior is active and would otherwise be silently associated with a different contract state."},
	}
}
