package contract

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"echo-mcp/internal/state"
)

const paymentIntentDeclinedBody = `{"error":{"type":"card_error","code":"card_declined","decline_code":"generic_decline","message":"Your card was declined.","payment_intent":{"id":"pi_123","object":"payment_intent","status":"requires_payment_method"}}}`

func TestLoadOpenAPIFileAcceptsOpenAPI30JSON(t *testing.T) {
	validator, err := LoadOpenAPIFile(filepath.Join("testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}
	if validator == nil {
		t.Fatal("validator = nil")
	}
}

func TestLoadOpenAPIFileReportsMetadata(t *testing.T) {
	sourcePath := filepath.Join("testdata", "payment-intent-openapi.json")
	validator, err := LoadOpenAPIFile(sourcePath)
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}

	metadata := validator.Metadata()
	if metadata.SourcePath != sourcePath {
		t.Fatalf("SourcePath = %q, want %q", metadata.SourcePath, sourcePath)
	}
	if metadata.OpenAPIVersion != "3.0.3" {
		t.Fatalf("OpenAPIVersion = %q, want 3.0.3", metadata.OpenAPIVersion)
	}
	if metadata.OperationsCount != 1 {
		t.Fatalf("OperationsCount = %d, want 1", metadata.OperationsCount)
	}
	if metadata.SchemasCount != 0 {
		t.Fatalf("SchemasCount = %d, want 0 component schemas", metadata.SchemasCount)
	}
}

func TestLoadOpenAPIFileCountsComponentSchemas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "component-schemas-openapi.json")
	document := `{
  "openapi": "3.0.3",
  "paths": {
    "/widgets": {
      "get": {
        "responses": {
          "200": {
            "description": "ok",
            "content": {
              "application/json": {
                "schema": {"$ref": "#/components/schemas/Widget"}
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Widget": {"type": "object"},
      "Error": {"type": "object"}
    }
  }
}`
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	validator, err := LoadOpenAPIFile(path)
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}
	metadata := validator.Metadata()
	if metadata.SchemasCount != 2 {
		t.Fatalf("SchemasCount = %d, want 2 component schemas", metadata.SchemasCount)
	}
}

