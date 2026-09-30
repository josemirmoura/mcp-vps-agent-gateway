# Première exécution de Portico MCP

## Objectif

L’installation est déclarative et orientée terminal. L’état opérateur principal est :

~~~text
.env                   # contient VPS_AGENT_SCOPE_ROOT
config/policy.yaml     # policy logique capacités/ressources
~~~

Docker Compose démarre le paquet. Il n’existe pas d’installateur parallèle cachant la configuration.

## Frontière prise en charge

L’utilisateur exécute les commandes directement sur la VPS. Aucun développeur, shell distant contrôlé par ChatGPT, mot de passe VPS ou clé SSH privée n’est requis.

Le mot de passe opérateur OAuth est une credential du service : il est saisi localement, sans écho, dans `setup-integrated-auth.sh`.

## Entrée guidée

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

`install.sh` orchestre les composants transparents existants, montre l’autorité effective avant le runtime et peut reprendre après une interruption.

## Phase 1 — Bootstrap

Le profil recommandé est **Standard** : plafond physique `/opt`, aucune racine de projet statique. Le plafond est seulement la limite maximale du système de fichiers, pas une autorisation. Un autre chemin absolu peut être choisi.

~~~bash
bash scripts/init.sh --scope /opt --dynamic-baseline
~~~

Avant toute mutation, `scripts/preflight.py` vérifie host, Docker/Compose, outils, systemd, RAM et edge. Un prérequis obligatoire manquant arrête le processus avant de créer l’état.

Le bootstrap crée secrets locaux aléatoires, ID d’instance stable, `config/policy.yaml`, plafond sélectionné, utilisateur non-root réel pour le shell confiné, répertoire d’état et validation Compose.

## Plafond physique

`compose.yaml` monte uniquement `VPS_AGENT_SCOPE_ROOT` sous `/host`. Le Broker impose que les chemins filesystem, cwd shell et Compose restent dans ce plafond. Toute tentative d’évasion échoue fermée.

Standard avec `--dynamic-baseline` n’autorise pas automatiquement `/opt`. Whole Host nécessite `VPS_AGENT_WHOLE_HOST=1` et `compose.host.yaml`, avec plafond `/`, sans activer Full.

## Phase 2 — Autorité MCP

Standard démarre sans projet autorisé. `permissions.discover_scope` peut révéler uniquement les noms des répertoires immédiatement sous le plafond, jamais leur contenu.

~~~text
permissions.request_root_access
 -> demande Broker en attente
 -> elicitation MCP / confirmation native du client
 -> délégation liée au subject
~~~

- `read` : lecture filesystem ;
- `work` : lecture/écriture + cwd shell confiné ;
- `compose` : ajoute les actions Compose déjà permises par la policy statique.

### Fichiers protégés

`.env` et `.env.*` restent verrouillés même dans un projet autorisé. `.env.example`, `.env.sample` et `.env.template` restent des modèles lisibles.

`permissions.request_sensitive_access` crée une exception séparée, temporaire, exact-path, liée au subject, auditée, expirante et révocable. Les jobs shell Scoped masquent les chemins protégés et les aliases hardlink détectés, sauf autorisation temporaire exacte.

Après un avertissement renforcé, une délégation peut viser le plafond lui-même. Elle ne peut jamais dépasser le plafond et n’ouvre toujours pas les secrets protégés.

systemd, Docker/Compose, réseau, paquets, utilisateurs/groupes, firewall et élévation restent séparément contrôlés.

## Phase 3 — Démarrage

~~~bash
docker compose up -d --build
~~~

~~~text
Gateway: non-root, pas de host root
Broker: privilégié, host monté sous /host, aucun port de contrôle distant
~~~

Docker est le mécanisme de packaging ; la policy du Broker reste la frontière d’autorisation.

## Phase 4 — Vérification locale

~~~bash
bash scripts/verify.sh
~~~

Vérifie Compose, santé Broker/Gateway, intégrité audit, authentification et un `system.info` sans danger. Succès local = runtime prêt, **pas installation terminée**.

## Phase 5 — OAuth intégré + endpoint public

Le DNS est une action manuelle externe. Portico guide la création A/AAAA et valide la résolution.

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Un Traefik existant unique et sûr est réutilisé ; sinon le Traefik intégré démarre si 80/443 sont libres. ZITADEL + PostgreSQL démarrent, l’identité opérateur dédiée non-admin est créée, username/e-mail sont affichés, audience ressource MCP et client privé d’introspection sont créés, DCR est activé, Gateway/Broker sont liés au subject puis la vérification publique s’exécute.

Succès :

~~~text
INTEGRATED AUTH: READY
~~~

À relancer après changement DNS/proxy/OAuth :

~~~bash
bash scripts/verify-public.sh
~~~

Cela vérifie HTTPS, discovery OAuth/OIDC, metadata de ressource protégée, DCR/PKCE, introspection privée et rejet fail-closed du MCP non authentifié.

## Phase 6 — Capacité ChatGPT

Vérifiez que le compte/workspace réel expose Developer Mode et la création d’app MCP personnalisée. OpenAI contrôle le déploiement et l’UI.

## Phase 7 — Tutoriel ChatGPT Web

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Création de l’app et login OAuth se font dans ChatGPT Web avec l’identité OAuth dédiée, jamais avec des credentials VPS/SSH.

## Phase 8 — Test réel

Le script fixe une base d’audit. L’utilisateur termine ChatGPT à son rythme puis appuie sur Entrée. ChatGPT doit appeler `system.info`.

Succès :

~~~text
appel MCP réel ChatGPT
+ subject authentifié attendu
+ policy allow
+ exécution réussie
+ audit Broker
+ chaîne d’audit valide
~~~

Alors seulement :

~~~text
CHATGPT WEB CONNECTION VERIFIED
INSTALLATION COMPLETE
~~~

## Mise à jour et suppression

~~~bash
bash scripts/update.sh
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Supprimer aussi le checkout :

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

Les projets délégués et ressources tierces ne sont pas supprimés simplement parce que Portico pouvait les gérer.
