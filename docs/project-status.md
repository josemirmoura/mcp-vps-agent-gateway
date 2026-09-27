# Project status and maturity

## Current stage

**PRE-ALPHA / EXECUTABLE REFERENCE IMPLEMENTATION**

There is now executable Go reference code with repeatable GitHub-runner validation.

There is still:

- no stable production release
- no production compatibility promise
- no long-running real-VPS deployment history
- no completed ChatGPT Web product-surface Gate 0A
- no production-enabled Full/admin shell

Laboratory evidence has reached the Gate 0B and Gate 1 behaviors. Formal maturity remains pre-R1 until the actual target client/distribution path is proven.

## Maturity ladder

### R0 — Architecture only

Documentation exists. No executable reference implementation.

### R1 — Product path + Gate 0B

- Gate 0A records the actual ChatGPT/client integration route.
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

The implementation has clean-room evidence for Gate 0B and the typed Gate 1 systemd action.

The next product milestone is **Gate 0A on the actual ChatGPT surface**, followed by a real-VPS Scoped pilot. See [implementation-validation.md](implementation-validation.md).
