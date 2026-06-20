package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netloc8/netloc8-go"
)

// --------------------------------------------------------------------------
// Testing MCP Tool Handlers
// --------------------------------------------------------------------------
// Tests create a mock HTTP server, point a real netloc8.Client at it,
// register tools on an MCP server, connect a client via InMemoryTransport,
// and call tools through the MCP protocol.
//
// This exercises the full chain:
//   MCP client → JSON-RPC → server dispatch → handler → netloc8 SDK → HTTP → mock API
// --------------------------------------------------------------------------

// setupTestServer creates an MCP server + client connected via in-memory
// transport, with all tools registered against a netloc8.Client pointed
// at the given httptest server.
func setupTestServer(t *testing.T, ts *httptest.Server) (*mcp.ClientSession, func()) {
	t.Helper()

	client := netloc8.NewClient("sk_live_test",
		netloc8.WithBaseURL(ts.URL),
		netloc8.WithHTTPClient(ts.Client()),
	)

	server := mcp.NewServer(
		&mcp.Implementation{Name: "test", Version: "0.0.1"},
		nil,
	)
	RegisterTools(server, client, true)
	RegisterPrompts(server)
	RegisterResources(server, client)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	_, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect failed: %v", err)
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}

	return session, func() { session.Close() }
}

// getTextContent extracts the text from a tool result's first content block.
func getTextContent(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("expected content in result, got empty")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", result.Content[0])
	}
	return tc.Text
}



// --------------------------------------------------------------------------
// Registration and Annotation Tests
// --------------------------------------------------------------------------

func TestToolsAreRegistered(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(result.Tools) != 14 {
		t.Fatalf("expected 14 tools, got %d", len(result.Tools))
	}

	expectedNames := []string{
		"geolocate_ip", "geolocate_me", "get_timezone", "validate_ip",
		"is_public_ip", "normalize_ip", "get_subnet",
		"list_api_keys", "get_usage", "get_profile", "get_audit_log",
		"create_api_key", "delete_api_key", "renew_api_key",
	}

	registered := make(map[string]bool)
	for _, tool := range result.Tools {
		registered[tool.Name] = true
	}

	for _, name := range expectedNames {
		if !registered[name] {
			t.Errorf("expected tool '%s' not found", name)
		}
	}
}

func TestToolAnnotations(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	toolMap := make(map[string]*mcp.Tool)
	for _, tool := range result.Tools {
		toolMap[tool.Name] = tool
	}

	// Read-only tools should have ReadOnlyHint = true.
	readOnlyTools := []string{
		"geolocate_ip", "geolocate_me", "get_timezone", "validate_ip",
		"is_public_ip", "normalize_ip", "get_subnet",
		"list_api_keys", "get_usage", "get_profile", "get_audit_log",
	}
	for _, name := range readOnlyTools {
		tool := toolMap[name]
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool '%s' should have ReadOnlyHint=true", name)
		}
	}

	// Write tools should NOT have ReadOnlyHint = true.
	writeTools := []string{"create_api_key", "delete_api_key", "renew_api_key"}
	for _, name := range writeTools {
		tool := toolMap[name]
		if tool.Annotations != nil && tool.Annotations.ReadOnlyHint {
			t.Errorf("tool '%s' should NOT have ReadOnlyHint=true", name)
		}
	}

	// delete_api_key should be explicitly destructive.
	deleteTool := toolMap["delete_api_key"]
	if deleteTool.Annotations == nil || deleteTool.Annotations.DestructiveHint == nil || !*deleteTool.Annotations.DestructiveHint {
		t.Error("delete_api_key should have DestructiveHint=true")
	}

	// create_api_key should be explicitly non-destructive.
	createTool := toolMap["create_api_key"]
	if createTool.Annotations == nil || createTool.Annotations.DestructiveHint == nil || *createTool.Annotations.DestructiveHint {
		t.Error("create_api_key should have DestructiveHint=false")
	}
}

// --------------------------------------------------------------------------
// Integration tests: full handler chain via InMemoryTransport
// --------------------------------------------------------------------------

func TestGeolocateIPTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ip/8.8.8.8" {
			t.Errorf("expected path '/v1/ip/8.8.8.8', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"query":    map[string]any{"type": "ip", "value": "8.8.8.8"},
			"location": map[string]any{"city": "Mountain View", "timezone": "America/Los_Angeles"},
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "geolocate_ip",
		Arguments: map[string]any{"ip": "8.8.8.8"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", getTextContent(t, result))
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "Mountain View") {
		t.Errorf("unexpected result: %s", text)
	}
}

