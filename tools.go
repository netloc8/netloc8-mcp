package main

// --------------------------------------------------------------------------
// Tool Definitions and Handlers
// --------------------------------------------------------------------------
// Each tool wraps a method from the netloc8-go SDK. The SDK returns typed
// Go structs, which we marshal to JSON for the AI to read.
//
// This is intentional: the SDK handles auth, error parsing, and HTTP
// transport. The MCP layer only handles tool registration and result
// formatting.
// --------------------------------------------------------------------------

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netloc8/netloc8-go"
)

// ==========================================================================
// Helpers
// ==========================================================================

func boolPtr(b bool) *bool {
	return &b
}

// upsellNote is appended to geo tool responses in unauthenticated mode.
const upsellNote = "\n\nNote: This result includes country-level data. " +
	"Sign up for a free API key at https://netloc8.com for city-level detail, " +
	"coordinates, timezone, and ASN."

// appendUpsell adds the upsell note to an existing tool result.
func appendUpsell(r *mcp.CallToolResult) {
	r.Content = append(r.Content, &mcp.TextContent{Text: upsellNote})
}

// jsonResult marshals a typed value to indented JSON and wraps it in a
// CallToolResult. This is the bridge between the SDK (typed structs) and
// the MCP protocol (JSON text for the AI).
func jsonResult(v any) *mcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(fmt.Errorf("failed to marshal result: %w", err))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}
}

// errResult wraps an error in a CallToolResult with IsError=true.
func errResult(err error) *mcp.CallToolResult {
	r := &mcp.CallToolResult{}
	r.SetError(err)
	return r
}

// ==========================================================================
// Input Types
// ==========================================================================
// These structs define the JSON Schema for each tool's parameters.
// The MCP SDK auto-generates the schema from struct tags and validates
// the AI's input before calling the handler.

// GeolocateIPInput defines the parameters for the geolocate_ip tool.
type GeolocateIPInput struct {
	IP string `json:"ip" jsonschema:"the IPv4 or IPv6 address to geolocate (e.g. 8.8.8.8 or 2001:4860:4860::8888)"`
}

// EmptyInput is used for tools that take no parameters.
type EmptyInput struct{}

// IPInput is reused by tools that only need an IP address parameter.
type IPInput struct {
	IP string `json:"ip" jsonschema:"the IPv4 or IPv6 address to look up"`
}

// AuditLogInput defines optional filtering/pagination parameters.
type AuditLogInput struct {
	Limit  int    `json:"limit,omitempty"  jsonschema:"maximum number of entries to return (default 20, max 100)"`
	Offset int    `json:"offset,omitempty" jsonschema:"number of entries to skip for pagination (default 0)"`
	Action string `json:"action,omitempty" jsonschema:"filter by action type (e.g. create_key, delete_key)"`
}

// CreateAPIKeyInput defines the parameters for creating a new API key.
type CreateAPIKeyInput struct {
	Name    string `json:"name"           jsonschema:"a human-readable name for the key (e.g. Production Backend)"`
	KeyType string `json:"type,omitempty" jsonschema:"key type: secret (default, full access) or publishable (geo:read only)"`
}

// KeyIDInput is used by tools that operate on a specific API key.
type KeyIDInput struct {
	KeyID string `json:"key_id" jsonschema:"the ID (hash) of the API key — get this from list_api_keys"`
}

// ==========================================================================
// RegisterTools adds all tool definitions and handlers to the MCP server.
// ==========================================================================

func RegisterTools(server *mcp.Server, client *netloc8.Client, authenticated bool) {
	// ── Read-only geo tools (work with or without an API key) ──
	registerGeolocateIP(server, client, authenticated)
	registerGeolocateMe(server, client, authenticated)
	registerGetTimezone(server, client, authenticated)
	registerValidateIP(server)

	// ── Local utility tools (no API calls) ──
	registerIsPublicIP(server)
	registerNormalizeIP(server)
	registerGetSubnet(server)

	// ── Account tools (require API key) ──
	if authenticated {
		registerListAPIKeys(server, client)
		registerGetUsage(server, client)
		registerGetProfile(server, client)
		registerGetAuditLog(server, client)

		registerCreateAPIKey(server, client)
		registerDeleteAPIKey(server, client)
		registerRenewAPIKey(server, client)
	}
}

