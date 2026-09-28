# Pre-release security hardening matrix

Reviewed for `0.1.0-rc.2` on 2026-09-28.

This matrix records concrete pre-release controls. A checked item means the control exists in code/tests or is intentionally documented. Final release readiness still depends on PR CI and the owner's clean-install human gate.

## Public transport and identity

- [x] HTTPS required for the supported public path.
- [x] OAuth/OIDC discovery validated before ChatGPT connection.
- [x] RFC 9728 protected-resource metadata published.
- [x] Dynamic Client Registration enabled only on the dedicated identity project.
- [x] PKCE S256 advertised and checked.
- [x] dedicated MCP resource audience required.
- [x] private RFC 7662 token introspection used by Gateway.
- [x] stable subject revalidated by Broker.
- [x] invalid/inactive access token rejected.
- [x] expired access token rejected by regression test.
- [x] wrong issuer rejected.
- [x] wrong audience rejected.
- [x] revoked/inactive access tokens are rejected from RFC 7662 introspection state; ZITADEL documents revoked tokens as inactive at introspection.
- [x] unauthenticated MCP returns fail-closed 401 with OAuth resource challenge.
- [x] DCR and OAuth endpoints have edge rate limiting.
- [x] application-level concurrent request ceiling exists.
- [x] public OAuth dependency pinned to ZITADEL `v4.19.1`, reviewed as the current latest stable release on 2026-09-28.

## Privilege boundary

- [x] Gateway runs non-root.
- [x] Gateway does not receive `/host`.
- [x] Gateway does not receive Docker socket.
- [x] Broker is local-only over Unix domain socket.
- [x] Broker validates Unix peer credentials.
- [x] Broker owns privileged SQLite.
- [x] Broker re-authorizes subject + tool + resource + action.
- [x] Scoped is default.
- [x] Whole Host is a separate physical-mount decision.
- [x] Full is disabled by default.
- [x] unrestricted network remains separate from Full.

## Files, jobs and replay

- [x] path traversal negative tests.
- [x] symlink escape negative tests.
- [x] physical scope root canonicalization.
- [x] sandboxed scoped shell/jobs.
- [x] runtime/output/task/memory limits.
- [x] operation journal and idempotency semantics.
- [x] fencing-token resource locks.
- [x] revoke-all invalidates grants and pending approvals and cancels grant-backed jobs.
- [x] non-replay-safe operations are not blindly retried.

## Secrets, logs and privacy

- [x] repository secret scan with gitleaks.
- [x] repository-wide development-artifact sanitization guard.
- [x] gosec source scan.
- [x] govulncheck.
- [x] image vulnerability scan with Trivy in package acceptance.
- [x] diagnostic bundle secret redaction and CI leakage checks.
- [x] first-party analytics/tracking absent.
- [x] bundled ZITADEL telemetry explicitly disabled.
- [x] user documentation forbids sharing VPS/SSH credentials.
- [x] privacy/data-location documentation exists.

## Release supply chain

- [x] SemVer `VERSION` file.
- [x] release tag must equal `v$VERSION`.
- [x] source release artifact has SHA-256 checksum.
- [x] GHCR release images build for amd64/arm64.
- [x] release images request BuildKit SBOM/provenance attestations.
- [ ] separate project-managed cryptographic signing key is not implemented for the first RC; no signing claim is made.

## Maturity boundaries

- [x] Scoped is the supported product path.
- [x] real ChatGPT Web + integrated OAuth completion gate has been exercised.
- [x] Full/R5 is not presented as production-ready.
- [ ] long-running real-workload reliability remains a later R4 maturity requirement.
- [ ] final clean-install human gate remains owner-operated and must pass before `v0.1.0` is frozen.
