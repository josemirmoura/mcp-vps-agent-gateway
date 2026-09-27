# ChatGPT integration

Checked: 2026-09-27.

## Target

~~~text
ChatGPT Web
   -> custom MCP app
   -> HTTPS /mcp endpoint
   -> Gateway
   -> Broker
   -> VPS
~~~

The server architecture is MCP-client independent. ChatGPT product availability is a separate completion gate.

## Current OpenAI product constraint

OpenAI currently documents full MCP support, including write/modify actions, for ChatGPT Business, Enterprise and Edu on ChatGPT web.

Official source:
https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

The same official page states that Pro users can connect MCPs with read/fetch permissions, but Full MCP is not currently available to Pro.

Do not infer write support for another plan/surface merely because the MCP server works with MCP Inspector or another client.

ChatGPT connects to remote MCP servers. A local/private-network MCP cannot be attached directly unless a supported secure tunnel/remote route is used.

## Current documented connection flow

For an eligible workspace/surface:

1. Enable Developer Mode in the workspace/user settings described by the current OpenAI documentation.
2. Go to Apps and create a custom app.
3. Provide the remote HTTPS MCP endpoint.
4. Select/configure authentication.
5. Scan tools.
6. Create the app.
7. In a new web chat, select or mention the app.
8. Call a harmless tool such as system.info.
9. Verify the matching authenticated call in Broker audit.

The OpenAI UI and permissions are version-sensitive. Recheck the official page at release/setup time.

## Our package completion rule

scripts/connect-chatgpt.sh shows the tutorial and then waits for an audited system.info call from the configured subject.

Showing the tutorial is not success.

~~~text
tutorial shown
 -> ChatGPT app connected
 -> system.info invoked from ChatGPT
 -> expected subject authenticated
 -> Broker policy allows
 -> execution succeeds
 -> audit record observed
 -> installation complete
~~~

If the target ChatGPT plan/workspace cannot invoke the required tool, the script times out and installation remains incomplete.

## Authentication

For OAuth/OIDC deployments validate issuer, audience/resource, signature, expiration, scopes, and subject.

If the OpenAI integration requires refresh-token support, follow the current OpenAI guidance for offline_access/refresh-token issuance.

Static bearer auth exists for laboratory/private validation. Do not assume it is the final authentication mechanism accepted by every ChatGPT app surface.

## Transport

Use MCP Streamable HTTP over remote HTTPS at /mcp.

Do not invent a custom WebSocket protocol.

## Security

ChatGPT host confirmations and app permissions are additional UX/safety controls. They never replace Broker authorization.

Tool definitions may be frozen/reviewed by the ChatGPT workspace. If the MCP tool schema changes, refresh/review the app actions using the current OpenAI workspace flow.
