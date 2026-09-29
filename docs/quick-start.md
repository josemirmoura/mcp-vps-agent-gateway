# Quick Start

The supported installation is terminal-first and uses the repository's transparent Docker Compose and shell scripts.

## Requirements

Before running the guided flow:

- Linux VPS; Ubuntu 24.04 LTS is the release-candidate validated target;
- Docker Engine 24+ and Docker Compose v2;
- at least 2 GB RAM for the bundled ZITADEL path;
- Git, OpenSSL, Python 3 and curl;
- public DNS for the MCP hostname, TCP 80/443 available and valid HTTPS;
- **ChatGPT Plus or higher**, with Developer Mode and custom MCP app creation actually exposed in ChatGPT Web for that account.

OpenAI controls plan availability and rollout. Its current documentation describes full MCP write/modify support for Business, Enterprise and Edu. Accounts that expose only read/fetch MCP permissions remain limited to those ChatGPT-side capabilities. Recheck the current OpenAI product surface before public setup.

## Release candidate

Until the first stable tag is frozen after the final operator acceptance gate:

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

The guided flow performs:

~~~text
Environment
 -> Scope
 -> Effective authority
 -> Containers
 -> Local verification
 -> Secure public access
 -> Connect ChatGPT
 -> real audited MCP call
 -> INSTALLATION COMPLETE
~~~

The default profile is **Project**. It delegates one filesystem root and leaves unrelated administrative capabilities disabled.

The installer can be safely rerun after an interrupted phase. Existing `.env`, policy, identity state and instance identity are preserved by the underlying lifecycle scripts.

## Authority profiles

### Project

~~~bash
bash scripts/install.sh --profile project --scope /opt/my-app
~~~

### Custom

Choose a physical filesystem ceiling and edit `config/policy.yaml` to restrict individual roots, services, Docker resources and capabilities.

~~~bash
bash scripts/install.sh --profile custom --scope /opt
~~~

### Delegated roots under a broader ceiling

For a multi-project VPS, the physical ceiling can be broader than the logical authority. For example, keep the Broker physically bounded by `/opt` while delegating only selected project directories:

~~~bash
bash scripts/init.sh --scope /opt
bash scripts/delegate-root.sh add /opt/project-a --access work --apply
bash scripts/delegate-root.sh add /opt/project-b --access compose --apply
bash scripts/delegate-root.sh list
~~~

`work` grants filesystem read/write plus scoped shell cwd for that directory. `compose` adds Compose inspect/manage for the same directory. `read` grants filesystem read only.

Changing the physical ceiling does not authorize the ceiling itself. Existing logical policy roots are preserved. To revoke a project later:

~~~bash
bash scripts/delegate-root.sh remove /opt/project-a --apply
~~~

The helper refuses to delegate the entire physical ceiling unless `--allow-ceiling` is explicitly supplied.

### Whole Host

Whole Host changes the Broker's **physical filesystem ceiling** to `/`.

It does not enable Full, unrestricted networking, package administration, user administration or firewall administration by itself.

~~~bash
bash scripts/install.sh --profile whole-host
~~~

The guided flow requires explicit confirmation before startup.

## Local-only validation

~~~bash
bash scripts/install.sh \
  --profile project \
  --scope /opt/vps-agent-sandbox \
  --local-only
~~~

This intentionally stops before public OAuth and ChatGPT. It must not be described as a completed installation.

## Stable releases

After `v0.1.0` is frozen, production-oriented installs should use a tagged Git checkout rather than `main`:

~~~bash
git clone --branch v0.1.0 \
  https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

The Git checkout is intentional: the existing update mechanism preserves fast-forward verification, state migration checks, backup and automatic rollback.
