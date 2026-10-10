# Community v0.1.0: integration and release evidence ledger

**Scope:** one compatible AI chat, one Linux computer, Scoped authority. This ledger is a release-readiness contract, not a notice of general availability. Public Community facts only; private Cloud plans and owner infrastructure details do not belong here.

**Round-two integration snapshot (2026-10-09, 23:59 UTC onward):** public `main` = `4212f30f7c145c8bbd0043b42bdc7dd95e466cd5`; **staging PR #99** = `9da45838517d6e4baa5e63236027f6a7471c66cb`. The second-round code changes from Blocks 1–4 are already combined in staging; `main` and owner VPS remain unchanged. Final exact-SHA recheck: **18/18 GitHub Actions workflows completed with SUCCESS** for `9da45838517d6e4baa5e63236027f6a7471c66cb`, including `docker-package-acceptance` run `38009010754`. These are automated staging results, not owner acceptance or a release signature. The newest verified published release remains `v0.1.0-rc.6`, and `VERSION=0.1.0-rc.7` alone is not a published signed RC7. Staging CI does not certify official MCP conformance, real client UX, legal rights or release signatures.

## Third-round independent integration checkpoint (2026-10-10)

The round-two SHA `9da45838517d6e4baa5e63236027f6a7471c66cb` remains a historical baseline. The newer staging commit **`7426484bd690b072b8efcaa8728bf67194657de5`** incorporates Block 1 (#114 and #116), Block 3 (#117), Block 5 release-documentation (#115), and Block 2 MCP capability/OAuth evidence (#118). At this exact staging SHA **18/18 GitHub Actions workflows completed SUCCESS**. Each input PR was independently checked before integration, including #118 **11/11 SUCCESS** at `2b2db1a3ea0a917a61dc94dbc52e471cc984916c`.

Block 4's guided installer authority-decline fix **PR #119** passed **8/8 workflows SUCCESS** on `3383deaa2f1b18e07ffef9170a86257e1ac00315` and was merged **only to staging** as `2fb2218f714cc5a0f44b4ed6183036ffd891c841`. The decline path previously exited with status 0; it now returns 2 with PT-BR/EN resume guidance and real Bash/PTY negative tests. The full third-round integration subsequently reached `835d9a0a486e35775543b53523ef4640b4d66519` with **18/18 workflows SUCCESS**. This is a historical checkpoint, not the latest fourth-round reviewed code SHA below.

## Fourth-round auditor remediation and final integrated code checkpoint (2026-10-10)

Independent audit [PR #99](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/99#issuecomment-6092147948) found release/update publication risk (AUD-01), incomplete exact-source provenance (AUD-02), and missing fail-closed handling for failed Broker audit writes (AUD-03). **All three findings now have implementation changes merged only into staging**, not public `main` or the owner's operating machine:

- **Gate C hardening, [PR #121](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/121)**: `Origin` enforcement blocks malicious browser requests at MCP transport before OAuth or execution, with real disposable OAuth→Unix IPC→Broker negative tests. PR CI **10/10 SUCCESS**, staging merge `6d5116c4417a7dac6c24d87680fa8e00f5addd6d`. The official frozen 37-scenario score remains **not passed**.
- **AUD-01/02, [PR #125](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/125)**: update preflight now requires a completed non-draft published GitHub Release, exact remote/local tag commit, signed source/provenance SHA and reproducible `git archive`, and signed immutable image digests *before* stopping services or producing operational backup. Release workflow defers the public tag until images and source are signed. PR CI **8/8 SUCCESS**, staging merge `ee30ac05f21ba1ab6729ee2531eba7d93e97647b`; staging for that SHA **18/18 SUCCESS**. Fixture-based cosign/crane/gh tests are **not evidence of a genuine published signed release**.
- **AUD-03, [PR #126](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/126)**: mutating Broker operations now write a durable `intent` before an external effect; a failed final audit returns `reconcile_required`, fences subsequent mutations in the running Broker, and requires operator reconciliation. Serialized mutation path and negative fault-injection/concurrency tests plus [recovery policy](runtime-semantics-and-recovery.md) are included. PR CI **9/9 SUCCESS**, staging merge `d5f54e4b69b2fb57397bfcbff063ba2f037c505a`. In-memory audit-degraded fencing does **not** automatically verify reconciliation across Broker restarts; external effects are not transactionally atomic with SQLite.
- **Final reviewed code tree at `d5f54e4b69b2fb57397bfcbff063ba2f037c505a`: 18/18 GitHub Actions workflows completed SUCCESS on this exact SHA**, zero failures, cancellation or pending. This establishes **Gate B for automated staging evidence only**. This ledger's own subsequent documentation commit also needs its own exact-SHA CI and status note on PR #99.

The official conformance suite and real-client tests are independent of this technical hardening. The latest published release remains the previously verified unsigned-for-this-new-contract `v0.1.0-rc.6`; `VERSION=0.1.0-rc.7` in source does not create a released signed candidate. **No main merge, production update, VPS/DNS change, tag, image push, or release was authorized in this integration.**

The official MCP 2026-07-28 full-suite baseline is unchanged: normal production Gateway **8 PASS / 2 FAIL / 3 SKIPPED / 24 NOT_TESTED** and test-only fixture **35 PASS / 1 FAIL / 1 SKIPPED / 0 NOT_TESTED**; both `release_gate_passed=false`. The [capability audit](mcp-release-capability-audit.md) identifies unadvertised optional completions, synthetic-fixture requirements and OAuth boundary evidence, without modifying the frozen suite or issuing a certification claim.

**Four valid gate statuses only:** PASS, FAIL, NOT TESTED, and NOT APPLICABLE (with explicit justification). The gate table below uses NOT TESTED wherever the newest final candidate still lacks required evidence, even when an earlier commit already passed related tests. Gate C remains FAIL under the complete frozen-suite requirement. No stable release may be published with mandatory gates outstanding.

## Integration inputs and review responsibilities

| Input | Working PRs | Dependency | Ready to merge into a release candidate? |
|---|---|---|---|
| Operator approval and Broker authority | #88/#93/#101 and round-two #113 | Second-round authorization changes merged into staging, including replay/logout regressions | Automated browser/Broker evidence present; real ChatGPT desktop/mobile owner acceptance pending |
| Runtime version, patched Go, MCP, OAuth | #89/#90/#91/#98 and round-two #110 | Version/toolchain and fixture-only MCP tests merged into staging | Patched Go/security CI present; production conformance gate FAIL, details below |
| Update/backup recovery | #97/#100 and round-two #111 | Rollback staged restore and revoked-grant fence merged in staging | Disposable Broker/volumes CI present; real owner install and recovery gate pending |
| Release source and GHCR signatures | #95, #96 | Require genuine signed assets from an authorized published release; mock cosign/crane covers behavior only | Pending first signed RC |
| EN/PT-BR installation | #85/#102 and round-two #112 | Complete bilingual guides and first-run smoke merged into staging | Disposable bilingual CI present; clean install of an authorized signed tag and owner UX pending |
| Coordinated candidate/release | staging #99; third-round #114–#120, Gate C #121, auditor corrections #125/#126 | Fourth-round integrated code `d5f54e4...` (18/18 SUCCESS); PR #99 tracks latest doc-only SHA and its separate recheck | MCP frozen conformance, real signed RC, real client acceptance, legal and owner gates remain pending |

Never cherry-pick stacked commits blindly or overwrite a peer branch. Trace each contribution to a PR/SHA, resolve overlapping files once, and test the resulting integrated tree independently.

## Required release gates

| Gate | Evidence and pass criteria | Current |
|---|---|---|
| A. Rights/license and publishing authority | license, attribution/NOTICE, contributor/ownership boundaries legally approved for **this** candidate; historical grants preserved | NOT TESTED: owner/legal review outstanding |
| B. Reproducible integrated build/security | exact SHA; CI Go race/vet/build for amd64/arm64, `govulncheck`, `gosec`, secret scan, Docker scans/SBOM, OAuth, IPC/Broker, lifecycle, negative security cases; all required checks green | PASS: automated staging for reviewed fourth-round **code** SHA `d5f54e4b69b2fb57397bfcbff063ba2f037c505a`, **18/18 workflows SUCCESS**, zero failures; following ledger-only documentation SHA must be rechecked separately before treating that SHA as final |
| C. MCP protocol and client support | official upstream requirements pinned, table of PASS/FAIL/SKIPPED/NOT TESTED/NOT APPLICABLE; no simulated client presented as real | FAIL: normal Gateway **8 PASS / 2 FAIL / 3 SKIPPED / 24 NOT_TESTED**; fixture-only **35 PASS / 1 FAIL / 1 SKIPPED / 0 NOT_TESTED**; both `release_gate_passed=false`, #79 open |
| D. Signed candidate | tag and commit match VERSION; actual source checksum+Sigstore bundle and GHCR image digests/signatures independently verified against expected release workflow/issuer; scan provenance and SBOM | NOT TESTED: no new signed RC published |
| E. Installation and operator acceptance | clean supported Linux clone from the exact released tag; HTTPS+OAuth; real compatible AI MCP call; scoped grant, denial, expiration, protected secrets, revocation and audit; desktop and mobile UI where available; secure HTTPS/SSH fallback | NOT TESTED for integrated candidate |
| F. Recovery and promotion | controlled backup/rollback and stop/tar failure rehearsal; no unexpected privilege/identity loss; CI candidate SHA **and separately main SHA after merge**; operator release approval | NOT TESTED |

Classification: **PASS**, **FAIL**, **SKIPPED**, **NOT TESTED**, or **NOT APPLICABLE**. A workflow's green badge is not an operator's acceptance, and synthetic signing does not authenticate a real published artifact. Any failed mandatory gate blocks publication or promotion. Do not convert untested evidence into PASS.

## Exact-candidate evidence record (fill in before release)

| Field | Required recorded value |
|---|---|
| Integration PR / reviewed SHA | [staging #99](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/99), reviewed fourth-round code `d5f54e4b69b2fb57397bfcbff063ba2f037c505a`, **18/18 GitHub Actions workflows SUCCESS**; ledger-only successor SHA and CI tracked in PR #99; **not** a published release SHA |
| Post-merge `main` SHA and CI URLs | `TBD` |
| Release tag and target commit | `TBD` |
| Architecture and package version | `TBD` |
| Source tarball checksum, bundle and verification | `TBD` |
| Gateway/Broker image names and **immutable digest** | `TBD` |
| Image cosign verification, signer issuer/identity and SBOM | `TBD` |
| MCP requirements and recorded exceptions | [documented measured matrix](mcp-2026-07-28-requirements-matrix.md), **release gate blocked**; full applicability review and real-client acceptance still required |
| Authorized clean-install target, redacted audit and OAuth evidence | `TBD` |
| Desktop/mobile client (version, capabilities, approve/deny) | `TBD` |
| Backup/recovery and rollback test | `TBD` |
| License/IP/notice sign-off | `TBD` |
| Operator approval to publish exact release | `TBD` |

Do not put credentials, access tokens, unredacted user or tenant identifiers, confidential paths, private security logic or operator `.env` contents into this public record.

## Safe promotion and rollback path

1. Freeze and review an integration commit. Run all required CI and negative tests **against that SHA**. Review the Linux/Scoped product boundary, repository ownership, security findings, and actual PR diffs.
2. Coordinate merge to protected `main` only after required review/authorization, then verify **new** CI on the post-merge SHA. An earlier PR check does not cover this merge. The publication workflow accepts **only owner-operated `workflow_dispatch` from main**, with the current reviewed 40-character SHA and an exact typed confirmation; Git tag pushes alone must never publish.
3. With rights resolved and an explicitly authorized release operation, create a version-tagged candidate using the documented workflow, then independently verify its **published** source and image artifacts by digest, checksum, Sigstore identity and expected issuer. Refuse unsigned/missing assets. Never infer immutable tag provenance solely from a successful verification of one blob.
4. Perform the [operator acceptance](operator-acceptance.md) on a disposable/recoverable supported Linux system from the exact published tag. For an MCP client without embedded/natively supported approval, verify the portal or SSH workflow with operator identity, explicit approve/deny and Broker enforcement; never claim generic native elicitation support.
5. Preserve rollback points for code, policy, audit, identity database/volumes and credentials. Exercise controlled `scripts/update.sh` failure recovery in disposable infrastructure first. Restore the previously accepted tag/volumes via the documented updater, checking audit, OAuth and Broker policy; **purge is not rollback**.
6. Stable `v0.1.0` promotion only follows explicit final owner authorization and complete evidence A–F. Changing license/notices requires a new tested immutable candidate.

Related: [release policy](releases.md), [execution plan](execution-plan.md), [operator acceptance](operator-acceptance.md), [security release](security-release.md), and GitHub Community issues #4/#57/#65/#68/#79/#81/#84/#92/#122/#123/#124.
