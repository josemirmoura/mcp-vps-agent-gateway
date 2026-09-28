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

## Public OAuth preflight

Before opening the ChatGPT app-creation flow, run:

~~~bash
bash scripts/verify-public.sh
~~~

The preflight checks the actual public boundary rather than only local containers:

1. HTTPS health endpoint is reachable.
2. RFC 9728 protected-resource metadata identifies the expected resource and Authorization Server.
3. Required scopes are present in protected-resource metadata.
4. OIDC discovery exposes the exact issuer plus HTTPS authorization/token endpoints.
5. PKCE S256 is advertised.
6. Token endpoint authentication methods are published.
7. CIMD or DCR is available, unless a predefined OAuth client was explicitly configured.
8. An unauthenticated /mcp request returns HTTP 401 with a WWW-Authenticate challenge pointing to the protected-resource metadata.
9. If VPS_AGENT_TEST_ACCESS_TOKEN is supplied, a real authenticated system.info call must pass.

The verifier warns when offline_access is not advertised because long-lived ChatGPT connectivity may then require reauthentication.

This preflight cannot prove the interactive browser authorization itself. That remains the final ChatGPT-side gate.

## Current documented connection flow

For an eligible workspace/surface:

1. Enable Developer Mode in the workspace/user settings described by the current OpenAI documentation.
2. Go to Apps and create a custom app.
3. Provide the remote HTTPS MCP endpoint.
4. Select OAuth authentication and complete the provider authorization flow.
5. Scan tools.
6. Create the app.
7. In a new web chat, select or mention the app.
8. Call a harmless tool such as system.info.
9. Verify the matching authenticated call in Broker audit.

The OpenAI UI and permissions are version-sensitive. Recheck the official page at release/setup time.

## Our package completion rule

scripts/connect-chatgpt.sh runs the public OAuth preflight, shows the connection tutorial and then waits for an audited system.info call from the configured subject.

Showing the tutorial is not success.

~~~text
public OAuth preflight passes
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

For the public ChatGPT route, configure VPS_AGENT_AUTH_MODE=oidc. The Gateway publishes RFC 9728 protected-resource metadata and validates issuer, audience/resource, signature, expiration, scopes and subject. The Broker then independently checks the expected subject and policy.

The Gateway uses the official MCP Go SDK bearer middleware. Failed unauthenticated requests include a WWW-Authenticate challenge containing the protected-resource metadata URL, which is independently checked by scripts/verify-public.sh.

For ChatGPT-compatible Authorization Server requirements, including PKCE S256 and CIMD/DCR/predefined clients, see docs/authentication.md and the current OpenAI authentication guide.

If the OpenAI integration requires refresh-token support, follow the current OpenAI guidance for offline_access/refresh-token issuance.

Static bearer auth exists only for local/laboratory acceptance. The public write-capable ChatGPT path uses OAuth/OIDC. Current OpenAI guidance states that ChatGPT does not present custom API keys to MCP servers.

## Transport

Use MCP Streamable HTTP over remote HTTPS at /mcp.

Do not invent a custom WebSocket protocol.

## Security

ChatGPT host confirmations and app permissions are additional UX/safety controls. They never replace Broker authorization.

Tool definitions may be frozen/reviewed by the ChatGPT workspace. If the MCP tool schema changes, refresh/review the app actions using the current OpenAI workspace flow.