func TestLoadOpenAPIFileReportsUnreadableFileDiagnostic(t *testing.T) {
	_, err := LoadOpenAPIFile(filepath.Join(t.TempDir(), "missing-openapi.json"))
	if err == nil {
		t.Fatal("LoadOpenAPIFile() error = nil, want unreadable file diagnostic")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if loadErr.Code != "unreadable_file" {
		t.Fatalf("Code = %q, want unreadable_file", loadErr.Code)
	}
}

func TestLoadOpenAPIFileReportsInvalidJSONDiagnostic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid-openapi.json")
	if err := os.WriteFile(path, []byte(`{"openapi":`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadOpenAPIFile(path)
	if err == nil {
		t.Fatal("LoadOpenAPIFile() error = nil, want invalid JSON diagnostic")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if loadErr.Code != "invalid_json" {
		t.Fatalf("Code = %q, want invalid_json", loadErr.Code)
	}
	if strings.Contains(loadErr.Message, `{"openapi":`) {
		t.Fatalf("diagnostic leaks file contents: %q", loadErr.Message)
	}
}

func TestLoadOpenAPIFileReportsUnsupportedOpenAPIDiagnostic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "openapi-31.json")
	if err := os.WriteFile(path, []byte(`{"openapi":"3.1.0","paths":{"/payments":{"get":{"responses":{"200":{"description":"ok"}}}}}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadOpenAPIFile(path)
	if err == nil {
		t.Fatal("LoadOpenAPIFile() error = nil, want unsupported OpenAPI diagnostic")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if loadErr.Code != "unsupported_openapi_version" {
		t.Fatalf("Code = %q, want unsupported_openapi_version", loadErr.Code)
	}
}

func TestLoadOpenAPIFileReportsMissingPathsDiagnostic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-paths.json")
	if err := os.WriteFile(path, []byte(`{"openapi":"3.0.3","paths":{}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadOpenAPIFile(path)
	if err == nil {
		t.Fatal("LoadOpenAPIFile() error = nil, want missing paths diagnostic")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if loadErr.Code != "missing_paths" {
		t.Fatalf("Code = %q, want missing_paths", loadErr.Code)
	}
}

func TestOpenAPIValidatorAcceptsMatchingPaymentIntentFailure(t *testing.T) {
	validator, err := LoadOpenAPIFile(filepath.Join("testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}

	err = validator.ValidateResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusPaymentRequired,
		ContentType: "application/json",
		Body:        paymentIntentDeclinedBody,
	})
	if err != nil {
		t.Fatalf("ValidateResponseRule() error = %v", err)
	}
}

func TestOpenAPIValidatorRejectsUnsupportedResponseStatus(t *testing.T) {
	validator, err := LoadOpenAPIFile(filepath.Join("testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}

	err = validator.ValidateResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        paymentIntentDeclinedBody,
	})
	if err == nil {
		t.Fatal("ValidateResponseRule() error = nil, want response status rejection")
	}
}

func TestOpenAPIValidatorRejectsMissingRequiredResponseField(t *testing.T) {
	validator, err := LoadOpenAPIFile(filepath.Join("testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}

	err = validator.ValidateResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusPaymentRequired,
		ContentType: "application/json",
		Body:        `{"error":{"type":"card_error","code":"card_declined"}}`,
	})
	if err == nil {
		t.Fatal("ValidateResponseRule() error = nil, want missing required field rejection")
	}
}

func TestOpenAPIValidatorRejectsMultipleJSONValues(t *testing.T) {
	validator, err := LoadOpenAPIFile(filepath.Join("testdata", "payment-intent-openapi.json"))
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}

	err = validator.ValidateResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusPaymentRequired,
		ContentType: "application/json",
		Body:        paymentIntentDeclinedBody + ` {}`,
	})
	if err == nil {
		t.Fatal("ValidateResponseRule() error = nil, want multiple JSON values rejection")
	}
}

func TestOpenAPICompatibilityCorpus(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		rule        state.ResponseRule
		wantErr     bool
		wantCode    string
		wantMessage string
	}{
		{
			name:    "inline response schema validates",
			fixture: "inline-response.json",
			rule: responseRule(http.MethodGet, "/widgets/wgt_123", http.StatusOK,
				`{"id":"wgt_123","name":"Launch checklist"}`),
		},
		{
			name:    "local ref response schema validates",
			fixture: "local-ref-response.json",
			rule: responseRule(http.MethodGet, "/widgets/wgt_123", http.StatusOK,
				`{"id":"wgt_123","kind":"widget","status":"active"}`),
		},
		{
			name:    "nested local ref validates",
			fixture: "nested-local-ref.json",
			rule: responseRule(http.MethodGet, "/widgets/wgt_123", http.StatusOK,
				`{"id":"wgt_123","owner":{"id":"usr_123","email":"owner@example.test"}}`),
		},
		{
			name:    "array of referenced objects validates",
			fixture: "array-ref-response.json",
			rule: responseRule(http.MethodGet, "/widgets", http.StatusOK,
				`{"items":[{"id":"wgt_123","name":"Launch checklist"}]}`),
		},
		{
			name:    "required field missing fails",
			fixture: "required-field.json",
			rule: responseRule(http.MethodGet, "/widgets/wgt_123", http.StatusOK,
				`{"id":"wgt_123"}`),
			wantErr:     true,
			wantMessage: "$.name is required by the OpenAPI response schema",
		},
		{
			name:    "enum mismatch fails",
			fixture: "enum-response.json",
			rule: responseRule(http.MethodGet, "/widgets/wgt_123", http.StatusOK,
				`{"id":"wgt_123","status":"deleted"}`),
			wantErr:     true,
			wantMessage: "$.status must be one of",
		},
		{
			name:    "nullable field accepts null",
			fixture: "nullable-response.json",
			rule: responseRule(http.MethodGet, "/widgets/wgt_123", http.StatusOK,
				`{"id":"wgt_123","nickname":null}`),
		},
		{
			name:    "unresolved ref fails clearly",
			fixture: "unresolved-ref.json",
			rule: responseRule(http.MethodGet, "/widgets/wgt_123", http.StatusOK,
				`{"id":"wgt_123"}`),
			wantErr:     true,
			wantCode:    "contract_validation_failed",
			wantMessage: `schema path "$" contains unresolved $ref "#/components/schemas/MissingWidget"`,
		},
		{
			name:    "cyclic ref fails clearly",
			fixture: "cyclic-ref.json",
			rule: responseRule(http.MethodGet, "/nodes/node_123", http.StatusOK,
				`{"child":{}}`),
			wantErr:     true,
			wantCode:    "contract_validation_failed",
			wantMessage: `schema path "$.child" contains cyclic $ref "#/components/schemas/Node"`,
		},
		{
			name:    "unsupported composition fails clearly",
			fixture: "unsupported-composition.json",
			rule: responseRule(http.MethodGet, "/widgets/wgt_123", http.StatusOK,
				`{"id":"wgt_123","name":"Launch checklist"}`),
			wantErr:     true,
			wantCode:    "unsupported_contract_feature",
			wantMessage: `schema path "$" contains unsupported oneOf`,
		},
		{
			name:    "provider neutral example library API validates",
			fixture: "example-library-api.json",
			rule: responseRule(http.MethodGet, "/books/book_123", http.StatusOK,
				`{"id":"book_123","title":"Distributed Systems","status":"available","author":{"id":"auth_123","name":"Ada Example"},"subtitle":null}`),
		},
		{
			name:    "provider neutral example library API rejects invalid enum",
			fixture: "example-library-api.json",
			rule: responseRule(http.MethodGet, "/books/book_123", http.StatusOK,
				`{"id":"book_123","title":"Distributed Systems","status":"lost","author":{"id":"auth_123","name":"Ada Example"}}`),
			wantErr:     true,
			wantMessage: "$.status must be one of",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			validator, err := LoadOpenAPIFile(corpusFixture(tt.fixture))
			if err != nil {
				t.Fatalf("LoadOpenAPIFile() error = %v", err)
			}

			err = validator.ValidateResponseRule(tt.rule)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("ValidateResponseRule() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateResponseRule() error = nil, want rejection")
			}
			if tt.wantCode != "" {
				var validationErr *ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("error = %T, want *ValidationError", err)
				}
				if validationErr.Code != tt.wantCode {
					t.Fatalf("Code = %q, want %q", validationErr.Code, tt.wantCode)
				}
				if !strings.Contains(strings.Join(validationErr.Diagnostics, "\n"), tt.wantMessage) {
					t.Fatalf("diagnostics missing %q: %+v", tt.wantMessage, validationErr.Diagnostics)
				}
				return
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("error = %q, want %q", err, tt.wantMessage)
			}
		})
	}
}

