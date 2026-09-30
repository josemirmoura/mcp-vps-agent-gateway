# Status productization — v0.1.0-rc.3

Dokumen ini melacak checklist public productization terhadap branch release-candidate saat ini. Ini evidence/status, bukan pengganti canonical architecture atau security docs.

## Selesai

- [x] implementation/status diselaraskan dengan real ChatGPT Web E2E evidence;
- [x] README product-oriented;
- [x] repository sanitization guard diperluas;
- [x] canonical product name/tagline/version;
- [x] SemVer VERSION + CHANGELOG;
- [x] guided transparent terminal install;
- [x] UX Project / Custom / Whole Host;
- [x] effective authority summary sebelum runtime;
- [x] Whole Host terpisah dari Full;
- [x] unrestricted network terpisah dari Full;
- [x] integrated OAuth/OIDC;
- [x] audit dan observability terpisah;
- [x] status/health/log/audit/diagnostic bundle;
- [x] secret redaction;
- [x] stable-by-default update;
- [x] `scripts/version.sh --check`;
- [x] backup/migration/rollback;
- [x] safe remove/purge;
- [x] productized landing;
- [x] privacy/telemetry;
- [x] compatibility/support matrix;
- [x] issue/PR templates;
- [x] tag ↔ VERSION validation;
- [x] checksum;
- [x] container SBOM/provenance;
- [x] OAuth regression coverage;
- [x] revoked/inactive token rejection;
- [x] OAuth/DCR rate limiting;
- [x] docs link checker;
- [x] guided installer lifecycle acceptance;
- [x] Full/R5 tidak diklaim production.

## Validation gates

PR, reference CI, integrated auth, Docker package, lifecycle, ephemeral Scoped, ephemeral VPS, simultaneous independent instances, dan final cross-review semuanya green.

## Sengaja belum selesai

- [ ] stable `v0.1.0`;
- [ ] final owner clean-install acceptance;
- [ ] long-running R4 evidence;
- [ ] Full/R5 production maturity;
- [ ] project-managed cryptographic signing key.

## Metadata GitHub

README, Pages, release policy, badges/links, dan repository content ditangani di branch ini. Description/topics/homepage adalah GitHub metadata dan tidak boleh dianggap selesai tanpa mutation evidence.
