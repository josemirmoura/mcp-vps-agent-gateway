# Carte de la documentation

La documentation canonique se trouve dans `docs/`. Ce dossier est la traduction française officielle. En cas d’écart avec le fichier anglais de la même version, l’anglais reste la référence technique jusqu’à correction.

## Opérateur / utilisateur

1. [quick-start.md](quick-start.md) — chemin d’installation le plus court.
2. [installation-contract.md](installation-contract.md) — frontière normative de l’installation.
3. [installer-flow.md](installer-flow.md) — déroulement complet de l’installation.
4. [product-model.md](product-model.md) — Project, Standard et Whole Host.
5. [chatgpt-integration.md](chatgpt-integration.md) — connexion ChatGPT et porte de finalisation.
6. [authentication.md](authentication.md) — OAuth/OIDC intégré.
7. [operations.md](operations.md) — état, logs, audit, mise à jour, suppression.
8. [troubleshooting.md](troubleshooting.md) — dépannage.
9. [faq.md](faq.md) — questions fréquentes.
10. [compatibility.md](compatibility.md) — environnements validés et non validés.
11. [privacy.md](privacy.md)
12. [releases.md](releases.md)
13. [support.md](support.md)
14. [operator-acceptance.md](operator-acceptance.md)

## Développement / sécurité

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

Ordre de précédence en cas de conflit : architecture, contrat d’installation, état courant du projet, schéma de policy, durcissement sécurité, sémantique runtime, documents de support, documents historiques.

**Architecture en une phrase :** deux processus Go, un Gateway MCP non privilégié et un Broker local privilégié reliés par socket Unix ; le Broker contrôle autorisation, état et exécution privilégiée.
