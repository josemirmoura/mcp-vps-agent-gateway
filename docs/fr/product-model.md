# Modèle produit : toolbox complet, autorité Scoped

## Décision centrale

Livrer un catalogue complet et contrôler l’autorité par policy server-side.

Pas de binaires limited/project/full séparés :

~~~text
mêmes binaires
+ même catalogue MCP
+ policy différente
= autorité effective différente
~~~

Le LLM ne choisit jamais le scope.

## Scope détenu par l’utilisateur

L’opérateur décide exactement quelle part de VPS est déléguée. Les exemples n’accordent rien ; seule la policy explicite confirmée est autoritative.

~~~text
un dossier:
/opt/my-app

plusieurs:
/opt/app-a
/var/www/site
/srv/data

filesystem entier:
/
~~~

Même principe indépendamment pour systemd, Docker, réseau, packages, users/groups et ressources admin.

## Profils

### Standard

Recommandé pour hosts multi-projets. Plafond physique par défaut `/opt`, aucune racine de projet autorisée à l’installation.

Portico ne découvre que les noms immédiats. Contenu verrouillé jusqu’à approbation `read`, `work` ou `compose`. Les secrets protégés restent une frontière imbriquée.

### Project

Autonomie dans une racine comme `/opt/my-app`: CRUD, shell Scoped avec cwd dans la racine, Docker/systemd sélectionnés et destinations réseau sélectionnées.

### Policy avancée/statique

Plusieurs racines/groupes peuvent être définis directement. Standard guidé préfère la délégation runtime au YAML initial.

### Whole host

Autorise explicitement des ressources host-wide. C’est une policy, pas un build différent. Peut inclure `/`, systemd large, Docker, packages, users/groups, firewall/network et shell admin temporaire. Les capacités dangereuses restent explicites et auditées.

## Catalogue de capacités

### Filesystem

`file.list/stat/read/mkdir/write/patch/copy/move/remove/remove_recursive/hash/chmod/chown`.

Les capabilities existent même désactivées. Suppression récursive et changements de permissions larges sont séparés.

### Commands/jobs

`shell.exec`, `job.start/status/tail/cancel`.

Le shell doit être sandboxed server-side ; un path en YAML ne suffit pas. Policy se traduit en cwd roots, ProtectSystem, ReadWritePaths/ReadOnlyPaths, ProtectHome, PrivateTmp, cgroups, MemoryMax, TasksMax, timeout, output limits et network policy.

### systemd

`service.list/status/logs/start/stop/restart/reload/enable/disable`, limité par units/actions canoniques.

### Docker/Compose

`docker.list/inspect/logs/start/stop/restart`, `compose.config/pull/up/down`. Opérations typées préférées ; aucun Docker socket dans Gateway.

### Diagnostics

`system.info/health/disk/memory`, `process.list/inspect`, `network.listen/check`, `journal.read`.

### Administration

Disponibles mais désactivées par défaut : package manager, users/groups, firewall, chmod/chown larges, `shell.exec_admin`.

## Scope multidimensionnel

Filesystem n’est qu’un axe. Policy scope séparément roots, systemd, Docker, shell cwd, réseau, package actions, users/groups, firewall et admin temporaire.

Un Project peut avoir CRUD complet dans `/opt/my-app` et aucune autorité sur nginx, Docker, apt ou Internet.

## chmod et opérations destructives

`file.chmod`, y compris 0777, peut être exposé si policy l’autorise. Defaults n’activent pas world-writable. `file.remove_recursive` reste distinct ; chmod/chown host-wide est une décision admin séparée.

## UX de configuration

`config/policy.yaml` est autoritatif et lisible humain/IA : ressources, preset, filesystem, shell/sandbox, systemd, Docker/Compose, réseau, administration, approval/elevation, auth/exposition. Compose/runtime rejettent l’invalide. Policy reste éditable sans réinstallation.

## Packaging

~~~text
Gateway container
  non-root
  no /host
  no Docker socket
        |
        | Unix socket
        v
Broker container
  privileged host-control boundary
  host mounted at /host
        |
        v
VPS
~~~

Broker n’est pas un sandbox autour du host : c’est la frontière privilégiée server-side emballée dans Docker. Docker apporte packaging/lifecycle, Gateway reste non-root, Broker reçoit host root uniquement pour les actions autorisées, n’expose pas d’API TCP distante, et policy décide les ressources. Monter `/host` n’accorde rien au LLM.

Flux principal :

~~~bash
bash scripts/install.sh
~~~

Orchestration transparente de bootstrap, Compose, vérification, OAuth et connexion ChatGPT. Standard/Project/Whole Host, autorité affichée, reprise possible. Les commandes individuelles restent accessibles. [installation-contract.md](installation-contract.md) définit la frontière.

## Installation complète seulement après ChatGPT

Le tutoriel seul ne suffit pas. Après runtime/policy, afficher endpoint, auth, scope effectif et étape suivante.

Premier run :

1. vérifier HTTPS ;
2. vérifier auth ;
3. test server-side inoffensif ;
4. afficher autorité ;
5. tutoriel ChatGPT courant ;
6. attendre la connexion ;
7. exiger un appel inoffensif depuis ChatGPT ;
8. vérifier subject, policy, exécution, audit ;
9. seulement alors terminer.

Les surfaces ChatGPT changent ; tutoriel versionné/revalidé.

~~~text
clone / bundle
 -> choisir autorité
 -> démarrer Gateway + Broker
 -> valider policy/sécurité
 -> OAuth + HTTPS
 -> tutoriel ChatGPT
 -> connexion
 -> appel E2E réel
 -> audit
 -> installation terminée
~~~
