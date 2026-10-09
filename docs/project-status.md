# Project status and maturity

## Current stage

**PRE-RELEASE PRODUCTIZATION / CORE CHATGPT E2E PROVED; OPERATOR APPROVAL UX NOT YET ACCEPTED**

The Docker-first Go implementation now has repeatable clean-runner validation across the core Scoped runtime, package lifecycle and host-operation path.

There is still:

- no stable production release
- no production compatibility promise
- no sufficiently long real-VPS reliability history for a production-maturity claim
- no production claim for Full/admin shell

Laboratory evidence covers the Docker package end-to-end on clean Ubuntu runners: filesystem CRUD, authorization denial, sandboxed shell/jobs, host systemd, host Docker/Compose, diagnostics, lifecycle and audit. On 2026-09-28 the supported integrated self-hosted OAuth path was also exercised against ChatGPT Web on a real target VPS through an authenticated `system.info` call observed by the Broker and recorded in the audit chain. This closes Gate 0A for the supported product path without creating a stable-production claim.

## Maturity ladder

### R0 — Architecture only

Documentation exists. No executable reference implementation.

### R1 — Product path + Gate 0B

- Gate 0A records the actual ChatGPT/client integration route. **Completed 2026-09-28 for the integrated OAuth + ChatGPT Web path.**
- MCP Inspector passes.
- Safe read/write POC works only in the disposable test root.

### R2 — Typed privileged pilot

- Gateway and Broker are separate processes.
- One non-critical service can be inspected/restarted through typed tools.
- Unauthorized resources fail closed.
- Local audit exists.

### R3 — Scoped pilot

- One real stack runs under explicit Scoped policy.
- Durable operational behavior has been tested for several days.
- Recovery and denial cases are exercised.
- Routine work does not require Full.

### R4 — Hardened Scoped production

- authentication required by the deployment is validated
- Broker-owned durable state is tested
- jobs/retries/locks are tested
- secret delivery is tested
- backup/recovery is tested
- operational monitoring exists

The project may be described as production-capable for its documented Scoped use case only after R4.

### R5 — Elevated/Full production

In addition to R4:

- Full feature flag explicitly enabled
- out-of-band approval works
- temporary grants/expiry work
- revoke-all works
- network elevation is separately controlled
- tamper-evident audit + remote checkpointing works
- shell/admin recovery behavior is tested

## Full default

Reference default:

~~~yaml
features:
  full_mode_enabled: false
~~~

Full is not required to call the project successful.

A strong Scoped implementation is a valid production endpoint.

## Evidence over popularity

Stars and forks are community signals, not production evidence.

Prefer:

- reproducible builds
- automated tests
- releases
- deployment history
- documented recovery tests
- incident learnings
- dependency maintenance
- security review
- external review

## Current next milestone

The implementation has clean-runner evidence through the broad Scoped toolbox and Docker packaging, plus real ChatGPT Web/OAuth evidence and a real scoped write/read/delete proof in the operator environment.

**There are several parallel stable blockers, not only licensing:** legal/license and authorship/IP (#57/#68), real desktop/mobile operator approval (#65), complete MCP conformance (#79), runtime-version consistency and patched-Go evidence (#81/#92), Portuguese quick-start parity (#84), signed immutable RC provenance and exact-candidate operator acceptance. RC6 was published under Apache-2.0; the RC7 source/runtime is not yet a published signed release (checked 2026-10-09). If license/public notices change, cut a new immutable RC, rerun the automated matrix and repeat acceptance.

Owner acceptance remains a required final technical gate. It includes real OAuth/client connection, audit, **native approval only when the client advertises elicitation, otherwise independent operator-authenticated approval UX**, scoped operations, protected-secret denial/temporary exception/revocation and lifecycle checks. The connected ChatGPT session returned `operator_fallback` and did not show a native confirmation; this must not be counted as approval success.

Long-running reliability remains a later production-maturity requirement. See [operator-acceptance.md](operator-acceptance.md), [execution-plan.md](execution-plan.md) and [implementation-validation.md](implementation-validation.md).
