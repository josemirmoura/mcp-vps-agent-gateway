# MCP VPS Agent Gateway

Security-first reference architecture for connecting ChatGPT or another MCP client to a Linux VPS without making the model a trusted security boundary.

> **Status: PRE-ALPHA / EXECUTABLE REFERENCE IMPLEMENTATION.** Go code now exists and is validated on disposable GitHub-hosted Ubuntu runners. There is still no stable production release.

## What this project is

A design and implementation path for this target:

~~~text
ChatGPT Web / MCP client
        |
        | Streamable HTTP
        v
vps-agent-gateway      non-root
        |
        | Unix socket
        v
vps-agent-broker       privileged, local-only
        |
        v
Linux / systemd / Docker
~~~

The Gateway handles MCP, authentication and schemas.

The Broker owns the real security boundary: policy, state, jobs, filesystem, Docker/systemd, secrets and privileged execution.

> **The LLM is never the security boundary.**

## What this project is not

Today this repository is not:

- plug-and-play software
- a Docker image
- a production release
- a generic root shell for AI
- a promise that every ChatGPT plan supports private MCP write access

The complete design is a north star. The repository now contains an MVP-first Go reference implementation, but it is not yet production-ready.

## Modes

- **Controlled** — inspection plus narrowly gated changes.
- **Scoped** — autonomous work inside an explicit perimeter.
- **Full** — optional temporary capability bundle, disabled by default.

Routine work should happen in Scoped. Full is exceptional.

## Current reference stack

- **Gateway:** Go + official MCP Go SDK
- **Broker:** Go
- **IPC:** Unix Domain Socket
- **Transport:** MCP Streamable HTTP
- **State:** SQLite, Broker-only
- **Isolation:** systemd transient units + cgroups
- **Secrets:** systemd credentials and/or root-owned files
- **Extra sandboxing:** Landlock when available
- **Ingress:** existing reverse proxy or supported private tunnel

No Kubernetes, Redis, service mesh or separate policy daemon is required.

## Quick start

### 1. Clone

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
~~~

### 2. Read the canonical docs

Start with:

~~~text
docs/README.md
docs/project-status.md
docs/mvp-first.md
docs/architecture.md
AGENTS.md
~~~

### 3. Do Gate 0A before privileged code

Confirm the actual ChatGPT product path you will use.

As of 2026-09-26, OpenAI documents private full MCP write/modify in Developer Mode for Business, Enterprise and Edu. Plus Web should not be assumed to support a private custom write MCP.

See [docs/chatgpt-integration.md](docs/chatgpt-integration.md).

### 4. Validate the executable reference implementation

~~~bash
go test -race ./...
go build ./cmd/...
~~~

The GitHub workflows additionally validate Docker, systemd, vulnerability scanning and real MCP-to-Broker Linux effects. See [docs/implementation-validation.md](docs/implementation-validation.md).

The Gate 0B baseline exposes:

~~~text
system.info
file.read_test
file.write_test
~~~

Restrict file access to:

~~~text
/tmp/vps-agent-poc/
~~~

Do not add root, Docker, SQLite, Full, approval or generic shell yet.

### 5. Continue gate-by-gate

~~~text
Read AGENTS.md, docs/README.md, docs/mvp-first.md and docs/implementation-validation.md.
Inspect the current implementation and CI evidence.
Advance only the next unproven gate; do not enable Full or generic admin shell early.
~~~

## The implementation ladder

~~~text
Gate -1   Adopt / adapt / build
Gate 0A   Prove ChatGPT product surface
Gate 0B   Safe MCP read/write POC
Gate 1    One typed privileged action
Gate 2    One real Scoped stack
Gate 3    Durable state/jobs/secrets
Gate 4    Broader validated writes
Gate 5    Optional temporary elevation
~~~

Each gate must earn the next layer of complexity.

## Core security properties

- Gateway never runs as root.
- Gateway never receives the Docker socket.
- Broker re-authorizes every privileged call.
- Policy is deny-by-default and authoritative inside the Broker.
- SQLite is opened only by the Broker.
- Replay-safe writes use infrastructure-managed idempotency.
- Non-replay-safe writes are never blindly retried.
- Filesystem authorization resists traversal and symlink escape.
- Tool results are untrusted data.
- Downstream MCP servers cannot be registered dynamically by the model.
- Full is disabled by default and never implies unrestricted network access.
- Secrets are not exposed through generic read tools.

## Documentation

The canonical reading order and precedence rules are in [docs/README.md](docs/README.md).

The most important documents are:

- [Architecture](docs/architecture.md)
- [MVP-first implementation](docs/mvp-first.md)
- [ChatGPT integration](docs/chatgpt-integration.md)
- [Threat model](docs/threat-model.md)
- [Security hardening](docs/security-hardening-v2.md)
- [Runtime semantics and recovery](docs/runtime-semantics-and-recovery.md)

## Build vs adopt

Before implementing a major component, evaluate existing solutions.

This project is justified when the required combination of self-hosting, ChatGPT Web, server-side policy, Scoped autonomy and operator-controlled privileged execution is not already available in an acceptable form.

See [docs/build-vs-adopt.md](docs/build-vs-adopt.md).

## License

Apache-2.0. See [LICENSE](LICENSE).

Português: [README.pt-BR.md](README.pt-BR.md).
