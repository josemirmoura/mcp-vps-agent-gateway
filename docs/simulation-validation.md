# Simulation validation

Date: 2026-09-26
Seed: 26092026
Scope: executable architectural model, not production implementation.

## Why simulate

The repository is still docs-first. Simulation cannot prove Linux syscalls, systemd, OAuth, Docker behavior or ChatGPT product compatibility, but it can falsify logical invariants before the Go implementation exists.

The model deliberately includes vulnerable baselines so the suite proves it can detect insecure design rather than only asserting success.

## Recorded results

Targeted adversarial scenarios covered:

- path traversal
- path-prefix confusion
- confused deputy
- agent self-approval
- implicit unrestricted network in Full
- foreign-subject grant reuse
- expired-grant reuse
- duplicate/replay behavior
- blind retry of non-replay-safe actions
- tool-definition poisoning
- tool-result policy injection
- audit tampering
- grant/job expiry
- resource-lock ABA race
- crash window around idempotency

Random adversarial run:

~~~text
seed: 26092026
operations: 100000
logical invariant violations: 0
result: PASS
~~~

Audit stress:

~~~text
events: 10000
chain valid before tamper: yes
mutated event: 4322
tamper detected: yes
~~~

Grant/job boundary:

~~~text
grant expiry: 130
requested deadline: 999
effective deadline: 130

t=101  running
t=129  running
t=130  terminated
t=131  terminated
~~~

Concurrency stress:

~~~text
random lock events: 50000
fencing invariant violations: 0
result: PASS
~~~

## Two specification gaps discovered

### Resource-lock ABA race

A naive TTL lock can fail when:

~~~text
A acquires token 1
A stalls and expires
B acquires token 2
A wakes and performs a stale release
~~~

Without fencing, A can accidentally remove B's live lock.

The architecture now requires:

~~~text
resource
owner/action_id
monotonic fencing_token
expires_at
~~~

Release and protected commits compare the current owner/token.

### Idempotency crash window

A naive sequence:

~~~text
external effect
CRASH
write idempotency result
~~~

can duplicate the effect after retry.

The architecture now requires an operation journal:

~~~text
PENDING -> external effect -> DONE
~~~

PENDING is committed before the external effect. If a crash happens after the effect but before DONE, retry becomes reconcile-required rather than blindly re-executing.

## What the simulation validates

It provides evidence for the logical consistency of:

- deny-by-default resource authorization
- subject/capability/grant binding
- network separation from Full
- replay policy
- out-of-band approval boundary
- tool-result distrust
- tool-definition change detection
- audit-chain integrity model
- grant/job deadline semantics
- fenced resource locks
- crash-aware operation journaling

## What it does not validate

Real implementation gates still include:

- actual openat2 and directory-FD behavior
- symlink races under the Linux kernel
- Landlock/systemd/cgroup containment
- Docker/systemd adapters
- SQLite durability under real power loss
- OAuth/OIDC
- MCP Go SDK behavior in our implementation
- reverse proxy behavior
- ChatGPT Plus Web write-capable integration
- real VPS workload recovery

Those belong to Gate 0A, Gate 0B, Gate 1 and later integration/security testing.

## Reproduce

~~~bash
python3 sim/architecture_sim.py
python3 sim/architecture_sim.py --fuzz 100000 --lock-fuzz 50000 --seed 26092026
~~~

The simulator uses only the Python standard library and is intentionally separate from production Go code.


## Continuous validation

The repository includes:

~~~text
.github/workflows/architecture-simulation.yml
~~~

Changes to the simulator or core architecture/security/runtime documents trigger the adversarial simulation in GitHub Actions. This keeps the architectural model executable as the design evolves.
