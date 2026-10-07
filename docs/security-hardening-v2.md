# Security hardening

These rules define the security posture beyond the base architecture.

## 1. Authorization belongs to the Broker

The Gateway is not trusted to authorize privileged work.

Every privileged operation binds:

~~~text
subject
+ canonical tool
+ resource
+ action
+ policy
+ grant when required
~~~

No ambient authority is inherited from a previous tool call or conversation step.

## 2. Permission expansion is requested in-band and approved out-of-band

Routine work uses Scoped.

The MCP client may request a new filesystem root inside the preconfigured physical ceiling. The request itself grants nothing. The Broker stores a typed pending approval containing subject, root, access profile and optional lifetime. Only the separate operator approval boundary may activate it.

Active dynamic root delegations are Broker-owned state. They do not edit `policy.yaml`, do not widen the physical ceiling and do not enable actions absent from the static policy. Revocation may be initiated in-band because it only reduces the requesting subject's dynamic authority. Temporary delegation expiry is enforced server-side, and shell jobs depending on a temporary root cannot outlive that delegation.

Generic MCP tool annotations and host confirmation policy remain UX signals rather than the Broker security boundary.

For clients that advertise standard MCP elicitation, root and protected-file delegation use the client's native confirmation surface. The Gateway returns a multi-round-trip elicitation request with opaque request state; the model receives neither a self-approval tool nor a usable approval token. After accept/decline/cancel, the Broker independently binds the decision to the pending request, authenticated subject, requested resource, access profile and approval expiry. Clients without elicitation leave the request pending for the separate operator fallback.

### Administrative elevation



If elevation is enabled later:

- MCP can request elevation
- MCP cannot approve elevation
- operator approval uses a separate authenticated web flow or equivalent trusted surface
- use step-up auth and preferably MFA/passkey
- bind approval to a one-time nonce/request
- apply rate limits, cooldowns and duplicate coalescing
- approval URLs expire
- GET parameters never encode approval decisions

Full remains disabled until R5 criteria are met.

## 3. Full is capability-based

Full is a convenience label for explicit capabilities.

Examples:

~~~text
shell.admin
filesystem.read:any
filesystem.write:any
docker.admin
systemd.admin
~~~

network.unrestricted is separate and must never be implied.

## 4. Replay safety

Every state-changing tool is classified:

### replay-safe
- infrastructure-managed idempotency identity required
- same identity + same normalized request returns the stored result
- same identity + different request conflicts

### non-replay-safe
- never blindly retried
- use action IDs
- use resource locks/state
- require risk-appropriate confirmation/authorization

The LLM is not responsible for inventing a stable idempotency key.

## 5. SQLite ownership

Only the Broker opens the privileged SQLite database.

Gateway and operator approval routes use narrow Broker APIs.

Transactions are short. External commands never run inside open database transactions.

## 6. Filesystem safety

Never authorize paths using string prefixes.

Preferred:
- openat2 with restrictive resolution flags

Fallback:
- secure directory-FD walk using openat/fstatat/O_NOFOLLOW semantics

If neither is safely available, privileged writes fail closed.

## 7. Kernel capability detection

Broker health reports relevant capabilities.

If Landlock is unavailable:
- retain systemd/cgroup sandboxing
- report degraded defense-in-depth
- do not disable other controls

If openat2 is unavailable:
- use the documented safe fallback or fail closed

## 8. Tool and result trust

Tool output, logs, web content and downstream MCP results are untrusted data.

They never:

- mutate policy
- create/extend grants
- change trust tier
- register a new MCP server
- expose secrets
- bypass network policy

Downstream MCP servers are configured out-of-band and material tool changes are fingerprinted/reviewed.

## 9. Secret handling

Default VPS mechanisms:

- root-owned files outside Git
- systemd credentials

Gateway cannot read plaintext secret storage.

No generic secret.read_plaintext tool.

Redact likely credentials from logs/output where practical.

## 10. Audit grows with privilege

### R1/R2
Structured local audit is sufficient.

### R3/R4
Add durable ordering/integrity checks and tested retention.

### R5 / Full
Require:
- monotonically sequenced records
- hash-chain integrity
- checkpoint/signature or authenticated chain head
- remote anchoring/forwarding
- detection of sequence gaps or chain breaks

Do not claim local root logs are tamper-proof.

## 11. Mandatory negative tests

Before the relevant maturity stage passes, test:

- path traversal denied
- symlink escape denied
- unauthorized service/stack denied
- wrong subject denied
- stale/revoked grant denied
- replay-safe retry executes once
- non-replay-safe action not blindly retried
- malicious tool result cannot grant capability
- arbitrary downstream MCP URL rejected
- Gateway cannot open Docker socket
- Gateway cannot open privileged SQLite
- missing kernel features never cause insecure silent fallback
- stale resource-lock owner cannot release or commit after a newer fencing token exists
- crash after external effect but before idempotency completion does not cause blind duplicate execution

Before R5 also test:

- elevation spam rate-limited
- self-approval impossible
- Full without network capability lacks unrestricted egress
- audit tampering detectable
- revoke-all works
