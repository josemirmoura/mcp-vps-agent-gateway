# Changelog

All notable changes to MCP VPS Agent Gateway are documented here.

The project follows Semantic Versioning. Pre-release versions may change while the release candidate is being validated.

## [Unreleased]

### Planned

- final clean-install operator acceptance gate before the first stable release
- freeze `v0.1.0` after the operator acceptance gate passes

## [0.1.0-rc.3] - 2026-09-28

### Fixed

- restored the two missing clauses from the canonical Apache License 2.0 text
- GitHub now detects the repository license as `Apache-2.0`

### Documentation

- repositioned the public landing page and READMEs around the direct ChatGPT Web -> Linux VPS use case and the manual terminal-copying problem it solves
- clarified that GitHub is the source, distribution and update channel, not a runtime relay between ChatGPT Web and the VPS
- added explicit public-page metadata and structured software-source description for clearer search and machine discovery

### Scope

No runtime, authorization, installation or security-boundary behavior changed from RC2. RC3 exists so the operator acceptance gate tests the exact source tree intended for stable promotion, including the canonical license correction and final public product presentation.

## [0.1.0-rc.2] - 2026-09-28

### Fixed

- release validation now supplies deterministic non-secret Compose configuration instead of depending on a local .env
- release validation checks the integrated OAuth Compose model used by the supported public path
- release workflow uses the Go version declared by go.mod and marks prerelease tags as GitHub pre-releases

### Release engineering note

The `v0.1.0-rc.1` tag was created, but its release workflow stopped before image publication or GitHub Release creation because Compose validation lacked required environment values. The tag is retained as immutable history and is not the acceptance candidate.

## [0.1.0-rc.1] - 2026-09-28

### Added

- integrated self-hosted OAuth/OIDC with ZITADEL and PostgreSQL
- Dynamic Client Registration and PKCE discovery for MCP clients
- private audience-bound token introspection
- guided terminal installation orchestration
- explicit Project, Custom and Whole Host authority profiles
- operator health, diagnostics and redacted diagnostic bundles
- release/version/support/privacy/compatibility documentation

### Validated

- real ChatGPT Web authenticated `system.info` completion gate
- scoped filesystem, shell/jobs, systemd, Docker and Compose operations
- update with backup, migration validation and automatic rollback
- safe remove, reinstall and purge lifecycle
- simultaneous independent instances
- race detector, vulnerability scanning, secret scanning, image scanning and SBOM generation

### Security

- Gateway remains non-root and does not receive the Docker socket or host root
- Broker remains the privileged local authorization boundary
- Full remains disabled by default and is not claimed as production-ready
- unrestricted networking remains a separate capability

[Unreleased]: https://github.com/josemirmoura/mcp-vps-agent-gateway/compare/v0.1.0-rc.3...HEAD
[0.1.0-rc.3]: https://github.com/josemirmoura/mcp-vps-agent-gateway/releases/tag/v0.1.0-rc.3
[0.1.0-rc.2]: https://github.com/josemirmoura/mcp-vps-agent-gateway/releases/tag/v0.1.0-rc.2
[0.1.0-rc.1]: https://github.com/josemirmoura/mcp-vps-agent-gateway/tree/v0.1.0-rc.1
