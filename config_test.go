package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netloc8/netloc8-go"
)

func TestLoadConfig_Success(t *testing.T) {
	t.Setenv("NETLOC8_API_KEY", "sk_live_test123")
	t.Setenv("NETLOC8_API_URL", "https://custom.api.example.com")

	cfg := LoadConfig()

	if cfg.APIKey != "sk_live_test123" {
		t.Errorf("expected APIKey = 'sk_live_test123', got: '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://custom.api.example.com" {
		t.Errorf("expected BaseURL = 'https://custom.api.example.com', got: '%s'", cfg.BaseURL)
	}
	if !cfg.Authenticated {
		t.Error("expected Authenticated = true when API key is set")
	}
}

func TestLoadConfig_DefaultBaseURL(t *testing.T) {
	t.Setenv("NETLOC8_API_KEY", "sk_live_test123")
	os.Unsetenv("NETLOC8_API_URL")

	cfg := LoadConfig()

	if cfg.BaseURL != netloc8.DefaultBaseURL {
		t.Errorf("expected default BaseURL = '%s', got: '%s'", netloc8.DefaultBaseURL, cfg.BaseURL)
	}
}

func TestLoadConfig_MissingAPIKey(t *testing.T) {
	os.Unsetenv("NETLOC8_API_KEY")

	cfg := LoadConfig()

	if cfg.APIKey != "" {
		t.Errorf("expected empty APIKey, got: '%s'", cfg.APIKey)
	}
	if cfg.Authenticated {
		t.Error("expected Authenticated = false when no API key is set")
	}
	if cfg.BaseURL == "" {
		t.Error("expected BaseURL to have a default value")
	}
}

func TestLoadConfig_UnauthenticatedToolCount(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

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
	defer session.Close()

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	// 4 geo tools + 3 local tools = 7
	if len(result.Tools) != 7 {
		t.Fatalf("expected 7 tools in unauthenticated mode, got %d", len(result.Tools))
	}

	// Account tools should NOT be present.
	for _, tool := range result.Tools {
		switch tool.Name {
		case "list_api_keys", "get_usage", "get_profile", "get_audit_log",
			"create_api_key", "delete_api_key", "renew_api_key":
			t.Errorf("account tool '%s' should not be registered in unauthenticated mode", tool.Name)
		}
	}
}
