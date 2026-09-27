# ChatGPT integration

## Target

Desired experience:

~~~text
ChatGPT Web
   -> plugin/app/MCP
   -> VPS Agent Gateway
   -> Broker
   -> VPS
~~~

The server architecture is client-independent. The ChatGPT distribution path is a separate product constraint.

## Current product constraint

**Checked: 2026-09-26.**

OpenAI currently documents full private MCP support, including write/modify actions in Developer Mode, for ChatGPT Business, Enterprise and Edu:

https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

The plugin directory is available across ChatGPT plans, but availability and capabilities vary by plan, surface, region, role and the apps included in a plugin:

https://help.openai.com/en/articles/20001256-plugins-in-chatgpt-and-codex

Therefore the project must not assume that a private custom write-capable MCP can be attached directly to ChatGPT Plus Web.

## Route A — Private development

Use a plan/workspace that currently supports private full MCP write in Developer Mode.

This is the simplest private testing route.

## Route B — Plus Web

If Plus Web is a hard requirement, the practical path is an eligible plugin/app whose remote MCP capabilities are available on that surface.

This can require:

- stable public HTTPS MCP endpoint
- plugin/app packaging
- authentication
- review/submission requirements
- actual availability on the target Plus account

Do not mark Plus Web complete until the plugin/app is visible there and a write tool executes successfully.

## Route C — Protocol development

Use MCP Inspector while the ChatGPT distribution path is unresolved.

This lets Gateway/Broker engineering proceed without pretending the product surface is already available.

## Gate 0A artifact

Before privileged implementation, record:

~~~text
target_surface:
target_plan:
integration_route:
read_available:
write_available:
private_or_published:
tested_date:
~~~

## Transport

Use MCP Streamable HTTP over HTTPS, typically at /mcp.

Official guidance:

https://developers.openai.com/plugins/concepts/mcp-server
https://developers.openai.com/plugins/build/mcp-server

Do not create a custom WebSocket protocol.

## Authentication

For private user data or write actions, use the authentication required by the current MCP/OpenAI integration.

For OAuth/OIDC, validate:

- issuer
- audience/resource
- signature
- expiration
- scopes
- subject

Authentication identifies the caller. Broker policy still decides authorization.

## Tool ladder

Gate 0B:

~~~text
system.info
file.read_test
file.write_test
~~~

Gate 1:

~~~text
system.info
file.read_test
service.status
service.restart
~~~

Do not expose the full north-star tool catalog before the gates justify it.

## Host confirmations

Host confirmations improve UX and safety. They do not replace server-side authorization.


## Installer handoff

The product installer must treat the **verified ChatGPT Web connection** as the final completion gate.

Showing the connection tutorial happens first. It does not complete installation.

The installer should output the configured MCP endpoint, authentication method and effective policy summary, then guide the user through the currently supported ChatGPT Web connection path.

After the user connects ChatGPT, the installer/runbook must require one harmless end-to-end tool call from ChatGPT itself and verify the corresponding authenticated subject, policy decision, execution result and Broker audit event.

Only after those checks pass may the installer declare setup complete.

Exact UI instructions are version-sensitive and must be checked against current official OpenAI documentation at release/install time.
