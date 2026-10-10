# Portico MCP Community

**Connect one compatible web AI chat to one Linux computer with server-enforced policy and audit.**

This repository contains the public Portico Community runtime.

The current validated product path connects ChatGPT Web or another compatible MCP client to a single Linux host through an unprivileged Gateway and a privileged local Broker. The machine owner defines authority; the Broker enforces it.

> **Status: pre-release Community candidate.** Ubuntu 24.04 workflows validate disposable runtime/package paths. A real user-facing ChatGPT Web/mobile acceptance, verified signed release assets, stable reliability and the final Community license are separate open gates.

## Community product boundary

Portico Community is intentionally focused:

```text
one compatible web AI chat
        |
        | HTTPS + MCP Streamable HTTP + OAuth/OIDC
        v
Portico Gateway
        |
        | protected local Unix socket
        v
Portico Broker
        |
        v
one Linux computer
```

That Linux computer can be a VPS, workstation or server.

The Community package is self-hosted. It does not require a managed Portico Cloud control plane.

## What it solves

Without a trusted execution path, server work with an AI becomes a manual relay: generate command, switch to terminal, run it, copy results back and repeat.

Portico closes that loop while keeping authorization on the owner-controlled machine. Depending on local policy, the AI client can inspect files, diagnose services, work with Docker/Compose, execute bounded shell/jobs and use other typed operations.

The supported path does not require handing the AI a VPS password, private SSH key or Docker socket.

## Security model

**The LLM is never the security boundary. The machine decides.**

Core properties:

- Gateway runs non-root;
- Gateway does not receive the host filesystem root or Docker socket;
- Broker is reachable only through a protected local Unix socket;
- Broker re-authorizes privileged requests against local policy;
- filesystem traversal and symlink escapes are denied;
- writes use operation journaling/idempotency semantics where applicable;
- destructive or administrative capabilities require explicit policy;
- protected secrets are not returned through generic tools;
- audit records privileged operations.

## Requirements

For the current supported path:

- Linux host; Ubuntu 24.04 LTS is the release-candidate validated target;
- Docker Engine 24+ and Docker Compose v2;
- Git, OpenSSL, Python 3 and curl;
- public DNS + valid HTTPS for the MCP endpoint;
- a compatible web AI/MCP client with the required custom MCP capability available.

Client capability can vary by account, workspace, plan and rollout. Compatibility claims are evidence-based, not inferred only from a subscription label. See [Compatibility](docs/compatibility.md).

## Quick start

```bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
```

The guided flow covers prerequisites, authority scope, containers, local verification, HTTPS/OAuth, AI-client connection and a real audited MCP call. It prints `INSTALLATION COMPLETE` only after the end-to-end completion gate succeeds.

See:

- [Quick Start](docs/quick-start.md) | [Português (Brasil)](docs/quick-start.pt-BR.md)
- [First installation, step-by-step acceptance](docs/first-install-acceptance.md) | [Português (Brasil)](docs/first-install-acceptance.pt-BR.md)
- [Installation contract](docs/installation-contract.md) | [Português (Brasil)](docs/installation-contract.pt-BR.md)
- [Installation flow](docs/installer-flow.md) | [Português (Brasil)](docs/installer-flow.pt-BR.md)
- [ChatGPT integration](docs/chatgpt-integration.md) | [Português (Brasil)](docs/chatgpt-integration.pt-BR.md)

## Authority

The package exposes a broad toolbox, while the owner chooses what the AI may operate.

Examples:

```text
one project:       /opt/my-app
several roots:     /opt/app + /var/www/site + /srv/data
whole filesystem:  /
```

Filesystem is only one dimension. Policy separately governs shell roots, systemd units, Docker/Compose resources, network, packages, users/groups, firewall and temporary administration.

## Operations

```bash
bash scripts/diagnose.sh status
bash scripts/diagnose.sh health
bash scripts/diagnose.sh logs 200
bash scripts/diagnose.sh audit 100

bash scripts/update.sh
bash scripts/remove.sh safe
```

Stable SemVer tags are the normal production update channel after stable release. The updater preserves operator configuration/state, validates the new runtime and rolls back on verification failure where supported.

## Public roadmap

This repository may evolve the Community runtime and public interoperability contracts over time. Broader Portico products may add managed multi-node capabilities separately.

Public roadmap commitments are intentionally limited to features that are released, required for interoperability, or explicitly approved for public announcement. See [Public roadmap](docs/roadmap-multinode-control-plane.md).

## Optional Portico Cloud connector

The public node runtime now includes an optional outbound Cloud connector. It enrolls a Linux node with a local Ed25519 identity, leases signed managed tasks, renews long-running leases and submits every operation to the same local Broker policy boundary.

See [Portico Cloud node connector](docs/cloud-node-connector.md).

## Documentation

- [Documentation map](docs/README.md)
- [Product model](docs/product-model.md)
- [Architecture](docs/architecture.md)
- [Security release matrix](docs/security-release.md)
- [Privacy and telemetry](docs/privacy.md)
- [Operations](docs/operations.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Release policy](docs/releases.md)
- [Support](docs/support.md)

## License

The current public pre-release line is distributed under **Apache-2.0**. See [LICENSE](LICENSE).

The licensing model for future stable Community releases is under review. Historical Apache-2.0 grants remain governed by the terms under which those versions were published.

Português: [README.pt-BR.md](README.pt-BR.md).