func TestOpenAPIValidatorResolvesLocalResponseSchemaRef(t *testing.T) {
	validator := loadTempOpenAPIContract(t, `{
  "openapi": "3.0.3",
  "paths": {
    "/v1/payment_intents": {
      "post": {
        "responses": {
          "200": {
            "description": "PaymentIntent",
            "content": {
              "application/json": {
                "schema": {"$ref": "#/components/schemas/payment_intent"}
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
        "type": "object",
        "required": ["id", "object", "status"],
        "properties": {
          "id": {"type": "string"},
          "object": {"type": "string", "enum": ["payment_intent"]},
          "status": {"type": "string", "enum": ["requires_payment_method", "succeeded"]}
        }
      }
    }
  }
}`)

	err := validator.ValidateResponseRule(state.ResponseRule{
		ID:          "payment-intent-created",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"id":"pi_123","object":"payment_intent","status":"succeeded"}`,
	})
	if err != nil {
		t.Fatalf("ValidateResponseRule() error = %v", err)
	}
}

func TestOpenAPIValidatorResolvesNestedLocalResponseSchemaRef(t *testing.T) {
	validator := loadTempOpenAPIContract(t, `{
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
                  "type": "object",
                  "required": ["customer"],
                  "properties": {
                    "customer": {"$ref": "#/components/schemas/customer"}
                  }
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
      "customer": {
        "type": "object",
        "required": ["id", "object"],
        "properties": {
          "id": {"type": "string"},
          "object": {"type": "string", "enum": ["customer"]}
        }
      }
    }
  }
}`)

	err := validator.ValidateResponseRule(state.ResponseRule{
		ID:          "payment-intent-with-customer",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"customer":{"id":"cus_123","object":"customer"}}`,
	})
	if err != nil {
		t.Fatalf("ValidateResponseRule() error = %v", err)
	}
}

