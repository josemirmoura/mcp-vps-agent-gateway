# MCP VPS Agent Gateway

Security-first MCP control plane for letting ChatGPT or another MCP client work on a Linux VPS with authority explicitly chosen by the VPS owner.

> **Status: Docker-first release candidate for automated acceptance.** Clean Ubuntu 24.04 workflows validate the package end to end. A stable release still requires the external ChatGPT Web OAuth gate and a real audited call against the target deployment.

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

Requirements: Linux VPS, Docker Engine, Docker Compose plugin, Git, OpenSSL and Python 3.

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git &&
cd mcp-vps-agent-gateway &&
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/vps-agent-sandbox &&
bash scripts/init.sh --scope /opt/vps-agent-sandbox &&
docker compose up -d --build &&
bash scripts/verify.sh
~~~

The Quick Start deliberately delegates only `/opt/vps-agent-sandbox`; replace that path with the directory you want the MCP to control. The `&&` chain stops at the first failed step.

The bootstrap creates random local secrets, a stable instance ID, local state and an **untracked** operator policy from config/policy.example.yaml. `--scope` persists `VPS_AGENT_SCOPE_ROOT` in `.env` and migrates template policy paths from the previous scoped root. The selected directory must already exist; bootstrap now fails before any Docker build if it does not. The user still chooses the authority explicitly.

Local verification proves health, invalid-token denial, a real MCP system.info call and audit-chain integrity. **Installation is still not complete.**

### Whole-host filesystem authority

Scoped is the default. To deliberately expose the whole host filesystem to the Broker, use the explicit override:

~~~bash
sed -i 's/^VPS_AGENT_WHOLE_HOST=.*/VPS_AGENT_WHOLE_HOST=1/' .env
docker compose -f compose.yaml -f compose.host.yaml up -d --build
~~~

`compose.host.yaml` is the deliberate whole-host switch and sets the Broker's physical ceiling to `/`. Server-side policy still controls which MCP operations are allowed.
### Public HTTPS + ChatGPT

For a write-capable ChatGPT connection, use a standards-based OAuth/OIDC authorization server and set the public endpoint, issuer, audience/resource and expected token subject in .env. Static bearer auth is intentionally local/lab only.

~~~dotenv
VPS_AGENT_DOMAIN=mcp.example.com
VPS_AGENT_PUBLIC_URL=https://mcp.example.com/mcp
VPS_AGENT_AUTH_MODE=oidc
VPS_AGENT_OIDC_ISSUER=https://auth.example.com
VPS_AGENT_OIDC_AUDIENCE=https://mcp.example.com/mcp
VPS_AGENT_SUBJECT=<expected-token-subject>
~~~

Then:

~~~bash
docker compose -f compose.yaml -f compose.https.yaml up -d --build
bash scripts/verify-public.sh
bash scripts/connect-chatgpt.sh
~~~

The public verifier requires HTTPS, OAuth discovery and fail-closed unauthenticated access. The connection script then shows the current ChatGPT Web flow and waits for a **new audited system.info call from ChatGPT**.

~~~text
tutorial shown
 -> OAuth app connected in ChatGPT
 -> ChatGPT calls the MCP
 -> Gateway authenticates
 -> Broker verifies subject + policy
 -> authorized VPS operation succeeds
 -> audit records it
 -> INSTALLATION COMPLETE
~~~

If the real ChatGPT call never arrives, setup remains incomplete.

Current OpenAI guidance checked on 2026-09-27 documents full MCP write/modify support on ChatGPT Web for Business, Enterprise and Edu workspaces. Pro is limited to read/fetch MCP access. Authenticated write-capable MCP servers are expected to use OAuth 2.1; ChatGPT does not present custom API keys.

Official references:
- https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt
- https://developers.openai.com/plugins/build/auth
## Toolbox

The current reference implementation includes complete scoped filesystem CRUD, durable sandboxed shell/jobs, typed systemd, Docker/Compose, diagnostics, packages, users/groups, UFW, out-of-band elevation, Broker-owned SQLite, operation journaling, fencing locks, and tamper-evident audit.

Capabilities existing in the package does not mean they are enabled. config/policy.yaml decides.

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
## Documentation

- [Product model](docs/product-model.md)
- [Docker first-run flow](docs/installer-flow.md)
- [Architecture](docs/architecture.md)
- [Authentication](docs/authentication.md)
- [Multi-instance](docs/multi-instance.md)
- [ChatGPT integration](docs/chatgpt-integration.md)
- [Threat model](docs/threat-model.md)
- [Security hardening](docs/security-hardening-v2.md)

## License

Apache-2.0. See [LICENSE](LICENSE).

Português: [README.pt-BR.md](README.pt-BR.md).
