# État de productisation — v0.1.0-rc.3

Ce document suit la checklist de productisation publique de la branche release-candidate. C’est de l’évidence/état, pas un remplacement des documents canoniques d’architecture ou de sécurité.

## Terminé

- [x] implémentation/état alignés sur l’E2E réel ChatGPT Web ;
- [x] README EN/PT-BR orienté produit ;
- [x] garde de sanitization étendue ;
- [x] nom, tagline et version canoniques ;
- [x] VERSION SemVer + CHANGELOG ;
- [x] installation terminal guidée/transparente ;
- [x] UX Project / Custom / Whole Host ;
- [x] résumé d’autorité avant runtime ;
- [x] Whole Host séparé de Full ;
- [x] réseau sans restriction séparé de Full ;
- [x] OAuth/OIDC intégré documenté ;
- [x] audit et observabilité séparés ;
- [x] status/health/log/audit/bundle ;
- [x] redaction des diagnostics ;
- [x] canal stable par défaut ;
- [x] check via `scripts/version.sh --check` ;
- [x] backup, migration, rollback automatique ;
- [x] safe remove et purge ;
- [x] landing productisée ;
- [x] confidentialité/télémétrie ;
- [x] matrice compatibilité/support ;
- [x] templates issue/PR ;
- [x] validation tag ↔ VERSION ;
- [x] checksum ;
- [x] SBOM/provenance conteneur ;
- [x] couverture de régression OAuth ;
- [x] rejet explicite token révoqué/inactif ;
- [x] rate limiting OAuth/DCR ;
- [x] checker de liens ;
- [x] acceptance lifecycle installateur ;
- [x] Full/R5 exclu des claims production.

## Gates de validation

PR, CI de référence, auth intégrée, paquet Docker, lifecycle, Scoped éphémère, preuve VPS, instances simultanées et cross-review final sont verts.

## Délibérément non terminé

- [ ] tag/release stable `v0.1.0` ;
- [ ] acceptance clean-install finale du propriétaire ;
- [ ] preuve R4 longue durée ;
- [ ] maturité production Full/R5 ;
- [ ] clé cryptographique de signature gérée par le projet.

## Métadonnées GitHub

README, Pages, release policy, badges/links et contenu sont couverts ici. Description/topics/homepage sont des métadonnées GitHub et ne doivent pas être marquées complètes sans preuve de mutation.
