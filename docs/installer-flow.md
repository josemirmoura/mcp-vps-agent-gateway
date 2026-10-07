# Portico MCP first-run flow

## Product objective

Installation is declarative and terminal-first.

The operator controls two operator/AI-readable local files plus one physical scope variable:

~~~text
.env                    # includes VPS_AGENT_SCOPE_ROOT
config/policy.yaml     # logical capability/resource policy
~~~

Both are local operator state. config/policy.yaml is created from the versioned config/policy.example.yaml template and is intentionally kept out of Git.

Then Docker Compose starts the package. No separate wizard or native installer owns the configuration.

## Supported user installation boundary

This document describes the supported end-user installation. It does not require a developer, a remote shell controlled by ChatGPT, or disclosure of VPS credentials.

The user runs the documented commands directly on the VPS. The tutorial never requires the VPS password, a private SSH key, unrestricted remote administrative access, or unrelated secrets to be supplied to ChatGPT or to the project maintainers.

The dedicated OAuth operator password is different: it is a service credential required by the integrated identity stack. It is entered locally into `setup-integrated-auth.sh` without terminal echo and is not supplied to ChatGPT.

See [installation-contract.md](installation-contract.md) for the normative boundary between supported installation and development-only procedures.

## Guided entry point

The normal user starts with:

~~~bash
bash scripts/install.sh
~~~

This script is an orchestration layer, not a second installer implementation. It calls the same transparent components described below, shows the effective authority before runtime startup, pauses only for unavoidable operator decisions, and can be rerun after an interrupted phase.

Advanced operators can still execute the phases individually.

## Phase 1 — Bootstrap

The supported guided entry point is:

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

The recommended interactive profile is **Standard**: physical ceiling `/opt` with no static project roots. The ceiling is only the maximum filesystem boundary; it does not grant project read/write authority. A different absolute ceiling may be selected when the operator organizes applications elsewhere.

Advanced operators can reproduce that bootstrap explicitly:

~~~bash
bash scripts/init.sh --scope /opt --dynamic-baseline
~~~

Before bootstrap mutates Portico state, the guided installer runs `scripts/preflight.py`. It checks the supported host/runtime prerequisites and, for the public path, whether an existing Traefik can be reused or whether the bundled proxy can safely own ports 80/443. Missing mandatory prerequisites stop before `.env`, policy or runtime state is created.

The bootstrap then checks Docker Compose, creates `.env` with random local secrets and a stable instance ID when needed, creates `config/policy.yaml` from the versioned template, persists the selected physical ceiling, configures the confined shell to use a real non-root host user, creates the local state directory, and validates Compose syntax.

## Physical filesystem ceiling

The default `compose.yaml` bind-mounts only `VPS_AGENT_SCOPE_ROOT` into the Broker under `/host`. The Broker validates that filesystem, shell cwd and Compose paths in policy remain inside that physical root. A policy escape makes the Broker fail closed.

Changing `VPS_AGENT_SCOPE_ROOT` on an existing installation changes only this physical ceiling. Existing logical roots in `config/policy.yaml` are preserved by default. The Standard fresh-install path uses `--dynamic-baseline`, so choosing `/opt` does not create a static `/opt` authorization. `scripts/init.sh --migrate-policy-root` remains an explicit opt-in for a real project-root move.

For deliberate whole-host filesystem authority, set `VPS_AGENT_WHOLE_HOST=1` in `.env` **and** add `compose.host.yaml`. The persisted flag lets lifecycle commands reuse the same deployment mode; the override sets the Broker's physical root to `/`. Neither is part of the default Scoped command.
## Phase 2 — The user chooses MCP authority

With the recommended Standard profile, no project root is authorized during installation.

Portico can use `permissions.discover_scope` to list only the immediate directory names below the physical ceiling. This discovery surface does not open files or descend into those directories. It exists so the assistant can explain what it can see and request the exact project needed instead of asking the operator to remember server paths.

After the MCP is connected, project roots are granted through explicit runtime approval:

~~~text
permissions.request_root_access
        -> pending Broker request
        -> native MCP elicitation / host-owned confirmation UI
        -> active subject-bound delegation
~~~

`read` grants filesystem read. `work` grants filesystem read/write plus scoped shell cwd. `compose` adds Compose authority only for Compose actions already enabled by the static action policy. On clients that advertise MCP elicitation, Portico asks the client to render the confirmation natively; no custom approval iframe is part of the normal ChatGPT path. Clients without elicitation leave the request pending for the separate operator fallback.

Protected secret-bearing paths form a second boundary inside authorized projects. Their temporary exception uses the same native MCP elicitation flow when the client supports it. By default `.env` and `.env.*` remain locked while `.env.example`, `.env.sample` and `.env.template` remain ordinary readable templates. Reading or modifying a protected path requires a separate temporary `permissions.request_sensitive_access` approval. The grant is exact-path, subject-bound, auditable, expiring and independently revocable. Scoped shell jobs mask protected paths, including hardlink aliases discovered inside the delegated roots, unless a temporary protected-file `work` grant explicitly exposes that exact path.

Advanced operators may still define static roots with `scripts/delegate-root.sh`. Manual editing of `config/policy.yaml` is not required by the normal guided installation.

