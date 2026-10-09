# Community productization status

Current source baseline: `VERSION=0.1.0-rc.7` on the development branch. **Not an immutable published release.**

Last published GitHub release verified on 2026-10-09: **`v0.1.0-rc.6` (pre-release)**. The RC7 runtime was separately observed on the operator VPS; a live runtime version is not release provenance.

## Implemented or evidenced

- [x] Public runtime Gateway (non-root) → Unix socket → authoritative local Broker, policy and audit;
- [x] EN/PT-BR README and guided installation foundation (translation parity tracked in #84);
- [x] Integrated OAuth/OIDC proof against a real ChatGPT Web connection;
- [x] Scoped write/read/delete proof in the operator's ChatGPT environment (2026-10-06);
- [x] Local diagnostics, state backup/update/rollback tooling and safe remove/purge flows (still require exact-final-RC acceptance);
- [x] Preliminary automated clean-runner acceptance and security checks on selected historical PR heads;
- [x] MCP native elicitation code with fail-closed unsupported-client behavior;
- [x] Signed-release automation **implemented in source**, but no signed RC7 release verified;
- [x] Immutable RC6 pre-release published, without retroactive Sigstore signatures.

## Blockers before stable Community v0.1.0

- [ ] **#57 / #68**: owner/legal approval of final license, authorship, rights and Community/Cloud boundary;
- [ ] **#65**: client-tested operator approval/denial in readable desktop AND mobile UI. The operator's ChatGPT session returned `operator_fallback` (no native elicitation advertised); a source-level native flow alone cannot pass this gate. Complete independent operator-authorized HTTPS fallback or prove a supported native client, with real Broker/audit and owner acceptance;
- [ ] **#79**: full relevant official MCP conformance against actual Gateway (the stateless smoke does not constitute full certification);
- [ ] **#81 / #92**: release identity consistency and patched Go toolchain verified in the precise candidate and complete security CI;
- [ ] **#84**: validated PT-BR tutorial/content parity;
- [ ] **Release provenance**: freeze final RC after legal/code fixes; publish immutable tag plus signed image/source artifacts; independently verify signatures and SHA/digests (draft #95/#96 work must pass before reliance);
- [ ] **Owner exact-tag acceptance**: fresh Linux install, OAuth/HTTPS, scoped and protected-file allow/deny/revoke on desktop/mobile, audit, diagnostics, failure, update/rollback, safe removal/reinstall;
- [ ] **Stable promotion**: pass release gates, freeze `v0.1.0`, verify signed outputs and update public installation instructions.

## Outside v0.1.0 scope

Multi-node paid SaaS, native Windows/macOS support, high-risk Full/R5 maturity and a formal long-running R4 production/SLA claim are separate future scopes. Do not claim them as stable Community functionality.

## Evidence discipline

A checked item means there is code or **specific scoped evidence**, not necessarily complete live acceptance. Re-check CI against the **exact head SHA**; failed-before-steps GitHub Actions runs are not test results. A source version, healthy `system.info` and a published release are three different facts. Do not declare 100% until all stable blockers are accepted.

Public source-of-truth for operational release gates: [execution-plan](execution-plan.md), [operator acceptance](operator-acceptance.md), [releases](releases.md).
