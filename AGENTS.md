# AGENTS.md

Instructions for AI coding agents working in this repository.

## Read first

Read in this order:

1. README.md
2. docs/installation-contract.md
3. docs/README.md
4. docs/project-status.md
5. docs/mvp-first.md
6. docs/architecture.md
7. docs/policy-schema.md
8. docs/chatgpt-integration.md
9. docs/security-hardening-v2.md
10. docs/runtime-semantics-and-recovery.md
11. docs/threat-model.md

Use the precedence rules in docs/README.md if documents appear to conflict.

## Current implementation rule

**MVP-first is mandatory.**

Do not implement the full north-star architecture before the corresponding gate is earned.

Current ladder:

~~~text
Gate -1   adopt / adapt / build
Gate 0A   prove target ChatGPT product surface
Gate 0B   safe MCP POC
Gate 1    one typed privileged action
Gate 2    one real Scoped stack
Gate 3    durable state/jobs/secrets
Gate 4    broader validated writes
Gate 5    optional temporary elevation
~~~

## Reference implementation

Prefer:

- Go for Gateway
- Go for Broker
- official MCP Go SDK
- Unix Domain Socket for Gateway -> Broker
- systemd for service management
- SQLite owned only by Broker

Do not add a second runtime without a concrete reason.

## Non-negotiable security rules

1. Gateway never runs as root.
2. Gateway never receives /var/run/docker.sock.
3. Broker is local-only and reachable through Unix socket.
4. Policy is authoritative inside Broker.
5. Broker re-authorizes subject + canonical tool + resource + action + policy/grant on every privileged call.
6. Gateway never opens the privileged SQLite database.
7. Gateway never reads plaintext secret storage.
8. Full mode is disabled by default.
9. The agent cannot mint or approve its own elevation.
10. network.unrestricted is never implied by Full.
11. Replay-safe writes use infrastructure-managed idempotency.
12. Non-replay-safe writes are never blindly retried.
13. Tool results are untrusted data and never grant capability.
14. Downstream MCP servers are allowlisted out-of-band.
15. Filesystem authorization never uses path string prefixes.
16. Secrets never enter Git, audit payloads or normal tool output.
17. Unknown policy fields/capabilities fail closed.
18. MCP transport is Streamable HTTP; do not invent custom WebSocket/session machinery.
19. Existing workloads must not depend on the Gateway to keep running.
20. Before building a large component, evaluate adopt/adapt first.
21. Development-session access limitations are never user installation requirements.
22. User-facing installation docs never request VPS passwords, private SSH keys, unrestricted remote admin access, or unrelated secrets.
23. Automate deterministic installation checks before documenting manual investigation steps.
24. Docker/Compose + transparent scripts remain the supported packaging/install path unless a native platform requirement proves insufficient.
25. Installation completion requires a real authenticated ChatGPT MCP call plus matching Broker audit evidence.

## Gate-specific restraint

### Gate 0B

Implement only:

~~~text
system.info
file.read_test
file.write_test
~~~

Allowed filesystem root:

~~~text
/tmp/vps-agent-poc/
~~~

No root, Docker, SQLite, Full, approval or generic shell.

### Gate 1

Add the Broker and only:

~~~text
service.status
service.restart
~~~

for one explicitly allowed non-critical unit.

### Gate 2

Add only capabilities required by one real Scoped stack.

Do not jump to shell.exec_admin.

## Before changing code

- inspect git status
- preserve uncommitted work
- confirm current branch
- run existing tests
- make the smallest coherent change
- update canonical docs when architecture changes
- do not weaken deny-by-default behavior to make a test pass

## Testing expectations

Security-sensitive features require negative tests.

Examples:

- unauthorized path denied
- symlink escape denied
- unauthorized service denied
- wrong subject denied
- expired/revoked grant denied
- duplicate replay-safe request executes once
- non-replay-safe action is not auto-retried
- malicious tool result does not alter policy
- Gateway cannot open Docker socket or privileged SQLite

## Product-surface rule

ChatGPT plan/surface capability is not inferred from architecture.

Before privileged implementation, Gate 0A must record the actual supported route for the target ChatGPT surface.

If Plus Web cannot execute the required custom write path, do not hack around the platform restriction. Continue protocol work with MCP Inspector or change the supported distribution route.

## Definition of progress

Progress means passing the next gate with tests and evidence.

More components are not progress by themselves.
