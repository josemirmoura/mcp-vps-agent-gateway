# Documentation map

The repository keeps documentation in `docs/` as the canonical source. It is intentionally split by audience rather than duplicated into a separate Wiki.

## Operator / user

Start here:

1. [quick-start.md](quick-start.md) — shortest supported path into the guided terminal flow.
2. [installation-contract.md](installation-contract.md) — normative boundary between supported installation and development-only procedures.
3. [installer-flow.md](installer-flow.md) — complete installation phases and manual boundaries.
4. [product-model.md](product-model.md) — Project, Custom, Whole Host and capability model.
5. [chatgpt-integration.md](chatgpt-integration.md) — ChatGPT product-surface and completion gate.
6. [authentication.md](authentication.md) — integrated OAuth/OIDC path.
7. [operations.md](operations.md) — status, health, logs, audit, update, rollback and removal.
8. [troubleshooting.md](troubleshooting.md) — common failure paths.
9. [faq.md](faq.md) — common authority/security questions.
10. [compatibility.md](compatibility.md) — supported/tested/untested environments.
11. [privacy.md](privacy.md) — data, logs, secrets and telemetry.
12. [releases.md](releases.md) — SemVer and stable/RC/development channels.
13. [support.md](support.md) — support boundaries.

## Developer / security

- [project-status.md](project-status.md) — current maturity and next milestone.
- [architecture.md](architecture.md) — canonical runtime and trust boundaries.
- [policy-schema.md](policy-schema.md) — authorization model and policy shape.
- [threat-model.md](threat-model.md) — threat model.
- [security-hardening-v2.md](security-hardening-v2.md) — hardening requirements.
- [security-release.md](security-release.md) — pre-release hardening matrix.
- [runtime-semantics-and-recovery.md](runtime-semantics-and-recovery.md) — durability, replay and recovery.
- [tool-trust-and-confused-deputy.md](tool-trust-and-confused-deputy.md) — downstream/tool trust.
- [transport-and-aggregation.md](transport-and-aggregation.md) — transport constraints.
- [implementation-validation.md](implementation-validation.md) — executable evidence.
- [simulation-validation.md](simulation-validation.md) — historical architecture simulation evidence.
- [mvp-first.md](mvp-first.md) — historical gate sequence and current gate state.
- [implementation-runbook.md](implementation-runbook.md) — historical implementation sequence.
- [build-vs-adopt.md](build-vs-adopt.md) — original adopt/adapt/build decision.

## Precedence

If documents appear to conflict, use this order:

1. architecture.md
2. installation-contract.md for installation/user-experience questions
3. current project-status.md
4. policy-schema.md
5. security-hardening-v2.md
6. runtime-semantics-and-recovery.md
7. supporting documents
8. historical implementation/simulation documents

Update historical documents when they can mislead current product status. Never weaken the canonical architecture to preserve stale text.

## Architecture in one sentence

Two Go processes: one unprivileged MCP Gateway and one privileged local Broker, connected by a Unix socket; the Broker owns authorization, state and privileged execution.

## Current productization strategy in one sentence

Preserve the validated Scoped runtime, make authority explicit and easy to operate, distribute through stable SemVer releases, and keep Full/R5 outside production claims until its separate maturity requirements are met.
