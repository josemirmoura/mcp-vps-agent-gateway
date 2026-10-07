# Architecture

## Goal

Build a secure control and communication layer that allows AI clients and AI agents to discover, operate and collaborate across owner-authorized computers and services, while keeping each destination machine, not the model, in control of authorization and privilege.

The current v0.1 reference runtime proves this model on one Linux host. The north-star architecture extends the same security invariants to multiple nodes, heterogeneous operating systems and AI-to-AI delegation.

ChatGPT Web is the operator's priority interactive client, but the architecture remains vendor-neutral and compatible with other suitable MCP clients and agents.

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
| optional operator web UI       |
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

## North-star multi-node runtime

The proven single-host runtime is the foundation for the planned multi-node architecture:

~~~text
web AI clients / local orchestrators / automation
                    |
                    v
           Portico Control Plane
       identity / discovery / routing
          audit aggregation / policy context
                    |
         +----------+----------+
         |          |          |
         v          v          v
      Node A      Node B      Node C
      Linux       Windows     Linux
         |          |          |
      Broker      Broker      Broker
         |          |          |
 files/RAG/AI   apps/AI     DB/GPU/AI
         \________ AI <-> AI ________/
~~~

The control plane routes requests and exposes stable node-aware capabilities. It does not become a global root authority. Every destination node re-authorizes the authenticated subject, tool, resource, action and delegation against its own local policy before execution.

AI-to-AI communication is a first-class target capability. An authorized orchestrator or agent may discover another agent, submit bounded work and receive durable results on the same node or another node. Delegation never implies authority inheritance: an agent receives only the explicit capability/context granted for that task.

See [vision.md](vision.md) and [roadmap-multinode-control-plane.md](roadmap-multinode-control-plane.md).

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

## Product capability model

The product ships one broad tool/capability catalog. Effective authority is selected by policy, not by building different binaries.

A user may authorize:

- one project root
- several selected roots/resources
- the whole host

For multi-project deployments, the physical ceiling and logical roots are deliberately separate. The ceiling is the maximum boundary, not a read grant. The Standard profile may expose only the immediate directory names below that ceiling through a discovery-only Broker operation so the client can request the correct project without opening it. The static policy supplies the baseline roots. Additional roots can be represented as Broker-owned dynamic delegations bound to the authenticated subject, an access profile (`read`, `work`, or `compose`) and an optional expiry. Creating a pending request is not authorization. On clients that advertise MCP elicitation, the Gateway returns a multi-round-trip elicitation request and the MCP client renders its own native confirmation surface. The opaque approval state is returned only through the protocol round trip and the Broker independently validates the pending request, authenticated subject and one-time approval token before activating the delegation. The confirmation tools are not published in the model-visible tool catalog. Clients without elicitation fail closed into the separate operator fallback. Revocation takes effect from Broker state without a container restart.

Secret-bearing files form a nested boundary inside an authorized root. Protected paths such as `.env` require a second exact-path, temporary, human-approved grant. Common template files remain ordinary project content. The Broker enforces this rule for generic filesystem operations, and the shell sandbox masks protected paths so a root delegation cannot be used as an alternate plaintext-secret retrieval path.

Filesystem scope is only one dimension. systemd units, Docker resources, shell roots, network destinations and administrative actions are independently scoped.

See [product-model.md](product-model.md).

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

A separate Approval Service process is not required by the reference design. For routine dynamic root and protected-file grants, the preferred path is native MCP elicitation rendered by the connected client. The Broker still owns the security decision: approval is bound to the authenticated subject and a one-time nonce/token, replay is prevented, and no model-visible MCP tool can approve its own request. A separate operator/admin route remains the fallback for clients without elicitation and an option for stronger step-up flows such as MFA or passkeys.

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
- idempotency / operation journal
- resource locks
- audit metadata

### Resource-lock fencing

A resource lock is not just `resource -> owner`.

Use at least:

```text
resource
owner/action_id
monotonic fencing_token
expires_at
```

Every acquire after expiry increments the fencing token. A stale owner may not release or commit work against a newer token.

This prevents an ABA race where job A's lock expires, job B acquires the resource, and a late release from A accidentally deletes B's lock.

### Idempotency crash window

Do not perform an external side effect and only then create the idempotency record.

Use an operation journal:

```text
PENDING -> external effect -> DONE
```

Create and commit `PENDING` before the external effect. If the Broker crashes after the effect but before `DONE`, a retry must not blindly re-execute. It enters reconciliation and either confirms the external state or returns an indeterminate/reconcile-required result.

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

## Downstream tool and agent trust

If downstream MCP servers or AI agents are aggregated:

- upstreams/agents are allowlisted or registered through an explicit trust flow
- tool and agent names are deterministic and namespaced
- material schema/capability changes are fingerprinted and reviewed
- tool and agent results are untrusted data
- results/messages never mutate policy, create leases, register trust or expose secrets
- delegated agents receive only explicitly bounded authority
- delegation chains preserve provenance and remain auditable across nodes

## First-release scope restraint

The first implementation deliberately does not attempt to deliver the entire north-star product at once. v0.1 is not yet:

- the multi-node control plane;
- the Windows node implementation;
- the AI-to-AI discovery/delegation fabric;
- a multi-tenant SaaS control plane;
- a distributed scheduler;
- a generic root-shell service;
- a Kubernetes project;
- a replacement for SSH;
- an attempt to support every possible MCP client.

These are maturity boundaries, not a rejection of the final multi-node/multi-AI product direction. The immediate goal remains:

> Safely prove and productize a valuable, auditable AI-to-machine control path on one Linux host, then expand the same security model gate by gate.
