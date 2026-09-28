# MVP-first implementation track

The full architecture is the north star. It is not the first milestone.

## Gate -1 — Adopt, adapt, or build

Evaluate existing products and open-source MCP servers first.

Choose deliberately:

- Adopt if an existing solution already satisfies the requirements.
- Adapt if an existing implementation is close enough.
- Build only when the required combination is missing.

The differentiating target here is:

~~~text
ChatGPT Web
+ self-controlled VPS
+ server-side policy
+ Scoped autonomy
+ optional temporary elevation
+ Broker under operator control
~~~

## Gate 0A — Prove the ChatGPT product surface

**Status: completed on 2026-09-28 for the supported integrated OAuth + ChatGPT Web path.**

This gate was required before the privileged product path could be considered validated.

Primary target: ChatGPT Web.

As of 2026-09-26, OpenAI documents full private MCP write/modify in Developer Mode for Business, Enterprise and Edu. Plugin availability varies by plan, surface, region and included app capabilities.

Do not assume that a private custom write-capable MCP can be attached directly to Plus Web.

Choose and validate one supported route:

1. Private development route — a workspace/plan that supports private full MCP write.
2. Plus Web route — an eligible plugin/app whose remote MCP write capability is available on Plus Web.
3. Protocol-only route — MCP Inspector while the distribution route is unresolved.

### Gate 0A success

Record:

~~~text
target_surface:
target_plan:
integration_route:
read_available:
write_available:
private_or_published:
tested_date:
~~~

The supported route must continue to be revalidated when ChatGPT product surfaces change. If a future target account does not expose custom MCP registration, stop at that product boundary or change only the distribution route. Do not weaken the server.

## Gate 0B — Safe MCP POC

Build the smallest server as an unprivileged user.

Reference implementation: Go + official MCP Go SDK.

Expose only:

~~~text
system.info
file.read_test
file.write_test
~~~

Restrict file access to:

~~~text
/tmp/vps-agent-poc/
~~~

Do not add root, Docker, systemd writes, SQLite, external secrets, Full, approval or generic shell.

### Gate 0B success

- MCP Inspector passes
- the chosen ChatGPT route discovers the tools when available
- read works
- write works when the client/product permits it
- forbidden paths fail
- errors are understandable
- retry/reconnect behavior is safe

## Gate 1 — One typed privileged action

Add the minimal Broker.

~~~text
vps-agent-gateway  (non-root)
        |
   Unix socket
        v
vps-agent-broker   (privileged)
~~~

Expose:

~~~text
system.info
file.read_test
service.status
service.restart
~~~

service.restart is limited to one disposable or non-critical unit.

Still exclude generic admin shell, Full, approval UI, broad Docker administration and remote audit infrastructure.

### Gate 1 success

The client can inspect, restart and re-check one test service, produce an audit record and fail safely on unauthorized units.

## Gate 2 — Scoped real-stack pilot

Define one explicit Scoped policy for one real stack.

Add only capabilities demonstrated by real need, such as:

~~~text
system.info
file.read
service.status
service.restart
docker.logs
docker.action(restart)
job.status
~~~

Use it for several days and measure:

- task frequency
- success/failure rate
- operator corrections
- false denials
- missing capabilities
- recovery time

Gate 2 passes when Scoped solves useful work without routine elevation.

## Gate 3 — Durability

Add only when needed:

- SQLite owned by Broker
- durable jobs
- infrastructure-generated idempotency identity
- logical resource locks
- systemd credentials / secret references
- stronger structured audit

## Gate 4 — Broader writes

Based on demonstrated need, add:

- file.write / file.patch
- validated configuration updates
- additional typed Docker/systemd actions
- sandboxed shell.exec inside Scoped roots

A generic shell is not a prerequisite for useful operation.

## Gate 5 — Temporary elevation

Only if Scoped is demonstrably insufficient:

- elevation requests
- out-of-band human approval
- temporary capability leases
- revoke-all
- remote audit anchoring
- Full feature flag

Only after those controls pass should shell.exec_admin be considered.

## Rule

Every new component needs evidence from the previous gate.

Optimize first for this question:

> Can the actual target client safely and reliably complete valuable work on one real server?
