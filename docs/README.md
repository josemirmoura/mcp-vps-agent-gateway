# Documentation map

This repository is the public documentation source for the **Portico Community/runtime**.

It is not the canonical location for confidential commercial strategy, private SaaS implementation or internal product planning.

## Operator / user

Start here:

1. [quick-start.md](quick-start.md) - shortest supported path into the guided terminal flow.
2. [installation-contract.md](installation-contract.md) - normative user-installation boundary.
3. [installer-flow.md](installer-flow.md) - complete installation phases.
4. [product-model.md](product-model.md) - authority and capability model.
5. [chatgpt-integration.md](chatgpt-integration.md) - ChatGPT/MCP connection path.
6. [authentication.md](authentication.md) - OAuth/OIDC path.
7. [operations.md](operations.md) - status, health, logs, audit, update, rollback and removal.
8. [troubleshooting.md](troubleshooting.md) - common failure paths.
9. [faq.md](faq.md) - common security/authority questions.
10. [compatibility.md](compatibility.md) - supported/tested/untested environments.
11. [privacy.md](privacy.md) - data, logs, secrets and telemetry.
12. [releases.md](releases.md) - SemVer and stable/RC/development channels.
13. [support.md](support.md) - Community support boundaries.
14. [operator-acceptance.md](operator-acceptance.md) - owner-operated final acceptance gate.
15. [vision.md](vision.md) - public product principles.
16. [roadmap-multinode-control-plane.md](roadmap-multinode-control-plane.md) - intentionally high-level public roadmap.

## Developer / security

- [project-status.md](project-status.md) - current public runtime maturity.
- [productization-status.md](productization-status.md) - public Community productization status.
- [execution-plan.md](execution-plan.md) - active public Community release path.
- [architecture.md](architecture.md) - runtime and trust boundaries.
- [policy-schema.md](policy-schema.md) - authorization model.
- [threat-model.md](threat-model.md) - threat model.
- [security-hardening-v2.md](security-hardening-v2.md) - hardening requirements.
- [security-release.md](security-release.md) - pre-release hardening matrix.
- [runtime-semantics-and-recovery.md](runtime-semantics-and-recovery.md) - durability, replay and recovery.
- [tool-trust-and-confused-deputy.md](tool-trust-and-confused-deputy.md) - downstream/tool trust.
- [transport-and-aggregation.md](transport-and-aggregation.md) - transport constraints.
- [implementation-validation.md](implementation-validation.md) - executable evidence.

## Precedence

For Community/runtime questions:

1. architecture.md
2. installation-contract.md
3. current project-status.md
4. policy-schema.md
5. security-hardening-v2.md
6. runtime-semantics-and-recovery.md
7. supporting public documents

## Public scope

Current Community promise:

**one compatible web AI chat <-> one Linux computer**

Broader commercial/SaaS planning is maintained outside this public repository.

- [Portico Cloud node connector](cloud-node-connector.md) — optional outbound managed-Cloud enrollment, signed task transport, and local Broker handoff.
