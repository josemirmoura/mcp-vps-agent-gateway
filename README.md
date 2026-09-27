# MCP VPS Agent Gateway

Security-first MCP control plane for letting ChatGPT or another MCP client work on a Linux VPS with authority explicitly chosen by the VPS owner.

> **Status: pre-alpha / executable Docker reference package.** The package is under active validation on clean GitHub-hosted Ubuntu machines. It is not yet a stable production release.

## The idea

One package ships the broad toolbox. **You decide what portion of the VPS the MCP may control.**

~~~text
one project:       /opt/my-app
several roots:     /opt/app + /var/www/site + /srv/data
whole filesystem:  /
~~~

Filesystem scope is only one dimension. The policy separately controls shell roots, systemd units, Docker/Compose resources, network, packages, users/groups, firewall, and temporary administration.

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

## Quick start

Requirements: Linux VPS, Docker Engine, Docker Compose plugin, Git, OpenSSL.

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway

bash scripts/init.sh
~~~

Then edit config/policy.yaml and .env.

~~~bash
docker compose up -d --build
bash scripts/verify.sh
~~~

Optional automatic HTTPS with Caddy:

~~~bash
docker compose -f compose.yaml -f compose.https.yaml up -d --build
~~~

Set VPS_AGENT_PUBLIC_URL in .env, then:

~~~bash
bash scripts/connect-chatgpt.sh
~~~

The connection script shows the ChatGPT Web tutorial and then waits for a **real audited system.info call from ChatGPT**. Showing the tutorial does not complete setup.

~~~text
tutorial shown
 -> user connects ChatGPT
 -> ChatGPT calls the MCP
 -> Broker sees expected subject
 -> policy allows the call
 -> audit records it
 -> INSTALLATION COMPLETE
~~~

If the real ChatGPT call never arrives, setup remains incomplete.

## Toolbox

The current reference implementation includes complete scoped filesystem CRUD, durable sandboxed shell/jobs, typed systemd, Docker/Compose, diagnostics, packages, users/groups, UFW, out-of-band elevation, Broker-owned SQLite, operation journaling, fencing locks, and tamper-evident audit.

Capabilities existing in the package does not mean they are enabled. config/policy.yaml decides.

Destructive actions such as recursive delete, chmod/chown, package management, user management, firewall changes, unrestricted network, and administrative shell are explicit policy choices. file.chmod can set 0777 when the owner enables that capability.

## Security properties

- Gateway runs non-root and does not receive /host.
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

## Documentation

- [Product model](docs/product-model.md)
- [Docker first-run flow](docs/installer-flow.md)
- [Architecture](docs/architecture.md)
- [ChatGPT integration](docs/chatgpt-integration.md)
- [Threat model](docs/threat-model.md)
- [Security hardening](docs/security-hardening-v2.md)

## License

Apache-2.0. See [LICENSE](LICENSE).

Português: [README.pt-BR.md](README.pt-BR.md).
