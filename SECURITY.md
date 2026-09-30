# Security Policy

## Supported versions

Security fixes target the newest supported release line.

| Version | Status |
| --- | --- |
| 0.1.x | release-candidate / initial public line |
| main | development, not a stable support promise |

## Reporting a vulnerability

Do **not** open a public issue containing exploit details, credentials or sensitive infrastructure information.

Use the repository's private Security reporting/advisory surface when it is available. If private reporting is not exposed for the repository, contact the maintainer through the GitHub profile without publishing the exploit, then establish a private channel before sending sensitive reproduction material.

Include when possible:

- affected version/component;
- reproduction conditions;
- expected vs actual behavior;
- security impact;
- suggested mitigation.

Never send real production passwords, private SSH keys or unrelated secrets as reproduction material.

## Security posture

This project assumes AI-generated actions, remote clients and external content can be hostile.

Deliberate security requirements include:

- non-root MCP Gateway;
- no direct Docker socket or host root for Gateway;
- local-only privileged Broker over Unix socket;
- deny-by-default Broker authorization;
- exact subject/resource/action re-authorization;
- human-approved temporary elevation when enabled;
- filesystem traversal and symlink-escape resistance;
- physical-ceiling discovery that exposes directory names only, never project contents;
- protected secret paths remain locked inside delegated roots unless separately and temporarily approved;
- hardlink-aware protected-file checks and shell namespace masking;
- sandboxed scoped shell execution;
- replay/idempotency controls;
- redacted operational diagnostics;
- no secrets committed to the repository;
- Full disabled by default;
- unrestricted network separate from Full.

The supported public path additionally validates integrated OAuth/OIDC issuer, active state, expiration, dedicated audience and required scopes through private token introspection.

A change that weakens one of these properties is a security-sensitive architectural change and requires negative tests.
