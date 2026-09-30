# Productization status — v0.1.0-rc.3

この文書は current release-candidate branch に対する public productization checklist の evidence/status です。Canonical architecture/security docs の代替ではありません。

## Completed

- [x] implementation/status と real ChatGPT Web E2E evidence を整合;
- [x] product entry point 中心の README;
- [x] repository sanitization guard 拡張;
- [x] canonical product name/tagline/version;
- [x] SemVer VERSION + CHANGELOG;
- [x] transparent guided terminal installation;
- [x] Project / Custom / Whole Host UX;
- [x] runtime 前の effective authority summary;
- [x] Whole Host と Full を分離;
- [x] unrestricted network と Full を分離;
- [x] integrated OAuth/OIDC;
- [x] audit と operational observability を分離;
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
- [x] release tag ↔ VERSION validation;
- [x] checksum;
- [x] container SBOM/provenance;
- [x] OAuth regression coverage;
- [x] revoked/inactive token rejection;
- [x] OAuth/DCR rate limiting;
- [x] docs link checker;
- [x] guided installer lifecycle acceptance;
- [x] Full/R5 production claim exclusion.

## Validation gates

PR、reference CI、integrated auth、Docker package、lifecycle、ephemeral Scoped、ephemeral VPS、simultaneous instances、final cross-review は green。

## Deliberately incomplete

- [ ] stable `v0.1.0`;
- [ ] owner final clean-install acceptance;
- [ ] long-running R4 evidence;
- [ ] Full/R5 production maturity;
- [ ] project-managed release signing key。

## GitHub storefront metadata

README、Pages、release policy、badges/links、repository content はこの branch で扱います。Description/topics/homepage は GitHub metadata のため、mutation evidence なしに complete としません。