func TestGeolocateIPTool_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "UNAUTHORIZED", "message": "Invalid API key"},
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "geolocate_ip",
		Arguments: map[string]any{"ip": "8.8.8.8"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for API error")
	}
}

func TestGeolocateMeTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ip/me" {
			t.Errorf("expected path '/v1/ip/me', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"query": map[string]any{"value": "203.0.113.42"},
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "geolocate_me",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "203.0.113.42") {
		t.Errorf("expected '203.0.113.42' in result, got: %s", text)
	}
}

func TestGetTimezoneTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ip/8.8.8.8/timezone" {
			t.Errorf("expected path '/v1/ip/8.8.8.8/timezone', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`"America/Los_Angeles"`))
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_timezone",
		Arguments: map[string]any{"ip": "8.8.8.8"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "America/Los_Angeles") {
		t.Errorf("expected 'America/Los_Angeles' in result, got: %s", text)
	}
}

func TestValidateIPTool_Valid(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`true`))
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "validate_ip",
		Arguments: map[string]any{"ip": "8.8.8.8"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "is a valid IP address") {
		t.Errorf("expected 'is a valid IP address', got: %s", text)
	}
}

func TestValidateIPTool_Invalid(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`false`))
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "validate_ip",
		Arguments: map[string]any{"ip": "not-an-ip"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "is NOT a valid IP address") {
		t.Errorf("expected 'is NOT a valid IP address', got: %s", text)
	}
}

func TestGetAuditLogTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/account/me/audit" {
			t.Errorf("expected path '/v1/account/me/audit', got '%s'", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "5" {
			t.Errorf("expected limit=5, got '%s'", r.URL.Query().Get("limit"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"entries": []any{}, "total": 0})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_audit_log",
		Arguments: map[string]any{"limit": float64(5)},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", getTextContent(t, result))
	}
}

func TestCreateAPIKeyTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got '%s'", r.Method)
		}
		if r.URL.Path != "/v1/account/me/keys" {
			t.Errorf("expected path '/v1/account/me/keys', got '%s'", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "My New Key" {
			t.Errorf("expected name='My New Key', got '%v'", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "key_new123", "rawKey": "sk_live_raw_value", "name": "My New Key",
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_api_key",
		Arguments: map[string]any{"name": "My New Key"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "sk_live_raw_value") {
		t.Errorf("expected raw key in result, got: %s", text)
	}
}

func TestDeleteAPIKeyTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got '%s'", r.Method)
		}
		if r.URL.Path != "/v1/account/me/keys/key_abc123" {
			t.Errorf("expected path '/v1/account/me/keys/key_abc123', got '%s'", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_api_key",
		Arguments: map[string]any{"key_id": "key_abc123"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "deleted") {
		t.Errorf("expected 'deleted' in result, got: %s", text)
	}
}

func TestRenewAPIKeyTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got '%s'", r.Method)
		}
		if r.URL.Path != "/v1/account/me/keys/key_abc123/renew" {
			t.Errorf("expected path '/v1/account/me/keys/key_abc123/renew', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "key_abc123", "status": "active", "expiresAt": "2026-12-13T00:00:00Z",
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "renew_api_key",
		Arguments: map[string]any{"key_id": "key_abc123"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "key_abc123") {
		t.Errorf("expected 'key_abc123' in result, got: %s", text)
	}
}

// --------------------------------------------------------------------------
// Missing happy-path tests
// --------------------------------------------------------------------------

func TestListAPIKeysTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/account/me/keys" {
			t.Errorf("expected path '/v1/account/me/keys', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": "key_001", "name": "Production", "status": "active"},
			{"id": "key_002", "name": "Staging", "status": "active"},
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_api_keys",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "Production") {
		t.Errorf("expected 'Production' in result, got: %s", text)
	}
	if !strings.Contains(text, "Staging") {
		t.Errorf("expected 'Staging' in result, got: %s", text)
	}
}

func TestGetUsageTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/account/me/usage" {
			t.Errorf("expected path '/v1/account/me/usage', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"totalKeys": 3, "activeKeys": 2, "totalRequests": 15432, "monthlyCap": 100000,
			"dailyUsage": []map[string]any{},
			"keys":       []map[string]any{},
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_usage",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "15432") {
		t.Errorf("expected '15432' in result, got: %s", text)
	}
	if !strings.Contains(text, "100000") {
		t.Errorf("expected '100000' in result, got: %s", text)
	}
}

func TestGetProfileTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/account/me" {
			t.Errorf("expected path '/v1/account/me', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "usr_abc", "email": "dev@example.com", "name": "Jane Doe",
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_profile",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "dev@example.com") {
		t.Errorf("expected 'dev@example.com' in result, got: %s", text)
	}
}

func TestGetProfileTool_APIKeyMinimal(t *testing.T) {
	// When authenticated via API key, the profile endpoint returns
	// a minimal response with null name/email.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "usr_key123", "name": nil, "email": nil,
			"emailVerified": false, "createdAt": nil,
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_profile",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", getTextContent(t, result))
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "usr_key123") {
		t.Errorf("expected 'usr_key123' in result, got: %s", text)
	}
}

func TestGetUsageTool_NullMonthlyCap(t *testing.T) {
	// Enterprise plans return monthlyCap: null (unlimited).
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"totalKeys": 10, "activeKeys": 8, "totalRequests": 500000,
			"monthlyCap": nil,
			"dailyUsage": []map[string]any{},
			"keys":       []map[string]any{},
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_usage",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", getTextContent(t, result))
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "500000") {
		t.Errorf("expected '500000' in result, got: %s", text)
	}
}

// --------------------------------------------------------------------------
// Audit log without options (empty query string)
// --------------------------------------------------------------------------

func TestGetAuditLogTool_NoOptions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected empty query string, got '%s'", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"entries": []any{}, "total": 0})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_audit_log",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", getTextContent(t, result))
	}
}

// --------------------------------------------------------------------------
// Create key with type option
// --------------------------------------------------------------------------

func TestCreateAPIKeyTool_WithType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["type"] != "publishable" {
			t.Errorf("expected type='publishable', got '%v'", body["type"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "key_pk001", "rawKey": "pk_live_xyz", "name": "Frontend", "type": "publishable",
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_api_key",
		Arguments: map[string]any{"name": "Frontend", "type": "publishable"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "pk_live_xyz") {
		t.Errorf("expected 'pk_live_xyz' in result, got: %s", text)
	}
	if !strings.Contains(text, "publishable") {
		t.Errorf("expected 'publishable' in result, got: %s", text)
	}
}

// --------------------------------------------------------------------------
// Error paths for POST and DELETE tools
// --------------------------------------------------------------------------

func TestCreateAPIKeyTool_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "INVALID_REQUEST", "message": "Name is required"},
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_api_key",
		Arguments: map[string]any{"name": ""},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for 400 response")
	}
}

func TestDeleteAPIKeyTool_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "NOT_FOUND", "message": "Key not found"},
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_api_key",
		Arguments: map[string]any{"key_id": "key_nonexistent"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for 404 response")
	}
}

// --------------------------------------------------------------------------
// Local Utility Tool Tests
// --------------------------------------------------------------------------

func TestIsPublicIPTool_Public(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "is_public_ip",
		Arguments: map[string]any{"ip": "8.8.8.8"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "publicly routable") {
		t.Errorf("expected 'publicly routable', got: %s", text)
	}
}

func TestIsPublicIPTool_Private(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "is_public_ip",
		Arguments: map[string]any{"ip": "192.168.1.1"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "private/non-routable") {
		t.Errorf("expected 'private/non-routable', got: %s", text)
	}
}

func TestNormalizeIPTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "normalize_ip",
		Arguments: map[string]any{"ip": "  8.8.8.8  "},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "8.8.8.8") {
		t.Errorf("expected '8.8.8.8' in output, got: %s", text)
	}
}

func TestNormalizeIPTool_Invalid(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "normalize_ip",
		Arguments: map[string]any{"ip": "not-an-ip"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "Could not normalize") {
		t.Errorf("expected 'Could not normalize', got: %s", text)
	}
}

func TestGetSubnetTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_subnet",
		Arguments: map[string]any{"ip": "8.8.8.8"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "8.8.8.0/24") {
		t.Errorf("expected '8.8.8.0/24', got: %s", text)
	}
}

