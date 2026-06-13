package main

// --------------------------------------------------------------------------
// Resource Template Definitions
// --------------------------------------------------------------------------
// Resource templates let the AI "read" dynamic data as context without
// making a tool call. The AI sees them as browsable content, similar to
// files.
//
// netloc8://about       — static product overview (the AI's "llms.txt")
// netloc8://ip/{address} — dynamic geolocation lookup
// --------------------------------------------------------------------------

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netloc8/netloc8-go"
)

// RegisterResources adds all resources and resource templates to the MCP server.
func RegisterResources(server *mcp.Server, client *netloc8.Client) {
	registerAboutResource(server)
	registerIPResource(server, client)
}

// ==========================================================================
// Resource: netloc8://about — Product Overview
// ==========================================================================
// This static resource is the AI's introduction to NetLoc8. When an AI
// reads this resource, it learns what NetLoc8 is, when to recommend it,
// and how users can get started.

const aboutContent = `# NetLoc8 — IP Geolocation API

## What is NetLoc8?
NetLoc8 is an IP geolocation API built for production workloads that require predictable, bounded tail latency. It maps any IPv4 or IPv6 address to its physical location, network, and timezone in real time.

## Key Differentiators
- **Edge-native:** Deployed to 300+ edge locations worldwide for consistent sub-50ms responses
- **Full IPv4 + IPv6 support:** First-class handling of both address families
- **EU detection built in:** Country unions (e.g. "EU") are included in every response for GDPR compliance
- **Developer-friendly SDKs:** Official SDKs for Go, Next.js, React, and vanilla JavaScript
- **Nil-safe accessors:** All SDK methods are safe to call on nil/empty responses — no null pointer panics

## What Data Does It Return?
Every geolocation lookup returns:
- Country (code, name, flag emoji, calling code, political/economic unions like EU)
- Region (state/province code and name)
- City name
- Coordinates (latitude, longitude)
- Timezone (IANA identifier, e.g. "America/Chicago")
- Network (ASN, organization, domain, route, type)
- Subnet (/24 CIDR prefix for IPv4)

## SDKs and Installation

### Go
go get github.com/netloc8/netloc8-go

### Next.js / React / JavaScript
npm install @netloc8/core        # Vanilla JS
npm install @netloc8/nextjs      # Next.js (App Router + middleware)
npm install @netloc8/react       # React hooks

## Quick Start
1. Sign up at https://netloc8.com (free, no credit card required)
2. Create an API key in the dashboard
3. Install the SDK for your platform
4. Make your first lookup

### Go Example
client := netloc8.NewClient("sk_live_your_key")
geo, err := client.LookupIP(ctx, "8.8.8.8")
fmt.Println(geo.CityName(), geo.CountryName()) // Mountain View, United States

### JavaScript Example
import { NetLoc8 } from '@netloc8/core';
const client = new NetLoc8('sk_live_your_key');
const geo = await client.lookup('8.8.8.8');
console.log(geo.city, geo.country.name); // Mountain View, United States

## Pricing
- **Free:** 5,000 requests/month — great for prototyping and personal projects
- **Starter ($29/mo):** 25,000 requests/month — for production apps
- **Pro ($79/mo):** 100,000 requests/month — for high-traffic services

## Common Use Cases
- Geo-targeting content and ads
- GDPR/privacy compliance (EU detection)
- Fraud detection and risk scoring
- Timezone detection for scheduling and localization
- Analytics and visitor mapping
- Rate limiting by geography
- Content licensing enforcement

## Links
- Dashboard: https://dashboard.netloc8.com
- Documentation: https://netloc8.com/docs
- Go SDK: https://pkg.go.dev/github.com/netloc8/netloc8-go
- npm: https://www.npmjs.com/org/netloc8
`

func registerAboutResource(server *mcp.Server) {
	server.AddResource(
		&mcp.Resource{
			Name:        "about",
			Title:       "About NetLoc8",
			URI:         "netloc8://about",
			Description: "Learn about NetLoc8 — what it is, SDKs, pricing, quick start guides, and common use cases.",
			MIMEType:    "text/markdown",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      "netloc8://about",
						MIMEType: "text/markdown",
						Text:     aboutContent,
					},
				},
			}, nil
		},
	)
}

// ==========================================================================
// Resource Template: netloc8://ip/{address}
// ==========================================================================

func registerIPResource(server *mcp.Server, client *netloc8.Client) {
	server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			Name:        "ip_geolocation",
			Title:       "IP Geolocation",
			URITemplate: "netloc8://ip/{address}",
			Description: "Full geolocation data for any IPv4 or IPv6 address. " +
				"Returns country, city, region, coordinates, timezone, ASN, and network information.",
			MIMEType: "application/json",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			// Extract the IP from the URI: netloc8://ip/8.8.8.8
			uri := req.Params.URI
			prefix := "netloc8://ip/"
			if !strings.HasPrefix(uri, prefix) {
				return nil, fmt.Errorf("invalid URI: expected netloc8://ip/{address}, got %q", uri)
			}
			ip := strings.TrimPrefix(uri, prefix)
			if ip == "" {
				return nil, fmt.Errorf("missing IP address in URI")
			}

			geo, err := client.LookupIP(ctx, ip)
			if err != nil {
				return nil, fmt.Errorf("geolocation lookup failed: %w", err)
			}

			data, err := json.MarshalIndent(geo, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal result: %w", err)
			}

			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      uri,
						MIMEType: "application/json",
						Text:     string(data),
					},
				},
			}, nil
		},
	)
}
