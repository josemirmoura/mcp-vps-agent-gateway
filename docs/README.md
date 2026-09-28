# Documentation map

This repository separates normative design from supporting material.

## Read in this order

1. [installation-contract.md](installation-contract.md) — normative supported-installation boundary and development/user separation.
2. [project-status.md](project-status.md) — maturity and what is actually usable.
3. [build-vs-adopt.md](build-vs-adopt.md) — Gate -1: adopt, adapt, or build.
4. [mvp-first.md](mvp-first.md) — implementation sequence and stop/go gates.
5. [architecture.md](architecture.md) — primary technical source of truth.
6. [policy-schema.md](policy-schema.md) — authorization model and policy shape.
7. [chatgpt-integration.md](chatgpt-integration.md) — ChatGPT product-surface constraints.
8. [threat-model.md](threat-model.md), [security-hardening-v2.md](security-hardening-v2.md), [runtime-semantics-and-recovery.md](runtime-semantics-and-recovery.md), [tool-trust-and-confused-deputy.md](tool-trust-and-confused-deputy.md), and [transport-and-aggregation.md](transport-and-aggregation.md) — deeper security/runtime material.
9. [implementation-runbook.md](implementation-runbook.md) — detailed build checklist.

## Precedence

If documents appear to conflict, use this order:

1. architecture.md
2. installation-contract.md for installation/user-experience questions
3. mvp-first.md
4. policy-schema.md
5. security-hardening-v2.md
6. runtime-semantics-and-recovery.md
7. supporting documents
8. implementation-runbook.md

The runbook may lag a newer architecture decision. Update the runbook rather than weakening the canonical architecture.

## Architecture in one sentence

Two Go processes: one unprivileged MCP Gateway and one privileged local Broker, connected by a Unix socket; the Broker owns authorization, state and privileged execution.

## Implementation strategy in one sentence

The real ChatGPT product path and the broad Scoped runtime are now implemented and validated; current work productizes that path without widening authority or promoting Full/R5 before its maturity requirements are met.
