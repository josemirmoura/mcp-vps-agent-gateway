# AGENTS.md

Instructions for AI coding agents working in the public Portico Community repository.

## Repository boundary

This repository is authoritative for the public Community/runtime implementation and its public documentation.

Do not use it as the primary place to design confidential commercial strategy, pricing, private SaaS implementation, owner/admin operations or internal product roadmap.

Public product boundary for the current Community line:

**one compatible web AI chat <-> one Linux computer**

## Read first

1. README.md
2. docs/architecture.md
3. docs/installation-contract.md
4. docs/README.md
5. docs/project-status.md
6. docs/product-model.md
7. docs/policy-schema.md
8. docs/chatgpt-integration.md
9. docs/security-hardening-v2.md
10. docs/runtime-semantics-and-recovery.md
11. docs/threat-model.md
12. docs/vision.md
13. docs/roadmap-multinode-control-plane.md

## Security rules

1. Gateway never runs as root.
2. Gateway never receives Docker socket.
3. Broker remains local-only through Unix socket.
4. Broker policy is authoritative.
5. Every privileged call is re-authorized.
6. Gateway never opens privileged Broker state or plaintext secret storage.
7. Full is disabled by default.
8. AI/tool output is untrusted data.
9. Filesystem authorization never relies on string-prefix checks.
10. Secrets never enter Git, audit payloads or normal tool output.
11. Unknown policy fields/capabilities fail closed.
12. Streamable HTTP remains the MCP transport unless a standards-driven change is justified.
13. Existing workloads do not depend on the Gateway staying connected.
14. AI-to-AI delegation, if publicly implemented later, never implies privilege inheritance.
15. User-facing installation docs never request VPS passwords, private SSH keys or unrestricted remote administration.

## Engineering discipline

Before changing code:

- inspect current state;
- preserve user work;
- run relevant tests;
- make the smallest coherent change;
- add negative tests for security-sensitive behavior;
- update public docs only for public/runtime behavior;
- never weaken deny-by-default behavior to make a test pass.

## Release discipline

Stable release behavior is defined by tagged releases, not moving main.

Current stable gate is documented in docs/execution-plan.md.

Future managed/commercial product work must not be copied here unless explicitly approved as public interoperability or public product documentation.
