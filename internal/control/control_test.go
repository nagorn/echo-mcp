package control

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"echo-mcp/internal/state"
)

func TestLocalPlaneIdentifiesMCPControlPlane(t *testing.T) {
	var plane Plane = New(state.New())

	if got := plane.Protocol(); got != "mcp" {
		t.Fatalf("Protocol() = %q, want %q", got, "mcp")
	}
}

func TestLocalPlaneConfiguresResponseRuleInMemory(t *testing.T) {
	store := state.New()
	plane := New(store)

	rule := state.ResponseRule{
		ID:         "rule-payment-ok",
		Method:     http.MethodGet,
		Path:       "/payments/123",
		StatusCode: http.StatusAccepted,
		Body:       `{"status":"accepted"}`,
	}

	if err := plane.ConfigureResponseRule(rule); err != nil {
		t.Fatalf("ConfigureResponseRule() error = %v", err)
	}

	matched, ok := store.MatchResponseRule(http.MethodGet, "/payments/123")
	if !ok {
		t.Fatal("MatchResponseRule() ok = false, want true")
	}
	if matched != rule {
		t.Fatalf("matched rule = %+v, want %+v", matched, rule)
	}
}

func TestLocalPlaneReportsInactiveContractStatus(t *testing.T) {
	plane := New(state.New())

	status := plane.ContractStatus()
	if status.Active {
		t.Fatal("ContractStatus().Active = true, want false")
	}
	if status.Message != "No OpenAPI contract is currently loaded." {
		t.Fatalf("message = %q", status.Message)
	}
}

func TestLocalPlaneLoadsContractAndReportsStatus(t *testing.T) {
	plane, sourcePath := newTestPlaneWithContractRoot(t)

	result, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{
		Path:           sourcePath,
		ContractID:     "stripe",
		ValidationMode: ValidationModeStrict,
	})
	if err != nil {
		t.Fatalf("LoadOpenAPIContract() error = %v", err)
	}
	if !result.Loaded {
		t.Fatal("Loaded = false, want true")
	}
	if result.ContractID != "stripe" {
		t.Fatalf("ContractID = %q, want stripe", result.ContractID)
	}
	if result.OpenAPIVersion != "3.0.3" {
		t.Fatalf("OpenAPIVersion = %q, want 3.0.3", result.OpenAPIVersion)
	}
	if result.OperationsCount != 1 {
		t.Fatalf("OperationsCount = %d, want 1", result.OperationsCount)
	}
	if result.SchemasCount != 0 {
		t.Fatalf("SchemasCount = %d, want 0 component schemas", result.SchemasCount)
	}

	status := plane.ContractStatus()
	if !status.Active {
		t.Fatal("ContractStatus().Active = false, want true")
	}
	if status.ContractID != "stripe" {
		t.Fatalf("status.ContractID = %q, want stripe", status.ContractID)
	}
	if status.ValidationMode != ValidationModeStrict {
		t.Fatalf("status.ValidationMode = %q, want strict", status.ValidationMode)
	}
	if status.LoadedAt.IsZero() {
		t.Fatal("LoadedAt is zero, want load timestamp")
	}
}

func TestLocalPlaneRejectsContractSwitchWhenBehaviorIsActive(t *testing.T) {
	store := state.New()
	plane, sourcePath := newTestPlaneWithContractRootAndStore(t, store)
	if _, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{Path: sourcePath, ContractID: "stripe"}); err != nil {
		t.Fatalf("LoadOpenAPIContract() error = %v", err)
	}

	if err := plane.ConfigureResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusPaymentRequired,
		ContentType: "application/json",
		Body:        `{"error":{"type":"card_error","code":"card_declined","decline_code":"generic_decline","message":"Your card was declined.","payment_intent":{"id":"pi_123","object":"payment_intent","status":"requires_payment_method"}}}`,
	}); err != nil {
		t.Fatalf("ConfigureResponseRule() error = %v", err)
	}

	_, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{Path: sourcePath, ContractID: "stripe-v2"})
	if err == nil {
		t.Fatal("LoadOpenAPIContract() error = nil, want active behavior rejection")
	}
	if !strings.Contains(err.Error(), "reset_required") {
		t.Fatalf("error = %q, want reset_required", err)
	}
}

