# Connect Portico MCP to AI clients

This guide lists AI applications that provide **remote MCP connections**. Availability varies by client version and plan. **A client's advertised MCP capability does not establish Portico end-to-end interoperability.** The current Community installation acceptance specifically requires a real authenticated ChatGPT Web `system.info` call audited by the Portico Broker.

## Shared connection information

- **Suggested name:** Portico MCP
- **Suggested description:** Connects AI clients to a Linux host with controlled access and approvals.
- **Server URL:** `https://YOUR-DOMAIN/mcp` (replace with the HTTPS endpoint from the installer).
- **Transport:** MCP Streamable HTTP over HTTPS.
- **Authentication:** OAuth/OIDC, using automatic discovery where supported.
- **Credentials:** use the dedicated Portico OAuth operator username and password created during setup. Never enter VPS SSH/root credentials or internal access tokens into a chat.

In Standard mode, `/opt` is a physical ceiling, **not permission to read every directory**. Projects require separate authorization enforced by the Broker.

## ChatGPT Web

**Status:** real end-to-end acceptance in progress (#153).

1. Open **Settings > Apps** and create a **custom MCP server** (menu names may vary).
2. Set **Name** to `Portico MCP` and the optional description to `Connects AI clients to a Linux host with controlled access and approvals.`.
3. Select **Server URL** and enter `https://YOUR-DOMAIN/mcp`.
4. Select **OAuth** authentication. Keep automatically discovered advanced OAuth settings unless the client reports a specific error.
5. Finish setup, complete the Portico operator OAuth login and authorize the connection.
6. Enable Portico in a new chat and send: `Call system.info on my VPS MCP and tell me the hostname.`
7. Back in the VPS terminal running `bash scripts/connect-chatgpt.sh`, press **Enter** to check the Broker audit event.

Completing the registration or seeing an OAuth login page **does not** complete the installation. The installer requires an authenticated, audited tool call.

[Detailed ChatGPT integration guide](chatgpt-integration.md).

## Claude (Web / Desktop / Cowork)

**Status:** vendor documents remote MCP connectors; **Portico + Claude E2E has not been verified**.

1. Go to **Customize > Connectors** and **Add custom connector**.
2. Enter a name and the remote HTTPS MCP URL `https://YOUR-DOMAIN/mcp`.
3. Review discovered OAuth settings; select a compatible login and client-registration option offered by Claude.
4. Authenticate with the Portico operator account, enable the connector in the chat, and try read-only `system.info`.
5. Keep actual test evidence before claiming interoperability.

[Claude's official custom remote MCP guide](https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp).

## Cursor

**Status:** Cursor advertises Streamable HTTP remote MCP with OAuth; **Portico + Cursor E2E has not been verified**.

In Cursor MCP settings, add a remote HTTP MCP server named `portico` using `https://YOUR-DOMAIN/mcp`. Do not select a local `stdio` server. Authenticate with OAuth when prompted, confirm the discovered tools, and try `system.info` before considering it supported.

[Cursor MCP docs](https://docs.cursor.com/context/model-context-protocol).

## Visual Studio Code / GitHub Copilot

**Status:** VS Code documents HTTP MCP servers with OAuth; **Portico + VS Code E2E has not been verified**.

Run **MCP: Add Server** and select a remote HTTP server, or use a workspace `.vscode/mcp.json`:

```json
{
  "servers": {
    "portico": {
      "type": "http",
      "url": "https://YOUR-DOMAIN/mcp"
    }
  }
}
```

Follow OAuth sign-in if prompted, verify tools are discovered and test read-only `system.info`. Do not hardcode OAuth credentials or secrets in the config.

[VS Code: Add and manage MCP servers](https://code.visualstudio.com/docs/agent-customization/mcp-servers) · [MCP configuration reference](https://code.visualstudio.com/docs/agents/reference/mcp-configuration).

## Local MCP-capable agent hosts and other AI providers

Portico is **model/provider neutral** at the protocol boundary. An LLM or API endpoint cannot directly invoke MCP unless an agent host/client implements the requisite MCP transport, tool schema and authentication. A local agent runtime, including a future Hermes/Nara host, may be evaluated using the same Portico endpoint or an explicitly reviewed local bridge; **Portico + Hermes/Nara E2E has not been verified**.

For any new host: document exact version, supported transport, OAuth/client identity, discovered tools; execute authenticated `system.info` and a read-only operation; verify matching Broker audit and a denial case; record findings in [compatibility](compatibility.md). Do not bypass TLS, OAuth or local Broker policy to accommodate an agent.

## Safety and troubleshooting

Check DNS, valid HTTPS, the `/mcp` path, OAuth discovery and successful consent. Do not bypass OAuth/TLS or paste credentials into a chat to make a client connect. Directory grants and Broker audit controls remain independent of the chosen client.

*Vendor supports remote MCP* is not the same as *Portico E2E verified*. The latter requires a real authorized tool invocation with audit evidence.

Follow [Community E2E #153](https://github.com/josemirmoura/portico-community/issues/153) and [multi-client documentation issue #159](https://github.com/josemirmoura/portico-community/issues/159).
