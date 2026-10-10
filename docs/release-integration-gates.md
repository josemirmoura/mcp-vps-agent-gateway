# Community v0.1.0: integration and release evidence ledger

**Scope:** one compatible AI chat, one Linux computer, Scoped authority. This ledger is a release-readiness contract, not a notice of general availability. Public Community facts only; private Cloud plans and owner infrastructure details do not belong here.

**Round-two integration snapshot (2026-10-09, 23:59 UTC onward):** public `main` = `4212f30f7c145c8bbd0043b42bdc7dd95e466cd5`; **staging PR #99** = `9da45838517d6e4baa5e63236027f6a7471c66cb`. The second-round code changes from Blocks 1–4 are already combined in staging; `main` and owner VPS remain unchanged. Final exact-SHA recheck: **18/18 GitHub Actions workflows completed with SUCCESS** for `9da45838517d6e4baa5e63236027f6a7471c66cb`, including `docker-package-acceptance` run `38009010754`. These are automated staging results, not owner acceptance or a release signature. The newest verified published release remains `v0.1.0-rc.6`, and `VERSION=0.1.0-rc.7` alone is not a published signed RC7. Staging CI does not certify official MCP conformance, real client UX, legal rights or release signatures.

## Integration inputs and review responsibilities

| Input | Working PRs | Dependency | Ready to merge into a release candidate? |
|---|---|---|---|
| Operator approval and Broker authority | #88/#93/#101 and round-two #113 | Second-round authorization changes merged into staging, including replay/logout regressions | Automated browser/Broker evidence present; real ChatGPT desktop/mobile owner acceptance pending |
| Runtime version, patched Go, MCP, OAuth | #89/#90/#91/#98 and round-two #110 | Version/toolchain and fixture-only MCP tests merged into staging | Patched Go/security CI present; production conformance gate FAIL, details below |
| Update/backup recovery | #97/#100 and round-two #111 | Rollback staged restore and revoked-grant fence merged in staging | Disposable Broker/volumes CI present; real owner install and recovery gate pending |
| Release source and GHCR signatures | #95, #96 | Require genuine signed assets from an authorized published release; mock cosign/crane covers behavior only | Pending first signed RC |
| EN/PT-BR installation | #85/#102 and round-two #112 | Complete bilingual guides and first-run smoke merged into staging | Disposable bilingual CI present; clean install of an authorized signed tag and owner UX pending |
| Coordinated candidate/release | staging #99, rounds #109 through #113, third-round reviews #114/#115 | Five second-round blocks combined at `9da4583...` (18/18 SUCCESS); subsequent third-round staging revisions require separate CI on their exact new SHA | MCP conformance, genuinely signed RC, real client acceptance, legal and owner gates remain pending |

Never cherry-pick stacked commits blindly or overwrite a peer branch. Trace each contribution to a PR/SHA, resolve overlapping files once, and test the resulting integrated tree independently.

## Required release gates

| Gate | Evidence and pass criteria | Current |
|---|---|---|
| A. Rights/license and publishing authority | license, attribution/NOTICE, contributor/ownership boundaries legally approved for **this** candidate; historical grants preserved | NOT TESTED / owner review required |
| B. Reproducible integrated build/security | exact SHA; CI Go race/vet/build for amd64/arm64, `govulncheck`, `gosec`, secret scan, Docker scans/SBOM, OAuth, IPC/Broker, lifecycle, negative security cases; all required checks green | PARTIAL: round-two `9da4583...` has **18/18 staging workflows SUCCESS**; still no post-merge main or actual release build/signed assets |
| C. MCP protocol and client support | official upstream requirements pinned, table of PASS/FAIL/SKIPPED/NOT TESTED/NOT APPLICABLE; no simulated client presented as real | FAIL/PARTIAL: Gateway normal **8 PASS/2 FAIL/3 SKIPPED/24 NOT TESTED**; isolated test fixture **35 PASS/1 FAIL/1 SKIPPED/0 NOT TESTED**; **both `release_gate_passed=false`**. Fixture result is not Gateway production conformance, #79 open |
| D. Signed candidate | tag and commit match VERSION; actual source checksum+Sigstore bundle and GHCR image digests/signatures independently verified against expected release workflow/issuer; scan provenance and SBOM | NOT TESTED: no new signed RC published |
| E. Installation and operator acceptance | clean supported Linux clone from the exact released tag; HTTPS+OAuth; real compatible AI MCP call; scoped grant, denial, expiration, protected secrets, revocation and audit; desktop and mobile UI where available; secure HTTPS/SSH fallback | NOT TESTED for integrated candidate |
| F. Recovery and promotion | controlled backup/rollback and stop/tar failure rehearsal; no unexpected privilege/identity loss; CI candidate SHA **and separately main SHA after merge**; operator release approval | NOT TESTED |

Classification: **PASS**, **FAIL**, **SKIPPED**, **NOT TESTED**, or **NOT APPLICABLE**. A workflow's green badge is not an operator's acceptance, and synthetic signing does not authenticate a real published artifact. Any failed mandatory gate blocks publication or promotion. Do not convert untested evidence into PASS.

## Exact-candidate evidence record (fill in before release)

| Field | Required recorded value |
|---|---|
| Integration PR / reviewed SHA | [staging #99](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/99), `9da45838517d6e4baa5e63236027f6a7471c66cb`, **18/18 GitHub Actions workflows SUCCESS**, **not** final release SHA |
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

Related: [release policy](releases.md), [execution plan](execution-plan.md), [operator acceptance](operator-acceptance.md), [security release](security-release.md), and GitHub Community issues #4/#57/#65/#68/#79/#81/#84/#92.
