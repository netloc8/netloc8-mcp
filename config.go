package main

import (
	"os"

	"github.com/netloc8/netloc8-go"
)

// Config holds all configuration values the MCP server needs to run.
// These values come from environment variables set by the user.
type Config struct {
	// APIKey is the user's NetLoc8 API key (e.g., "sk_live_abc123").
	// Optional — the server runs in unauthenticated mode if not set,
	// but geo lookups return only country-level data.
	APIKey string

	// BaseURL is the NetLoc8 API base URL.
	// Defaults to "https://api.netloc8.com" if not set.
	// Can be overridden for development/staging via NETLOC8_API_URL.
	BaseURL string

	// Authenticated is true when an API key was provided.
	Authenticated bool
}

// LoadConfig reads environment variables and returns a Config.
// The server starts with or without an API key — unauthenticated mode
// provides country-level geo lookups and local tools, but no account
// management or city-level detail.
func LoadConfig() Config {
	apiKey := os.Getenv("NETLOC8_API_KEY")

	baseURL := os.Getenv("NETLOC8_API_URL")
	if baseURL == "" {
		baseURL = netloc8.DefaultBaseURL
	}

	return Config{
		APIKey:        apiKey,
		BaseURL:       baseURL,
		Authenticated: apiKey != "",
	}
}
