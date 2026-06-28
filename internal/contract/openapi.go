// Package contract validates configured behavior against developer-provided
// external dependency contracts.
package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"echo-mcp/internal/state"
)

const mediaTypeApplicationJSON = "application/json"

// OpenAPIContract validates response rules against one OpenAPI 3.0.x document.
type OpenAPIContract struct {
	metadata    OpenAPIMetadata
	paths       map[string]map[string]openAPIOperation
	rawDocument any
}

// OpenAPIMetadata describes the loaded OpenAPI contract.
type OpenAPIMetadata struct {
	SourcePath               string
	OpenAPIVersion           string
	OperationsCount          int
	SchemasCount             int
	UnsupportedFeatureCounts map[string]int
}

// LoadError is a safe, structured OpenAPI load diagnostic.
type LoadError struct {
	Code        string
	Message     string
	Diagnostics []string
	Err         error
}

func (e *LoadError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *LoadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ValidationError is a safe, structured OpenAPI validation diagnostic.
type ValidationError struct {
	Code        string
	Message     string
	Diagnostics []string
	Err         error
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *ValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type openAPIDocument struct {
	OpenAPI    string                                 `json:"openapi"`
	Paths      map[string]map[string]openAPIOperation `json:"paths"`
	Components openAPIComponents                      `json:"components"`
}

type openAPIComponents struct {
	Schemas map[string]openAPISchema `json:"schemas"`
}

type openAPIOperation struct {
	Responses map[string]openAPIResponse `json:"responses"`
}

type openAPIResponse struct {
	Content map[string]openAPIMediaType `json:"content"`
}

type openAPIMediaType struct {
	Schema openAPISchema `json:"schema"`
}

type openAPISchema struct {
	Ref                  string                   `json:"$ref"`
	Type                 string                   `json:"type"`
	Required             []string                 `json:"required"`
	Properties           map[string]openAPISchema `json:"properties"`
	Items                *openAPISchema           `json:"items"`
	Enum                 []json.RawMessage        `json:"enum"`
	Nullable             bool                     `json:"nullable"`
	AdditionalProperties json.RawMessage          `json:"additionalProperties"`
	OneOf                []openAPISchema          `json:"oneOf"`
	AnyOf                []openAPISchema          `json:"anyOf"`
	AllOf                []openAPISchema          `json:"allOf"`
	Not                  *openAPISchema           `json:"not"`
}

// LoadOpenAPIFile loads one OpenAPI 3.0.x JSON contract from a local file.
func LoadOpenAPIFile(path string) (*OpenAPIContract, error) {
	return LoadOpenAPIFileWithSourcePath(path, path)
}

// LoadOpenAPIFileWithSourcePath reads path while reporting sourcePath in metadata.
func LoadOpenAPIFileWithSourcePath(path string, sourcePath string) (*OpenAPIContract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &LoadError{
			Code:        "unreadable_file",
			Message:     fmt.Sprintf("OpenAPI contract file %q is unreadable", sourcePath),
			Diagnostics: []string{"The OpenAPI contract file could not be read."},
			Err:         err,
		}
	}

	var document openAPIDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, &LoadError{
			Code:        "invalid_json",
			Message:     fmt.Sprintf("OpenAPI contract file %q is not valid JSON", sourcePath),
			Diagnostics: []string{err.Error()},
			Err:         err,
		}
	}
	var rawDocument any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&rawDocument); err != nil {
		return nil, &LoadError{
			Code:        "invalid_json",
			Message:     fmt.Sprintf("OpenAPI contract file %q is not valid JSON", sourcePath),
			Diagnostics: []string{err.Error()},
			Err:         err,
		}
	}
	if !strings.HasPrefix(document.OpenAPI, "3.0.") {
		return nil, &LoadError{
			Code:        "unsupported_openapi_version",
			Message:     fmt.Sprintf("OpenAPI version %q is not supported; only OpenAPI 3.0.x JSON is supported", document.OpenAPI),
			Diagnostics: []string{"Supported OpenAPI versions: 3.0.x JSON."},
		}
	}
	if len(document.Paths) == 0 {
		return nil, &LoadError{
			Code:        "missing_paths",
			Message:     "OpenAPI contract must define non-empty paths",
			Diagnostics: []string{"The OpenAPI document did not include any paths."},
		}
	}
	unsupportedFeatures := scanUnsupportedFeatures(data)

	return &OpenAPIContract{
		metadata: OpenAPIMetadata{
			SourcePath:               sourcePath,
			OpenAPIVersion:           document.OpenAPI,
			OperationsCount:          countOperations(document.Paths),
			SchemasCount:             countComponentSchemas(document.Components.Schemas),
			UnsupportedFeatureCounts: unsupportedFeatures,
		},
		paths:       document.Paths,
		rawDocument: rawDocument,
	}, nil
}

