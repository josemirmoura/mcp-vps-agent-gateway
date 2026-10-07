# Pre-release security hardening matrix

Reviewed for the `0.1.0-rc.3` closeout line on 2026-09-30.

This matrix records concrete pre-release controls. A checked item means the control exists in code/tests or is intentionally documented. Final stable-release readiness still depends on green PR CI and the owner's clean-install operator acceptance gate against the current candidate.

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
- [x] application-level concurrent request ceiling exists (default 64; excess requests fail with HTTP 429).
- [x] MCP request bodies are capped at 1 MiB and Broker IPC decoding is capped at 2 MiB.
- [x] public OAuth dependency pinned to ZITADEL `v4.19.1`, reviewed as the current latest stable release on 2026-09-28.

## Privilege boundary

- [x] Gateway runs non-root.
- [x] Gateway does not receive `/host`.
- [x] Gateway does not receive Docker socket.
- [x] Broker is local-only over Unix domain socket.
- [x] Broker validates Unix peer credentials with `SO_PEERCRED`.
- [x] package acceptance proves that a process with socket group access but the wrong UID is rejected before Broker request handling.
- [x] Broker owns privileged SQLite.
- [x] Broker re-authorizes subject + tool + resource + action.
- [x] Scoped is default.
- [x] Whole Host is a separate physical-mount decision.
- [x] Full is disabled by default.
- [x] unrestricted network remains separate from Full.

## Files, jobs and replay

- [x] path traversal negative tests.
- [x] symlink escape negative tests.
- [x] recursive removal does not follow symlink escapes.
- [x] move/rename through a symlink directory outside the root is denied.
- [x] concurrent file↔symlink swap attempts never return outside-root content.
- [x] hardlink unlink semantics are covered so removal inside an authorized root does not alter the outside hardlink name/content.
- [x] physical scope root canonicalization.
- [x] sandboxed scoped shell/jobs.
- [x] runtime/output/task/memory limits.
- [x] operation journal and idempotency semantics.
- [x] fencing-token resource locks.
- [x] revoke-all invalidates grants and pending approvals and cancels grant-backed jobs.
- [x] non-replay-safe operations are not blindly retried.
- [x] network allowlist mode fails closed in the sandbox until hostname/IP egress enforcement is implemented; unrestricted egress remains a separate explicit authority.

## Secrets, logs and privacy

- [x] repository secret scan with gitleaks.
- [x] repository-wide development-artifact sanitization guard.
- [x] gosec source scan.
- [x] govulncheck.
- [x] bounded Go fuzzing for secure filesystem root selection and policy parsing.
- [x] high-volume adversarial architecture simulation for grants, replay and locking.
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
- [x] release workflow is configured for keyless GitHub/Sigstore signed provenance attestations for GHCR image digests and the source package.
- [x] no long-lived project private signing key is stored; signing identity is derived from short-lived GitHub OIDC credentials.
- [ ] live release evidence for the new signed-attestation workflow is still required on the next release candidate before this control is claimed as release-proven.

## Maturity boundaries

- [x] Scoped is the supported product path.
- [x] real ChatGPT Web + integrated OAuth completion gate has been exercised.
- [x] Full/R5 is not presented as production-ready.
- [ ] long-running real-workload reliability remains a later R4 maturity requirement.
- [ ] final clean-install operator acceptance gate remains owner-operated and must pass before `v0.1.0` is frozen.
