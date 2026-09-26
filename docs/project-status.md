# Project status and maturity

## Current maturity

**Stage: pre-alpha / docs-first reference architecture.**

As of the current repository state:

- there is no production-ready implementation
- there is no supported binary
- there is no Docker image
- there is no stable API compatibility promise
- there is no production release
- the documented security model has not yet been validated by a long-running production deployment of this repository

The repository is intentionally a design and implementation guide first.

## What is ready

- architecture
- threat model
- security hardening rules
- policy model
- transport guidance
- MVP-first gates
- runtime/recovery semantics
- agent implementation instructions

## What is not ready

- install-and-run package
- production-grade Broker
- production-grade MCP Gateway
- Approval Service
- stable Full mode
- long-term compatibility guarantees

## Production-readiness levels

### R0 — Documentation only

Architecture exists. No executable reference implementation.

### R1 — Gate 0 POC

Real MCP client can discover and execute safe test read/write tools in a disposable directory.

### R2 — Typed privileged pilot

A minimal Broker performs one tightly scoped privileged action against a non-critical test service.

### R3 — Scoped production pilot

A real application stack is operated under an explicit Scoped policy with audit and recovery tests.

### R4 — Hardened production

Required production controls are validated, including authentication, durable state, recovery, secret handling and security gates.

### R5 — Elevated/Full production

Temporary administrative capabilities have undergone dedicated security review and production validation.

Do not describe the project as production-ready before R4.

## Full mode default

Full mode is **disabled by default**.

An implementation must require an explicit server-side enablement before Full capability requests are even accepted.

Recommended default:

```yaml
features:
  full_mode_enabled: false
```

Enabling Full should require:

- Gate 0, Gate 1 and Gate 2 passed
- security/recovery tests passed
- out-of-band approval working
- emergency `revoke-all` tested
- explicit operator decision

## Maturity signals

Stars and forks can be useful community signals but are not security or production-readiness guarantees.

Prefer objective evidence:

- releases
- reproducible builds
- automated tests
- security policy
- published threat model
- real deployment history
- incident/recovery evidence
- maintenance activity
- external review

This repository should report its maturity honestly as it evolves.