func TestLocalPlaneRejectsUnloadWhenBehaviorIsActiveUnlessForced(t *testing.T) {
	store := state.New()
	plane, sourcePath := newTestPlaneWithContractRootAndStore(t, store)
	if _, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{Path: sourcePath, ContractID: "stripe"}); err != nil {
		t.Fatalf("LoadOpenAPIContract() error = %v", err)
	}
	if err := plane.ConfigureResponseRule(state.ResponseRule{
		ID:          "stripe-like-paymentintent-card-declined",
		Method:      http.MethodPost,
		Path:        "/v1/payment_intents/pi_123/confirm",
		StatusCode:  http.StatusPaymentRequired,
		ContentType: "application/json",
		Body:        `{"error":{"type":"card_error","code":"card_declined","decline_code":"generic_decline","message":"Your card was declined.","payment_intent":{"id":"pi_123","object":"payment_intent","status":"requires_payment_method"}}}`,
	}); err != nil {
		t.Fatalf("ConfigureResponseRule() error = %v", err)
	}

	_, err := plane.UnloadOpenAPIContract(UnloadOpenAPIContractCommand{})
	if err == nil {
		t.Fatal("UnloadOpenAPIContract() error = nil, want active behavior rejection")
	}
	if !strings.Contains(err.Error(), "reset_required") {
		t.Fatalf("error = %q, want reset_required", err)
	}

	result, err := plane.UnloadOpenAPIContract(UnloadOpenAPIContractCommand{Force: true})
	if err != nil {
		t.Fatalf("UnloadOpenAPIContract(force) error = %v", err)
	}
	if !result.Unloaded {
		t.Fatal("Unloaded = false, want true")
	}
	if result.PreviousContractID != "stripe" {
		t.Fatalf("PreviousContractID = %q, want stripe", result.PreviousContractID)
	}
	if plane.ContractStatus().Active {
		t.Fatal("ContractStatus().Active = true after forced unload, want false")
	}
}

func TestLocalPlaneLoadsRelativePathInsideContractRoot(t *testing.T) {
	plane, sourcePath := newTestPlaneWithContractRoot(t)

	result, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{Path: sourcePath})
	if err != nil {
		t.Fatalf("LoadOpenAPIContract() error = %v", err)
	}
	if !result.Loaded {
		t.Fatal("Loaded = false, want true")
	}
	if result.SourcePath != sourcePath {
		t.Fatalf("SourcePath = %q, want relative source path %q", result.SourcePath, sourcePath)
	}

	status := plane.ContractStatus()
	if !status.ContractRootConfigured {
		t.Fatal("ContractRootConfigured = false, want true")
	}
	if status.ContractRootSource != ContractRootSourceEnv {
		t.Fatalf("ContractRootSource = %q, want %q", status.ContractRootSource, ContractRootSourceEnv)
	}
}

func TestLocalPlaneLoadsAbsolutePathInsideContractRoot(t *testing.T) {
	root := t.TempDir()
	sourcePath := writeContractFixture(t, root, filepath.Join("contracts", "payment-intent-openapi.json"))
	manager, err := NewContractManagerWithContractRoot(root, ContractRootSourceEnv)
	if err != nil {
		t.Fatalf("NewContractManagerWithContractRoot() error = %v", err)
	}
	plane := NewWithContractManager(state.New(), manager, nil)

	result, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{Path: sourcePath})
	if err != nil {
		t.Fatalf("LoadOpenAPIContract() error = %v", err)
	}
	if result.SourcePath != filepath.Join("contracts", "payment-intent-openapi.json") {
		t.Fatalf("SourcePath = %q, want relative display path", result.SourcePath)
	}
	if strings.Contains(result.SourcePath, root) {
		t.Fatalf("SourcePath leaks contract root: %q", result.SourcePath)
	}
}

func TestLocalPlaneNormalizesInRootTraversalSourcePath(t *testing.T) {
	root := t.TempDir()
	relativePath := filepath.Join("contracts", "payment-intent-openapi.json")
	writeContractFixture(t, root, relativePath)
	manager, err := NewContractManagerWithContractRoot(root, ContractRootSourceEnv)
	if err != nil {
		t.Fatalf("NewContractManagerWithContractRoot() error = %v", err)
	}
	plane := NewWithContractManager(state.New(), manager, nil)
	traversalPath := filepath.Join("..", filepath.Base(root), relativePath)

	result, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{Path: traversalPath})
	if err != nil {
		t.Fatalf("LoadOpenAPIContract() error = %v", err)
	}
	if result.SourcePath != relativePath {
		t.Fatalf("SourcePath = %q, want normalized relative display path %q", result.SourcePath, relativePath)
	}
	if strings.Contains(result.SourcePath, "..") || strings.Contains(result.SourcePath, root) {
		t.Fatalf("SourcePath leaks traversal or contract root: %q", result.SourcePath)
	}
}