// ==========================================================================
// Tool: geolocate_ip — SDK: client.LookupIP
// ==========================================================================

func registerGeolocateIP(server *mcp.Server, client *netloc8.Client, authenticated bool) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "geolocate_ip",
			Title: "Geolocate IP Address",
			Description: "Look up geolocation data for an IPv4 or IPv6 address using NetLoc8's edge network. " +
				"Returns country, city, region, coordinates, timezone, ASN, and EU membership status. " +
				"Use this when you need the physical location of an IP address. " +
				"Example: ip=\"8.8.8.8\" → United States, Mountain View, CA, America/Los_Angeles.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input GeolocateIPInput) (*mcp.CallToolResult, any, error) {
			geo, err := client.LookupIP(ctx, input.IP)
			if err != nil {
				return errResult(err), nil, nil
			}
			result := jsonResult(geo)
			if !authenticated {
				appendUpsell(result)
			}
			return result, nil, nil
		},
	)
}

// ==========================================================================
// Tool: geolocate_me — SDK: client.LookupMe
// ==========================================================================

func registerGeolocateMe(server *mcp.Server, client *netloc8.Client, authenticated bool) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "geolocate_me",
			Title: "Geolocate My IP",
			Description: "Look up geolocation data for this machine's public IP address using NetLoc8. " +
				"Returns the same rich data as geolocate_ip (country, city, coordinates, ASN, EU status). " +
				"Useful for determining the developer's location or the server's egress point. " +
				"No arguments required.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, any, error) {
			geo, err := client.LookupMe(ctx)
			if err != nil {
				return errResult(err), nil, nil
			}
			result := jsonResult(geo)
			if !authenticated {
				appendUpsell(result)
			}
			return result, nil, nil
		},
	)
}

// ==========================================================================
// Tool: get_timezone — SDK: client.Timezone
// ==========================================================================

func registerGetTimezone(server *mcp.Server, client *netloc8.Client, authenticated bool) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "get_timezone",
			Title: "Get Timezone",
			Description: "Get the IANA timezone string for an IP address. " +
				"Lighter than a full geolocate_ip call when you only need the timezone. " +
				"Useful for scheduling, date formatting, or timezone-aware code. " +
				"Example: ip=\"8.8.8.8\" → \"America/Los_Angeles\".",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input IPInput) (*mcp.CallToolResult, any, error) {
			tz, err := client.Timezone(ctx, input.IP)
			if err != nil {
				return errResult(err), nil, nil
			}
			msg := fmt.Sprintf("Timezone: %s", tz)
			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: msg},
				},
			}
			if !authenticated {
				appendUpsell(result)
			}
			return result, nil, nil
		},
	)
}

// ==========================================================================
// Tool: validate_ip — SDK: netloc8.ParseIP (local)
// ==========================================================================

func registerValidateIP(server *mcp.Server) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "validate_ip",
			Title: "Validate IP Address",
			Description: "Check if a string is a valid IPv4 or IPv6 address. " +
				"Returns true or false. Useful for input validation before making a geo lookup. " +
				"Example: ip=\"8.8.8.8\" → valid, ip=\"not-an-ip\" → invalid. " +
				"Runs locally — no API call, no quota usage.",
			Annotations: localAnnotations,
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input IPInput) (*mcp.CallToolResult, any, error) {
			_, valid := netloc8.ParseIP(input.IP)

			msg := fmt.Sprintf("%s is NOT a valid IP address", input.IP)
			if valid {
				msg = fmt.Sprintf("%s is a valid IP address", input.IP)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: msg},
				},
			}, nil, nil
		},
	)
}

// ==========================================================================
// Tool: list_api_keys — SDK: client.ListKeys
// ==========================================================================

func registerListAPIKeys(server *mcp.Server, client *netloc8.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "list_api_keys",
			Title: "List API Keys",
			Description: "List all API keys on the authenticated NetLoc8 account. " +
				"Shows key prefix, name, type, scopes, and status. " +
				"Raw key values are never returned — only metadata.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, any, error) {
			keys, err := client.ListKeys(ctx)
			if err != nil {
				return errResult(err), nil, nil
			}
			return jsonResult(keys), nil, nil
		},
	)
}