// Metadata reports loaded contract metadata.
func (c *OpenAPIContract) Metadata() OpenAPIMetadata {
	if c == nil {
		return OpenAPIMetadata{}
	}
	return c.metadata
}

// ValidateResponseRule validates one configured behavior rule against the contract.
func (c *OpenAPIContract) ValidateResponseRule(rule state.ResponseRule) error {
	if c == nil {
		return nil
	}

	operation, err := c.findOperation(rule.Method, rule.Path)
	if err != nil {
		return err
	}
	if operation.Responses == nil {
		return fmt.Errorf("no OpenAPI operation matches %s %s", rule.Method, rule.Path)
	}

	response, ok := operation.Responses[strconv.Itoa(rule.StatusCode)]
	if !ok {
		response, ok = operation.Responses["default"]
	}
	if !ok {
		return fmt.Errorf("OpenAPI operation %s %s does not define response status %d", rule.Method, rule.Path, rule.StatusCode)
	}

	if len(response.Content) == 0 {
		return nil
	}

	media, ok := response.Content[mediaTypeApplicationJSON]
	if !ok {
		return fmt.Errorf("OpenAPI response for %s %s status %d does not define supported application/json content", rule.Method, rule.Path, rule.StatusCode)
	}
	if normalizeContentType(rule.ContentType) != mediaTypeApplicationJSON {
		return fmt.Errorf("outcome.content_type must be %q for %s %s status %d", mediaTypeApplicationJSON, rule.Method, rule.Path, rule.StatusCode)
	}

	var body any
	decoder := json.NewDecoder(strings.NewReader(rule.Body))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return fmt.Errorf("outcome.body must be valid JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("outcome.body must contain one JSON value")
	}

	state := &schemaValidationState{resolving: map[string]bool{}}
	if err := c.validateSchema(media.Schema, body, "$", state); err != nil {
		return err
	}

	return nil
}

type schemaValidationState struct {
	resolving map[string]bool
}

func (c *OpenAPIContract) findOperation(method string, path string) (openAPIOperation, error) {
	httpMethod := strings.ToLower(method)

	if operations, ok := c.paths[path]; ok {
		if operation, ok := operations[httpMethod]; ok {
			return operation, nil
		}
	}

	templates := make([]string, 0, len(c.paths))
	for template := range c.paths {
		templates = append(templates, template)
	}
	sort.Strings(templates)

	var matchedTemplates []string
	var matchedOperation openAPIOperation
	for _, template := range templates {
		if template == path || !pathMatches(template, path) {
			continue
		}
		operation, ok := c.paths[template][httpMethod]
		if ok {
			matchedTemplates = append(matchedTemplates, template)
			matchedOperation = operation
		}
	}

	if len(matchedTemplates) > 1 {
		return openAPIOperation{}, fmt.Errorf("ambiguous OpenAPI route for %s %s: %s", method, path, strings.Join(matchedTemplates, ", "))
	}
	if len(matchedTemplates) == 1 {
		return matchedOperation, nil
	}

	return openAPIOperation{}, nil
}

