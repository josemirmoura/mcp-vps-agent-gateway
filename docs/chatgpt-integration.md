# ChatGPT integration

## Client independence

Keep the server architecture independent from ChatGPT. Treat ChatGPT/MCP as an adapter over the same domain and broker.

This prevents product-plan changes from forcing a server redesign.

## Connection patterns

Depending on current OpenAI product availability, a deployment may use:

### Private MCP path

Use when the account/workspace supports private MCP with the required write capabilities.

Possible shape:

```text
ChatGPT
  |
Private MCP / Plugin
  |
Secure MCP transport
  |
Gateway bound locally or privately
```

### Published plugin path

If private write-capable MCP is not available for the target plan, the same Gateway can be exposed through a reviewable HTTPS endpoint and packaged according to the current plugin requirements.

```text
ChatGPT
  |
Published plugin
  |
HTTPS + OAuth/OIDC
  |
MCP Gateway
```

## Required POC

Before integrating production services, expose only:

- `system.info`
- `file.write_test`
- `permissions.status`

`file.write_test` must write only inside a disposable test directory.

Validate:

1. read works;
2. write works;
3. forbidden write fails;
4. host confirmations behave as expected;
5. auth identity is correct.

If the client plan blocks write tools, do not weaken server security or disguise writes as reads. Change the client integration path, not the core architecture.

## Tool set

Suggested first tools:

Read-only:

```text
system.info
system.health
file.read
service.list
service.status
docker.list
docker.inspect
docker.logs
job.status
job.tail
permissions.status
audit.recent
```

Write:

```text
file.write
file.patch
shell.exec
service.action
docker.action
job.start
job.cancel
permissions.request_elevation
```

Full-only:

```text
shell.exec_admin
```

## Authentication

Validate at least:

- issuer
- audience
- expiration
- scopes
- subject
- signature

Use client-authentication mechanisms supported by the current platform when available. Do not rely only on source IP, User-Agent, a static shared header or a secret URL.
