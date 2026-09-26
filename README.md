# MCP VPS Agent Gateway

A security-first reference architecture for connecting an AI assistant such as ChatGPT to a Linux VPS through MCP, while keeping authorization and privilege enforcement on the server.

> **Status:** reference architecture with an MVP-first implementation path. The full design is a north star, not the first milestone.

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
  transport-and-aggregation.md
  mvp-first.md
  runtime-semantics-and-recovery.md
  tool-trust-and-confused-deputy.md
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


## Quick start

> This repository is currently a reference architecture and implementation guide, not a finished binary. The fastest path is to build **Gate 0** first and prove your real MCP client before adding privileged features.

### 1. Clone

```bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
```

### 2. Read the minimum set

```text
docs/mvp-first.md
docs/security-hardening-v2.md
docs/transport-and-aggregation.md
AGENTS.md
```

### 3. Build only Gate 0

Your first implementation should expose only:

```text
system.info
file.read_test
file.write_test
```

with read/write restricted to a disposable directory such as:

```text
/tmp/vps-agent-poc/
```

Do **not** add root access, Docker control, OAuth, Full mode, Approval Service or a generic admin shell yet.

### 4. Test with your real MCP client

Success means:

- the client discovers the tools
- read works
- write works when the client/product permits it
- forbidden paths fail
- reconnects do not corrupt state

If your real client cannot execute the required write tool, stop there and fix only the client integration path.

### 5. Hand it to a coding agent

You can give an AI coding agent this instruction:

```text
Read AGENTS.md and docs/mvp-first.md.
Implement Gate 0 only.
Do not implement privileged execution, Docker access, Full mode,
OAuth, Approval Service or any feature beyond Gate 0.
Add tests for path confinement and clear errors.
```

After Gate 0 passes, continue with Gate 1 and Gate 2 in [docs/mvp-first.md](docs/mvp-first.md).


## Start here

1. Read [docs/architecture.md](docs/architecture.md).
2. Read [docs/threat-model.md](docs/threat-model.md).
3. Read [docs/security-hardening-v2.md](docs/security-hardening-v2.md).
4. Read [docs/policy-schema.md](docs/policy-schema.md).
5. Read [docs/transport-and-aggregation.md](docs/transport-and-aggregation.md).
6. Read [docs/mvp-first.md](docs/mvp-first.md).
7. Read [docs/runtime-semantics-and-recovery.md](docs/runtime-semantics-and-recovery.md).
8. Read [docs/tool-trust-and-confused-deputy.md](docs/tool-trust-and-confused-deputy.md).
9. Follow [docs/implementation-runbook.md](docs/implementation-runbook.md).
10. If ChatGPT is your client, read [docs/chatgpt-integration.md](docs/chatgpt-integration.md).
11. If an AI coding agent is implementing the project, make it read [AGENTS.md](AGENTS.md) first.

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
