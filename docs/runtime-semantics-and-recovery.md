# Runtime semantics and recovery

This document defines operational behavior that must be explicit before production.

## 1. Grant expiry during long-running jobs

An elevated job derives an immutable job grant at start.

Default:

~~~text
job_deadline <= grant_expiry
~~~

When the grant expires:

1. stop accepting new privileged mutations from that job
2. send graceful termination to the managed process/cgroup
3. after a bounded grace period, force termination
4. persist the final state
5. emit audit

Routine Scoped jobs do not need an elevated grant.

If a specific administrative job must outlive the interactive grant, require a separate action-specific completion grant with its own bounded deadline.

## 2. SQLite concurrency

SQLite is intentionally single-writer and acceptable initially.

Rules:

- only Broker opens the DB
- transactions are short
- never keep a transaction open while shell/deploy/migration/Docker work runs
- persist transition, commit, perform external work, persist next transition
- long-lived exclusion uses logical resource locks with deadlines, not long DB transactions
- enable WAL
- use busy timeout/backoff for short contention

Migrate only if measured contention becomes material.

### Resource-lock fencing

Locks with TTL/deadline must include a monotonically increasing fencing token.

A stale job can never release a resource or commit a protected step using an older token. Release is compare-and-delete on resource + owner/action_id + fencing token.

This closes the classic ABA race:

```text
A acquires token 1
A stalls and expires
B acquires token 2
A wakes up late
A must NOT release/overwrite B
```

### Idempotency crash window

For replay-protected effects, create an operation journal row before the external side effect:

```text
PENDING -> effect -> DONE
```

If the Broker crashes after the effect but before DONE, the next attempt does not re-execute automatically. It enters action-specific reconciliation or returns an indeterminate/reconcile-required result.

## 3. Idempotency is infrastructure metadata

Do not depend on the LLM to invent a stable idempotency key.

Preferred behavior:

- Gateway/integration layer generates or derives retry identity
- bind it to subject + canonical tool + normalized request hash + invocation identity
- use a client/transport invocation identity when available
- if no stable retry identity exists, do not auto-retry state-changing operations

Identical arguments do not prove a retry; the user may legitimately request the same action twice.

## 4. Secret management

The MVP has no secret-reading tools.

Default VPS mechanisms:

- root-owned files outside the repository
- systemd credentials

Broker resolves secret_ref internally and injects the value only into the target process.

Rules:

- Gateway cannot read plaintext secret storage
- model-facing tools cannot enumerate secret values
- secrets never enter audit payloads
- redact known secrets from stdout/stderr/logs where practical

External secret managers are optional adapters when a real need appears.

## 5. Approval friction

Routine unattended work uses pre-authorized Scoped policy.

Out-of-band approval is reserved for exceptional elevation.

This avoids approval fatigue while keeping administrative access deliberate.

## 6. Recovery

### Broker restarts during a job

- jobs run in managed systemd units/cgroups where appropriate
- every job has a deadline
- Broker restart reconciles persisted state with live units/processes
- orphan jobs are adopted for monitoring or terminated by policy

### SQLite corruption

Fail closed:

1. stop new Gateway/Broker writes
2. invalidate pending approvals
3. treat elevated grants as revoked
4. preserve damaged DB
5. restore latest verified backup
6. reconcile active job units
7. re-enable only after integrity checks

### Broker audit write failure (AUD-03)

Privileged or potentially state-changing requests follow a fail-closed audit
protocol. The Broker commits an append-only hash-chain `intent` event (bound to
the request invocation ID when present) **before** invoking an external effect;
it records a separate `allow` or `deny` outcome afterward. New or unknown
tool names are **mutating by default** unless explicitly reviewed as read-only.
Potentially mutating requests are serialized through this boundary.

- If durable intent cannot be written, the Broker rejects the operation
  **before** any external mutation.
- If the effect happened but the outcome cannot be written, the response is
  `reconcile_required`, **not** a success or a claim that the effect was
  rolled back. The intent remains in the audit chain.
- On any audit append failure, the Broker latches a degraded state and refuses
  subsequent state-changing requests, even if SQLite immediately recovers.
  A new Broker process is required **after deliberate operator reconciliation**.
  An operator must not retry unsafe actions automatically or clear the database
  to bypass the fence.
- The operator should independently check the affected resource's actual
  state, match audit `intent` to its invocation ID and operation journal,
  inspect hash-chain integrity, repair storage safely, and decide whether a
  restart is warranted. For operations with idempotency keys, replays use the
  stored operation status; without stable keys, do not assume retry safety.
- Hash-chained intent and result are distinct database commits; they do not
  make an external shell command, Docker call or filesystem write an atomic
  transaction. A crash may leave a valid unmatched `intent` that requires
  explicit inspection.

Audit events contain metadata, never request-body secrets. This gate does not
replace transactional approval/grant resolution, OAuth checks or revocation.

### Erroneously issued grant

Provide an operator command outside the AI surface:

~~~text
vps-agent revoke-all
~~~

It must:

- revoke all elevated grants
- block new elevated starts
- terminate or quarantine affected active jobs according to emergency policy
- emit an audit record/checkpoint appropriate to the maturity stage

Rotate credentials separately if credential compromise is suspected.

### Gateway route compromise

Stopping/removing the MCP route must not stop application workloads.

## 7. Backups

At the stage where these assets exist, back up:

- policy configuration
- SQLite state
- audit checkpoints
- deployment unit files

Do not include plaintext secrets unless intentionally encrypted.

## 8. Readiness

### Scoped production
Require:

- Gate 0A/0B and the typed privileged path passed;
- Scoped real-stack behavior proven for the documented use case;
- recovery tested for enabled state/jobs;
- secret path tested if secrets are used;
- Gateway/Broker removal leaves workloads running;
- release compatibility/support boundaries documented;
- long-running reliability evidence appropriate to the production claim.

The release candidate satisfies the implementation/E2E gates but does not yet claim the long-running R4 reliability history.

### Full production
Additionally require:

- grant/job expiry semantics tested
- revoke-all tested
- approval flow tested
- remote audit checkpointing tested
- network elevation separation tested
