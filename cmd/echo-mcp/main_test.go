package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"echo-mcp/internal/control"
	"echo-mcp/internal/state"
)

func TestHTTPAddrFromEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "default address when unset",
			env:  map[string]string{},
			want: ":8080",
		},
		{
			name: "configured address when set",
			env: map[string]string{
				"ECHO_MCP_HTTP_ADDR": "127.0.0.1:18080",
			},
			want: "127.0.0.1:18080",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := httpAddrFromEnvironment(func(key string) string {
				return tt.env[key]
			})

			if got != tt.want {
				t.Fatalf("httpAddrFromEnvironment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadContractManagerFromEnvironmentReportsStartupLoadedContract(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourcePath := copyStartupContractFixture(t, root, filepath.Join("contracts", "payment-intent-openapi.json"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	manager, err := loadContractManagerFromEnvironment(func(key string) string {
		if key == "ECHO_MCP_OPENAPI_FILE" {
			return filepath.Join("contracts", "payment-intent-openapi.json")
		}
		if key == "ECHO_MCP_CONTRACT_ROOT" {
			return root
		}
		return ""
	}, logger)
	if err != nil {
		t.Fatalf("loadContractManagerFromEnvironment() error = %v", err)
	}

	status := manager.Status()
	if !status.Active {
		t.Fatal("status.Active = false, want true")
	}
	if status.SourcePath != filepath.Join("contracts", "payment-intent-openapi.json") {
		t.Fatalf("SourcePath = %q, want relative source path", status.SourcePath)
	}
	if !status.ContractRootConfigured {
		t.Fatal("ContractRootConfigured = false, want true")
	}
	if status.ContractRootSource != "ECHO_MCP_CONTRACT_ROOT" {
		t.Fatalf("ContractRootSource = %q, want ECHO_MCP_CONTRACT_ROOT", status.ContractRootSource)
	}
	if status.ValidationMode != control.ValidationModeStrict {
		t.Fatalf("ValidationMode = %q, want strict", status.ValidationMode)
	}

	plane := control.NewWithContractManager(state.New(), manager, nil)
	if err := plane.ConfigureResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"ok":true}`,
	}); err == nil {
		t.Fatal("ConfigureResponseRule() error = nil, want startup-loaded contract validation rejection")
	}

	if _, err := os.Stat(sourcePath); err != nil {
		t.Fatalf("copied startup fixture not found: %v", err)
	}
}

func TestLoadContractManagerFromEnvironmentRejectsOutsideContractRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outsideRoot := t.TempDir()
	outsidePath := copyStartupContractFixture(t, outsideRoot, "outside-openapi.json")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	_, err := loadContractManagerFromEnvironment(func(key string) string {
		if key == "ECHO_MCP_OPENAPI_FILE" {
			return outsidePath
		}
		if key == "ECHO_MCP_CONTRACT_ROOT" {
			return root
		}
		return ""
	}, logger)
	if err == nil {
		t.Fatal("loadContractManagerFromEnvironment() error = nil, want outside-root rejection")
	}
	if !strings.Contains(err.Error(), "contract_path_not_allowed") {
		t.Fatalf("error = %q, want contract_path_not_allowed", err)
	}
}

func copyStartupContractFixture(t *testing.T, root string, relativePath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "contract", "testdata", "payment-intent-openapi.json"))
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
	return fullPath
}