func countOperations(paths map[string]map[string]openAPIOperation) int {
	count := 0
	for _, operations := range paths {
		for method := range operations {
			if isHTTPMethod(method) {
				count++
			}
		}
	}
	return count
}

func countComponentSchemas(schemas map[string]openAPISchema) int {
	return len(schemas)
}

func isHTTPMethod(method string) bool {
	switch strings.ToLower(method) {
	case "get", "put", "post", "delete", "options", "head", "patch", "trace":
		return true
	default:
		return false
	}
}

func pathMatches(template string, path string) bool {
	templateParts := splitPath(template)
	pathParts := splitPath(path)
	if len(templateParts) != len(pathParts) {
		return false
	}

	for i := range templateParts {
		templatePart := templateParts[i]
		pathPart := pathParts[i]
		if strings.HasPrefix(templatePart, "{") && strings.HasSuffix(templatePart, "}") && len(templatePart) > 2 {
			if pathPart == "" {
				return false
			}
			continue
		}
		if templatePart != pathPart {
			return false
		}
	}

	return true
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func (c *OpenAPIContract) validateSchema(schema openAPISchema, value any, path string, state *schemaValidationState) error {
	if schema.Ref != "" {
		return c.validateReferencedSchema(schema.Ref, value, path, state)
	}
	if value == nil && schema.Nullable {
		return nil
	}
	if err := rejectUnsupportedSchemaConstructs(schema, path); err != nil {
		return err
	}

	schemaType := schema.Type
	if schemaType == "" && (len(schema.Properties) > 0 || len(schema.Required) > 0) {
		schemaType = "object"
	}

	switch schemaType {
	case "object":
		objectValue, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object", path)
		}
		for _, required := range schema.Required {
			if _, ok := objectValue[required]; !ok {
				return fmt.Errorf("%s.%s is required by the OpenAPI response schema", path, required)
			}
		}
		propertyNames := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			propertyNames = append(propertyNames, name)
		}
		sort.Strings(propertyNames)
		for _, name := range propertyNames {
			propertyValue, ok := objectValue[name]
			if !ok {
				continue
			}
			if err := c.validateSchema(schema.Properties[name], propertyValue, path+"."+name, state); err != nil {
				return err
			}
		}
		if err := validateAdditionalProperties(schema, objectValue, path); err != nil {
			return err
		}
	case "array":
		arrayValue, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s must be an array", path)
		}
		if schema.Items != nil {
			for index, item := range arrayValue {
				if err := c.validateSchema(*schema.Items, item, fmt.Sprintf("%s[%d]", path, index), state); err != nil {
					return err
				}
			}
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be a string", path)
		}
	case "integer":
		number, ok := value.(json.Number)
		if !ok {
			return fmt.Errorf("%s must be an integer", path)
		}
		if _, err := strconv.ParseInt(number.String(), 10, 64); err != nil {
			return fmt.Errorf("%s must be an integer", path)
		}
	case "number":
		number, ok := value.(json.Number)
		if !ok {
			return fmt.Errorf("%s must be a number", path)
		}
		if _, err := strconv.ParseFloat(number.String(), 64); err != nil {
			return fmt.Errorf("%s must be a number", path)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a boolean", path)
		}
	case "":
		return nil
	default:
		return fmt.Errorf("%s uses unsupported schema type %q", path, schemaType)
	}

	if err := validateEnum(schema, value, path); err != nil {
		return err
	}

	return nil
}

func rejectUnsupportedSchemaConstructs(schema openAPISchema, path string) error {
	if len(schema.OneOf) > 0 {
		return unsupportedSchemaFeatureError(path, "oneOf", "")
	}
	if len(schema.AnyOf) > 0 {
		return unsupportedSchemaFeatureError(path, "anyOf", "")
	}
	if len(schema.AllOf) > 0 {
		return unsupportedSchemaFeatureError(path, "allOf", "")
	}
	if schema.Not != nil {
		return unsupportedSchemaFeatureError(path, "not", "")
	}

	return nil
}

