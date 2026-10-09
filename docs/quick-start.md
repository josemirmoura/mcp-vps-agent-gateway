# Portico MCP Quick Start

[Português (Brasil)](quick-start.pt-BR.md) · [Installation contract](installation-contract.md)

Portico MCP is installed from the terminal using the repository's transparent Docker Compose and shell scripts.

## Requirements

Before running the guided flow:

- Linux VPS; Ubuntu 24.04 LTS is the release-candidate validated target;
- Docker Engine 24+ and Docker Compose v2;
- at least 2 GB RAM for the bundled ZITADEL path;
- Git, OpenSSL, Python 3 and curl;
- public DNS for the MCP hostname, TCP 80/443 available and valid HTTPS;
- a ChatGPT Web account/workspace where Developer Mode and custom MCP app creation are actually exposed.

OpenAI controls plan/workspace availability and rollout. Treat the actual product surface as the compatibility gate rather than assuming a plan label guarantees capabilities. Current official documentation describes full MCP write/modify support for Business, Enterprise and Edu; separately, on 2026-10-06 the operator's ChatGPT Plus environment exposed Portico write-capable tools and completed a real scoped write/read/delete proof. That observation applies to the tested environment only. Recheck the current product surface and official documentation before public setup.

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
Prerequisites
 -> Scope
 -> Effective authority
 -> Containers
 -> Local verification
 -> Secure public access
 -> Connect ChatGPT
 -> real audited MCP call
 -> INSTALLATION COMPLETE
~~~

Before creating Portico state, the installer runs an explicit prerequisite preflight. It checks the supported Linux/runtime basics, Docker Engine/Compose, host tools, systemd, memory and (for the public path) the edge-proxy situation. Missing mandatory prerequisites stop the flow with an official upstream installation link.

If one existing Traefik is detected, Portico reuses it. If none is detected and ports 80/443 are free, Portico provisions its bundled Traefik. It never replaces an unknown service already occupying those ports.

The installer can be safely rerun after an interrupted phase. Existing local operator state is preserved by the lifecycle scripts unless an explicit purge is requested.

## Recommended authority model: Standard

The default interactive choice is **Standard**.

Standard uses:

~~~text
physical filesystem ceiling: /opt
static project roots:        none
project authority:           granted later through explicit approval
~~~

The physical ceiling defines where Portico MCP may ever be allowed to operate. It does **not** authorize `/opt` itself at install time.

Later, the operator may explicitly approve either a subdirectory or the exact physical ceiling. Approving the exact ceiling is shown with a stronger warning because it grants the selected access profile to all current and future paths under that ceiling until revocation.

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
host-owned MCP elicitation when supported and verified
        |
        v
Broker applies the policy and activates only an authorized delegation
~~~

The native in-chat approval surface is not universally available in ChatGPT and has not completed desktop/mobile acceptance for the final Community candidate. If the client does not support that flow, the request stays pending and the operator must use only a separately deployed and authenticated approval channel supported by the exact installed runtime. A URL or request ID cannot authorize anything.

The model cannot approve its own permission expansion. On elicitation-capable clients, opaque approval state travels only through the protocol round trip; the model does not receive a self-approval tool or usable approval token. The Broker binds the decision to the authenticated subject and pending request.

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

## Public hostname and OAuth operator

For the public ChatGPT path, the guided installer explains how to create a DNS hostname such as `mcp.example.com`, point an A/AAAA record to the VPS and enter the hostname without `https://` or `/mcp`. It verifies DNS before continuing.

The integrated OAuth setup creates a dedicated Portico operator identity. The installer shows both:

~~~text
Username: vps-operator
Email:    operator@example.com
~~~

The OAuth login screen uses the username. This identity is separate from Linux/SSH/root credentials.

## ChatGPT completion

After public OAuth verification, `scripts/connect-chatgpt.sh` shows a short connection tutorial with the exact MCP endpoint and operator username.

Interactive installs do not have a hidden countdown. Complete the ChatGPT app/OAuth setup at your own pace, send the harmless `system.info` test call, then return to the terminal and press Enter. If the audited call is not present yet, Portico explains what to check and lets you retry.

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

## Release selection and installation safety

The `v0.1.0` stable release is **not yet published**. Do not try to clone a nonexistent stable tag or treat a moving `main` checkout as a signed release.

For engineering/preview inspection, use the clone command at the top of this guide. For a production-oriented installation **after an exact release is published and verified**, choose the published immutable tag and verify the matching source and image signatures/digests first. The installer uses a Git checkout because controlled updates need fast-forward checks, backup and rollback, subject to actual lifecycle acceptance.

Until then, this is a pre-stable candidate, not a guarantee of successful recovery or completed mobile authorization UX. Final operator acceptance includes a clean Linux install, an authenticated real MCP client call and Broker audit proof.