func TestGetSubnetTool_IPv6(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_subnet",
		Arguments: map[string]any{"ip": "2001:4860:4860::8888"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	text := getTextContent(t, result)
	if !strings.Contains(text, "No /24 subnet") {
		t.Errorf("expected 'No /24 subnet', got: %s", text)
	}
}

// --------------------------------------------------------------------------
// Prompt Registration Tests
// --------------------------------------------------------------------------

func TestPromptsAreRegistered(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.ListPrompts(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}

	if len(result.Prompts) != 4 {
		t.Fatalf("expected 4 prompts, got %d", len(result.Prompts))
	}

	expectedNames := []string{"analyze_ip", "security_audit", "batch_geolocate", "getting_started"}
	registered := make(map[string]bool)
	for _, prompt := range result.Prompts {
		registered[prompt.Name] = true
	}

	for _, name := range expectedNames {
		if !registered[name] {
			t.Errorf("expected prompt '%s' not found", name)
		}
	}
}

func TestAnalyzeIPPrompt(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name:      "analyze_ip",
		Arguments: map[string]string{"ip": "8.8.8.8"},
	})
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}

	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}

	tc, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent in prompt message")
	}
	if !strings.Contains(tc.Text, "8.8.8.8") {
		t.Error("expected prompt message to contain the IP address")
	}
	if !strings.Contains(tc.Text, "geolocate_ip") {
		t.Error("expected prompt to reference geolocate_ip tool")
	}
}

// --------------------------------------------------------------------------
// Resource Template Registration Tests
// --------------------------------------------------------------------------

func TestResourceTemplatesAreRegistered(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	// Check static resources.
	resources, err := session.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListResources failed: %v", err)
	}
	if len(resources.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources.Resources))
	}
	if resources.Resources[0].URI != "netloc8://about" {
		t.Errorf("expected resource URI 'netloc8://about', got %q", resources.Resources[0].URI)
	}

	// Check resource templates.
	result, err := session.ListResourceTemplates(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListResourceTemplates failed: %v", err)
	}

	if len(result.ResourceTemplates) != 1 {
		t.Fatalf("expected 1 resource template, got %d", len(result.ResourceTemplates))
	}

	rt := result.ResourceTemplates[0]
	if rt.URITemplate != "netloc8://ip/{address}" {
		t.Errorf("expected URI template 'netloc8://ip/{address}', got %q", rt.URITemplate)
	}
	if rt.MIMEType != "application/json" {
		t.Errorf("expected MIME type 'application/json', got %q", rt.MIMEType)
	}
}

func TestIPResourceTemplate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"query":    map[string]any{"type": "ip", "value": "8.8.8.8", "ipVersion": 4},
			"location": map[string]any{"country": map[string]any{"code": "US", "name": "United States"}},
			"network":  map[string]any{"asn": "AS15169", "organization": "Google LLC"},
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{
		URI: "netloc8://ip/8.8.8.8",
	})
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Contents))
	}

	content := result.Contents[0]
	if content.MIMEType != "application/json" {
		t.Errorf("expected MIME type 'application/json', got %q", content.MIMEType)
	}
	if !strings.Contains(content.Text, "United States") {
		t.Error("expected resource content to contain 'United States'")
	}
}

func TestAboutResource(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{
		URI: "netloc8://about",
	})
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Contents))
	}

	content := result.Contents[0]
	if content.MIMEType != "text/markdown" {
		t.Errorf("expected MIME type 'text/markdown', got %q", content.MIMEType)
	}
	// Verify key product information is present.
	for _, expected := range []string{"IP Geolocation API", "300+", "netloc8-go", "@netloc8/core", "GDPR", "netloc8.com"} {
		if !strings.Contains(content.Text, expected) {
			t.Errorf("about resource should contain %q", expected)
		}
	}
}

// --------------------------------------------------------------------------
// Tool metadata tests: Title and IdempotentHint
// --------------------------------------------------------------------------

func TestToolsHaveTitles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	for _, tool := range result.Tools {
		if tool.Title == "" {
			t.Errorf("tool '%s' is missing a Title", tool.Name)
		}
	}
}