func (c *OpenAPIContract) validateReferencedSchema(ref string, value any, path string, state *schemaValidationState) error {
	if !strings.HasPrefix(ref, "#/") {
		return unsupportedNonLocalRefError(path, ref)
	}
	if state == nil {
		state = &schemaValidationState{resolving: map[string]bool{}}
	}
	if state.resolving[ref] {
		return cyclicRefError(path, ref)
	}

	state.resolving[ref] = true
	defer delete(state.resolving, ref)

	resolved, err := c.resolveLocalSchemaRef(ref)
	if err != nil {
		return unresolvedRefError(path, ref)
	}
	return c.validateSchema(resolved, value, path, state)
}

func (c *OpenAPIContract) resolveLocalSchemaRef(ref string) (openAPISchema, error) {
	target, err := resolveJSONPointer(c.rawDocument, strings.TrimPrefix(ref, "#"))
	if err != nil {
		return openAPISchema{}, err
	}
	payload, err := json.Marshal(target)
	if err != nil {
		return openAPISchema{}, err
	}
	var schema openAPISchema
	if err := json.Unmarshal(payload, &schema); err != nil {
		return openAPISchema{}, err
	}
	return schema, nil
}

func resolveJSONPointer(document any, pointer string) (any, error) {
	if pointer == "" {
		return document, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("JSON pointer must start with /")
	}

	current := document
	for _, rawToken := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		token := strings.ReplaceAll(strings.ReplaceAll(rawToken, "~1", "/"), "~0", "~")
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[token]
			if !ok {
				return nil, fmt.Errorf("JSON pointer token %q not found", token)
			}
			current = next
		case []any:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, fmt.Errorf("JSON pointer array index %q not found", token)
			}
			current = typed[index]
		default:
			return nil, fmt.Errorf("JSON pointer token %q cannot be resolved", token)
		}
	}
	return current, nil
}

func validateEnum(schema openAPISchema, value any, path string) error {
	if len(schema.Enum) == 0 {
		return nil
	}

	valueJSON, err := canonicalJSON(value)
	if err != nil {
		return err
	}
	for _, raw := range schema.Enum {
		enumValue, err := decodeRawJSONValue(raw)
		if err != nil {
			return fmt.Errorf("%s contains invalid enum value", path)
		}
		enumJSON, err := canonicalJSON(enumValue)
		if err != nil {
			return err
		}
		if valueJSON == enumJSON {
			return nil
		}
	}

	return fmt.Errorf("%s must be one of %s", path, enumValuesDescription(schema.Enum))
}

