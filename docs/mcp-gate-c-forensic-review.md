# Gate C forensic review: frozen MCP 2026-07-28 (Block 2)

Source baseline: Community PR #99, commit `835d9a0a486e35775543b53523ef4640b4d66519`, upstream `modelcontextprotocol/conformance@c37eec888e1c6ff140af79987a40008548b7cc5f` (runner 0.2.0-alpha.12). **No modification of pinned requirements, exit codes, or scenario statuses is permitted.** Release gate remains `false` unless *all 37 frozen server scenarios* pass with exit 0, or a separately approved release-criterion supersedes the inappropriate full-suite claim.

## Primary raw evidence independently inspected

- Normal production Gateway, [run 38011721235](https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/38011721235), ZIP artifact `11653486773`, artifact digest `sha256:c416f8ef7a0e022230aa6e2dcea6ab8638596fd31fce44aa501ad97844e3f10c`: **8 PASS / 2 FAIL / 3 SKIPPED / 24 NOT_TESTED**, runner 1; `release_gate_passed=false`.
- Same source commit, test-only isolated Gateway fixture, [run 38011721237](https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/38011721237), ZIP artifact `11654345065`, artifact digest `sha256:4c795c1ea53cf940142b38254898f44689d6a4fb46cf8a840192b5666df7d68d`: **35 PASS / 1 FAIL / 1 SKIPPED / 0 NOT_TESTED**, runner 1, `release_gate_passed=false`.
- Reviewed original `evidence/*matrix.json`, individual `results*/server-*/checks.json` and standard-output log. Artifacts also contain non-scored scenario directories: these do not replace the 37 frozen required-server entries. Client-role requirements do not apply to this resource server.
- Both production and lab artifacts are observations of the **same source SHA**, not of the same compiled server binary or capabilities. Lab's `test_*` tools, synthetic prompts and resources must **never** appear in the normal production tool catalog.

## Exact scenario inventory and original classifications

| # | Upstream server scenario | Normal | Lab-only fixture | Why this difference exists |
|---:|---|---|---|---|
| 1 | `server-stateless` | NOT_TESTED | SKIPPED | 24 real checks PASS; 4 unavailable synthetic probes; 2 list-change checks skipped. |
| 2 | `completion-complete` | FAIL | FAIL | Optional completions not advertised; -32601/404 verified separately; frozen runner demands method. |
| 3 | `tools-list` | PASS | PASS | Production tool catalog actually interrogated. |
| 4 | `tools-call-simple-text` | NOT_TESTED | PASS | Synthetic content tool required. |
| 5 | `tools-call-image` | NOT_TESTED | PASS | Synthetic content tool required. |
| 6 | `tools-call-audio` | NOT_TESTED | PASS | Synthetic content tool required. |
| 7 | `tools-call-embedded-resource` | NOT_TESTED | PASS | Synthetic content tool required. |
| 8 | `tools-call-mixed-content` | NOT_TESTED | PASS | Synthetic content tool required. |
| 9 | `tools-call-error` | NOT_TESTED | PASS | Synthetic error tool required. |
| 10 | `tools-call-with-progress` | NOT_TESTED | PASS | Synthetic progress tool required. |
| 11 | `server-sse-multiple-streams` | PASS | PASS | Production transport interrogated. |
| 12 | `resources-list` | PASS | PASS | Production resource endpoint interrogated. |
| 13 | `resources-read-text` | NOT_TESTED | PASS | No synthetic test:// static text resource in production. |
| 14 | `resources-read-binary` | NOT_TESTED | PASS | No synthetic test:// binary resource in production. |
| 15 | `resources-templates-read` | NOT_TESTED | PASS | Synthetic template absent in production. |
| 16 | `sep-2164-resource-not-found` | PASS | PASS | Normal resource-not-found behavior measured. |
| 17 | `prompts-list` | PASS | PASS | Normal prompts/list endpoint measured. |
| 18 | `prompts-get-simple` | NOT_TESTED | PASS | Synthetic prompt required. |
| 19 | `prompts-get-with-args` | NOT_TESTED | PASS | Synthetic prompt with arguments required. |
| 20 | `prompts-get-embedded-resource` | NOT_TESTED | PASS | Synthetic embedded prompt required. |
| 21 | `prompts-get-with-image` | NOT_TESTED | PASS | Synthetic image prompt required. |
| 22 | `dns-rebinding-protection` | PASS | PASS | Production Origin handling measured. |
| 23 | `caching` | SKIPPED | PASS | Normal server has no synthetic resources/read cache fixture. |
| 24 | `input-required-result-basic-elicitation` | NOT_TESTED | PASS | Synthetic InputRequiredResult tool required. |
| 25 | `input-required-result-basic-sampling` | NOT_TESTED | PASS | Synthetic InputRequiredResult tool required. |
| 26 | `input-required-result-basic-list-roots` | NOT_TESTED | PASS | Synthetic InputRequiredResult tool required. |
| 27 | `input-required-result-request-state` | NOT_TESTED | PASS | Synthetic state tool required. |
| 28 | `input-required-result-multiple-input-requests` | NOT_TESTED | PASS | Synthetic multi-input tool required. |
| 29 | `input-required-result-multi-round` | FAIL | PASS | Raw production check expects a fixture InputRequiredResult; lab completes 4 checks. |
| 30 | `input-required-result-missing-input-response` | SKIPPED | PASS | Production yields WARNING rather than fixture's re-request behavior. |
| 31 | `input-required-result-non-tool-request` | NOT_TESTED | PASS | Synthetic request fixture required. |
| 32 | `input-required-result-result-type` | NOT_TESTED | PASS | Synthetic typed result fixture required. |
| 33 | `input-required-result-unsupported-methods` | PASS | PASS | Normal unsupported-method behavior measured. |
| 34 | `input-required-result-tampered-state` | NOT_TESTED | PASS | Synthetic signed requestState fixture required. |
| 35 | `input-required-result-capability-check` | NOT_TESTED | PASS | Synthetic client-capability tool required. |
| 36 | `input-required-result-ignore-extra-params` | SKIPPED | PASS | Production yields WARNING; lab tests its synthetic tool. |
| 37 | `input-required-result-validate-input` | PASS | PASS | Normal validation behavior measured. |

