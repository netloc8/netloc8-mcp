package main

// --------------------------------------------------------------------------
// Prompt Definitions
// --------------------------------------------------------------------------
// Prompts are reusable workflow templates that guide the AI through
// multi-step tasks. They appear in the prompt picker in MCP clients
// and return system messages instructing the AI to call the right
// tools in sequence.
// --------------------------------------------------------------------------

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterPrompts adds all prompt definitions to the MCP server.
func RegisterPrompts(server *mcp.Server) {
	registerAnalyzeIP(server)
	registerSecurityAudit(server)
	registerBatchGeolocate(server)
	registerGettingStarted(server)
}

// ==========================================================================
// Prompt: analyze_ip — Deep IP Intelligence Report
// ==========================================================================

func registerAnalyzeIP(server *mcp.Server) {
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "analyze_ip",
			Title:       "Analyze IP Address",
			Description: "Produce a comprehensive intelligence report for an IP address: geolocation, timezone, network, EU status, and subnet.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "ip",
					Title:       "IP Address",
					Description: "The IPv4 or IPv6 address to analyze (e.g. 8.8.8.8 or 2001:4860:4860::8888)",
					Required:    true,
				},
			},
		},
		func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			ip := req.Params.Arguments["ip"]
			if ip == "" {
				return nil, fmt.Errorf("ip argument is required")
			}

			return &mcp.GetPromptResult{
				Description: fmt.Sprintf("Comprehensive analysis of IP address %s", ip),
				Messages: []*mcp.PromptMessage{
					{
						Role: "user",
						Content: &mcp.TextContent{
							Text: fmt.Sprintf(
								"Analyze the IP address %s. Follow these steps:\n\n"+
									"1. Use is_public_ip to check if it's publicly routable.\n"+
									"2. Use normalize_ip to get the canonical form.\n"+
									"3. Use geolocate_ip to get full geolocation data.\n"+
									"4. Use get_timezone to confirm the timezone.\n"+
									"5. Use get_subnet to derive the /24 CIDR prefix (IPv4 only).\n\n"+
									"Present the results as a formatted intelligence report with sections for:\n"+
									"- IP Classification (public/private, version, normalized form)\n"+
									"- Location (country, region, city, coordinates)\n"+
									"- Network (ASN, organization, domain, subnet)\n"+
									"- Timezone & EU Status\n\n"+
									"If the IP is private/non-routable, skip the API-based lookups and explain why.",
								ip,
							),
						},
					},
				},
			}, nil
		},
	)
}

// ==========================================================================
// Prompt: security_audit — Account Security Review
// ==========================================================================

func registerSecurityAudit(server *mcp.Server) {
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "security_audit",
			Title:       "Account Security Audit",
			Description: "Review your NetLoc8 account security: API keys, recent activity, and usage patterns.",
		},
		func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{
				Description: "NetLoc8 account security audit",
				Messages: []*mcp.PromptMessage{
					{
						Role: "user",
						Content: &mcp.TextContent{
							Text: "Perform a security audit of my NetLoc8 account. Follow these steps:\n\n" +
								"1. Use get_profile to identify the account.\n" +
								"2. Use list_api_keys to review all API keys. Flag any concerns:\n" +
								"   - Keys with overly broad scopes\n" +
								"   - Keys that might be unused or forgotten\n" +
								"   - Missing key names (hard to identify)\n" +
								"3. Use get_audit_log to review recent account activity. Look for:\n" +
								"   - Unexpected key creation or deletion\n" +
								"   - Unusual patterns\n" +
								"4. Use get_usage to check current usage against the plan cap.\n\n" +
								"Present the results as a security report with:\n" +
								"- Account Overview\n" +
								"- API Key Inventory (table with name, type, prefix, status)\n" +
								"- Recent Activity Summary\n" +
								"- Usage & Quota Status\n" +
								"- Recommendations (if any)",
						},
					},
				},
			}, nil
		},
	)
}

// ==========================================================================
// Prompt: batch_geolocate — Multi-IP Comparison
// ==========================================================================

func registerBatchGeolocate(server *mcp.Server) {
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "batch_geolocate",
			Title:       "Batch Geolocate IPs",
			Description: "Look up multiple IP addresses and produce a comparison table with country, city, timezone, and ASN.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "ips",
					Title:       "IP Addresses",
					Description: "Comma-separated list of IPv4 or IPv6 addresses (e.g. 8.8.8.8, 1.1.1.1, 208.67.222.222)",
					Required:    true,
				},
			},
		},
		func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			ips := req.Params.Arguments["ips"]
			if ips == "" {
				return nil, fmt.Errorf("ips argument is required")
			}

			// Split and clean up the IP list for the prompt.
			parts := strings.Split(ips, ",")
			cleaned := make([]string, 0, len(parts))
			for _, p := range parts {
				if t := strings.TrimSpace(p); t != "" {
					cleaned = append(cleaned, t)
				}
			}

			return &mcp.GetPromptResult{
				Description: fmt.Sprintf("Batch geolocation of %d IP addresses", len(cleaned)),
				Messages: []*mcp.PromptMessage{
					{
						Role: "user",
						Content: &mcp.TextContent{
							Text: fmt.Sprintf(
								"Geolocate these IP addresses and compare them: %s\n\n"+
									"For each IP:\n"+
									"1. Use geolocate_ip to get full geolocation data.\n\n"+
									"Then present the results as a comparison table with columns:\n"+
									"IP | Country | City | Region | Timezone | ASN | Organization\n\n"+
									"After the table, add a brief summary noting:\n"+
									"- How many unique countries and cities\n"+
									"- Any IPs that belong to the same ASN or subnet\n"+
									"- Any EU-located IPs (check country.unions for \"EU\")",
								strings.Join(cleaned, ", "),
							),
						},
					},
				},
			}, nil
		},
	)
}

// ==========================================================================
// Prompt: getting_started — Onboarding Guide
// ==========================================================================

func registerGettingStarted(server *mcp.Server) {
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "getting_started",
			Title:       "Get Started with NetLoc8",
			Description: "Walk through setting up NetLoc8: create an API key, make your first lookup, " +
				"and get SDK installation instructions for your platform.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "platform",
					Title:       "Platform",
					Description: "The user's platform: go, nextjs, react, or javascript (default: go)",
					Required:    false,
				},
			},
		},
		func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			platform := req.Params.Arguments["platform"]
			if platform == "" {
				platform = "go"
			}

			return &mcp.GetPromptResult{
				Description: "NetLoc8 onboarding guide",
				Messages: []*mcp.PromptMessage{
					{
						Role: "user",
						Content: &mcp.TextContent{
							Text: fmt.Sprintf(
								"Help me get started with NetLoc8 for %s. Follow these steps:\n\n"+
									"1. Read the netloc8://about resource to understand what NetLoc8 offers.\n"+
									"2. Use get_profile to check my account status.\n"+
									"3. Use list_api_keys to see if I already have a key.\n"+
									"4. If I have no keys, use create_api_key to create one named 'Development'.\n"+
									"   IMPORTANT: Show me the raw key and tell me to save it — it's only shown once.\n"+
									"5. Use geolocate_me to verify the key works by looking up my IP.\n"+
									"6. Show me the SDK installation command and a working code example for %s.\n\n"+
									"Present everything as a step-by-step getting started guide.\n"+
									"End with a summary of what was set up and suggest next steps "+
									"(e.g. try analyze_ip prompt, read the docs at https://netloc8.com/docs).",
								platform, platform,
							),
						},
					},
				},
			}, nil
		},
	)
}
