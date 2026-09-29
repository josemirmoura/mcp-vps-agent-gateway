# Transport and tool aggregation

## Transport

Use MCP Streamable HTTP at a stable HTTPS endpoint, typically /mcp.

Do not design a custom WebSocket protocol and do not use legacy HTTP+SSE as the primary architecture.

MCP 2026-07-28 has a stateless protocol core. Let the official SDK handle protocol negotiation and compatibility with older revisions.

Application jobs, grants and audit state never depend on a transport connection remaining open.

## Disconnects and retries

- replay-safe writes use infrastructure-managed idempotency
- non-replay-safe writes are never blindly retried
- long work returns a durable job handle
- clients can reconnect and query job.status/job.tail
- grant validity is checked server-side
- transport identity is not authorization identity

## Jobs and MCP Tasks

The Broker's internal job state machine is canonical:

~~~text
job.start
job.status
job.tail
job.cancel
~~~

MCP 2026-07-28 defines Tasks as an extension. If the target client and SDK support that extension reliably, add an adapter from internal jobs to Tasks.

Do not make Broker execution semantics depend on one client extension.

## Tool names

First-party tools use stable domain names:

~~~text
system.info
system.health
file.read
file.write
docker.logs
docker.action
service.status
service.action
job.status
permissions.list_root_access
permissions.request_root_access
permissions.revoke_root_access
permissions.request_elevation
~~~

## Downstream aggregation

Aggregation is not a first-version goal.

If added later, expose imported tools as:

~~~text
<upstream>.<tool>
~~~

Requirements:

- preserve upstream identity and original name
- deterministic mapping across restarts
- reject collisions explicitly
- never first-wins/last-wins silently
- policy/audit use canonical names
- normalization collisions require explicit aliases

## Reverse proxy

When public ingress is needed:

- terminate TLS at the existing reverse proxy or Gateway
- expose only required routes
- preserve MCP headers
- configure appropriate request/stream timeouts
- avoid buffering that breaks protocol behavior
- never expose the Broker

## Observability

Track:

- discovery/negotiation failures
- tool-call latency and errors
- auth failures
- policy denials
- transport errors
- duplicate/retry decisions
- active/durable jobs
- downstream availability if aggregation is enabled

Never log bearer tokens or known secrets.
