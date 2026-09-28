# Changelog

All notable changes to MCP VPS Agent Gateway are documented here.

The project follows Semantic Versioning. Pre-release versions may change while the release candidate is being validated.

## [Unreleased]

### Planned

- final clean-install human gate before the first stable release
- freeze `v0.1.0` after the human gate passes

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

[Unreleased]: https://github.com/josemirmoura/mcp-vps-agent-gateway/compare/v0.1.0-rc.1...HEAD
[0.1.0-rc.1]: https://github.com/josemirmoura/mcp-vps-agent-gateway/releases/tag/v0.1.0-rc.1
