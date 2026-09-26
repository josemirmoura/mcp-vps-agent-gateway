# Transport and tool aggregation

This document defines the network transport and tool-naming rules for production deployments.

## 1. Transport

Use **MCP Streamable HTTP** at a stable HTTPS endpoint, typically `/mcp`.

Do not design a new WebSocket protocol and do not use legacy SSE as the primary architecture.

Current MCP SDKs and OpenAI plugin guidance use Streamable HTTP. Older protocol revisions may internally use session-aware streaming behavior, while the 2026-07-28 protocol core is designed to be stateless and self-describing.

### Compatibility strategy

- Prefer the current official MCP SDK instead of implementing the wire protocol manually.
- Support the protocol revisions provided by that SDK.
- For modern 2026-07-28 requests, do not invent sticky-session requirements.
- If compatibility with older 2025-era clients is enabled and the SDK uses session IDs for them, follow the SDK's session handling rather than adding a custom session layer.
- Application jobs, leases and authorization state must never depend on a transport connection remaining open.

Transport state and business state are separate:

```text
HTTP request/stream may disappear
        |
        v
job / lease / audit state remains server-side
```

## 2. Disconnects and retries

Network loss must not corrupt an operation.

Rules:

- replay-safe writes use mandatory idempotency keys
- non-replay-safe writes are never blindly retried
- long-running work returns a durable job handle
- clients can reconnect and query `job.status` / `job.tail`
- lease validity is checked server-side on every privileged operation

Do not use transport session identity as an authorization identity.

## 3. Long-running tasks

The core project uses its own durable job abstraction:

```text
job.start
job.status
job.tail
job.cancel
```

This deliberately decouples task lifetime from HTTP lifetime.

Protocol-level task extensions may be adopted later when they are mature across the target clients, without changing the execution model.

## 4. Tool-name collisions

An MCP server must not silently expose two tools with the same effective name.

Tool names should use stable domain namespaces. Dots are valid in MCP tool names and fit this project well:

```text
system.info
system.health
file.read
file.write
docker.logs
docker.action
service.status
service.action
job.status
permissions.request_elevation
```

### Aggregating downstream MCP servers

If the Gateway later acts as an MCP multiplexer/proxy for multiple downstream servers, expose imported tools under a deterministic namespace:

```text
<upstream>.<tool>
```

Examples:

```text
github.search
database.search
browser.search
```

Requirements:

- preserve the original upstream server ID and original tool name in internal metadata
- reject startup/reload if two tools resolve to the same exposed name
- do not silently let the first or last registration win
- keep names within MCP tool-name constraints
- keep a deterministic mapping across restarts
- include the canonical exposed name in audit events
- policy rules target canonical exposed names, not ambiguous display titles

### Namespace normalization

Normalization must itself be collision-safe.

For example, if `foo-bar` and `foo_bar` would both normalize to the same prefix, do not merge them silently. Require an explicit alias in configuration.

## 5. Reverse proxy

When a public endpoint is required:

- terminate TLS at the existing reverse proxy or at the Gateway
- forward only the `/mcp` route needed by the Gateway
- preserve required MCP headers
- configure timeouts compatible with Streamable HTTP
- do not buffer streaming responses in a way that breaks the protocol
- keep the privileged Broker unreachable from the network

HTTP/2 may be used by the proxy/client stack, but it is not a custom MCP requirement.

## 6. Health and observability

Track at least:

- MCP initialization/discovery failures
- tool-call latency and failure count
- auth failures
- policy denials
- transport disconnects
- duplicate idempotency attempts
- active/durable jobs
- downstream MCP availability if aggregation is enabled

Transport failures must be observable without leaking bearer tokens or tool arguments containing secrets.
