# Arsitektur

## Tujuan

Memberi AI client akses operasional yang berguna ke Linux VPS sambil menjaga VPS, bukan model, sebagai pengendali authorization dan privilege.

> **LLM tidak pernah menjadi security boundary.**

## Canonical runtime

Dua proses Go milik project:

~~~text
ChatGPT / MCP client
        |
        | MCP Streamable HTTP
        v
+-----------------------------+
| vps-agent-gateway           |
| unprivileged                |
| MCP + auth + schemas        |
+-------------+---------------+
              |
              | Unix Domain Socket
              v
+-------------+---------------+
| vps-agent-broker            |
| privileged, local-only      |
| authoritative policy        |
| SQLite + locks + jobs       |
| secrets + audit             |
| files + systemd + Docker    |
| sandboxed execution         |
+-------------+---------------+
              |
              v
     Linux / systemd / Docker
~~~

Reverse proxy atau private tunnel adalah ingress infrastructure, bukan service project ketiga.

## Mengapa Go

Official MCP Go SDK Tier 1 dan mendukung MCP 2026-07-28. Satu bahasa mengurangi packaging/dependency/maintenance sambil mempertahankan process-level privilege boundary.

## Gateway

Berjalan non-root. Boleh expose MCP, validasi OAuth/OIDC/schema, normalize tools/resources, non-authoritative preflight, dan call Broker via Unix socket.

Tidak boleh root, menerima Docker socket, membuka privileged SQLite, membaca plaintext secret store, menjadi authoritative authorization point, atau approve elevation sendiri.

## Broker

Privileged security boundary. Setiap privileged call diotorisasi ulang terhadap:

~~~text
subject
+ canonical tool
+ canonical resource
+ action
+ current policy
+ grant/lease/job bila diperlukan
~~~

Broker owns policy evaluation, SQLite, idempotency, locks, durable jobs, safe filesystem, Docker/systemd, sandboxing, secret resolution, audit. Gateway dianggap untrusted deputy.

## Capability model

Satu broad catalog, effective authority dari policy. Operator dapat authorize satu root, beberapa root/resources, atau whole host.

Physical ceiling dan logical roots dipisah. Ceiling adalah maximum boundary, bukan read grant. Standard dapat discovery-only immediate directory names. Dynamic delegation adalah Broker-owned, subject-bound, profile `read`/`work`/`compose`, optional expiry.

Pending request bukan authorization. MCP elicitation membuat client render native human confirmation. Broker memvalidasi request, subject, one-time token. Confirmation tools tidak model-visible. Client tanpa elicitation fail-closed ke operator fallback. Revocation aktif dari Broker state tanpa restart.

Secret-bearing files menjadi nested boundary. `.env` butuh separate temporary exact-path approval. Template tetap normal. Shell sandbox mem-mask protected paths.

Filesystem hanya satu dimensi; systemd, Docker, shell, network, admin discope terpisah. Lihat [product-model.md](product-model.md).

## Policy / approval

Policy Engine berada di Broker, bukan daemon ketiga. Policy deny-by-default. Preset Controlled, Scoped, Full; Full default off.

Routine Scoped tanpa human interruption. Temporary elevation approval terjadi di luar action channel. Native elicitation adalah jalur utama; Broker bind approval ke subject + one-time nonce dan cegah replay/autoapproval. Separate admin route tetap fallback/MFA.

## Full

Mengembang menjadi explicit capabilities:

~~~text
shell.admin
filesystem.read:any
filesystem.write:any
docker.admin
systemd.admin
~~~

Unrestricted network tidak implied. Full temporary, revocable, gated.

## Transport

MCP Streamable HTTP pada stable HTTPS endpoint. Core 2026-07-28 stateless; jangan membuat custom WebSocket/session. Jobs/leases memakai handles dan SDK menangani negotiation.

## Filesystem

Jangan authorize berdasarkan string prefix. Prefer `openat2` restrictive flags. Fallback secure directory-FD walk dengan `openat/fstatat/O_NOFOLLOW` atau fail closed. Jangan silent fallback ke string checks.

## Process isolation

systemd transient units, cgroups, NoNewPrivileges, PrivateTmp, FS restrictions, MemoryMax, TasksMax, deadline, output limit, cancellation. Landlock defense-in-depth.

## Docker

Gateway tidak menerima `/var/run/docker.sock`. Docker dilakukan oleh typed Broker operations terhadap canonical stacks/actions.

## Jobs

~~~text
job.start
job.status
job.tail
job.cancel
~~~

Internal durable job model authoritative dan tidak tergantung koneksi HTTP. MCP Tasks dapat diadaptasi kemudian.

## State

SQLite hanya Broker, transaction pendek. State dapat berisi approvals, leases, jobs, idempotency journal, locks, audit metadata.

Lock minimal: resource, owner/action_id, monotonic fencing_token, expires_at. Reacquire setelah expiry increment token; stale owner tidak boleh release/commit terhadap token lebih baru.

Idempotency journal:

~~~text
PENDING -> external effect -> DONE
~~~

PENDING di-commit sebelum effect. Crash setelah effect sebelum DONE masuk reconciliation, bukan blind retry.

## Secrets

Native Linux first: root-owned files di luar repo, systemd credentials. Broker resolve refs dan inject hanya ke target process. Gateway/model-facing tools tidak expose plaintext.

## Audit maturity

Gate 1: structured local audit.
Scoped production: durable sequence/integrity.
Before Full: tamper-evident hash chain, remote checkpoint/forwarding, tested recovery.

## Downstream trust

Upstream allowlisted out-of-band, tool names deterministic/namespaced, schema changes fingerprinted/reviewed, result dianggap untrusted data. Result tidak mengubah policy, lease, server registration, atau secret exposure.

## Non-goals awal

Bukan universal gateway, multi-tenant control plane, distributed scheduler, generic root-shell service, Kubernetes project, SSH replacement, atau support semua MCP client.

> Tujuan pertama: buktikan dengan aman bahwa target AI client dapat melakukan operasi kecil, bernilai, dan auditable pada satu VPS.