func TestLocalPlaneRejectsTraversalOutsideContractRoot(t *testing.T) {
	plane, _ := newTestPlaneWithContractRoot(t)

	_, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{Path: filepath.Join("..", "secrets.json")})
	assertContractPathNotAllowed(t, err)
}

func TestLocalPlaneRejectsAbsolutePathOutsideContractRoot(t *testing.T) {
	plane, _ := newTestPlaneWithContractRoot(t)
	outsidePath := filepath.Join(t.TempDir(), "outside-openapi.json")
	if err := os.WriteFile(outsidePath, []byte(`{"openapi":"3.0.3","paths":{}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{Path: outsidePath})
	assertContractPathNotAllowed(t, err)
}

func TestLocalPlaneRejectsSymlinkEscapeFromContractRoot(t *testing.T) {
	root := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "outside-openapi.json")
	if err := os.WriteFile(outsidePath, []byte(`{"openapi":"3.0.3","paths":{"/x":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"type":"object"}}}}}}}}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	linkPath := filepath.Join(root, "linked-openapi.json")
	if err := os.Symlink(outsidePath, linkPath); err != nil {
		t.Skipf("Symlink() unsupported: %v", err)
	}
	manager, err := NewContractManagerWithContractRoot(root, ContractRootSourceEnv)
	if err != nil {
		t.Fatalf("NewContractManagerWithContractRoot() error = %v", err)
	}
	plane := NewWithContractManager(state.New(), manager, nil)

	_, err = plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{Path: "linked-openapi.json"})
	assertContractPathNotAllowed(t, err)
}

func TestLocalPlanePreservesUnreadableDiagnosticsInsideContractRoot(t *testing.T) {
	plane, _ := newTestPlaneWithContractRoot(t)

	_, err := plane.LoadOpenAPIContract(LoadOpenAPIContractCommand{Path: "missing-openapi.json"})
	if err == nil {
		t.Fatal("LoadOpenAPIContract() error = nil, want unreadable file")
	}
	if strings.Contains(err.Error(), "contract_path_not_allowed") {
		t.Fatalf("error = %q, want normal unreadable diagnostics for inside-root path", err)
	}
}

func TestLocalPlaneManualConfigureBehaviorUnchangedWithoutContract(t *testing.T) {
	store := state.New()
	plane, _ := newTestPlaneWithContractRootAndStore(t, store)
	rule := state.ResponseRule{
		ID:          "manual-response",
		Method:      http.MethodGet,
		Path:        "/manual",
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        `{"ok":true}`,
	}

	if err := plane.ConfigureResponseRule(rule); err != nil {
		t.Fatalf("ConfigureResponseRule() error = %v", err)
	}
	if _, ok := store.MatchResponseRule(http.MethodGet, "/manual"); !ok {
		t.Fatal("MatchResponseRule() ok = false, want configured manual rule")
	}
}

func newTestPlaneWithContractRoot(t *testing.T) (*LocalPlane, string) {
	t.Helper()
	return newTestPlaneWithContractRootAndStore(t, state.New())
}

func newTestPlaneWithContractRootAndStore(t *testing.T, store *state.Store) (*LocalPlane, string) {
	t.Helper()
	root := t.TempDir()
	writeContractFixture(t, root, filepath.Join("contracts", "payment-intent-openapi.json"))
	manager, err := NewContractManagerWithContractRoot(root, ContractRootSourceEnv)
	if err != nil {
		t.Fatalf("NewContractManagerWithContractRoot() error = %v", err)
	}
	return NewWithContractManager(store, manager, nil), filepath.Join("contracts", "payment-intent-openapi.json")
}

func writeContractFixture(t *testing.T, root string, relativePath string) string {
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
	return fullPath
}

func assertContractPathNotAllowed(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("LoadOpenAPIContract() error = nil, want contract path rejection")
	}
	var operationErr *OperationError
	if !errors.As(err, &operationErr) {
		t.Fatalf("error = %T, want *OperationError", err)
	}
	if operationErr.Code != "contract_path_not_allowed" {
		t.Fatalf("Code = %q, want contract_path_not_allowed", operationErr.Code)
	}
	if operationErr.Message != "contract path is outside the allowed contract root" {
		t.Fatalf("Message = %q, want outside-root message", operationErr.Message)
	}
	if len(operationErr.Diagnostics) != 1 || operationErr.Diagnostics[0] != "OpenAPI contract paths must resolve under the configured contract root." {
		t.Fatalf("Diagnostics = %+v, want safe outside-root diagnostic", operationErr.Diagnostics)
	}
}
