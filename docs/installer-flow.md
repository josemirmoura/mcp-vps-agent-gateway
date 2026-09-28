# Docker first-run flow

## Product objective

Installation is declarative and terminal-first.

The operator controls two human/AI-readable local files plus one physical scope variable:

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

## Phase 1 — Bootstrap

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/vps-agent-sandbox
bash scripts/init.sh --scope /opt/vps-agent-sandbox
~~~

The user supplies the scoped root explicitly. The bootstrap checks Docker Compose, creates .env with random local secrets and a stable instance ID when needed, creates config/policy.yaml from the versioned template, persists the selected scope, migrates template policy paths from the previous scope, creates the local state directory, and validates Compose syntax. If the scoped root does not exist, bootstrap fails before the Docker build instead of allowing Compose to fail later.

## Physical filesystem ceiling

The default `compose.yaml` bind-mounts only `VPS_AGENT_SCOPE_ROOT` into the Broker under `/host`. The Broker validates that filesystem, shell cwd and Compose paths in policy remain inside that physical root. A policy escape makes the Broker fail closed.

For deliberate whole-host filesystem authority, set `VPS_AGENT_WHOLE_HOST=1` in `.env` **and** add `compose.host.yaml`. The persisted flag lets lifecycle commands reuse the same deployment mode; the override sets the Broker's physical root to `/`. Neither is part of the default Scoped command.
## Phase 2 — The user chooses MCP authority

The operator may edit config/policy.yaml after bootstrap for finer-grained authority. The user decides exactly which VPS resources are delegated.

One root or several logical roots are valid when they fit under `VPS_AGENT_SCOPE_ROOT`. Whole-host `/` requires the explicit `compose.host.yaml` override.

systemd, Docker/Compose, shell, network, packages, users/groups, firewall, and temporary elevation are scoped separately.

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

The package does not control the user's DNS provider, so DNS is an unavoidable manual platform action. Create a DNS A/AAAA record pointing a hostname at the VPS. The setup script validates that the hostname resolves before it proceeds, and the public verifier later confirms valid HTTPS.

Then run:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

The script reuses a single existing Traefik when one is present. If none is present and ports 80/443 are free, it starts the bundled Traefik. It then starts ZITADEL + PostgreSQL, creates the dedicated non-admin operator identity, creates an MCP-only OAuth resource audience plus a private introspection client, enables MCP-compatible Dynamic Client Registration, configures the Gateway/Broker identity binding and runs the public verification.

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

## Phase 6 — Show the current ChatGPT Web tutorial

~~~bash
bash scripts/connect-chatgpt.sh
~~~

The ChatGPT product UI is an unavoidable manual platform action: the user must create/select the MCP app and complete the OAuth browser login. The script displays the endpoint, authentication mode, expected subject, and the current connection steps. The user signs in with the dedicated OAuth operator account, never with VPS/SSH credentials.

At this point:

~~~text
Tutorial shown.
Installation is NOT complete.
~~~

## Phase 7 — Verify the real ChatGPT connection

The connection script waits for an audited call from the configured subject.

The user asks ChatGPT to call system.info.

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

The package never deletes applications, services, containers, databases or files merely because it was authorized to manage them.
