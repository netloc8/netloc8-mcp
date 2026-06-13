# NetLoc8 MCP Server

An [MCP](https://modelcontextprotocol.io) server that gives AI assistants access to [NetLoc8](https://netloc8.com) — IP geolocation deployed to 300+ edge locations worldwide.

Works **with or without** an API key. Without a key, you get country-level geo data and local utility tools. With a free key, you unlock city-level detail, coordinates, timezone, and account management.

## Installation

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

| Prompt | Description |
|---|---|
| `analyze_ip` | Deep-dive analysis of a single IP address |
| `security_audit` | Security-focused review of an IP address |
| `batch_geolocate` | Geolocate and summarize a list of IPs |
| `getting_started` | Guided onboarding for new NetLoc8 users |

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
