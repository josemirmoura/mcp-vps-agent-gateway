# Cloud node to real local Broker acceptance

This regression runs in the **Community repository CI** with disposable credentials and paths, without production VPS access. It exercises the actual `cloudnode.Runner`, real IPC Unix socket, `broker.Broker`, `securefs.Manager`, local scoped policy, and append-only SQLite audit chain.

## Positive and negative scenarios

1. A synthetic enrolled Ed25519 identity performs signed Cloud HTTP heartbeat and lease requests. The HTTP fixture checks the signature on each request.
2. A remotely queued `file.read` task accesses a disposable allowed file inside the test scope. The Broker returns a successful result; the connector posts a signed completion.
3. A second task attempts `file.read` on `/etc/shadow` from outside the local scope. The Broker denies it, the connector returns the original denial, and no Cloud authority expands the local policy.
4. The task's untrusted Cloud `requested_by` must **not** become the local Broker principal. The locally configured subject is used for both requests.
5. The Broker's local append-only audit chain must validate, recording one allow and one deny against the exact originating task IDs.

## Limits

This integration uses a fake Cloud HTTP server and a **real Broker**, not a real PostgreSQL-backed Cloud. The private Cloud CI independently tests its durable queue, signed lease, completion and tenant audit. An additional cross-repository test will prove the joined real Cloud→connector→Broker→Cloud path.

The fixture runs under the test user's own UID, not with root privileges. This is not evidence of a root-owned production Broker, Android MCP host acceptance, or authorization of a production machine.
