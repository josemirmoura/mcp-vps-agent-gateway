# Executable implementation validation

Checked: 2026-09-27.

This document records evidence from clean GitHub-hosted Ubuntu runners. These runners are disposable machines, so successful runs prove reproducible behavior in a fresh Linux environment without touching a production VPS.

## Current evidence

### File write/read through real MCP -> Gateway -> Broker -> Linux

Workflow run:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36283767952

Result:

- Ubuntu 24.04 GitHub-hosted runner
- Gateway installed as non-root `vps-agent`
- Broker installed as root with group-gated Unix socket
- Streamable HTTP MCP on localhost
- `file.write_test` wrote a real file under Broker-owned state
- `file.read_test` returned the same 56-byte content
- external host check confirmed the file and SHA-256
- audit chain valid with 2 events

Observed artifact evidence:

~~~text
file content:
MCP VPS Agent Gateway passou pela VPS efemera do GitHub.

sha256:
572890f111b50050dd1c405e724a0cd2b31dee2df32bd52c30d3db6b1853eea1
~~~

### Negative filesystem proof: /etc/shadow

Workflow run:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36283831850

The proof request attempted `file.read_test` against `/etc/shadow`.

The workflow succeeds only when the MCP tool returns a denial/error and the audit chain remains valid.

Result: **PASS**.

### Real systemd restart through MCP

Workflow run:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36283884675

Sequence:

~~~text
service.status
service.restart
service.status
~~~

Observed Linux state:

~~~text
PID before: 3124
PID after:  3173
ActiveState after: active
~~~

Audit chain:

~~~text
events: 3
valid: true
head:
d7f82e6fd0f69478544fb30fee4d034f9d69b8b0615d06a7643099d59b6ac21d
~~~

Result: **PASS**.

## CI evidence

The reference implementation CI validates, on GitHub-hosted Linux runners:

- `go vet ./...`
- `go test -race -count=1 ./...`
- native command builds
- linux/amd64 and linux/arm64 release builds
- architecture adversarial simulation
- Docker typed-operation smoke against a real Docker daemon
- sanitized Docker inspect does not expose injected environment secrets
- systemd unit syntax/hardening verification
- transient systemd job smoke
- `govulncheck ./...`

Historical clean runs are visible in the Actions history. The latest commit must be green before merge.

## Privilege-boundary proof

The ephemeral environment intentionally uses:

~~~text
/run/vps-agent              root:vps-agent 0750
/run/vps-agent/broker.sock  root:vps-agent 0660
/var/lib/vps-agent          root:vps-agent 0750
~~~

Broker policy, Broker environment secrets and SQLite are installed root-only.

The proof workflow explicitly fails if the Gateway OS account can read:

- `/etc/vps-agent/broker.env`
- `/etc/vps-agent/policy.yaml`
- `/var/lib/vps-agent/state.db`

## What this proves

It proves that the current reference implementation can be built from a clean checkout and can exercise the intended chain:

~~~text
MCP client
  -> Streamable HTTP
  -> non-root Gateway
  -> protected Unix socket
  -> privileged Broker
  -> server-side policy
  -> real Linux operation
  -> Broker audit
~~~

## What this does not prove yet

It does not yet prove:

- the final ChatGPT Web distribution path (Gate 0A)
- deployment on the user's real VPS
- long-running production reliability
- a real external OIDC provider in the target environment
- remote audit anchoring
- Full/admin shell in production

Full remains disabled and `shell.exec_admin` intentionally returns not implemented until Gate 5.

## Reproducible interactive proof

The current request is stored in:

~~~text
proof/request.json
~~~

Supported proof request types:

- `file_write_read`
- `file_read_denied`
- `service_restart`

Changing the request in a pull-request branch triggers `.github/workflows/ephemeral-proof.yml`.

The workflow uploads:

- `proof/evidence.json`
- `proof/audit-status.json`
- Gateway/Broker/service journals
- socket/directory permissions
- before/after systemd state
- file content/hash when applicable

This is the mechanism used for human-requested acceptance demonstrations.
