# Portico MCP

**Connect ChatGPT Web directly to your Linux VPS through MCP, with authority you define and the server enforces.**

Portico MCP is an open-source, self-hosted bridge for letting ChatGPT Web or another MCP client inspect and operate a Linux VPS through the Model Context Protocol. OAuth authenticates the client, a non-root Gateway exposes the MCP surface, and a privileged local Broker re-authorizes host operations against operator-defined policy and records audit evidence.

> **Status: pre-release productization candidate.** Clean Ubuntu 24.04 workflows validate the package end to end, and the integrated self-hosted OAuth path has been verified through a real audited ChatGPT Web `system.info` call on 2026-09-28. A stable public release, compatibility promise, and long-running production reliability claim are still pending.

## What it solves

Without a trusted execution path, server work with an AI becomes a manual relay: ChatGPT suggests a command, you switch to a terminal, run it, copy logs back, rebuild context and repeat. Broad SSH-style access would remove some friction while creating a much larger trust problem.

Portico MCP closes that loop while keeping authorization on the VPS. From the conversation, ChatGPT can inspect files, diagnose services, analyze logs, work with Docker and perform other operations that the VPS owner explicitly enables.

~~~text
ChatGPT Web
    -> OAuth
    -> MCP Gateway
    -> Policy Broker
    -> Your Linux VPS
~~~

GitHub hosts the source, documentation, releases and update channel. **GitHub is not a runtime relay between ChatGPT Web and the VPS after installation.**

The supported flow does not require handing ChatGPT a VPS password, SSH private key or Docker socket.

## The idea

One package ships the broad toolbox. **You decide what portion of the VPS the MCP may control.**

~~~text
one project:       /opt/my-app
several roots:     /opt/app + /var/www/site + /srv/data
whole filesystem:  /
~~~

Filesystem scope is only one dimension. The policy separately controls shell roots, systemd units, Docker/Compose resources, network, packages, users/groups, firewall, and temporary administration.

For multi-project hosts, a broader physical ceiling such as `/opt` can contain several explicitly delegated roots. Widening the ceiling does not authorize it. Static roots remain in `config/policy.yaml`; production runtime delegation can be requested through MCP and, when the client supports elicitation, the MCP client itself renders the native human-confirmation surface. The Broker remains authoritative, and delegations can be listed and revoked from chat without rewriting policy or restarting the Broker. The local `scripts/delegate-root.sh` helper remains available for bootstrap and operator-managed static roots.

**The LLM is never the security boundary. The server decides.**

## Runtime

~~~text
ChatGPT Web / MCP client
        |
        | HTTPS + MCP Streamable HTTP
        v
Gateway container
non-root, no host root
        |
        | protected Unix socket
        v
Broker container
privileged host-control boundary
        |
        | host mounted at /host
        v
Linux / systemd / Docker / files
~~~

Docker is the packaging mechanism. The Broker is still a privileged component and must be treated like root on the delegated host. Server-side policy is authoritative.

## Requirements

Before starting the guided installation, have:

- a Linux VPS; Ubuntu 24.04 LTS is the release-candidate validated target;
- Docker Engine 24+ with Docker Compose v2;
- at least 2 GB RAM for the bundled ZITADEL OAuth path;
- Git, OpenSSL, Python 3 and curl;
- a public DNS hostname pointing to the VPS, TCP 80/443 available and valid HTTPS;
- a ChatGPT Web account/workspace where Developer Mode and custom MCP app creation are actually available.

OpenAI controls availability and permissions by plan, workspace and rollout. Current official OpenAI documentation lists **full MCP support, including write/modify actions, for Business, Enterprise and Edu**; Pro can connect custom MCPs with read/fetch permissions in developer mode. Portico MCP cannot elevate permissions that ChatGPT itself does not expose. Always verify the current product UI and official OpenAI documentation before relying on a specific plan label.

See [compatibility](docs/compatibility.md) for the supported/tested matrix.

## Quick start

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

The terminal now conducts the supported flow:

~~~text
Prerequisites
 -> Scope: Standard / Project / Whole Host
 -> effective authority review
 -> Containers
 -> Local verification
 -> HTTPS + integrated OAuth
 -> Connect ChatGPT
 -> real audited system.info
 -> INSTALLATION COMPLETE
~~~