func decodeRawJSONValue(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func canonicalJSON(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func enumValuesDescription(values []json.RawMessage) string {
	descriptions := make([]string, 0, len(values))
	for _, value := range values {
		descriptions = append(descriptions, string(value))
	}
	return strings.Join(descriptions, ", ")
}

func validateAdditionalProperties(schema openAPISchema, objectValue map[string]any, path string) error {
	if len(schema.AdditionalProperties) == 0 {
		return nil
	}

	allowed, ok := additionalPropertiesBoolean(schema.AdditionalProperties)
	if !ok {
		if hasAdditionalProperties(schema, objectValue) {
			return unsupportedSchemaFeatureError(path, "additionalProperties schema", "")
		}
		return nil
	}
	if allowed {
		return nil
	}

	propertyNames := make(map[string]struct{}, len(schema.Properties))
	for name := range schema.Properties {
		propertyNames[name] = struct{}{}
	}
	extraNames := make([]string, 0)
	for name := range objectValue {
		if _, ok := propertyNames[name]; !ok {
			extraNames = append(extraNames, name)
		}
	}
	sort.Strings(extraNames)
	if len(extraNames) > 0 {
		return fmt.Errorf("%s.%s is not allowed by the OpenAPI response schema", path, extraNames[0])
	}
	return nil
}

func additionalPropertiesBoolean(raw json.RawMessage) (bool, bool) {
	trimmed := strings.TrimSpace(string(raw))
	switch trimmed {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return true, false
	}
}

func hasAdditionalProperties(schema openAPISchema, objectValue map[string]any) bool {
	for name := range objectValue {
		if _, ok := schema.Properties[name]; !ok {
			return true
		}
	}
	return false
}

func unresolvedRefError(path string, ref string) *ValidationError {
	return &ValidationError{
		Code:    "contract_validation_failed",
		Message: "OpenAPI response schema contains unresolved $ref",
		Diagnostics: []string{
			fmt.Sprintf("schema path %q contains unresolved $ref %q", path, ref),
			"The behavior was not validated against the response schema.",
		},
	}
}

func cyclicRefError(path string, ref string) *ValidationError {
	return &ValidationError{
		Code:    "contract_validation_failed",
		Message: "OpenAPI response schema contains cyclic $ref",
		Diagnostics: []string{
			fmt.Sprintf("schema path %q contains cyclic $ref %q", path, ref),
			"The behavior was not validated against the response schema.",
		},
	}
}

func unsupportedNonLocalRefError(path string, ref string) *ValidationError {
	return &ValidationError{
		Code:    "unsupported_contract_feature",
		Message: fmt.Sprintf("OpenAPI response schema at %s uses unsupported non-local $ref", path),
		Diagnostics: []string{
			fmt.Sprintf("schema path %q contains unsupported non-local $ref %q", path, ref),
			"Only local internal OpenAPI JSON pointer refs are supported in this MVP.",
			"The behavior was not validated against the response schema.",
		},
	}
}

func unsupportedSchemaFeatureError(path string, feature string, value string) *ValidationError {
	detail := fmt.Sprintf("schema path %q contains unsupported %s", path, feature)
	limitation := fmt.Sprintf("%s validation is unsupported in this MVP.", feature)
	waitAction := fmt.Sprintf("Suggested next action: wait for %s support.", feature)
	if feature == "$ref" && value != "" {
		detail = fmt.Sprintf("schema path %q contains unsupported $ref %q", path, value)
		limitation = "$ref resolution is unsupported in this MVP."
		waitAction = "Suggested next action: wait for $ref resolution support."
	}

	return &ValidationError{
		Code:    "unsupported_contract_feature",
		Message: fmt.Sprintf("OpenAPI response schema at %s uses unsupported %s", path, feature),
		Diagnostics: []string{
			detail,
			limitation,
			"The behavior was not validated against the response schema.",
			"Suggested next action: use validation.mode=off with reason for intentional manual behavior.",
			"Suggested next action: use a reduced/inline schema fixture.",
			waitAction,
		},
	}
}

func scanUnsupportedFeatures(data []byte) map[string]int {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}

	counts := map[string]int{}
	scanUnsupportedFeatureValue(root, counts)
	return counts
}

func scanUnsupportedFeatureValue(value any, counts map[string]int) {
	switch typed := value.(type) {
	case map[string]any:
		if ref, ok := typed["$ref"].(string); ok {
			if !strings.HasPrefix(ref, "#/") {
				counts["non_local_refs"]++
			}
		}
		for _, feature := range []string{"allOf", "oneOf", "anyOf"} {
			if _, ok := typed[feature]; ok {
				counts[feature]++
			}
		}
		if additionalProperties, ok := typed["additionalProperties"]; ok {
			if _, ok := additionalProperties.(bool); !ok {
				counts["additional_properties_schema"]++
			}
		}
		for _, nested := range typed {
			scanUnsupportedFeatureValue(nested, counts)
		}
	case []any:
		for _, nested := range typed {
			scanUnsupportedFeatureValue(nested, counts)
		}
	}
}

func normalizeContentType(contentType string) string {
	mediaType, _, _ := strings.Cut(contentType, ";")
	return strings.ToLower(strings.TrimSpace(mediaType))
}
