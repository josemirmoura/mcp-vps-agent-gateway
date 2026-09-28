# Quick Start

The supported installation is terminal-first and uses the repository's transparent Docker Compose and shell scripts.

## Release candidate

Until the first stable tag is frozen after the final human gate:

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
git clone --branch v0.1.0 --depth 1 \
  https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

The Git checkout is intentional: the existing update mechanism preserves fast-forward verification, state migration checks, backup and automatic rollback.
