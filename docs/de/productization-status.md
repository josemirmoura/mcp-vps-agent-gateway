# Produktisierungsstatus — v0.1.0-rc.3

Dieses Dokument verfolgt die öffentliche Produktisierungs-Checkliste des Release-Candidate-Branches. Es ist Evidenz/Status, kein Ersatz für kanonische Architektur- oder Sicherheitsdokumente.

## Abgeschlossen

- [x] Implementierung/Status mit realer ChatGPT-Web-E2E-Evidenz abgeglichen;
- [x] README EN/PT-BR produktorientiert;
- [x] Repository-Sanitization erweitert;
- [x] kanonischer Produktname, Tagline, Version;
- [x] SemVer VERSION + CHANGELOG;
- [x] geführte transparente Terminalinstallation;
- [x] Project / Custom / Whole Host UX;
- [x] effektive Autorität vor Runtime-Start;
- [x] Whole Host von Full getrennt;
- [x] uneingeschränktes Netzwerk von Full getrennt;
- [x] integriertes OAuth/OIDC dokumentiert;
- [x] Audit und Observability getrennt dokumentiert;
- [x] status/health/log/audit/diagnostic bundle;
- [x] Secret-Redaktion;
- [x] stable-by-default Updatekanal;
- [x] Updatecheck via `scripts/version.sh --check`;
- [x] Backup, Migrationsvalidierung, automatischer Rollback;
- [x] safe remove und purge;
- [x] produktisierte Landing Page;
- [x] Datenschutz/Telemetrie;
- [x] Kompatibilitäts-/Supportmatrix;
- [x] Issue-/PR-Templates;
- [x] Tag ↔ VERSION Validierung;
- [x] Checksums;
- [x] Container SBOM/Provenance;
- [x] OAuth-Regressionsfälle;
- [x] explizite Ablehnung revokter/inaktiver Tokens;
- [x] OAuth/DCR Rate Limiting;
- [x] Docs-Linkchecker;
- [x] Installer-Lifecycle-Acceptance;
- [x] Full/R5 aus Produktionsclaims ausgeschlossen.

## Validierungsgates

CI, integrierte Auth, Docker-Paket, Lifecycle, Scoped-Acceptance, ephemere VPS, simultane Instanzen und abschließendes Cross-Review sind grün.

## Bewusst offen

- [ ] Stable `v0.1.0`;
- [ ] finaler Clean-Install-Acceptance-Gate des Owners;
- [ ] langfristige R4-Evidenz;
- [ ] Full/R5-Produktionsreife;
- [ ] projektverwalteter kryptographischer Signing Key.

## GitHub-Metadaten

README, Pages, Release Policy, Badges/Links und Repository-Inhalt werden hier behandelt. Description/Topics/Homepage sind GitHub-Metadaten und dürfen ohne Mutations-Evidenz nicht als abgeschlossen markiert werden.
