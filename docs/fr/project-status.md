# État et maturité du projet

## Étape actuelle

**PRE-RELEASE PRODUCTIZATION / CHATGPT E2E GATE COMPLETE**

L’implémentation Go Docker-first possède une validation répétable sur runners propres pour runtime Scoped, lifecycle et opérations host.

Il n’existe encore ni release stable production, ni promesse de compatibilité production, ni historique long sur VPS réelle, ni claim production pour Full/admin shell.

L’évidence labo couvre CRUD filesystem, denials, shell/jobs sandboxed, systemd host, Docker/Compose host, diagnostics, lifecycle et audit. Le 2026-09-28, le chemin OAuth intégré supporté a été exercé avec ChatGPT Web sur une VPS réelle via `system.info` authentifié vu par le Broker et inscrit dans la chaîne d’audit. Gate 0A est fermé sans claim stable.

## Échelle de maturité

### R0 — Architecture seule

Documentation, aucune implémentation exécutable.

### R1 — Chemin produit + Gate 0B

Gate 0A terminé pour OAuth intégré + ChatGPT Web, MCP Inspector passe, POC sûre read/write uniquement dans une racine jetable.

### R2 — Pilote privilégié typé

Gateway/Broker séparés, une action de service non critique via tools typées, deny fail-closed et audit local.

### R3 — Pilote Scoped

Un stack réel sous policy explicite, plusieurs jours d’usage, recovery/denials testés, pas de Full routinier.

### R4 — Production Scoped durcie

Auth du deployment, state durable, jobs/retries/locks, secret delivery, backup/recovery et monitoring testés. Alors seulement production-capable pour Scoped.

### R5 — Elevated/Full production

En plus de R4 : Full flag, approval out-of-band, grants temporaires/expiration, revoke-all, élévation réseau séparée, audit tamper-evident + checkpoint distant et recovery admin testé.

## Full par défaut

~~~yaml
features:
  full_mode_enabled: false
~~~

Full n’est pas requis pour le succès. Un Scoped solide est une cible production valide.

## Évidence plutôt que popularité

Stars/forks ne prouvent pas la production. Préférez builds reproductibles, tests, releases, historique deployment, recovery tests, incidents, maintenance dépendances, security review et revue externe.

## Prochain jalon

**Acceptance clean-install du propriétaire de `v0.1.0-rc.5`**. RC5 inclut elicitation MCP native, discovery-only du plafond, enforcement des secrets et safety annotations.

Bloquant pour stable `v0.1.0` : exact tag RC5, vrai OAuth ChatGPT, `system.info` audité, UX native, discovery sans fuite de contenu, opérations Scoped, denial/exception temporaire/révocation `.env`, révocation root et lifecycle. Fiabilité longue durée ensuite. Voir [operator-acceptance.md](operator-acceptance.md) et [implementation-validation.md](implementation-validation.md).
