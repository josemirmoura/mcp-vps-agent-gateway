# Status dan maturity project

## Tahap saat ini

**PRE-RELEASE PRODUCTIZATION / CHATGPT E2E GATE COMPLETE**

Implementasi Go Docker-first memiliki repeatable clean-runner validation untuk core Scoped runtime, package lifecycle, dan host operations.

Belum ada stable production release, production compatibility promise, long-running real-VPS history, atau production claim untuk Full/admin shell.

Lab evidence mencakup filesystem CRUD, authorization denial, sandboxed shell/jobs, host systemd, host Docker/Compose, diagnostics, lifecycle, dan audit. Pada 2026-09-28 integrated self-hosted OAuth path diuji terhadap ChatGPT Web pada VPS nyata melalui authenticated `system.info` yang dilihat Broker dan tercatat di audit chain. Gate 0A selesai tanpa stable-production claim.

## Maturity ladder

### R0 — Architecture only

Dokumentasi tanpa executable reference implementation.

### R1 — Product path + Gate 0B

Gate 0A integrated OAuth + ChatGPT Web selesai, MCP Inspector pass, safe read/write POC hanya pada disposable root.

### R2 — Typed privileged pilot

Gateway/Broker terpisah, satu service non-critical dapat diperiksa/restart lewat typed tools, unauthorized resource fail-closed, local audit tersedia.

### R3 — Scoped pilot

Satu real stack di explicit Scoped policy, durable behavior beberapa hari, recovery/denial diuji, routine work tidak butuh Full.

### R4 — Hardened Scoped production

Deployment auth, durable Broker state, jobs/retries/locks, secret delivery, backup/recovery, monitoring diuji. Baru setelah itu production-capable untuk documented Scoped use case.

### R5 — Elevated/Full production

Selain R4: Full feature flag, out-of-band approval, temporary grants/expiry, revoke-all, separate network elevation, tamper-evident audit + remote checkpointing, tested admin recovery.

## Full default

~~~yaml
features:
  full_mode_enabled: false
~~~

Full bukan syarat keberhasilan. Strong Scoped implementation adalah production endpoint valid.

## Evidence over popularity

Stars/forks bukan production evidence. Utamakan reproducible builds, automated tests, releases, deployment history, recovery tests, incident learnings, dependency maintenance, security review, external review.

## Milestone saat ini

Milestone adalah **owner clean-install acceptance of `v0.1.0-rc.5`**. RC5 membawa native MCP elicitation, discovery-only ceiling inventory, nested protected-secret enforcement, dan explicit public-tool safety annotations.

Blocker stable `v0.1.0`: exact RC5 tag, real ChatGPT OAuth, audited `system.info`, native approval UX, discovery tanpa content leakage, representative Scoped ops, `.env` denial/temporary exception/revocation, root revocation, lifecycle checks. Long-running reliability menyusul. Lihat [operator-acceptance.md](operator-acceptance.md) dan [implementation-validation.md](implementation-validation.md).
