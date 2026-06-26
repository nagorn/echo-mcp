// Package contract validates configured behavior against developer-provided
// external dependency contracts.
package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"echo-mcp/internal/state"
)

const mediaTypeApplicationJSON = "application/json"

// OpenAPIContract validates response rules against one OpenAPI 3.0.x document.
type OpenAPIContract struct {
	paths map[string]map[string]openAPIOperation
}

type openAPIDocument struct {
	OpenAPI string                                 `json:"openapi"`
	Paths   map[string]map[string]openAPIOperation `json:"paths"`
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
	Ref        string                   `json:"$ref"`
	Type       string                   `json:"type"`
	Required   []string                 `json:"required"`
	Properties map[string]openAPISchema `json:"properties"`
	Items      *openAPISchema           `json:"items"`
	OneOf      []openAPISchema          `json:"oneOf"`
	AnyOf      []openAPISchema          `json:"anyOf"`
	AllOf      []openAPISchema          `json:"allOf"`
	Not        *openAPISchema           `json:"not"`
}

// LoadOpenAPIFile loads one OpenAPI 3.0.x JSON contract from a local file.
func LoadOpenAPIFile(path string) (*OpenAPIContract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI contract %q: %w", path, err)
	}

	var document openAPIDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse OpenAPI contract %q: %w", path, err)
	}
	if !strings.HasPrefix(document.OpenAPI, "3.0.") {
		return nil, fmt.Errorf("OpenAPI version %q is not supported; only OpenAPI 3.0.x JSON is supported", document.OpenAPI)
	}
	if len(document.Paths) == 0 {
		return nil, fmt.Errorf("OpenAPI contract must define paths")
	}

	return &OpenAPIContract{paths: document.Paths}, nil
}

// ValidateResponseRule validates one configured behavior rule against the contract.
func (c *OpenAPIContract) ValidateResponseRule(rule state.ResponseRule) error {
	if c == nil {
		return nil
	}

	operation, ok := c.findOperation(rule.Method, rule.Path)
	if !ok {
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
	if decoder.More() {
		return fmt.Errorf("outcome.body must contain one JSON value")
	}

	if err := validateSchema(media.Schema, body, "$"); err != nil {
		return err
	}

	return nil
}

func (c *OpenAPIContract) findOperation(method string, path string) (openAPIOperation, bool) {
	httpMethod := strings.ToLower(method)

	if operations, ok := c.paths[path]; ok {
		if operation, ok := operations[httpMethod]; ok {
			return operation, true
		}
	}

	templates := make([]string, 0, len(c.paths))
	for template := range c.paths {
		templates = append(templates, template)
	}
	sort.Strings(templates)

	for _, template := range templates {
		if template == path || !pathMatches(template, path) {
			continue
		}
		operation, ok := c.paths[template][httpMethod]
		if ok {
			return operation, true
		}
	}

	return openAPIOperation{}, false
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

func validateSchema(schema openAPISchema, value any, path string) error {
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
			if err := validateSchema(schema.Properties[name], propertyValue, path+"."+name); err != nil {
				return err
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

	return nil
}

func rejectUnsupportedSchemaConstructs(schema openAPISchema, path string) error {
	if schema.Ref != "" {
		return fmt.Errorf("%s uses unsupported $ref", path)
	}
	if schema.Items != nil {
		return fmt.Errorf("%s uses unsupported array items", path)
	}
	if len(schema.OneOf) > 0 {
		return fmt.Errorf("%s uses unsupported oneOf", path)
	}
	if len(schema.AnyOf) > 0 {
		return fmt.Errorf("%s uses unsupported anyOf", path)
	}
	if len(schema.AllOf) > 0 {
		return fmt.Errorf("%s uses unsupported allOf", path)
	}
	if schema.Not != nil {
		return fmt.Errorf("%s uses unsupported not", path)
	}

	return nil
}

func normalizeContentType(contentType string) string {
	mediaType, _, _ := strings.Cut(contentType, ";")
	return strings.ToLower(strings.TrimSpace(mediaType))
}
