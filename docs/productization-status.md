# Productization status — v0.1.0-rc.1

This document tracks the public-productization checklist against the current release-candidate branch. It is evidence/status, not a replacement for the canonical architecture or security documents.

## Completed in the productization branch

- [x] current implementation/status reconciled with the real ChatGPT Web E2E evidence;
- [x] README EN/PT-BR rewritten around the product entry point;
- [x] repository sanitization guard extended beyond supported docs;
- [x] canonical product name, tagline and visible version;
- [x] SemVer VERSION + CHANGELOG;
- [x] guided transparent terminal installation;
- [x] Project / Custom / Whole Host authority UX;
- [x] effective authority summary before runtime startup;
- [x] Whole Host kept separate from Full;
- [x] unrestricted network kept separate from Full;
- [x] integrated OAuth/OIDC retained and documented;
- [x] audit and operational observability documented separately;
- [x] status/health/log/audit/diagnostic-bundle operator workflows;
- [x] diagnostic bundle redaction retained;
- [x] stable-by-default update channel;
- [x] explicit update-availability check via `scripts/version.sh --check`;
- [x] backup, migration validation and automatic rollback preserved;
- [x] safe remove and purge behavior preserved;
- [x] landing page productized;
- [x] privacy and telemetry documentation;
- [x] compatibility/support matrix;
- [x] issue templates and PR template;
- [x] release tag ↔ VERSION validation;
- [x] release checksum;
- [x] container SBOM/provenance requested by release workflow;
- [x] invalid/inactive/expired/wrong-issuer/wrong-audience OAuth regression coverage;
- [x] revoked/inactive OAuth token rejection made explicit at the introspection boundary;
- [x] OAuth/DCR edge rate limiting;
- [x] documentation link checker;
- [x] guided installer lifecycle acceptance job;
- [x] Full/R5 explicitly excluded from production claims.

## Validation gates for this branch

- [x] pull request opened;
- [x] reference implementation CI green;
- [x] integrated-auth component acceptance green;
- [x] Docker package acceptance green;
- [x] lifecycle acceptance including guided install green;
- [x] full ephemeral Scoped acceptance green;
- [x] ephemeral VPS proof green;
- [x] simultaneous independent-instance acceptance green;
- [x] final cross-review after CI evidence.

## Deliberately not completed here

- [ ] stable `v0.1.0` tag/release. It must follow the owner's final human gate.
- [ ] final clean-install human gate on the owner's target environment.
- [ ] long-running R4 production reliability evidence.
- [ ] Full/R5 production maturity.
- [ ] separate project-managed cryptographic release signing key.

## GitHub storefront metadata

README, Pages, release policy, badges/links and repository content are handled in this branch.

Repository-level description/topics/homepage are GitHub metadata rather than repository files. They should be reviewed in the final storefront pass; the current connected GitHub tool surface does not expose a repository-metadata mutation action, so this item must not be marked complete by automation without evidence.
