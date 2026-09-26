# Architecture

## Goal

Give an AI client useful operational access to a Linux VPS while keeping the VPS, not the model, in control of authorization and privilege.

> **The LLM is never the security boundary.**

## Canonical runtime

The reference design has two project-owned processes, both implemented in Go:

~~~text
ChatGPT / MCP client
        |
        | MCP Streamable HTTP
        v
+-----------------------------+
| vps-agent-gateway           |
| unprivileged                |
|                             |
| MCP + auth + schemas        |
| canonical tool names        |
| response shaping            |
| optional human web UI       |
+-------------+---------------+
              |
              | Unix Domain Socket
              v
+-------------+---------------+
| vps-agent-broker            |
| privileged, local-only      |
|                             |
| authoritative policy        |
| SQLite + locks + jobs       |
| secrets + audit             |
| files + systemd + Docker    |
| sandboxed execution         |
+-------------+---------------+
              |
              v
     Linux / systemd / Docker
~~~

The existing reverse proxy or a supported private tunnel is ingress infrastructure, not a third project service.

## Why Go for both processes

The official MCP Go SDK is Tier 1 and supports MCP specification 2026-07-28. One language reduces packaging, dependency and maintenance cost while keeping the process-level privilege boundary.

Another supported language is possible, but the reference implementation should not add runtime diversity without evidence that it helps.

## Gateway

The Gateway runs without root.

It may:

- expose MCP over Streamable HTTP
- validate OAuth/OIDC tokens when required
- validate input schemas
- normalize tool names and resources
- perform non-authoritative preflight checks
- call the Broker through a Unix socket
- later host a human approval web route

It must not:

- run as root
- access the Docker socket
- open the privileged SQLite database
- read plaintext secret storage
- become the authoritative authorization point
- approve its own elevation

## Broker

The Broker is the privileged security boundary.

Every privileged call is re-authorized against:

~~~text
subject
+ canonical tool
+ canonical resource
+ action
+ current policy
+ lease/job grant when required
~~~

The Broker owns:

- authoritative policy evaluation
- SQLite state
- idempotency
- resource locks
- durable jobs
- safe filesystem operations
- Docker/systemd operations
- process sandboxing
- secret resolution
- audit

The Gateway is treated as an untrusted deputy.

## Policy is a module, not a service

The Policy Engine lives inside the Broker. It is not a third daemon.

Policies are deny-by-default capability documents.

Presets:

- Controlled — inspection plus narrowly gated writes
- Scoped — autonomous actions inside an explicit perimeter
- Full — optional temporary capability bundle

Full is disabled by default.

## Approval is a flow, not necessarily a daemon

Routine work should use Scoped and require no human interruption.

If temporary elevation is later enabled, the agent may create a request, but approval occurs outside the MCP action channel.

A separate Approval Service process is not required by the reference design. A human route may live in the unprivileged Gateway if:

- the human uses OAuth/OIDC step-up, MFA or passkey
- approval is bound to a one-time nonce
- the Broker independently validates the signed assertion and nonce
- no MCP tool can approve the request
- replay is prevented

A separate approval service remains an option for larger deployments.

## Full

Full is not a single root bit.

It expands into explicit capabilities such as:

~~~text
shell.admin
filesystem.read:any
filesystem.write:any
docker.admin
systemd.admin
~~~

Unrestricted network egress is not implied and requires separate approval.

Full is disabled by default, temporary, revocable and unavailable until MVP and recovery gates pass.

## Transport

Use MCP Streamable HTTP at a stable HTTPS endpoint.

For MCP 2026-07-28 the protocol core is stateless. Do not build custom WebSocket or transport-session machinery. Jobs and leases use explicit application handles.

Let the official SDK handle protocol negotiation and backward compatibility.

## Filesystem

Never authorize paths with string-prefix checks.

Preferred implementation:

- openat2 with restrictive resolution flags

Fallback:

- carefully implemented directory-FD walk with openat/fstatat/O_NOFOLLOW semantics, or
- fail closed for privileged writes

Never silently fall back to string-based authorization.

## Process isolation

Baseline controls:

- systemd transient units
- cgroups
- NoNewPrivileges
- PrivateTmp
- filesystem restrictions
- MemoryMax
- TasksMax
- runtime deadline
- output limit
- cancellation

Landlock is defense-in-depth when available. Its absence is reported but does not disable baseline controls.

## Docker

The Gateway never gets /var/run/docker.sock.

Docker operations are typed Broker operations authorized against canonical stacks and actions.

## Jobs

The internal durable job model is authoritative:

~~~text
job.start
job.status
job.tail
job.cancel
~~~

Jobs do not depend on an HTTP connection remaining open.

If the target MCP client later supports the MCP Tasks extension reliably, an adapter may map internal jobs to that extension without changing Broker semantics.

## State

SQLite is the initial store and only the Broker opens it.

Transactions are short. Never keep a database transaction open while an external command, deploy, migration or Docker action runs.

State can include:

- approvals when enabled
- leases/grants
- jobs
- idempotency
- resource locks
- audit metadata

Migrate only if measured contention justifies it.

## Secrets

Prefer native Linux mechanisms first:

- root-owned files outside the repository
- systemd credentials

The Broker resolves secret references and injects values only into target processes.

The Gateway and model-facing tools do not expose plaintext secret retrieval.

## Audit by maturity

Gate 1:
- structured local audit in journald or append-oriented JSON

Scoped production:
- durable sequence/integrity checks

Before Full production:
- tamper-evident hash chain
- remote checkpoint/forwarding
- tested incident recovery

Remote audit infrastructure is not a Gate 0 prerequisite.

## Downstream tool trust

If downstream MCP servers are later aggregated:

- upstreams are allowlisted out-of-band
- tool names are deterministic and namespaced
- material schema/description changes are fingerprinted and reviewed
- tool results are untrusted data
- results never mutate policy, create leases, register servers or expose secrets

## First-version non-goals

The first implementation is not:

- a universal MCP gateway
- a multi-tenant control plane
- a distributed scheduler
- a generic root-shell service
- a Kubernetes project
- a replacement for SSH
- an attempt to support every MCP client

Its first goal is:

> Safely prove that the actual target AI client can perform a small, valuable, auditable operation on one VPS.
