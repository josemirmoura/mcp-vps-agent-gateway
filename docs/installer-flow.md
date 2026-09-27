# Installer and first-run flow

## Product objective

Installation should feel like one command even though the runtime preserves separate privilege boundaries.

The installer is responsible for turning operator intent into a validated server-side policy and a working ChatGPT connection.

## Phase 1 — Preflight

Check:

- supported Linux/systemd environment
- architecture
- required ports/HTTPS route
- existing Docker/systemd availability
- current user has installation authority
- conflicting installations
- backup/rollback location

## Phase 2 — Choose MCP authority

The user explicitly chooses the resources delegated to MCP.

The installer can offer presets, but every preset expands into an editable policy preview.

Examples:

~~~text
Filesystem:
[x] /opt/app
[x] /var/www/site
[ ] /

systemd:
[x] app.service
[x] nginx.service

Docker:
[x] app-stack

Shell:
[x] enabled inside selected roots

Network:
[x] api.github.com:443
[ ] unrestricted

Administration:
[ ] apt
[ ] users/groups
[ ] firewall
[ ] admin shell
~~~

The user owns this choice.

## Phase 3 — Capability selection

Expose the complete supported capability catalog and let policy enable only what the operator wants.

High-risk capabilities must be visually distinguished, especially:

- recursive delete
- chmod/chown
- package management
- firewall
- user management
- unrestricted network
- administrative shell

## Phase 4 — Policy preview and confirmation

Render a human-readable summary before activation.

Example:

~~~text
MCP WILL BE ABLE TO:
- read/write/delete under /opt/app
- run sandboxed commands with cwd under /opt/app
- restart app.service
- inspect/restart containers in app-stack

MCP WILL NOT BE ABLE TO:
- access /etc
- manage ssh.service
- administer users
- change firewall
- use unrestricted network
~~~

Require explicit confirmation.

## Phase 5 — Install runtime

Install:

- vps-agent-broker as native privileged systemd service
- vps-agent-gateway as non-root service or supported container deployment
- Unix socket permissions
- state directories
- policy
- authentication configuration
- TLS/reverse-proxy integration when required

## Phase 6 — Validate locally

Run:

- policy validation
- filesystem confinement checks
- Broker/Gateway connectivity
- privilege-boundary checks
- health/readiness
- audit verification
- harmless MCP tool call

Do not continue on security-critical failure.

## Phase 7 — Show ChatGPT Web connection tutorial

This is a mandatory installation stage, but it is **not** the completion stage.

Display:

- final MCP HTTPS endpoint
- configured authentication method
- exact effective scope
- current supported ChatGPT Web connection route

Then guide the operator step by step through connecting the MCP in ChatGPT Web.

At the end of this phase, the installer must report something equivalent to:

~~~text
Tutorial completed.
Waiting for ChatGPT connection verification...
Installation is NOT complete yet.
~~~

The tutorial must use current official OpenAI instructions appropriate to the user's ChatGPT surface at install/release time.

## Phase 8 — Verify the real ChatGPT connection

After the operator has followed the tutorial and connected ChatGPT Web, ask the user to send a harmless command from ChatGPT itself, such as:

~~~text
Read a test file from the authorized directory and tell me its contents.
~~~

Then verify on the server, automatically when possible:

- the MCP request arrived
- the expected subject was authenticated
- policy allowed the intended resource only
- the Broker executed it
- audit recorded it
- the returned result matches the expected test result

If any of these checks fail, installation remains incomplete and the installer should provide diagnostics/retry guidance.

Optionally run a negative proof against an unauthorized resource.

## Completion criterion

Installation finishes only after:

~~~text
runtime healthy
+ policy active
+ MCP endpoint reachable
+ ChatGPT connected
+ first end-to-end tool call verified
+ audit valid
~~~

Until every item above is verified, the installer must report the deployment as **incomplete**.

The final success message belongs **after Phase 8**, never after merely displaying the tutorial.
