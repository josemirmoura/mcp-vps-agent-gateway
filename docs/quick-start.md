# Portico MCP Quick Start

Portico MCP is installed from the terminal using the repository's transparent Docker Compose and shell scripts.

## Requirements

Before running the guided flow:

- Linux VPS; Ubuntu 24.04 LTS is the release-candidate validated target;
- Docker Engine 24+ and Docker Compose v2;
- at least 2 GB RAM for the bundled ZITADEL path;
- Git, OpenSSL, Python 3 and curl;
- public DNS for the MCP hostname, TCP 80/443 available and valid HTTPS;
- **ChatGPT Plus or higher**, with Developer Mode and custom MCP app creation actually exposed in ChatGPT Web for that account.

OpenAI controls plan availability and rollout. Recheck the current ChatGPT product surface before public setup.

## Start here

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

The installer detects the terminal locale. Portuguese (Brazil) and English are supported initially. You can override detection explicitly:

~~~bash
bash scripts/install.sh --lang pt-BR
# or
bash scripts/install.sh --lang en
~~~

The terminal banner identifies the product as **Portico MCP**, shows the version and includes:

~~~text
Feito por Josemir Moura | github.com/josemirmoura
~~~

## Guided flow

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

The installer can be safely rerun after an interrupted phase. Existing local operator state is preserved by the lifecycle scripts unless an explicit purge is requested.

## Recommended authority model: Standard

The default interactive choice is **Standard**.

Standard uses:

~~~text
physical filesystem ceiling: /opt
static project roots:        none
project authority:           granted later through explicit approval
~~~

The physical ceiling defines where Portico MCP may ever be allowed to operate. It does **not** authorize `/opt` itself.

Equivalent non-interactive command:

~~~bash
bash scripts/install.sh --profile custom --scope /opt
~~~

The CLI value remains `custom` for compatibility, while the interactive product label is **Standard**.

After ChatGPT is connected, a project is authorized dynamically:

~~~text
permissions.request_root_access
        |
        v
in-chat Authorize / Deny card
        |
        v
Broker activates read / work / compose for that root
~~~

The model cannot approve its own permission expansion. The approval token is delivered only to the app UI and the Broker binds the decision to the authenticated subject and pending request.

## Project-locked profile

Choose this when Portico MCP should never operate outside one project root.

~~~bash
bash scripts/install.sh \
  --profile project \
  --scope /opt/my-app
~~~

If the path does not exist, the interactive installer can create it explicitly, or you can use:

~~~bash
bash scripts/install.sh \
  --profile project \
  --scope /opt/my-app \
  --create-scope
~~~

## Whole Host

Whole Host changes the Broker's **physical filesystem ceiling** to `/`.

It does not enable Full, unrestricted networking, package administration, user administration or firewall administration by itself.

~~~bash
bash scripts/install.sh --profile whole-host
~~~

The guided flow requires explicit confirmation before startup.

## Shell execution user

Confined shell jobs run as a real non-root host user. The guided installer detects the invoking non-root user automatically.

To choose another existing non-root user:

~~~bash
bash scripts/install.sh --run-as deploy
~~~

Normal scoped shell jobs are never configured to run as root.

## Dynamic project roots

With Standard, no project root is authorized at installation time.

Once connected to ChatGPT, use:

~~~text
permissions.request_root_access
permissions.list_root_access
permissions.revoke_root_access
~~~

Access profiles:

- `read`: filesystem read;
- `work`: filesystem read/write plus scoped shell cwd;
- `compose`: `work` plus Compose operations already enabled by the static action policy.

Dynamic roots can be permanent or time-limited and take effect without restarting the Broker.

Advanced operators can still manage static roots from the terminal with `scripts/delegate-root.sh`, but this is not required by the normal guided installation.

## Local-only validation

To validate the package without public OAuth or ChatGPT:

~~~bash
bash scripts/install.sh \
  --profile custom \
  --scope /opt \
  --local-only \
  --yes
~~~

This starts with no static project roots and intentionally stops before public OAuth and ChatGPT. It must not be described as a completed installation.

## Removal

Safe removal preserves local operator configuration and audit state:

~~~bash
bash scripts/remove.sh safe
~~~

Full purge removes Portico MCP-owned runtime, volumes, local configuration/state and the default locally built Gateway/Broker images:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
bash scripts/remove.sh --purge
~~~

To also delete the Git checkout, use the separate explicit confirmation:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

The purge never deletes arbitrary delegated project directories, third-party images, applications, databases or services. The legacy `/opt/vps-agent-sandbox` directory is removed only when it is empty.

## Stable releases

After `v0.1.0` is frozen, production-oriented installs should use a tagged Git checkout rather than `main`:

~~~bash
git clone --branch v0.1.0 \
  https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

The Git checkout is intentional: the update mechanism preserves fast-forward verification, state migration checks, backup and automatic rollback.