func TestOpenAPIValidatorReportsUnresolvedLocalRef(t *testing.T) {
	validator := loadTempOpenAPIContract(t, `{
  "openapi": "3.0.3",
  "paths": {
    "/v1/payment_intents": {
      "post": {
        "responses": {
          "200": {
            "description": "PaymentIntent",
            "content": {
              "application/json": {
                "schema": {"$ref": "#/components/schemas/payment_intent"}
              }
            }
          }
        }
      }
    }
  }
}`)

	err := validator.ValidateResponseRule(state.ResponseRule{
		ID:          "payment-intent-created",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"id":"pi_123"}`,
	})
	assertValidationError(t, err, "contract_validation_failed", `schema path "$" contains unresolved $ref "#/components/schemas/payment_intent"`, "The behavior was not validated against the response schema.")
}

func TestOpenAPIValidatorReportsCyclicLocalRef(t *testing.T) {
	validator := loadTempOpenAPIContract(t, `{
  "openapi": "3.0.3",
  "paths": {
    "/foo": {
      "get": {
        "responses": {
          "200": {
            "description": "Foo",
            "content": {
              "application/json": {
                "schema": {"$ref": "#/components/schemas/Foo"}
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Foo": {
        "type": "object",
        "properties": {
          "child": {"$ref": "#/components/schemas/Foo"}
        }
      }
    }
  }
}`)

	err := validator.ValidateResponseRule(state.ResponseRule{
		ID:          "cyclic-foo",
		Method:      http.MethodGet,
		Path:        "/foo",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"child":{}}`,
	})
	assertValidationError(t, err, "contract_validation_failed", `schema path "$.child" contains cyclic $ref "#/components/schemas/Foo"`)
}

func TestOpenAPIValidatorRejectsInvalidEnum(t *testing.T) {
	validator := loadTempOpenAPIContract(t, `{
  "openapi": "3.0.3",
  "paths": {
    "/v1/payment_intents": {
      "post": {
        "responses": {
          "200": {
            "description": "PaymentIntent",
            "content": {
              "application/json": {
                "schema": {"$ref": "#/components/schemas/payment_intent"}
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
        "type": "object",
        "required": ["id", "object", "status"],
        "properties": {
          "id": {"type": "string"},
          "object": {"type": "string", "enum": ["payment_intent"]},
          "status": {"type": "string", "enum": ["requires_payment_method", "succeeded"]}
        }
      }
    }
  }
}`)

	err := validator.ValidateResponseRule(state.ResponseRule{
		ID:          "payment-intent-invalid-status",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"id":"pi_123","object":"payment_intent","status":"not_a_status"}`,
	})
	if err == nil {
		t.Fatal("ValidateResponseRule() error = nil, want enum rejection")
	}
	if !strings.Contains(err.Error(), "$.status must be one of") {
		t.Fatalf("error = %q, want enum diagnostic", err)
	}
}

func TestOpenAPIValidatorSupportsNullableProperties(t *testing.T) {
	validator := loadTempOpenAPIContract(t, `{
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
                  "type": "object",
                  "properties": {
                    "client_secret": {"type": "string", "nullable": true}
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}`)

	err := validator.ValidateResponseRule(state.ResponseRule{
		ID:          "payment-intent-null-client-secret",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"client_secret":null}`,
	})
	if err != nil {
		t.Fatalf("ValidateResponseRule() error = %v", err)
	}
}

func TestOpenAPIValidatorRejectsAdditionalPropertiesFalse(t *testing.T) {
	validator := loadTempOpenAPIContract(t, `{
  "openapi": "3.0.3",
  "paths": {
    "/strict": {
      "get": {
        "responses": {
          "200": {
            "description": "Strict",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "id": {"type": "string"}
                  },
                  "additionalProperties": false
                }
              }
            }
          }
        }
      }
    }
  }
}`)

	err := validator.ValidateResponseRule(state.ResponseRule{
		ID:          "strict-extra",
		Method:      http.MethodGet,
		Path:        "/strict",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"id":"obj_123","extra":true}`,
	})
	if err == nil {
		t.Fatal("ValidateResponseRule() error = nil, want additionalProperties rejection")
	}
	if !strings.Contains(err.Error(), "$.extra is not allowed by the OpenAPI response schema") {
		t.Fatalf("error = %q, want additionalProperties diagnostic", err)
	}
}

func TestOpenAPIValidatorSupportsArrayItems(t *testing.T) {
	validator := loadTempOpenAPIContract(t, `{
  "openapi": "3.0.3",
  "paths": {
    "/array": {
      "get": {
        "responses": {
          "200": {
            "description": "Array",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "payment_method_types": {
                      "type": "array",
                      "items": {"type": "string"}
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}`)

	err := validator.ValidateResponseRule(state.ResponseRule{
		ID:          "array-ok",
		Method:      http.MethodGet,
		Path:        "/array",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"payment_method_types":["card"]}`,
	})
	if err != nil {
		t.Fatalf("ValidateResponseRule() error = %v", err)
	}
}

func TestOpenAPIValidatorAcceptsStripePaymentIntentWhenFixtureAvailable(t *testing.T) {
	sourcePath := os.Getenv("ECHO_MCP_STRIPE_OPENAPI_FILE")
	if sourcePath == "" {
		t.Skip("set ECHO_MCP_STRIPE_OPENAPI_FILE to run the Stripe OpenAPI integration test")
	}

	validator, err := LoadOpenAPIFile(sourcePath)
	if err != nil {
		t.Fatalf("LoadOpenAPIFile(%q) error = %v", sourcePath, err)
	}

	valid := state.ResponseRule{
		ID:          "stripe-paymentintent-valid",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"id":"pi_123","object":"payment_intent","status":"succeeded","created":1710000000,"livemode":false,"amount":2000,"currency":"usd","payment_method_types":["card"],"client_secret":null}`,
	}
	if err := validator.ValidateResponseRule(valid); err != nil {
		t.Fatalf("valid Stripe PaymentIntent response rejected: %v", err)
	}

	invalid := valid
	invalid.ID = "stripe-paymentintent-invalid-status"
	invalid.Body = `{"id":"pi_123","object":"payment_intent","status":"not_a_status","created":1710000000,"livemode":false}`
	err = validator.ValidateResponseRule(invalid)
	if err == nil {
		t.Fatal("invalid Stripe PaymentIntent response accepted, want enum rejection")
	}
	if !strings.Contains(err.Error(), "$.status must be one of") {
		t.Fatalf("invalid Stripe PaymentIntent error = %q, want status enum diagnostic", err)
	}
}

func TestOpenAPIValidatorAcceptsGitHubMetaWhenFixtureAvailable(t *testing.T) {
	sourcePath := os.Getenv("ECHO_MCP_GITHUB_OPENAPI_FILE")
	if sourcePath == "" {
		t.Skip("set ECHO_MCP_GITHUB_OPENAPI_FILE to run the GitHub OpenAPI integration test")
	}

	validator, err := LoadOpenAPIFile(sourcePath)
	if err != nil {
		t.Fatalf("LoadOpenAPIFile(%q) error = %v", sourcePath, err)
	}

	valid := state.ResponseRule{
		ID:          "github-meta-valid",
		Method:      http.MethodGet,
		Path:        "/meta",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"verifiable_password_authentication":true}`,
	}
	if err := validator.ValidateResponseRule(valid); err != nil {
		t.Fatalf("valid GitHub /meta response rejected: %v", err)
	}

	invalid := valid
	invalid.ID = "github-meta-invalid"
	invalid.Body = `{}`
	err = validator.ValidateResponseRule(invalid)
	if err == nil {
		t.Fatal("invalid GitHub /meta response accepted, want required-field rejection")
	}
	if !strings.Contains(err.Error(), "$.verifiable_password_authentication is required") {
		t.Fatalf("invalid GitHub /meta error = %q, want required-field diagnostic", err)
	}
}

func loadTempOpenAPIContract(t *testing.T, document string) *OpenAPIContract {
	t.Helper()

	path := filepath.Join(t.TempDir(), "openapi.json")
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	validator, err := LoadOpenAPIFile(path)
	if err != nil {
		t.Fatalf("LoadOpenAPIFile() error = %v", err)
	}
	return validator
}

func corpusFixture(name string) string {
	return filepath.Join("testdata", "openapi-corpus", name)
}

func responseRule(method string, path string, statusCode int, body string) state.ResponseRule {
	return state.ResponseRule{
		ID:          "corpus-rule",
		Method:      method,
		Path:        path,
		StatusCode:  statusCode,
		ContentType: "application/json",
		Body:        body,
	}
}

func assertValidationError(t *testing.T, err error, code string, diagnostics ...string) {
	t.Helper()

	if err == nil {
		t.Fatalf("ValidateResponseRule() error = nil, want %s", code)
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %T, want *ValidationError", err)
	}
	if validationErr.Code != code {
		t.Fatalf("Code = %q, want %q", validationErr.Code, code)
	}
	joined := strings.Join(validationErr.Diagnostics, "\n")
	for _, diagnostic := range diagnostics {
		if !strings.Contains(joined, diagnostic) {
			t.Fatalf("diagnostics missing %q:\n%s", diagnostic, joined)
		}
	}
}
