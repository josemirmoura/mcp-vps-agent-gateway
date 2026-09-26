# Implementation runbook

This is the detailed build path. The canonical architecture is docs/architecture.md and the gate sequence is docs/mvp-first.md.

## 0. Ground rules

Before code:

1. read docs/README.md
2. complete Gate -1
3. complete Gate 0A
4. do not build beyond the current gate
5. use Go for both reference binaries
6. keep Gateway unprivileged and Broker local-only

## 1. Repository layout

Reference layout:

~~~text
cmd/
  vps-agent-gateway/
  vps-agent-broker/

internal/
  gateway/
    mcp/
    auth/
    tools/
    brokerclient/
  broker/
    api/
    authz/
    policy/
    files/
    systemd/
    docker/
    exec/
    jobs/
    state/
    secrets/
    audit/
  shared/
    protocol/
    errors/

policy/
  schema.json
  controlled.yaml
  scoped.example.yaml
  full.example.yaml

deploy/
  systemd/
  reverse-proxy/

tests/
  integration/
  security/
~~~

Runtime:

~~~text
/etc/vps-agent/       config/secrets metadata
/run/vps-agent/       broker.sock
/var/lib/vps-agent/   state/jobs/audit
~~~

## 2. Gate 0B implementation

Build only vps-agent-gateway.

Use the official MCP Go SDK and Streamable HTTP.

Tools:

~~~text
system.info
file.read_test
file.write_test
~~~

Filesystem root:

~~~text
/tmp/vps-agent-poc/
~~~

Requirements:

- non-root service account
- no Docker socket
- no SQLite
- no generic shell
- input schemas
- output size limits
- path confinement tests
- MCP Inspector test

Use an internal Executor interface so Gate 0B's local test executor can later be replaced by BrokerClient without rewriting tool handlers.

## 3. Gate 1 Broker

Add vps-agent-broker.

### IPC

Use a Unix Domain Socket.

Recommended:

~~~text
/run/vps-agent/
  owner: root
  group: vps-agent
  mode: 0750

/run/vps-agent/broker.sock
  owner: root
  group: vps-agent
  mode: 0660
~~~

Check peer credentials where practical.

### First Broker operations

~~~text
ping
system.info
service.status
service.restart
~~~

Allow only one explicitly configured non-critical unit.

Broker authorization is deny-by-default.

### Gate 1 audit

Structured journald or append-oriented JSON is sufficient.

Capture:

- request/action ID
- subject
- canonical tool
- canonical resource
- decision
- result/exit status
- duration

Do not log raw secrets.

## 4. Gate 2 Scoped pilot

Add policy parsing/validation inside Broker.

Unknown fields/capabilities fail closed.

Define one real stack.

Possible tools:

~~~text
file.read
service.status
service.restart
docker.logs
docker.action
job.status
~~~

Do not add unrestricted Docker socket access.

Docker remains Broker-only and typed.

Run for several days and record real missing capabilities.

## 5. Gate 3 durable state

Only now add SQLite if needed.

Only Broker opens:

~~~text
/var/lib/vps-agent/state.db
~~~

Enable WAL.

Suggested state:

- jobs
- idempotency
- resource locks
- audit metadata
- later approvals/grants

Rules:

- short transactions
- no external command inside DB transaction
- busy timeout/backoff
- logical locks with deadlines for long operations

### Idempotency

Gateway/integration layer creates retry identity as infrastructure metadata when a stable retry identity exists.

Bind to:

~~~text
subject + canonical tool + normalized request hash + invocation identity
~~~

Never infer retry merely because arguments are identical.

## 6. Jobs

Internal model:

~~~text
job.start
job.status
job.tail
job.cancel
~~~

A job runs in its own managed systemd unit/cgroup when appropriate.

Persist:

- owner
- policy/grant
- resource
- action
- unit/pid
- created/deadline
- status
- exit code

The MCP connection may disappear without losing job state.

## 7. Secrets

Prefer:

- root-owned files outside repo
- systemd LoadCredential / credential directory

Broker resolves secret references.

No plaintext secret retrieval tool.

Test that stdout/stderr/audit do not trivially expose known test secrets.

## 8. Gate 4 broader writes

Add only demonstrated capabilities.

### Files

Implement safe resolution and atomic writes.

Preferred openat2; secure fallback or fail closed.

For configuration changes:

~~~text
backup/temporary write
 -> native validator
 -> atomic replace
 -> reload/restart
 -> healthcheck
 -> rollback when safe
~~~

Examples:

~~~text
nginx -t
docker compose config
systemd unit validation
~~~

### Sandboxed shell.exec

Only if typed tools are insufficient.

Run under systemd transient unit/cgroup.

Enforce:

- cwd roots
- timeout
- MemoryMax
- TasksMax
- output limit
- cancellation
- filtered environment
- network policy

Landlock is optional defense-in-depth.

## 9. ChatGPT authentication/distribution

Follow docs/chatgpt-integration.md.

Do not add OAuth merely because the north-star architecture mentions it. Add the authentication required by the chosen deployment route.

For OAuth/OIDC validate issuer, audience/resource, signature, expiration, subject and scopes.

## 10. Gate 5 elevation

Only after Scoped proves insufficient.

Enable feature:

~~~yaml
features:
  full_mode_enabled: true
~~~

Add:

- elevation request
- human out-of-band approval
- one-time nonce
- temporary explicit capabilities
- expiry
- revoke-all
- separate network elevation
- enhanced audit

A human approval UI may be hosted by Gateway, but Broker validates the signed human identity/assertion and nonce independently.

Only after this is tested consider shell.exec_admin.

## 11. Lease and job semantics

Default:

~~~text
job_deadline <= grant_expiry
~~~

When an elevated grant expires:

- no new privileged mutations
- graceful termination
- force termination after bounded grace period
- persist final state
- audit

A special completion grant may be used only with explicit human approval.

## 12. Recovery

### Broker restart
- reconcile persisted jobs with systemd units/processes
- adopt monitoring or terminate according to policy

### SQLite corruption
- fail closed
- invalidate pending approvals
- treat elevated grants as revoked
- preserve damaged DB
- restore verified backup
- reconcile active jobs
- re-enable after integrity checks

### Emergency

Provide an operator command outside the AI surface:

~~~text
vps-agent revoke-all
~~~

It revokes elevated grants and blocks new elevated starts.

## 13. systemd hardening

Gateway:

- User=vps-agent
- NoNewPrivileges=yes
- ProtectSystem=strict
- ProtectHome=yes
- PrivateTmp=yes
- only required ReadWritePaths
- no Docker socket

Broker:

- root-owned
- no TCP listener
- Unix socket only
- minimal API
- fixed config/runtime paths
- journald
- restart-on-failure

Review with systemd-analyze security.

## 14. Production acceptance

### Scoped production
Require at least:

- Gate 0A/0B/1/2 passed
- negative authorization tests
- safe filesystem tests for enabled writes
- recovery test
- secrets test
- client path proven
- existing workloads survive Gateway/Broker stop

### Full production
Additionally:

- Full explicitly enabled
- approval tested
- expiry tested
- revoke-all tested
- network capability separation tested
- remote audit checkpointing tested
- admin shell containment/recovery tested

## 15. Rollback

The agent gateway must remain an optional control plane.

Stopping Gateway/Broker and removing its route must not stop existing application workloads.

That property is mandatory.