// ==========================================================================
// Tool: get_usage — SDK: client.GetUsage
// ==========================================================================

func registerGetUsage(server *mcp.Server, client *netloc8.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "get_usage",
			Title: "Get API Usage",
			Description: "Get API usage statistics for the authenticated NetLoc8 account. " +
				"Returns total requests, monthly cap, daily breakdown, and per-key usage. " +
				"Useful for checking if you're close to your plan limit. " +
				"Example: (no input) → Usage summary object.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, any, error) {
			usage, err := client.GetUsage(ctx)
			if err != nil {
				return errResult(err), nil, nil
			}
			return jsonResult(usage), nil, nil
		},
	)
}

// ==========================================================================
// Tool: get_profile — SDK: client.GetProfile
// ==========================================================================

func registerGetProfile(server *mcp.Server, client *netloc8.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "get_profile",
			Title: "Get Account Profile",
			Description: "Get the authenticated user's NetLoc8 profile. " +
				"Returns name, email, and account creation date.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, any, error) {
			profile, err := client.GetProfile(ctx)
			if err != nil {
				return errResult(err), nil, nil
			}
			return jsonResult(profile), nil, nil
		},
	)
}

// ==========================================================================
// Tool: get_audit_log — SDK: client.GetAuditLog
// ==========================================================================

func registerGetAuditLog(server *mcp.Server, client *netloc8.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "get_audit_log",
			Title: "Get Audit Log",
			Description: "Get the account's audit log — a chronological list of account actions. " +
				"Shows key creation, deletion, site changes, and other events. " +
				"Useful for debugging recent changes or security review.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input AuditLogInput) (*mcp.CallToolResult, any, error) {
			// Build SDK options from input fields.
			var opts []netloc8.AuditLogOption
			if input.Limit > 0 {
				opts = append(opts, netloc8.WithLimit(input.Limit))
			}
			if input.Offset > 0 {
				opts = append(opts, netloc8.WithOffset(input.Offset))
			}
			if input.Action != "" {
				opts = append(opts, netloc8.WithAction(input.Action))
			}

			log, err := client.GetAuditLog(ctx, opts...)
			if err != nil {
				return errResult(err), nil, nil
			}
			return jsonResult(log), nil, nil
		},
	)
}

// ==========================================================================
// Tool: create_api_key — SDK: client.CreateKey
// ==========================================================================

func registerCreateAPIKey(server *mcp.Server, client *netloc8.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "create_api_key",
			Title: "Create API Key",
			Description: "Create a new NetLoc8 API key. This is a WRITE OPERATION that modifies your account. " +
				"The raw key value is returned once — store it securely. " +
				"Supports 'secret' keys (full access) and 'publishable' keys (geo:read only, requires allowed origins).",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    false,
				DestructiveHint: boolPtr(false),
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input CreateAPIKeyInput) (*mcp.CallToolResult, any, error) {
			var opts []netloc8.CreateKeyOption
			if input.KeyType != "" {
				opts = append(opts, netloc8.WithKeyType(input.KeyType))
			}

			key, err := client.CreateKey(ctx, input.Name, opts...)
			if err != nil {
				return errResult(err), nil, nil
			}
			return jsonResult(key), nil, nil
		},
	)
}

// ==========================================================================
// Tool: delete_api_key — SDK: client.DeleteKey
// ==========================================================================

func registerDeleteAPIKey(server *mcp.Server, client *netloc8.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "delete_api_key",
			Title: "Delete API Key",
			Description: "Revoke (delete) an API key. This is a DESTRUCTIVE WRITE OPERATION — the key " +
				"immediately stops working and this cannot be undone. " +
				"Use list_api_keys first to find the key ID.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    false,
				DestructiveHint: boolPtr(true),
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input KeyIDInput) (*mcp.CallToolResult, any, error) {
			err := client.DeleteKey(ctx, input.KeyID)
			if err != nil {
				return errResult(err), nil, nil
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("API key %s has been deleted", input.KeyID)},
				},
			}, nil, nil
		},
	)
}

