# Active execution plan: Community v0.1.0 stable

Status: **pre-stable**. This plan measures 100% as the first **honestly accepted and signed stable Community release**, for **one compatible web AI chat connected to one Linux node in Scoped mode**. It does not imply Full/R5, commercial Cloud, Windows/macOS or long-running R4 SLA maturity.

## Baseline and evidence (2026-10-09)

- Core Go Gateway/Broker, installer, Linux Scope/policy, local audit, OAuth integration and several ephemeral acceptance suites already exist.
- An authenticated owner ChatGPT session confirmed `system.info`, `gateway_version=v0.1.0-rc.7` and valid local audit; a historical real disposable scoped file write/read/delete proof exists.
- Native MCP elicitation is implemented in the Gateway, but the tested ChatGPT session reported `approval_method=operator_fallback`: **no native dialog and no new grant**. A separate audited operator flow must be accepted.
- `VERSION` is `0.1.0-rc.7` while the newest **published GitHub Release is v0.1.0-rc.6** (checked 2026-10-09). RC7 is not a published immutable signed release.
- Security/build/source/image verification and adaptive approvals are in parallel development PRs. Their existence is not release acceptance; use each PR's **exact head SHA** and successful checks as evidence.

## Gate A: code/security stabilization (engineering)

- [ ] Resolve and merge nonoverlapping version identity fixes (#81; coordinate #89 and #90), including all MCP, binary, log and package identities.
- [ ] Upgrade to Go patch with fixed standard library vulnerabilities (#92). Run `go test -race`, `go vet`, `govulncheck`, `gosec` and Docker package lifecycle checks on the exact candidate.
- [ ] Complete official MCP conformance against the **actual disposable Gateway** (#79); record passing, skipped and untestable checks separately, without claiming complete conformance from stateless smoke.
- [ ] Finish OAuth security boundary (#91), input validation, fail-closed checks and negative cases.
- [ ] Harden update interrupted-backup recovery and verify mock + disposable real lifecycle failure drill before using the revised updater on an operational machine.
- [ ] Finish PT-BR installation tutorial parity (#84, PR #85) and ensure source README, installer and support docs agree.
- [ ] Resolve overlapping approval PRs (#88/#93) and avoid merging duplicate bridge/UI changes blindly. All mandatory checks green **on the final rebased SHA**, not just historical PR heads.

**Exit evidence:** a clean integrated Community main commit; complete green CI/security; linked actionable MCP conformance output; reproducible clean install/update/rollback tests; no known P0 code vulnerabilities.

## Gate B: usable and safe owner approval (engineering + operator/client acceptance)

- [ ] Select the adaptive approval architecture (#65): native MCP elicitation when capability is advertised; otherwise an independently authenticated HTTPS operator interface, with audited SSH CLI as the fallback.
- [ ] Reject by default when client/identity/CSRF/origin/session, scope, deadline, request binding or Broker authorization is absent. A link, MCP OAuth login or request ID alone never grants authority.
- [ ] Broker-local decisions for **approve AND deny** root and protected-file requests, expiry, revocation, exact ceiling warning, no self-approval by model, no grant on invalid or replayed requests.
- [ ] Validate the web operator surface with independent login, session hardening and responsive **Android/narrow browser and desktop** behavior; do not send Docker socket, administrative Broker tokens or approval secrets to a browser.
- [ ] Test real connected ChatGPT where available, record advertised MCP capabilities, fallback method and actual approval+denial UI. Native-elicitation support must not be assumed from a ChatGPT plan label.
- [ ] Complete threat review and adversarial tests against the real Broker with audit verification; capture owner visual acceptance with sanitized screenshots.

**Exit evidence:** user can approve and deny a disposable operation on desktop/mobile, explicitly reading root/access/physical ceiling/TTL; Broker grants only approved scope and records/revokes it. Client incompatibility is documented and safely handled, not hidden.

## Gate C: Community licensing, authorship and notices (owner/legal)

- [ ] Resolve #57 and #68 (authorship, contribution rights, trademark, final Community license and public/private boundaries), documenting the approval.
- [ ] If changing from the Apache-2.0 historical pre-release grant, update LICENSE/NOTICE and all public notices and **cut a new RC**. Do not rewrite historical release rights.
- [ ] Donations (#59) remain optional; only publish Ko-fi CTA after a real approved owner URL exists. A donation page is **not** a prerequisite for technical v0.1.0.

**Exit evidence:** final legal decision and exact public notices approved for the prospective stable source package.

## Gate D: immutable signed release candidate (engineering)

- [ ] Freeze a commit after A–C; run every CI/security/packaging workflow on that exact commit.
- [ ] Create/publish a new immutable RC tag through the authorized release workflow, without confusing source `VERSION` with a GitHub Release.
- [ ] Verify source package SHA-256 and Sigstore bundle independently (#95); verify all published multiarch GHCR images **by immutable digest and signing identity** (#96).
- [ ] Confirm README/install/update pin the **published final RC**, and archive evidence of actual digests, tag, commit and workflow run.

**Exit evidence:** a downloadable exact-tag RC, verified signatures for source and images, and a reproducible installation target.

## Gate E: owner clean-install and failure-path acceptance (owner + engineering)

- [ ] Use a disposable/owner-designated supported Ubuntu host and follow [operator acceptance](operator-acceptance.md) against the exact signed candidate.
- [ ] Prove DNS/TLS, integrated OAuth, ChatGPT/MCP connection, audited `system.info`, bounded scoped file/service/job operations, out-of-scope denial and protected secret deny→temporary allow→revoke.
- [ ] Complete both approved and denied mobile/desktop authorization experiences via the actually supported native/HTTPS channel.
- [ ] Prove diagnostics contain no secrets and tamper-evident audit is intact; test backup failure auto-restart, migration failure rollback, safe remove/reinstall and controlled purge on the disposable host.
- [ ] Record actual host/architecture, steps, pass/fail, logs without credentials, timestamps, approved tag and owner sign-off. Resolve any failure, recut a candidate if code changes and repeat relevant checks.

**Exit evidence:** signed owner acceptance record for the immutable candidate; no unresolved P0.

## Gate F: stable promotion

- [ ] Freeze the accepted commit, set stable `VERSION=0.1.0` appropriately, update changelog/release notes and repeat CI/security/packaging checks for the final stable commit.
- [ ] Publish immutable signed `v0.1.0`; independently verify source and image signatures/digests again; update public documentation to **stable pinned checkout**.
- [ ] Verify default stable update channel, operator install/rollback instructions and a non-secret post-release smoke test.
- [ ] Declare **Community v0.1.0 stable 100%** only when A–F have objective passing evidence and no known P0 blockers.

## Ordering and blockers

A and B can advance in parallel without source-file collisions. C requires owner/legal decision. D depends on A–C. E depends on D and a real owner/client/device. F depends on successful E.

Maintain release evidence in public PRs/issues and do not copy private Cloud commercial material into this Community project. Larger product maturity remains tracked independently after stable.
