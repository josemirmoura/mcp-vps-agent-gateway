# Docker first-run flow

## Product objective

Installation is declarative and terminal-first.

The operator edits two human/AI-readable files:

~~~text
.env
config/policy.yaml
~~~

Then Docker Compose starts the package. No separate wizard or native installer owns the configuration.

## Phase 1 — Bootstrap

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/init.sh
~~~

The bootstrap checks Docker Compose, creates .env with random local secrets when needed, creates the local state directory, and validates Compose syntax. It does not decide the scope.

## Phase 2 — The user chooses MCP authority

The operator edits config/policy.yaml. The user decides exactly which VPS resources are delegated.

One root, several roots, or / are all valid explicit choices.

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

## Phase 5 — HTTPS/public endpoint

ChatGPT needs a reachable remote HTTPS MCP endpoint.

Use the operator's existing reverse proxy/tunnel, or the optional Caddy override:

~~~bash
docker compose -f compose.yaml -f compose.https.yaml up -d --build
~~~

Set the final HTTPS /mcp URL in VPS_AGENT_PUBLIC_URL.

## Phase 6 — Show the current ChatGPT Web tutorial

~~~bash
bash scripts/connect-chatgpt.sh
~~~

The script displays endpoint, authentication mode, expected subject, and the current connection steps.

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

## Updates

~~~bash
git pull --ff-only
docker compose up -d --build
bash scripts/verify.sh
~~~

The policy and .env remain operator-controlled configuration.

## Removal

~~~bash
docker compose down -v
~~~

Deleting the repository/state/configuration is a separate explicit operator action. The package never automatically deletes the VPS resources it was authorized to manage.
