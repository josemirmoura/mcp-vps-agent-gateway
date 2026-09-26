# MVP-first implementation track

The complete architecture is the north star. It is **not** the first implementation milestone.

The project must prove the client path and real operational value before building Approval Service, Full mode, remote audit anchoring or advanced sandbox layers.

## Gate 0 — client compatibility before privileged code

Build the smallest possible MCP server, running as an unprivileged user.

Expose only:

```text
system.info
file.read_test
file.write_test
```

Restrictions:

- read/write only inside a disposable directory such as `/tmp/vps-agent-poc/`
- no root
- no Docker
- no systemd writes
- no SQLite
- no Approval Service
- no Full mode
- no external secrets

Connect this POC to the **actual target client**.

For this project, ChatGPT Web is the primary client target.

### Gate 0 success

- ChatGPT discovers the tools
- read works
- write works when the client/product permits it
- forbidden path fails
- errors are understandable
- reconnect does not corrupt state

If the real client cannot execute the required write tool, stop. Change only the client integration path. Do not build the privileged architecture yet.

## Gate 1 — prove one privileged operational action

After Gate 0 passes, add the minimal privileged Broker.

Keep only:

```text
system.info
file.read_test
service.status
service.restart
```

`service.restart` must be restricted to **one disposable or non-critical test service**.

At this point the architecture has two processes:

```text
MCP Gateway (non-root)
        |
   Unix socket
        v
Broker (privileged, minimal)
```

No Approval Service yet. No Full mode. No general admin shell.

### Gate 1 success

The real client can:

1. inspect the test service
2. restart it through a typed operation
3. verify final health
4. receive a precise audit record

## Gate 2 — Scoped value test

Add a small explicit Scoped policy for one real application stack.

Recommended first capabilities:

```text
system.info
file.read
docker.logs OR service.status
service.restart OR docker.action(restart)
job.status
```

Do not add a generic root shell yet.

Run real tasks for several days and record:

- task frequency
- success/failure rate
- operator corrections
- false denials
- missing capabilities
- recovery time

Only capabilities demonstrated by real work should be added.

## Gate 3 — add complexity only when justified

Add these components only after evidence:

- SQLite: when durable jobs/idempotency/leases are needed
- OAuth/OIDC: when the remote integration requires authenticated user identity
- Approval Service: when temporary administrative elevation becomes a real use case
- Full mode: only after Scoped proves insufficient
- Landlock: defense-in-depth after the baseline sandbox works
- remote audit anchoring: before broad administrative production use
- tool aggregation: only when multiple upstream MCP servers are actually connected

## Language choice

The architecture does not require two languages.

Using TypeScript for MCP and Go for the Broker is a valid target, but the MVP should minimize runtime diversity.

If the currently supported official SDK and implementation environment make it practical, two small Go binaries or another single-language split are acceptable, provided privilege separation remains intact.

Do not merge Gateway and privileged Broker into one root process merely to reduce line count.

## What not to optimize for

Do not optimize for:

- total feature completeness
- arbitrary root automation
- supporting every MCP client
- a universal MCP multiplexer
- theoretical scalability

Optimize first for:

> Can the actual AI client safely and reliably perform a small set of valuable operations on one real server?

That is the MVP.