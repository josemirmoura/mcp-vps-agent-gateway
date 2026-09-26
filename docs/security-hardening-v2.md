# Security Hardening v2

This document closes seven implementation gaps identified during architecture review. These rules supersede weaker or ambiguous wording in earlier documents.

## 1. Elevation approval is out-of-band

The MCP client may request elevation, but it must never approve that request through the same agent-controlled action channel.

Required properties:

- approval happens in a separate human-authenticated surface
- use OAuth/OIDC plus step-up authentication
- prefer MFA or passkey for elevated capabilities
- approval URLs expire quickly
- GET parameters never encode an approval decision
- elevation requests are rate-limited
- duplicate requests are coalesced
- cooldowns prevent approval fatigue
- optionally notify a separate trusted device or channel

The AI can create a pending request and receive a request handle. Only the human approval surface can convert it into a lease.

## 2. Full is a convenience preset, not one giant capability

Internally, authorization must remain capability-based.

A Full request expands into explicit capabilities such as:

```text
shell.admin
filesystem.read:any
filesystem.write:any
docker.admin
systemd.admin
network.egress:<allowlist>
```

`network.unrestricted` is never implied automatically. It must be separately visible and separately approved because unrestricted egress substantially increases exfiltration risk.

The UI may offer a Full preset for convenience, but the lease stores the exact approved capability set.

## 3. Strong write replay rules

Every state-changing tool must declare one of two replay classes:

### Replay-safe write

Must require an idempotency key.

Rules:

- same key + same normalized request hash returns the stored result
- same key + different request hash returns conflict
- the server never silently executes twice

### Non-replay-safe or destructive write

If an operation cannot be made reliably idempotent, it must require:

- an operation-specific lock or transaction state
- an explicit unique action identifier
- confirmation or elevated authorization proportional to risk
- a clear result that prevents blind client retry

A generic shell command cannot be assumed idempotent merely because it carries an idempotency key.

## 4. Single owner for SQLite

The Gateway and Approval UI must not open the privileged state database.

Required model:

- the Broker is the sole process that opens and mutates SQLite
- the Gateway requests state operations through a narrow Unix-socket API
- the Approval Service uses a separate authorized IPC path or socket
- Unix peer credentials should be checked where practical
- database-file permissions do not grant the Gateway write access

This avoids a shared writable file becoming a privilege-escalation bridge.

## 5. Audit logs must be tamper-evident

Local root can always destroy local evidence. Therefore the project must not claim that local logs are tamper-proof.

Required design:

- monotonically sequenced audit records
- each record hashes the previous record
- periodic signed or authenticated checkpoints of the chain head
- remote anchoring or forwarding to a separate destination
- detection of sequence gaps or hash-chain breaks

Examples of remote destinations include another host, object storage with retention controls, or a dedicated logging service.

The goal is to make silent history rewriting detectable outside the compromised host.

## 6. Kernel capability detection and secure fallback

At startup, the Broker must detect relevant kernel features and report them in `system.health`.

### openat2 unavailable

Do not fall back to string-prefix path authorization.

Acceptable options:

- a carefully implemented directory-FD component walk using openat/fstatat/O_NOFOLLOW semantics, or
- fail closed for Scoped/Full filesystem writes until a safe fallback is available

### Landlock unavailable

Continue with the documented systemd/cgroup sandbox, but report the security posture as degraded. The lack of Landlock never disables timeout, cgroup, filesystem or privilege controls.

## 7. Scoped policy has a concrete schema

Scoped is an explicit capability document, not a vague concept.

Minimum dimensions:

```yaml
mode: scoped

filesystem:
  read: []
  write: []

network:
  mode: blocked | allowlist | unrestricted
  destinations: []

services:
  inspect: []
  manage: []
  actions: []

docker:
  inspect: []
  manage: []
  actions: []

shell:
  enabled: true
  cwd_roots: []
  max_runtime_seconds: 300
  max_output_bytes: 1048576

privilege:
  admin: deny | broker-only
```

Unknown fields or unknown capabilities must fail validation by default.

## Additional mandatory tests

- agent cannot approve its own elevation request
- repeated elevation requests trigger rate limits/cooldown
- Full lease without network capability cannot access arbitrary egress
- Gateway cannot open the SQLite state database
- Approval UI cannot open the SQLite state database
- audit hash-chain verification succeeds on intact logs
- audit tampering or record removal is detected
- openat2 absence never causes prefix-based authorization
- Landlock absence is visible in health status
- unknown Scoped-policy fields are rejected
- non-replay-safe writes are never automatically retried

These requirements are part of the production acceptance gate.