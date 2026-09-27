# Multiple VPS instances in one ChatGPT workspace

This is an advanced scenario. The normal onboarding remains one VPS Agent instance per installation.

## Identity model

Tool names stay stable across installations. Do not rename tools per server.

Each installation instead gets:

- its own HTTPS MCP endpoint;
- its own OAuth client/resource relationship and credentials;
- a stable VPS_AGENT_INSTANCE_ID generated at bootstrap;
- an operator-visible VPS_AGENT_INSTANCE_NAME;
- instance identity in system.info, Broker audit events and structured logs.

Example:

~~~dotenv
VPS_AGENT_INSTANCE_NAME=VPS Agent | Loja
VPS_AGENT_PUBLIC_URL=https://mcp-loja.example.com/mcp
~~~

On another VPS:

~~~dotenv
VPS_AGENT_INSTANCE_NAME=VPS Agent | Blog
VPS_AGENT_PUBLIC_URL=https://mcp-blog.example.com/mcp
~~~

Register them as separate ChatGPT apps with matching visible names. Select or @mention the intended app rather than relying on ChatGPT to silently disambiguate identical tool names.

## Verification

For each app, call system.info and compare instance_id, instance_name and hostname with the expected VPS. Then inspect the Broker audit on that VPS:

~~~bash
bash scripts/diagnose.sh audit 20
~~~

The same instance_id/name should appear in new audit events and structured logs.

## Same-conversation use

If the current ChatGPT interface allows multiple draft/custom MCP apps in the same conversation, explicitly select or mention the intended app before an operation. UI behavior is product-surface behavior and may change independently of this server.

## Release gate

Automated tests prove instance identity survives Broker -> Gateway -> MCP and is recorded in audit/logs. The product-surface test still requires connecting at least two live apps to the same eligible ChatGPT workspace and manually verifying calls land on the intended VPS. That test cannot be simulated faithfully by the repository CI.
