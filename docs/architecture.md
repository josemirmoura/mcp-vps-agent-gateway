# Architecture

## Goal

Provide an AI client with useful operational access to a Linux server without making the model itself a trusted security component.

## Components

### MCP Gateway

Runs as an unprivileged service account.

Responsibilities:

- MCP transport
- tool schemas
- OAuth/OIDC token validation
- request validation
- policy pre-checks
- lease validation
- idempotency handling
- normalized responses
- minimal audit metadata

It must not:

- run as root
- open the Docker socket
- decide privileged policy alone
- expose secrets unnecessarily

### Policy Engine

Authoritative server-side policy.

Dimensions can include:

- filesystem read/write roots
- network policy
- Docker stack scope
- systemd unit scope
- shell privilege
- TTL
- human approval requirement

Recommended presets:

- Controlled
- Scoped
- Full

### Approval Service

Creates temporary capability leases after human authorization.

An agent may request elevation. It must not be able to approve its own request.

Recommended controls:

- OAuth/OIDC login
- step-up authentication
- MFA or passkey
- explicit scope display
- explicit TTL
- revocation

### Execution Broker

Small privileged daemon reachable only through a Unix Domain Socket.

Responsibilities:

- re-check authorization
- safe filesystem operations
- sandboxed process execution
- Docker operations
- systemd operations
- job management
- audit events

The broker should have a deliberately small API.

### Runtime state

Suggested paths:

```text
/etc/vps-agent/      sensitive configuration
/run/vps-agent/      Unix sockets
/var/lib/vps-agent/  SQLite, jobs, audit state
```

## Trust boundaries

```text
Internet / AI Client
        |
        | untrusted requests
        v
MCP Gateway
        |
        | authenticated + validated IPC
        v
Execution Broker
        |
        | privileged OS operations
        v
Linux kernel / Docker / systemd
```

Every boundary should validate independently.

## Full access

Full access is a temporary capability, not a permanent default.

A typical lease contains:

```json
{
  "subject": "user-id",
  "mode": "full",
  "scopes": [
    "shell.admin",
    "filesystem.full",
    "docker.admin",
    "systemd.admin",
    "network.unrestricted"
  ],
  "issued_at": "...",
  "expires_at": "..."
}
```

The client only needs an opaque lease handle.

## Filesystem

Do not authorize a path by string prefix.

Use safe path resolution. On modern Linux, `openat2()` with appropriate resolve flags can help constrain path traversal, magic links and symlink escapes.

## Process sandbox

Prefer native Linux mechanisms first:

- systemd transient units
- cgroups
- `NoNewPrivileges`
- filesystem protection
- private temp directories
- process limits
- memory limits
- runtime limits
- Landlock when available

## Docker

Do not give the MCP Gateway access to `/var/run/docker.sock`.

Docker control is effectively administrative. Route it through the broker and enforce stack-level policy.

## Long-running work

Use an explicit job model:

- `job.start`
- `job.status`
- `job.tail`
- `job.cancel`

A job should remain identifiable even if the MCP client disconnects.

## Storage

SQLite is enough for the first implementation:

- approvals
- leases
- jobs
- idempotency
- audit events

Use WAL mode and include the database in backup policy.
