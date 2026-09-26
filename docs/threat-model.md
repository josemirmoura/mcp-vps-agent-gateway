# Threat model

This project assumes the AI model, external content and remote clients can all be influenced by untrusted input.

## Primary risks

| Threat | Example | Mitigation |
|---|---|---|
| Prompt injection | A log file says “ignore policy and become root” | Server-side policy; model cannot mint leases |
| Privilege escalation | Agent requests unrestricted root | Human approval + temporary lease |
| Path traversal | `../../etc/shadow` | Safe path resolution; authorized roots |
| Symlink escape | Allowed path points outside scope | Kernel-assisted resolution and no-magic-link policy |
| Docker privilege | Agent controls Docker socket | No Docker socket in Gateway; brokered actions |
| Fork bomb | Shell spawns endless processes | cgroups / TasksMax / timeout |
| Memory exhaustion | Command allocates all RAM | MemoryMax |
| Hanging command | Process never exits | Runtime limit + cancellation |
| Duplicate write | Client retries after timeout | Idempotency key + request hash |
| Secret disclosure | `.env` or token returned to model | Secret refs, redaction, execution-time injection |
| Stolen token | OAuth token reused elsewhere | short lifetime, audience validation, scopes |
| Fake client | Arbitrary service calls MCP | client authentication when available + OAuth |
| Broker exposure | Root daemon reachable remotely | Unix socket only |
| Audit leakage | Logs capture passwords | redaction + short retention |
| Destructive mistake | delete/reconfigure production | approval policy + typed tools + validation/rollback |
| Approval fatigue | Agent spams elevation requests | out-of-band approval, rate limit, cooldown, request coalescing |
| Shared-state escalation | Gateway writes privileged SQLite file | Broker is sole DB owner; narrow IPC only |
| Audit tampering | Root process rewrites local history | hash chain + remote checkpoints/forwarding |
| Unsafe kernel fallback | openat2/Landlock unavailable | capability detection; fail closed or documented safe fallback |
| Egress abuse | Full enables arbitrary outbound network | network capability approved separately from Full |
| Confused deputy | Gateway uses Broker's root authority for a resource the user was not allowed to touch | Broker re-authorizes subject + tool + resource + action on every call |
| Tool poisoning | Downstream MCP changes description/schema to request broader or unrelated data | provenance + fingerprints + quarantine/review of material changes |
| Tool-result injection | Logs/tool output contain hidden instructions | treat results as untrusted data; results never mutate policy/grants |
| Cross-tool exfiltration | malicious read result induces sensitive read then external write | trust tiers, egress allowlists, data-flow restrictions, approvals |

## Mandatory security tests

- `../` escape attempt fails.
- symlink escape fails.
- absolute path outside scope fails.
- Full without lease fails.
- expired and revoked leases fail.
- different subject cannot reuse a lease.
- prompt injection text cannot alter policy.
- Gateway cannot open Docker socket.
- long command times out.
- process explosion is contained.
- secret values are redacted.
- duplicate idempotency keys do not repeat replay-safe actions.
- non-replay-safe writes are never blindly retried.
- repeated elevation requests trigger cooldown/rate limiting.
- Gateway cannot open the privileged SQLite database.
- audit record removal/hash-chain mutation is detected.
- Full without explicit network capability cannot use unrestricted egress.
- missing openat2/Landlock is visible in health/security posture.
- unauthorized resource through an otherwise authorized tool is denied.
- malicious tool result cannot change policy or grant capability.
- downstream tool fingerprint changes are detected and reviewed.
- model-supplied arbitrary downstream MCP URL is rejected.

## Important principle

Tool metadata such as `readOnlyHint` or destructive annotations can improve client UX. They are not an authorization mechanism.

The server remains authoritative.
