# MCP 2026-07-28: frozen required conformance evidence

Portico Community is an **MCP server** operating as an OAuth **resource server**. It is not being marketed as a native MCP client or OAuth authorization server. The protocol version is distinct from the Portico `VERSION` product SemVer.

## Source and test identity

- Official upstream implementation: `modelcontextprotocol/conformance`.
- Immutable upstream commit: `c37eec888e1c6ff140af79987a40008548b7cc5f` (`0.2.0-alpha.12`).
- Frozen requirements file: `requirements/2026-07-28.yaml` from that same commit.
- Required **server** scenarios: **37**. Required **client** scenarios: **32**, **NOT APPLICABLE** to Portico's server role.
- `not_scored` optional/pending scenarios do not count as required, and cannot be silently substituted for required tests.

## Execution and classification

Dedicated CI `.github/workflows/block2-mcp-requirements.yml` builds the **real Gateway** with a disposable local filesystem and `VPS_AGENT_AUTH_MODE=none` bound **only to loopback**. It builds the pinned official upstream runner from source via `npm ci` and executes:

```sh
node dist/index.js server --url http://127.0.0.1:18994/mcp --requirements 2026-07-28
```

The CI saves the official output and per-scenario `checks.json` artifacts. `scripts/mcp_conformance_matrix.py` emits a machine-readable report with:

| Classification | Meaning |
| --- | --- |
| **PASS** | Every required check in this scenario succeeded. |
| **FAIL** | A required check fails unexpectedly, or the report is corrupt. |
| **SKIPPED** | No failures, but some/all checks were not executed. Not a pass. |
| **NOT_TESTED** | No test evidence, or the official diagnostic tool is absent. Not a pass. |
| **NOT_APPLICABLE** | Client-role requirements when evaluating Portico's server, or optional non-scored extensions. |

In particular, the original stateless smoke on `main` reported **24 success, 4 diagnostic-only NOT TESTED, 2 skipped** across 30 checks. That one scenario is **not the 37-scenario requirements suite** and does not prove complete conformance.

The evidence workflow is intentionally named `mcp-requirements-evidence-not-release-gate`: a green report-generation job only proves that the pinned test harness ran and classified evidence. **It cannot declare release readiness by itself.** The report has an explicit `release_gate_passed` Boolean, which is true only when all 37 scored scenarios PASS and the official runner exits zero. For an enforced gate, invoke the matrix parser with `--enforce`; failures then exit nonzero.

## OAuth Resource Server

`internal/gateway/oauth_resource_smoke_test.go` tests the actual Gateway HTTP handler against disposable loopback introspection for protected-resource metadata, Bearer challenge, missing/expired/inactive tokens, issuer/audience/subject/scope denial and positive subject propagation. It is synthetic Resource Server evidence, **not** a live browser authorization-code+PKCE test and does not certify the external identity provider.

## Release blockers and coordination

- Fully reconcile and run the required scenario matrix for the **exact final integrated SHA**, identifying missing features/fixtures without exposing synthetic diagnostic tools on the production MCP catalog.
- Do not reinterpret tests requiring imaginary `test_*` diagnostic tools as production functional requirements without an explicit protocol/conformance review. They remain visible as NOT_TESTED until independently measured.
- Keep `gateway.go` and the authoritative Broker authorization changes with Block 1. Propose protocol changes via Block 5 handoff.
- Actual ChatGPT/MCP Apps compatibility, native elicitation and approved/denied permissions on mobile/desktop belong to Blocks 1 and 5.
- Promoting the release requires security scanners green, exact-RC signing, license approval and operator acceptance in addition to this evidence.
