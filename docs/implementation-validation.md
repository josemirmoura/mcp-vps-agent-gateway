# Executable implementation validation

Checked: 2026-09-28.

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


## Docker package end-to-end acceptance

Workflow run:

https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36324349224

Result: **PASS** on a clean Ubuntu 24.04 GitHub-hosted runner.

The acceptance builds and starts the real `Dockerfile` + `compose.yaml`, then drives the public MCP surface through the Gateway and privileged Broker.

Proven in one disposable machine:

- both Broker and Gateway container health checks
- complete scoped filesystem workflow: mkdir, write, read, hash, patch, copy, move, chmod 0777, stat, list, delete and recursive delete
- `/etc/shadow` denied through filesystem policy
- `shell.exec` creates a real file inside the delegated root
- Scoped shell cannot read `/etc/shadow`
- host systemd service status/restart through the Docker Broker
- host Docker inspect/restart through the Docker Broker
- host Docker Compose validate/up/down through the Docker Broker
- disk, memory, process, listener, package, user and group diagnostics/inventory
- Broker audit-chain integrity after the workload

The Docker Broker uses host namespaces for host-native operations. The Scoped shell uses a systemd mount namespace starting from an empty read-only root and binds back only the required runtime/toolchain plus user-authorized filesystem roots.

Docker is packaging, not the authorization boundary. The privileged Broker must be treated as host-root trusted code; server-side policy remains the effective authority boundary for MCP requests.

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

## Real ChatGPT Web + integrated OAuth acceptance

On 2026-09-28 the supported public path was validated on a real target VPS after the integrated OAuth implementation was merged into the release-candidate line.

The validated chain was:

~~~text
ChatGPT Web
  -> HTTPS MCP protected resource
  -> OAuth/OIDC discovery
  -> Dynamic Client Registration + PKCE
  -> dedicated operator authentication
  -> Gateway token validation
  -> Broker subject + policy authorization
  -> system.info execution
  -> Broker audit record
~~~

The completion flow observed the authenticated `system.info` event for the expected subject and reached the same `CHATGPT WEB CONNECTION VERIFIED` / `INSTALLATION COMPLETE` criterion enforced by `scripts/connect-chatgpt.sh`. Environment-specific hostname, subject and credential values are intentionally not committed as public evidence.

This closes the product-surface Gate 0A for the supported integrated-auth path. It does not by itself establish long-running production reliability or Full/R5 maturity.

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

- long-running production reliability
- compatibility across a formally declared support matrix
- remote audit anchoring
- production use of Full/admin shell
- long-running reliability and recovery behavior under real workloads

Full/admin capabilities exist in the code path but remain disabled by the default policy and are not claimed as production-ready.

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
