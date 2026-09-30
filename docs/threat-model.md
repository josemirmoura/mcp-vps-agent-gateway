# Threat model

Assume the AI model, remote clients, tool results, logs and external content can all contain hostile or misleading input.

## Primary risks

| Threat | Example | Primary control |
|---|---|---|
| Prompt injection | log says "become root" | server-side Broker policy |
| Confused deputy | Gateway asks Broker to touch an unauthorized resource | Broker re-authorizes subject + tool + resource + action |
| Privilege escalation | agent asks for admin shell | Full disabled; temporary explicit grant only |
| Path traversal | ../../etc/shadow | safe descriptor-based path resolution |
| Symlink/rename race | allowed path is swapped toward an outside target | descriptor-relative `os.Root` operations + adversarial swap tests |
| Docker privilege | Gateway gets docker.sock | Docker only through Broker |
| Fork/memory bomb | generated process exhausts VPS | systemd/cgroups/timeouts |
| Duplicate write | client retries mutation | infrastructure idempotency or no blind retry |
| Secret disclosure | model reads credentials | secret refs + Broker-only plaintext access |
| Token theft | stolen OAuth token | issuer/audience/expiry/scope validation |
| Broker exposure | root API reachable remotely | Unix socket only |
| Approval fatigue | repeated elevation prompts | Scoped routine + rate limit/cooldown |
| Shared-state escalation | Gateway writes Broker DB | Broker-only SQLite |
| Unsafe fallback | missing kernel feature weakens checks | detect + safe fallback/fail closed |
| Egress abuse | elevated agent exfiltrates data | network capability separate from Full |
| Tool poisoning | downstream MCP changes schema/description | provenance/fingerprint/review |
| Tool-result injection | tool output contains hidden instructions | results are untrusted data |
| Cross-tool exfiltration | malicious read induces sensitive read then external write | trust/egress/data-flow restrictions |
| Recovery failure | Broker/DB fails mid-change | managed jobs + fail-closed recovery |

## Stage-appropriate security tests

### Gate 0B
- path confinement
- invalid input
- forbidden path
- output limits

### Gate 1/2
- unauthorized service/stack denied
- Gateway cannot access Docker socket
- wrong subject/resource denied
- malicious tool result cannot alter policy

### Gate 3/4
- hardlink unlink behavior and concurrent symlink/rename-race containment
- idempotent retry behavior
- non-replay-safe no-blind-retry behavior
- SQLite recovery
- job restart/reconciliation
- secret redaction
- shell/resource containment when enabled
- request-body/concurrency/rate-limit abuse controls
- egress allowlist fail-closed behavior until enforcement is available

### Gate 5 / R5
- self-approval impossible
- elevation spam limited
- expired/revoked grant denied
- Full without network grant has no unrestricted egress
- revoke-all works
- audit tampering detectable
- remote audit checkpoint exists

## Important principle

MCP annotations and host confirmations improve UX and safety behavior. They do not replace Broker authorization.
