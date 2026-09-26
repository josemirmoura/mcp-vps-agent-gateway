# AGENTS.md

Instructions for AI coding agents working in this repository.

## Read first

Before editing:

1. `README.md`
2. `docs/architecture.md`
3. `docs/threat-model.md`
4. `docs/implementation-runbook.md`

## Non-negotiable rules

1. Native platform primitives first.
2. MCP Gateway never runs as root.
3. MCP Gateway never receives Docker socket access.
4. Privileged Broker is local and reachable only through a Unix socket.
5. Full access requires a human-approved temporary lease.
6. The agent cannot mint or approve its own elevation.
7. Authorization is enforced server-side.
8. Never commit secrets.
9. Never print secrets in logs or responses.
10. SQLite is sufficient initially.
11. Do not add Kubernetes, Redis, OPA, a service mesh or other infrastructure without demonstrated need.
12. Meaningful writes should be idempotent where practical.
13. Long-running operations use persistent jobs.
14. Critical configuration changes validate before apply.
15. MCP/ChatGPT is an adapter; the domain must remain client-independent.

## Implementation order

1. scaffold
2. Broker minimum
3. MCP Gateway minimum
4. Controlled/Scoped policy
5. secure filesystem
6. sandboxed shell
7. jobs
8. Docker/systemd
9. OAuth/OIDC
10. Approval Service
11. Full
12. client integration
13. gradual production

Do not start by implementing unrestricted root shell access.

## Before changing code

- inspect `git status`
- do not destroy uncommitted work
- run existing tests
- make the smallest coherent change
- update documentation when architecture changes
- preserve deny-by-default behavior

## Security gate

Before connecting a production host, confirm:

- path traversal is blocked
- symlink escape is blocked
- shell has timeout/cgroup limits
- Gateway has no Docker socket
- Scoped obeys resource boundaries
- Full without lease fails
- expired lease fails
- prompt injection cannot self-elevate
- secrets are redacted
- idempotency works