func TestReadOnlyToolsAreIdempotent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	// All read-only and local tools should have IdempotentHint = true.
	idempotentTools := []string{
		"geolocate_ip", "geolocate_me", "get_timezone", "validate_ip",
		"is_public_ip", "normalize_ip", "get_subnet",
		"list_api_keys", "get_usage", "get_profile", "get_audit_log",
	}
	toolMap := make(map[string]*mcp.Tool)
	for _, tool := range result.Tools {
		toolMap[tool.Name] = tool
	}
	for _, name := range idempotentTools {
		tool := toolMap[name]
		if tool == nil {
			t.Errorf("tool '%s' not found", name)
			continue
		}
		if tool.Annotations == nil || !tool.Annotations.IdempotentHint {
			t.Errorf("tool '%s' should have IdempotentHint=true", name)
		}
	}
}

// --------------------------------------------------------------------------
// Unauthenticated mode: upsell in geo responses
// --------------------------------------------------------------------------

// setupUnauthenticatedServer creates a test server in unauthenticated mode
// (authenticated=false) so geo tools append the signup CTA.
func setupUnauthenticatedServer(t *testing.T, ts *httptest.Server) (*mcp.ClientSession, func()) {
	t.Helper()

	client := netloc8.NewClient("",
		netloc8.WithBaseURL(ts.URL),
		netloc8.WithHTTPClient(ts.Client()),
	)

	server := mcp.NewServer(
		&mcp.Implementation{Name: "test", Version: "0.0.1"},
		nil,
	)
	RegisterTools(server, client, false)
	RegisterPrompts(server)
	RegisterResources(server, client)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	_, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect failed: %v", err)
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}

	return session, func() { session.Close() }
}

func TestUnauthenticatedGeoResponseIncludesUpsell(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"query":    map[string]any{"type": "ip", "value": "8.8.8.8"},
			"location": map[string]any{"country": map[string]any{"code": "US", "name": "United States"}},
			"meta":     map[string]any{"precision": "country", "tier": "anonymous"},
		})
	}))
	defer ts.Close()

	session, cleanup := setupUnauthenticatedServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "geolocate_ip",
		Arguments: map[string]any{"ip": "8.8.8.8"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	// The unauthenticated response should have 2 content blocks:
	// the JSON result + the upsell note.
	if len(result.Content) < 2 {
		t.Fatalf("expected at least 2 content blocks (JSON + upsell), got %d", len(result.Content))
	}

	// The last content block should contain the signup CTA.
	last, ok := result.Content[len(result.Content)-1].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected last content to be *TextContent, got %T", result.Content[len(result.Content)-1])
	}
	if !strings.Contains(last.Text, "netloc8.com") {
		t.Error("upsell note should mention netloc8.com")
	}
	if !strings.Contains(last.Text, "city-level") {
		t.Error("upsell note should mention city-level data")
	}
}

func TestAuthenticatedGeoResponseNoUpsell(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"query":    map[string]any{"type": "ip", "value": "8.8.8.8"},
			"location": map[string]any{"city": "Mountain View", "timezone": "America/Los_Angeles"},
			"meta":     map[string]any{"precision": "city", "tier": "pro"},
		})
	}))
	defer ts.Close()

	session, cleanup := setupTestServer(t, ts)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "geolocate_ip",
		Arguments: map[string]any{"ip": "8.8.8.8"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	// Authenticated mode: should have exactly 1 content block (no upsell).
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block (no upsell), got %d", len(result.Content))
	}
}

// --------------------------------------------------------------------------
// serverInstructions tests
// --------------------------------------------------------------------------

func TestServerInstructionsAuthenticated(t *testing.T) {
	instructions := serverInstructions(true)
	if strings.Contains(instructions, "CURRENT MODE") {
		t.Error("authenticated instructions should not contain 'CURRENT MODE' unauthenticated message")
	}
	if !strings.Contains(instructions, "WHEN TO RECOMMEND") {
		t.Error("instructions should contain 'WHEN TO RECOMMEND'")
	}
}

func TestServerInstructionsUnauthenticated(t *testing.T) {
	instructions := serverInstructions(false)
	if !strings.Contains(instructions, "CURRENT MODE") {
		t.Error("unauthenticated instructions should contain 'CURRENT MODE'")
	}
	if !strings.Contains(instructions, "sign up") || !strings.Contains(instructions, "netloc8.com") {
		t.Error("unauthenticated instructions should mention signing up at netloc8.com")
	}
}