Dynamic approval may target a subdirectory or, after a stronger explicit warning, the exact configured physical ceiling. The ceiling is still the hard maximum boundary; no dynamic approval can escape it. Granting the ceiling itself is intentionally broad because it covers current and future projects beneath it, but it still does not unlock protected secret files.

systemd, Docker/Compose actions, network, packages, users/groups, firewall, and temporary elevation remain separately constrained by static server policy.

## Phase 3 — Start

~~~bash
docker compose up -d --build
~~~

Runtime separation:

~~~text
Gateway: non-root, no host root
Broker: privileged, host mounted at /host, no remote control port
~~~

Docker packages the privileged Broker. Docker is not the Broker's authorization boundary. Policy is.

## Phase 4 — Local verification

~~~bash
bash scripts/verify.sh
~~~

This verifies Compose configuration, Broker health, Gateway health, audit integrity, authentication, and a harmless system.info call.

A local verification success means the runtime is ready. It does **not** mean installation is complete.

## Phase 5 — Integrated OAuth + public endpoint

ChatGPT needs a reachable remote HTTPS MCP endpoint, but the operator should not have to assemble an identity provider by hand.

The package does not control the user's DNS provider, so DNS is an unavoidable manual platform action. The guided setup explains how to create a subdomain, point a DNS A/AAAA record at the VPS and enter only the hostname. The setup script validates that the hostname resolves before it proceeds, and the public verifier later confirms valid HTTPS.

Then run:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

The script reuses a single existing Traefik when one is present. If none is present and ports 80/443 are free, it starts the bundled Traefik. It then starts ZITADEL + PostgreSQL, creates the dedicated non-admin operator identity, shows the OAuth username and email that will be used at login, creates an MCP-only OAuth resource audience plus a private introspection client, enables MCP-compatible Dynamic Client Registration, configures the Gateway/Broker identity binding and runs the public verification.

A successful phase ends with:

~~~text
INTEGRATED AUTH: READY
~~~

The setup already runs the public verification. It can be rerun independently after any DNS, proxy or OAuth change:

~~~bash
bash scripts/verify-public.sh
~~~

That command validates the public HTTPS certificate/route, OAuth/OIDC discovery, protected-resource metadata, DCR/PKCE expectations, private token introspection and fail-closed unauthenticated MCP behavior.

The script refuses to replace an unknown service already occupying 80/443.

## Phase 6 — Confirm ChatGPT MCP capability

Before starting the ChatGPT-side connection, inspect the actual feature surface of the target account/workspace.

Continue when ChatGPT exposes Developer Mode / Plugins with an option to create a custom MCP app. Do not reject an account solely from its plan name, because OpenAI product rollouts and documentation can change independently.

If custom MCP creation is absent, the VPS/public OAuth side may still be healthy, but the ChatGPT-side completion gate cannot run on that account until the feature becomes available.

## Phase 7 — Show the current ChatGPT Web tutorial

~~~bash
bash scripts/connect-chatgpt.sh
~~~

The ChatGPT product UI is an unavoidable manual platform action: the user must create/select the MCP app and complete the OAuth browser login. The script displays the endpoint, authentication mode, expected subject, and the current connection steps. The user signs in with the dedicated OAuth operator account, never with VPS/SSH credentials.

At this point:

~~~text
Tutorial shown.
Installation is NOT complete.
~~~

## Phase 8 — Verify the real ChatGPT connection

The connection script first records a validated audit baseline. In an interactive terminal, the user completes ChatGPT setup at their own pace and returns to press Enter when ready to verify.

The user asks ChatGPT to call system.info. Each Enter checks for a matching audited call after the fixed baseline. If it is not present, Portico explains what to check and allows another attempt without restarting the installation.

Success requires:

~~~text
real ChatGPT MCP call
+ expected authenticated subject
+ policy allow
+ successful execution
+ Broker audit record
+ valid audit chain
~~~

Only then:

~~~text
CHATGPT WEB CONNECTION VERIFIED
INSTALLATION COMPLETE
~~~

If the call does not arrive or fails authorization, installation remains incomplete.

## Development-only procedures

Development and acceptance may use temporary probes, ephemeral runners, project-specific self-hosted runners, ad-hoc curl/openssl diagnostics, development branches, or a human operator who runs commands because the development session has no VPS execution channel.

Those are **not installation steps**. They belong in PR/issue evidence or development notes and must not be copied into the supported user tutorial.

The release review runs `python3 scripts/check-installation-contract.py` to detect environment-specific development artifacts and prohibited credential-sharing instructions in the supported installation documents.

## Updates

~~~bash
bash scripts/update.sh
~~~

The update helper backs up .env, policy and Broker state, applies a fast-forward Git update, rebuilds, verifies, and rolls code/state back if verification fails. The policy and .env remain operator-controlled configuration.

## Removal

~~~bash
bash scripts/remove.sh safe
~~~

Safe removal stops the package while preserving configuration/audit state. Full purge is a separate explicit action:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

It also removes the default locally built Portico MCP Gateway/Broker images and removes the legacy `/opt/vps-agent-sandbox` only when that directory is empty.

Deleting the Git checkout itself requires a second explicit confirmation:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

The package never deletes arbitrary delegated project directories, third-party images, applications, services, containers, databases or files merely because it was authorized to manage them.
