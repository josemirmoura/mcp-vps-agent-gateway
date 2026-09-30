# Jalur implementasi MVP-first

Full architecture adalah north star, bukan milestone pertama.

## Gate -1 — Adopt, adapt, atau build

Evaluasi existing product dan open-source MCP server terlebih dahulu. Adopt jika sudah memenuhi, adapt bila dekat, build hanya bila kombinasi yang dibutuhkan hilang.

Target pembeda:

~~~text
ChatGPT Web
+ self-controlled VPS
+ server-side policy
+ Scoped autonomy
+ optional temporary elevation
+ Broker under operator control
~~~

## Gate 0A — Buktikan ChatGPT product surface

**Selesai 2026-09-28 untuk integrated OAuth + ChatGPT Web.**

Actual target client harus divalidasi sebelum privileged path. Jangan berasumsi private write-capable MCP dapat dipasang pada semua plan.

Rute:

1. workspace/plan dengan private full MCP write;
2. eligible app/plugin dengan remote write;
3. MCP Inspector saat distribusi belum selesai.

Catat:

~~~text
target_surface:
target_plan:
integration_route:
read_available:
write_available:
private_or_published:
tested_date:
~~~

Jika future account tidak menawarkan custom MCP, berhenti di boundary tersebut atau ubah distribusi saja. Jangan melemahkan server.

## Gate 0B — Safe MCP POC

Unprivileged Go + official SDK. Hanya:

~~~text
system.info
file.read_test
file.write_test
~~~

Filesystem hanya `/tmp/vps-agent-poc/`. Tanpa root, Docker, systemd write, SQLite, external secrets, Full, approval, generic shell.

Sukses: Inspector pass, tool discovery, read/write sesuai client, forbidden path fail, error jelas, retry/reconnect aman.

## Gate 1 — One typed privileged action

Minimal Broker via Unix socket. Tambahkan `service.status`/`service.restart` untuk unit disposable/non-critical. Generic admin shell, Full, approval UI, broad Docker admin, dan remote audit tetap di luar.

## Gate 2 — Scoped real-stack pilot

Explicit policy untuk satu real stack. Tambah capability berdasarkan kebutuhan: file.read, service status/restart, docker logs/restart, job.status. Gunakan beberapa hari dan ukur frekuensi, success/failure, operator correction, false denial, missing capability, recovery. Lolos saat Scoped berguna tanpa routine elevation.

## Gate 3 — Durability

Saat perlu: Broker SQLite, durable jobs, idempotency identity, logical locks, systemd credentials/secret refs, stronger structured audit.

## Gate 4 — Broader writes

Berdasarkan kebutuhan: `file.write`/`file.patch`, validated config updates, additional typed Docker/systemd actions, sandboxed `shell.exec` di Scoped roots. Generic shell bukan prasyarat.

## Gate 5 — Temporary elevation

Hanya jika Scoped tidak cukup: elevation requests, out-of-band human approval, temporary leases, revoke-all, remote audit anchoring, Full flag. Baru kemudian pertimbangkan `shell.exec_admin`.

## Rule

Setiap komponen baru membutuhkan evidence dari gate sebelumnya.

> Dapatkah actual target client menyelesaikan valuable work secara aman dan andal pada real server?