**Standard** is the recommended default: `/opt` is the physical ceiling, while project roots start unauthorized and are granted later through explicit approval. **Whole Host** changes the physical filesystem ceiling to `/`, but it does not enable Full or unrestricted networking. Filesystem and capabilities remain separate policy dimensions.

The orchestration is deliberately thin and transparent. The individual `init.sh`, Compose, `verify.sh`, OAuth and ChatGPT scripts remain usable and documented. See the [Quick Start](docs/quick-start.md) and [installation flow](docs/installer-flow.md).

The supported user flow is self-service on the user's own VPS. It never asks the user to expose a VPS password, private SSH key, unrestricted remote administrative access or unrelated secrets to ChatGPT or a maintainer.

For the release-candidate period, `main` remains development. After the final operator acceptance gate freezes `v0.1.0`, normal installs should use the tagged release checkout rather than `main`.

## Toolbox

The current reference implementation includes complete scoped filesystem CRUD, durable sandboxed shell/jobs, typed systemd, Docker/Compose, diagnostics, packages, users/groups, UFW, out-of-band elevation, Broker-owned SQLite, operation journaling, fencing locks, and tamper-evident audit.

Capabilities existing in the package does not mean they are enabled. `config/policy.yaml` sets the static baseline. Broker-owned dynamic root delegations may add narrower runtime resource authority only after an explicit approval request; they never widen the physical ceiling or enable actions absent from the static capability policy.

Destructive actions such as recursive delete, chmod/chown, package management, user management, firewall changes, unrestricted network, and administrative shell are explicit policy choices. file.chmod can set 0777 when the owner enables that capability.

## Security properties

- Gateway runs non-root and does not receive /host.
- Scoped packaging physically mounts only `VPS_AGENT_SCOPE_ROOT`; whole-host filesystem access requires the explicit `compose.host.yaml` override.
- Broker has no remote TCP API; Gateway reaches it through a Unix socket.
- Broker re-authorizes every privileged request against policy.
- Gateway never receives the Docker socket directly.
- filesystem resolution rejects traversal and symlink escapes.
- writes use operation journaling and idempotency semantics.
- resource mutations use locks/fencing where required.
- tool output is untrusted data and cannot grant authority.
- Full/elevation never implicitly grants unrestricted network.
- secrets are not returned through generic tools.

## Validation

GitHub Actions validates vet, race detector, govulncheck, adversarial simulation, real systemd jobs, real Docker operations, native Gateway -> Broker -> Linux acceptance, Docker package -> host acceptance, and negative authorization tests.

See [implementation validation](docs/implementation-validation.md).

## Operations

~~~bash
bash scripts/diagnose.sh status
bash scripts/diagnose.sh health
bash scripts/diagnose.sh logs 200
bash scripts/diagnose.sh audit 100

bash scripts/update.sh
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Safe removal preserves configuration and audit state. Purge requires explicit confirmation and removes only MCP-owned artifacts. Updates back up operator configuration/state, use fast-forward Git updates, verify the new runtime, and roll back code/state on verification failure.
## Planned evolution

The next major architectural direction is a **multi-node Portico control plane**: one stable MCP surface for ChatGPT to discover and operate multiple owner-authorized computers and servers, with local policy enforcement on every node. The plan covers Linux nodes, the SRV-IA as the first additional node, a native Windows node, local RAG/databases, GPU/compute jobs, aggregated audit and optional local-LLM fallback.

See [Multi-node control-plane roadmap](docs/roadmap-multinode-control-plane.md).

## Documentation

- [Quick Start](docs/quick-start.md)
- [Installation contract](docs/installation-contract.md)
- [Product model](docs/product-model.md)
- [Multi-node control-plane roadmap](docs/roadmap-multinode-control-plane.md)
- [Operations](docs/operations.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Compatibility](docs/compatibility.md)
- [Privacy and telemetry](docs/privacy.md)
- [Release policy](docs/releases.md)
- [Support](docs/support.md)
- [Architecture](docs/architecture.md)
- [Authentication](docs/authentication.md)
- [ChatGPT integration](docs/chatgpt-integration.md)
- [Security release matrix](docs/security-release.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for engineering principles and validation expectations, and [SECURITY.md](SECURITY.md) for private vulnerability-reporting guidance.

## License

Apache-2.0. See [LICENSE](LICENSE).

Português: [README.pt-BR.md](README.pt-BR.md).
