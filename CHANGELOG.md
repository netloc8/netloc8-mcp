# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [1.0.2] — 2026-06-24

### Changed

- User-facing description strings in `tools.go` updated to reference `location:read` instead of `geo:read` (`CreateAPIKeyInput` struct tag and `create_api_key` tool description).
- Bumped `github.com/netloc8/netloc8-go` dependency to `v1.2.2`.

## [1.0.1] — 2026-06-13

### Fixed

- Dockerfile: switched base image from `scratch` to `gcr.io/distroless/static-debian12` to satisfy `os.Stderr` usage at runtime (scratch has no `/dev/stderr` device node).

## [1.0.0] — 2026-06-13

### Added

- Initial release — 14 MCP tools and 4 prompts wrapping the NetLoc8 API.
- Tools: `geolocate_ip`, `geolocate_me`, `get_timezone`, `validate_ip`, `normalize_ip`, `is_public_ip`, `get_subnet`, `get_profile`, `list_api_keys`, `create_api_key`, `delete_api_key`, `renew_api_key`, `get_usage`, `get_audit_log`.
- Prompts: `analyze_ip`, `audit_review`, `key_rotation`, `usage_overview`.
- Resources: `netloc8://profile`, `netloc8://keys`, `netloc8://usage`, `netloc8://audit`.
