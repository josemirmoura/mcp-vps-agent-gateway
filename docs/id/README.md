# Peta dokumentasi

Dokumentasi kanonis berada di `docs/`. Folder ini adalah terjemahan resmi Bahasa Indonesia. Jika ada perbedaan dengan file Inggris pada versi yang sama, versi Inggris menjadi referensi teknis sampai terjemahan diperbaiki.

## Operator / pengguna

1. [quick-start.md](quick-start.md) — jalur instalasi terpendek.
2. [installation-contract.md](installation-contract.md) — batas normatif instalasi.
3. [installer-flow.md](installer-flow.md) — alur instalasi lengkap.
4. [product-model.md](product-model.md) — Project, Standard, dan Whole Host.
5. [chatgpt-integration.md](chatgpt-integration.md) — koneksi ChatGPT dan gate penyelesaian.
6. [authentication.md](authentication.md) — OAuth/OIDC terintegrasi.
7. [operations.md](operations.md) — status, log, audit, update, penghapusan.
8. [troubleshooting.md](troubleshooting.md) — pemecahan masalah.
9. [faq.md](faq.md) — FAQ.
10. [compatibility.md](compatibility.md) — lingkungan tervalidasi dan belum tervalidasi.
11. [privacy.md](privacy.md)
12. [releases.md](releases.md)
13. [support.md](support.md)
14. [operator-acceptance.md](operator-acceptance.md)

## Pengembang / keamanan

- [project-status.md](project-status.md)
- [architecture.md](architecture.md)
- [policy-schema.md](policy-schema.md)
- [threat-model.md](threat-model.md)
- [security-hardening-v2.md](security-hardening-v2.md)
- [security-release.md](security-release.md)
- [runtime-semantics-and-recovery.md](runtime-semantics-and-recovery.md)
- [tool-trust-and-confused-deputy.md](tool-trust-and-confused-deputy.md)
- [transport-and-aggregation.md](transport-and-aggregation.md)
- [implementation-validation.md](implementation-validation.md)
- [simulation-validation.md](simulation-validation.md)
- [mvp-first.md](mvp-first.md)
- [implementation-runbook.md](implementation-runbook.md)
- [build-vs-adopt.md](build-vs-adopt.md)
- [multi-instance.md](multi-instance.md)
- [productization-status.md](productization-status.md)
- [references.md](references.md)

Jika ada konflik, prioritasnya: arsitektur, kontrak instalasi, status proyek terkini, schema policy, security hardening, runtime semantics, dokumen pendukung, lalu dokumen historis.

**Arsitektur dalam satu kalimat:** dua proses Go, Gateway MCP tanpa privilege dan Broker lokal privileged yang terhubung lewat Unix socket; Broker menguasai otorisasi, state, dan eksekusi privileged.
