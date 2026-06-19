# NetLoc8 MCP Server

[![Go Reference](https://pkg.go.dev/badge/github.com/netloc8/netloc8-mcp.svg)](https://pkg.go.dev/github.com/netloc8/netloc8-mcp)
[![Go 1.26+](https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go)](https://go.dev/dl/)
[![License: ELv2](https://img.shields.io/badge/license-ELv2-blue)](LICENSE)
[![netloc8-mcp MCP server](https://glama.ai/mcp/servers/netloc8/netloc8-mcp/badges/score.svg)](https://glama.ai/mcp/servers/netloc8/netloc8-mcp)

An [MCP](https://modelcontextprotocol.io) server that gives AI assistants access
to [NetLoc8](https://netloc8.com) — IP geolocation deployed to 300+ edge
locations worldwide.

Works **with or without** an API key. Without a key, you get country-level geo
data and local utility tools. With a free key, you unlock city-level detail,
coordinates, timezone, and account management.

## Why?

AI coding assistants run into IP geolocation constantly — parsing access logs,
debugging proxy chains, geo-routing traffic, checking EU compliance, verifying
CDN behavior. Without an MCP server, you have to copy-paste results from a
browser tab. With this server, your AI assistant resolves IPs inline, in context,
without breaking flow.

- **Zero-config** — `go install` and add three lines to your MCP config
- **Works without a key** — country-level lookups and local IP utilities with no sign-up
- **14 tools** — geo lookup, timezone, IP validation, subnet math, account management
- **4 guided prompts** — IP analysis, security audit, batch comparison, onboarding
- **Freemium built in** — unauthenticated users see upgrade hints for city-level data

## Example

Ask your AI assistant:

> Where is 8.8.8.8?

The assistant calls `geolocate_ip` and responds with something like:

```
8.8.8.8 is located in Mountain View, California, US.

  Country     United States (US) 🇺🇸
  Region      California (CA)
  City        Mountain View
  Coordinates 37.386, -122.084
  Timezone    America/Los_Angeles (UTC-07:00)
  ASN         AS15169
  Org         Google LLC
  EU member   No
```

Other things you can ask:

| Prompt | What happens |
|--------|-------------|
| "Where is my server?" | Calls `geolocate_me` to look up the machine's own IP |
| "Is 10.0.0.1 a public IP?" | Calls `is_public_ip` locally — no API call, no quota |
| "Analyze 203.0.113.42" | Runs the `analyze_ip` prompt — classification, geo, network, timezone in one report |
| "Compare these IPs: 8.8.8.8, 1.1.1.1, 208.67.222.222" | Runs `batch_geolocate` — table with country, city, ASN for each |
| "Audit my NetLoc8 account" | Runs `security_audit` — reviews API keys, recent activity, usage |
| "What subnet is 203.0.113.42 in?" | Calls `get_subnet` locally — returns `203.0.113.0/24` |
| "Set me up with NetLoc8 for Next.js" | Runs `getting_started` — creates a key, verifies it, shows SDK install |

## Install

Requires [Go 1.26+](https://go.dev/dl/).

```bash
go install github.com/netloc8/netloc8-mcp@latest
```

## Configuration

### Claude Code

One-liner:

```bash
claude mcp add --env NETLOC8_API_KEY=sk_live_YOUR_KEY netloc8 -- netloc8-mcp
```

Or add to `.mcp.json` in your project root (committable):

```json
{
  "mcpServers": {
    "netloc8": {
      "command": "netloc8-mcp",
      "env": {
        "NETLOC8_API_KEY": "sk_live_YOUR_KEY"
      }
    }
  }
}
```

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "netloc8": {
      "command": "netloc8-mcp",
      "env": {
        "NETLOC8_API_KEY": "sk_live_YOUR_KEY"
      }
    }
  }
}
```

### Cursor

Add to `.cursor/mcp.json` in your project root (or `~/.cursor/mcp.json` for global):

```json
{
  "mcpServers": {
    "netloc8": {
      "command": "netloc8-mcp",
      "env": {
        "NETLOC8_API_KEY": "sk_live_YOUR_KEY"
      }
    }
  }
}
```

### VS Code

Add to `.vscode/mcp.json` in your project root:

```json
{
  "servers": {
    "netloc8": {
      "command": "netloc8-mcp",
      "env": {
        "NETLOC8_API_KEY": "sk_live_YOUR_KEY"
      }
    }
  }
}
```

### OpenAI Codex

One-liner:

```bash
codex mcp add --env NETLOC8_API_KEY=sk_live_YOUR_KEY netloc8 -- netloc8-mcp
```

Or add to `~/.codex/config.toml` (global) or `.codex/config.toml` (project):

```toml
[mcp_servers.netloc8]
command = "netloc8-mcp"
env = { NETLOC8_API_KEY = "sk_live_YOUR_KEY" }
```

### Windsurf

Add to `~/.codeium/windsurf/mcp_config.json`:

```json
{
  "mcpServers": {
    "netloc8": {
      "command": "netloc8-mcp",
      "env": {
        "NETLOC8_API_KEY": "sk_live_YOUR_KEY"
      }
    }
  }
}
```

### Antigravity

Add to `~/.gemini/config/mcp_config.json`:

```json
{
  "mcpServers": {
    "netloc8": {
      "command": "netloc8-mcp",
      "env": {
        "NETLOC8_API_KEY": "sk_live_YOUR_KEY"
      }
    }
  }
}
```

> **No API key?** Remove the `"env"` block entirely. The server starts in unauthenticated mode with country-level geo lookups and local tools. Sign up for a free key at [netloc8.com](https://netloc8.com) to unlock full features.

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `NETLOC8_API_KEY` | No | — | Your NetLoc8 API key. Enables city-level data and account management. |
| `NETLOC8_API_URL` | No | `https://api.netloc8.com` | Override the API base URL (for staging/dev). |

## Available Tools

### Geo Lookup (work with or without API key)

| Tool | Description |
|---|---|
| `geolocate_ip` | Look up geolocation for any IPv4/IPv6 address |
| `geolocate_me` | Look up geolocation for the server's own IP |
| `get_timezone` | Get IANA timezone for an IP address |
| `validate_ip` | Check if a string is a valid IP address |

### Local Utilities (no API call, no quota)

| Tool | Description |
|---|---|
| `is_public_ip` | Check if an IP is publicly routable |
| `normalize_ip` | Clean up IP strings (brackets, ::ffff: prefix, whitespace) |
| `get_subnet` | Derive /24 CIDR prefix from an IPv4 address |

### Account Management (requires API key)

| Tool | Description |
|---|---|
| `list_api_keys` | List all API keys on your account |
| `get_usage` | Get request counts and plan limits |
| `get_profile` | Get account profile (name, email) |
| `get_audit_log` | View account activity log |
| `create_api_key` | Create a new API key |
| `delete_api_key` | Revoke an API key (irreversible) |
| `renew_api_key` | Extend an API key's expiration |

## Prompts

Prompts are guided workflows that chain multiple tools together. Select them from
your MCP client's prompt picker, or just describe what you want — most clients
will match you to the right prompt automatically.

| Prompt | Description |
|---|---|
| `analyze_ip` | Deep-dive analysis of a single IP — classification, geolocation, network, timezone, EU status |
| `security_audit` | Review your NetLoc8 account — API key inventory, recent activity, usage, recommendations |
| `batch_geolocate` | Look up multiple IPs and produce a comparison table |
| `getting_started` | Guided onboarding — create a key, verify it, get SDK install instructions for your platform |

## Resources

| URI | Description |
|---|---|
| `netloc8://about` | Product overview, SDKs, pricing, and quick start guide |
| `netloc8://ip/{address}` | Geo data for a specific IP address |

## What You Get

| Feature | No API Key | Free Key | Paid Key |
|---|---|---|---|
| Country-level geo | ✅ | ✅ | ✅ |
| City, coordinates, timezone | — | ✅ | ✅ |
| Local utility tools | ✅ | ✅ | ✅ |
| Account management | — | ✅ | ✅ |
| Monthly requests | — | 5,000 | 25,000+ |

## Links

- [NetLoc8](https://netloc8.com) — Sign up (free, no credit card)
- [Documentation](https://netloc8.com/docs)
- [Go SDK](https://pkg.go.dev/github.com/netloc8/netloc8-go)
- [npm packages](https://www.npmjs.com/org/netloc8)
- [MCP Specification](https://modelcontextprotocol.io)

## License

[Elastic License 2.0 (ELv2)](LICENSE)