// ==========================================================================
// Tool: renew_api_key — SDK: client.RenewKey
// ==========================================================================

func registerRenewAPIKey(server *mcp.Server, client *netloc8.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "renew_api_key",
			Title: "Renew API Key",
			Description: "Renew an API key by resetting its expiration date. This is a WRITE OPERATION " +
				"that extends the key's validity. Non-destructive. " +
				"Use list_api_keys first to find the key ID.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    false,
				DestructiveHint: boolPtr(false),
				IdempotentHint:  true,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input KeyIDInput) (*mcp.CallToolResult, any, error) {
			key, err := client.RenewKey(ctx, input.KeyID)
			if err != nil {
				return errResult(err), nil, nil
			}
			return jsonResult(key), nil, nil
		},
	)
}

// ==========================================================================
// Local Utility Tools (no API calls — use SDK functions directly)
// ==========================================================================
// These tools run entirely locally using the netloc8-go SDK's utility
// functions. They don't consume API quota or require network access.

var localAnnotations = &mcp.ToolAnnotations{
	ReadOnlyHint:   true,
	IdempotentHint: true,
	OpenWorldHint:  boolPtr(false),
}

// ==========================================================================
// Tool: is_public_ip — SDK: netloc8.IsPublicIP
// ==========================================================================

func registerIsPublicIP(server *mcp.Server) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "is_public_ip",
			Title: "Check Public IP",
			Description: "Check if an IP address is publicly routable. " +
				"Rejects RFC1918, CGNAT, loopback, link-local, and ULA addresses. " +
				"Example: ip=\"8.8.8.8\" → public, ip=\"192.168.1.1\" → private. " +
				"Runs locally — no API call, no quota usage.",
			Annotations: localAnnotations,
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input IPInput) (*mcp.CallToolResult, any, error) {
			isPublic := netloc8.IsPublicIP(input.IP)

			msg := fmt.Sprintf("%s is a private/non-routable IP address", input.IP)
			if isPublic {
				msg = fmt.Sprintf("%s is a publicly routable IP address", input.IP)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: msg},
				},
			}, nil, nil
		},
	)
}

// ==========================================================================
// Tool: normalize_ip — SDK: netloc8.NormalizeIP
// ==========================================================================

func registerNormalizeIP(server *mcp.Server) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "normalize_ip",
			Title: "Normalize IP Address",
			Description: "Normalize an IP address string: strip brackets, remove IPv4-mapped prefix (::ffff:), " +
				"trim whitespace, and lowercase. Runs locally — no API call, no quota usage.",
			Annotations: localAnnotations,
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input IPInput) (*mcp.CallToolResult, any, error) {
			// ParseIP validates and normalizes. NormalizeIP alone doesn't reject invalid strings.
			_, valid := netloc8.ParseIP(input.IP)

			if !valid {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Could not normalize %q — not a valid IP address", input.IP)},
					},
				}, nil, nil
			}

			normalized := netloc8.NormalizeIP(input.IP)
			msg := normalized
			if normalized != input.IP {
				msg = fmt.Sprintf("%s → %s", input.IP, normalized)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: msg},
				},
			}, nil, nil
		},
	)
}

// ==========================================================================
// Tool: get_subnet — SDK: netloc8.Subnet
// ==========================================================================

func registerGetSubnet(server *mcp.Server) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:  "get_subnet",
			Title: "Get Subnet",
			Description: "Derive the /24 CIDR prefix from an IPv4 address (e.g. 8.8.8.8 → 8.8.8.0/24). " +
				"Returns empty for IPv6 addresses. " +
				"Runs locally — no API call, no quota usage.",
			Annotations: localAnnotations,
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input IPInput) (*mcp.CallToolResult, any, error) {
			subnet := netloc8.Subnet(input.IP)

			if subnet == "" {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("No /24 subnet for %s (IPv6 addresses and invalid inputs return empty)", input.IP)},
					},
				}, nil, nil
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Subnet: %s", subnet)},
				},
			}, nil, nil
		},
	)
}