### Evidence behind the two raw FAILs

**`completion-complete`:** original `checks.json` contains `completion-complete: FAILURE` with *method not found* and `wire-schema-valid: SUCCESS`. The official 2026-07-28 [completion specification](https://modelcontextprotocol.io/specification/2026-07-28/server/utilities/completion) says servers **that support completions** MUST advertise `completions`; the standard lists JSON-RPC `-32601` for an unsupported capability. Normal `server/discover` does **not** advertise `completions`; `TestMCPCompletionCapabilityIsAbsentAndMethodIsClosed` confirms `completion/complete` returns HTTP 404 / JSON-RPC `-32601`. This is a **frozen-runner FAIL** and a *proposed non-applicability on capability grounds*, not a verified production defect nor a retroactive PASS.

**`input-required-result-multi-round`:** normal `checks.json` has `sep-2322-multi-round-r1: FAILURE`, text *Expected InputRequiredResult with inputRequests and requestState*, plus `wire-schema-valid: SUCCESS`. A non-fixture production tool was asked to behave like `test_input_required_result_multi_round`. The separate isolated lab's same scenario achieved **4 SUCCESS / 0 FAILURE** with an explicitly HMAC-bound two-round `InputRequiredResult` fixture. That proves the SDK/transport can carry these messages under synthetic conditions; it does **not** verify the production Broker's real approval rules or client-specific UX.

**`server-stateless`:** normal original 30 checks are 24 SUCCESS, 4 FAILURE marked `Not testable` because `test_missing_capability`, `test_streaming_elicitation`, `test_logging_tool` are unavailable, plus 2 SKIPPED list-change notification checks. Lab has 28 SUCCESS, 0 FAILURE, 2 SKIPPED. In both, `server/discover`, version handling, metadata validation, missing-header errors, JSON-RPC errors and HTTP response codes have measured successes. Two SKIPPED checks are notifications for list changes: production tools catalogue is declared static (`listChanged=false`). The skip remains part of the raw result; review normative applicability before excluding it.

**`caching`, `missing-input-response`, `ignore-extra-params`:** production raw statuses contain SKIPPED/WARNING because the expected synthetic resource or tool behavior is absent; lab covers these with fixture-specific results. No conversion to PASS.

## Real protocol defect found by direct negative testing

The normal Gateway's initial `Handler` accepted `Origin: https://attacker.invalid` over `POST /mcp` with **HTTP 200**, actually invoking `system.info`; the new test [Gate C first run 38013189063](https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/38013189063) caught this. This is a **genuine production transport defect**: MCP 2026-07-28 [Streamable HTTP security](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http) states that invalid present `Origin` MUST receive 403.

The fix implements `guardMCPOrigin` before authentication and before the tool executor, solely on the MCP endpoint. Security constraints: the configured OAuth resource origin is the public trust anchor, never the untrusted `Host` header; without a configured public resource, a supplied Origin must be same-authority **loopback only**. Requests without an Origin continue to work for headless clients. Invalid, malformed, multi-valued, cross-scheme and DNS-rebinding-style origins return 403 before any executor call; redacted status has no untrusted Origin string. Tests use disposable localhost HTTP, no secrets or production Broker. Bloco 5 must review the single `gateway.go` wrapper-line alteration and integration consequences.

This fix was discovered **after** the original official artifact, so those original scores are immutable history. Re-run both frozen suites on this PR SHA and archive their new original artifacts; do not claim earlier runs demonstrate the correction.

## Production OAuth Resource Server and what is *not* tested

The separate `TestIntegratedOAuthResourceServerBoundaryE2E` exercises a real `Handler` with a disposable introspection server. It verifies protected resource metadata, Bearer 401 challenge including `resource_metadata`, positive active introspection and subject propagation, and negative cases for inactive, expired, missing/altered issuer or audience, missing expiration/subject, missing/prefix-confused scopes. This is **OAuth resource-server evidence**, separate from the normal conformance run's `VPS_AGENT_AUTH_MODE=none`.

PKCE, login through a browser, authorization-code exchange, AS metadata and end-user client experience remain integration responsibility of the **OAuth client/authorization server**, not proof of this resource server. Security of real provider configuration and authorization decisions still requires device/operator acceptance.

## Proposed *separate* normative release acceptance (approval required)

Keep two separately signed verdicts:

1. **Frozen official full-suite**: 37 scenarios with raw status, runner exit, `release_gate_passed`; until 37 PASS and exit 0, permanently record **BLOCKED**. Never make the release job silently ignore a failing official requirement.
2. **Community advertised-profile normative gate** (engineering/owner sign-off **pending**): evaluate capability-advertised server features and HTTP requirements against *normal* Gateway, using test-only isolated probes to reach synthetic edges. Require green real-Gateway tests for stateless metadata/header mismatch/Origin, `server/discover`, tools/list and advertised tool behavior, OAuth resource metadata and Bearer validation, denial before executor, schema and secret boundaries, no advertised-but-unimplemented capability, and repeat after integration at exact release SHA. Mark unadvertised completions **NOT APPLICABLE (proposed)**, test fixture dependencies **NOT TESTED ON NORMAL GATEWAY**, and unavailable list-change events **conditional/not triggered** with clear rationale. Each exception must be explicitly reviewed, never auto-approved.
3. A third **real-client acceptance** remains independent: ChatGPT/other supported client connection, OAuth handoff, and owner approval/denial/revocation via the actual supported UX, not a synthetic protocol runner.

To close issue #79 for release purposes, engineering must approve this exact **normative applicability rule** or bring every required capability into the normal advertised product and obtain a clean frozen run. In either case attach: exact integration SHA, CI workflow links, original redacted and full GitHub artifact IDs, all normative HTTP/OAuth negatives, reviewer decision, and retest after any new code change.

## Safety and interpretation

The `scripts/mcp_gate_c_diagnostic.py` companion verifies **every original check** against the frozen matrix and records source SHA, check ID/status and SHA-256/error-category only. Original error messages remain preserved in the GitHub Actions artifact but are never printed in the new public summary, preventing accidental leaking of future tokens/cookies. It rejects check-status drift, removed evidence and manipulated status counts. It does not waive failed scenarios or claim conformance.

This PR includes a narrowly scoped **MCP HTTP Origin transport correction** (one `gateway.go` wrapper line and an isolated helper), negative regressions, original-evidence diagnostics and Gate C documentation. Broker privileges, staged integration, `main`, production, VPS, credentials, release and signature workflows are untouched.
