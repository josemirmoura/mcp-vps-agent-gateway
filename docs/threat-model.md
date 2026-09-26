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
- duplicate idempotency keys do not repeat the action.

## Important principle

Tool metadata such as `readOnlyHint` or destructive annotations can improve client UX. They are not an authorization mechanism.

The server remains authoritative.
