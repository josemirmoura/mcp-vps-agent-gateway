# MCP VPS Agent Gateway

A security-first reference architecture for connecting an AI assistant such as ChatGPT to a Linux VPS through MCP, while keeping authorization and privilege enforcement on the server.

> **Status:** architecture and implementation runbook. Reference code can be added incrementally.

## Why this project exists

Remote AI agents become much more useful when they can do real operational work. Giving a model an unrestricted root shell, however, turns a prompt-injection failure into an infrastructure incident.

This project defines a safer architecture with three operational modes:

- **Controlled** — inspect freely and require approval for meaningful changes.
- **Scoped** — operate autonomously inside an explicitly authorized perimeter.
- **Full** — temporary administrative access granted by a human and automatically expired.

The core design rule is:

> **The LLM is never the security boundary.**

The model requests actions. The server decides whether they are allowed.

## Target architecture

```text
AI Client / ChatGPT
        |
        | MCP
        v
+-----------------------+
| MCP Gateway           |
| non-root              |
| auth + validation     |
+-----------+-----------+
            |
       Unix socket
            |
+-----------v-----------+
| Policy Engine         |
| Controlled / Scoped   |
| Full + temporary lease|
+-----------+-----------+
            |
       Unix socket
            |
+-----------v-----------+
| Execution Broker      |
| root-owned, minimal   |
| sandbox + files       |
| Docker + systemd      |
+-----------+-----------+
            |
            v
       Linux VPS
```

## Security properties

- MCP Gateway does not run as root.
- MCP Gateway never receives direct access to `/var/run/docker.sock`.
- Privileged operations go through a small local broker.
- Policies are enforced server-side.
- Full access requires a human-approved, time-limited lease.
- The agent cannot grant itself additional privileges.
- Shell commands are constrained by timeout, cgroups/systemd and output limits.
- Filesystem access must resist path traversal and symlink escapes.
- Replay-safe writes require idempotency keys; non-replay-safe/destructive writes require stronger confirmation and locking.
- Jobs can outlive a disconnected MCP request.
- Audit logs capture actions and decisions, not entire conversations.
- Secrets should be referenced or injected at execution time instead of returned to the model.

## Repository map

```text
docs/
  architecture.md
  implementation-runbook.md
  threat-model.md
  security-hardening-v2.md
  policy-schema.md
  chatgpt-integration.md

examples/policies/
  controlled.yaml
  scoped.yaml
  full.yaml

AGENTS.md
SECURITY.md
CONTRIBUTING.md
LICENSE
```

## Start here

1. Read [docs/architecture.md](docs/architecture.md).
2. Read [docs/threat-model.md](docs/threat-model.md).
3. Read [docs/security-hardening-v2.md](docs/security-hardening-v2.md).
4. Read [docs/policy-schema.md](docs/policy-schema.md).
5. Follow [docs/implementation-runbook.md](docs/implementation-runbook.md).
6. If ChatGPT is your client, read [docs/chatgpt-integration.md](docs/chatgpt-integration.md).
7. If an AI coding agent is implementing the project, make it read [AGENTS.md](AGENTS.md) first.

## Intended stack

A pragmatic first implementation can use:

- **MCP Gateway:** TypeScript + official MCP SDK
- **Privileged Broker:** Go
- **IPC:** Unix Domain Socket
- **State:** SQLite
- **Auth:** OAuth/OIDC
- **Process isolation:** systemd transient units + cgroups
- **Additional sandboxing:** Landlock where supported
- **Containers:** Docker or rootless Podman where appropriate

No Kubernetes, Redis, service mesh or policy DSL is required for the first version.

## Scope

This repository documents an architecture. It is not a promise that every ChatGPT plan or MCP host currently supports every write capability. Product availability can change. Keep the server architecture independent from the client adapter.

## License

Apache-2.0. See [LICENSE](LICENSE).

---

Português: veja [README.pt-BR.md](README.pt-BR.md).
