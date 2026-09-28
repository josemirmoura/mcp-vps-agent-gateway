# Project status and maturity

## Current stage

**PRE-RELEASE PRODUCTIZATION / CHATGPT E2E GATE COMPLETE**

The Docker-first Go implementation now has repeatable clean-runner validation across the core Scoped runtime, package lifecycle and host-operation path.

There is still:

- no stable production release
- no production compatibility promise
- no long-running real-VPS deployment history
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

The implementation has clean-room evidence through the broad Scoped toolbox and Docker packaging, plus a completed real ChatGPT Web OAuth end-to-end gate on 2026-09-28.

The current milestone is **public productization of the Scoped path**: synchronize documentation, sanitize the repository, provide a guided terminal flow, formalize versioning/releases and compatibility, complete pre-release security hardening, and prepare a release candidate for the owner's final clean-install human gate. Long-running reliability remains a later production-maturity requirement. See [implementation-validation.md](implementation-validation.md).
