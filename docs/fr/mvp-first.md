# Parcours d’implémentation MVP-first

L’architecture complète est le nord, pas le premier jalon.

## Gate -1 — Adopter, adapter ou construire

Évaluez d’abord produits existants et serveurs MCP open source. Adoptez si cela satisfait, adaptez si proche, construisez seulement si la combinaison manque.

Cible différenciante :

~~~text
ChatGPT Web
+ VPS sous contrôle propre
+ policy server-side
+ autonomie Scoped
+ élévation temporaire optionnelle
+ Broker contrôlé par l’opérateur
~~~

## Gate 0A — Prouver la surface ChatGPT

**Terminé le 2026-09-28 pour OAuth intégré + ChatGPT Web.**

Valider le vrai client cible avant le chemin privilégié. Ne pas supposer qu’un MCP privé write-capable fonctionne sur tout plan.

Routes :

1. workspace/plan avec private full MCP write ;
2. app/plugin éligible avec remote write ;
3. MCP Inspector tant que la distribution n’est pas résolue.

Enregistrer :

~~~text
target_surface:
target_plan:
integration_route:
read_available:
write_available:
private_or_published:
tested_date:
~~~

Si un futur compte ne propose pas custom MCP, arrêter à cette frontière ou changer seulement la distribution. Ne pas affaiblir le serveur.

## Gate 0B — POC MCP sûre

Serveur Go non privilégié + SDK officiel. Seulement :

~~~text
system.info
file.read_test
file.write_test
~~~

Filesystem limité à `/tmp/vps-agent-poc/`. Pas de root, Docker, write systemd, SQLite, secrets, Full, approval ou shell générique.

Succès : Inspector, discovery, read/write selon client, paths interdits refusés, erreurs claires, retry/reconnect sûrs.

## Gate 1 — Une action privilégiée typée

Broker minimal par Unix socket. Ajouter `service.status`/`service.restart` pour une unité non critique. Exclure shell admin générique, Full, approval UI, Docker large et audit distant.

## Gate 2 — Pilote Scoped réel

Policy explicite sur un stack. Ajouter uniquement besoins démontrés : file.read, status/restart service, Docker logs/restart, job.status. Mesurer fréquence, réussite, corrections, false denials, capacités manquantes et recovery. Gate passé quand Scoped fait du travail utile sans élévation routinière.

## Gate 3 — Durabilité

Si besoin : SQLite Broker, jobs durables, idempotency identity, locks, systemd credentials/secret refs, audit structuré renforcé.

## Gate 4 — Writes plus larges

Selon besoin : `file.write`/`file.patch`, configs validées, autres actions Docker/systemd typées et `shell.exec` sandboxed dans racines Scoped. Shell générique non requis.

## Gate 5 — Élévation temporaire

Seulement si Scoped est insuffisant : demandes d’élévation, approbation humaine out-of-band, leases temporaires, revoke-all, ancrage audit distant, flag Full. Puis seulement considérer `shell.exec_admin`.

## Règle

Chaque nouveau composant exige une preuve du gate précédent.

> Le vrai client cible peut-il effectuer un travail utile, sûr et fiable sur un vrai serveur ?
