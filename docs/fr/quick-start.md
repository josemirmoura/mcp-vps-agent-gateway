# Démarrage rapide Portico MCP

Portico MCP s’installe depuis le terminal avec Docker Compose et les scripts transparents du dépôt.

## Prérequis

- VPS Linux ; Ubuntu 24.04 LTS est la cible validée du RC ;
- Docker Engine 24+ et Docker Compose v2 ;
- au moins 2 Go RAM pour ZITADEL intégré ;
- Git, OpenSSL, Python 3 et curl ;
- DNS public, TCP 80/443 et HTTPS valide ;
- compte/workspace ChatGPT disposant réellement du mode développeur et des apps MCP personnalisées.

## Démarrer

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Langues officielles :

~~~text
en · pt-BR · es · de · fr · ja · id
~~~

Choix explicite :

~~~bash
bash scripts/install.sh --lang fr
~~~

## Flux guidé

~~~text
Prérequis
 -> Scope
 -> Autorité effective
 -> Conteneurs
 -> Vérification locale
 -> Accès public sécurisé
 -> Connecter ChatGPT
 -> appel MCP réel audité
 -> INSTALLATION COMPLETE
~~~

Le preflight vérifie host, Docker/Compose, outils, systemd, RAM et edge. Un Traefik existant est réutilisé s’il est identifiable en sécurité ; sinon le Traefik intégré peut prendre 80/443 s’ils sont libres.

## Profil recommandé : Standard

~~~text
plafond physique : /opt
racines statiques : aucune
autorité projet : approuvée ensuite à la demande
~~~

Le plafond définit la limite maximale et **n’autorise pas /opt**.

Autre plafond :

~~~bash
bash scripts/install.sh --profile custom --scope /srv/apps
~~~

`permissions.discover_scope` montre uniquement les noms des dossiers immédiats. L’accès est ensuite demandé via confirmation native MCP :

~~~text
permissions.request_root_access
 -> confirmation native MCP
 -> Broker active read / work / compose
~~~

L’IA ne peut pas approuver sa propre extension d’autorité.

### Secrets

`.env` et `.env.*` restent verrouillés même dans les projets autorisés. Les modèles tels que `.env.example` restent lisibles. Un vrai secret nécessite une approbation temporaire séparée via `permissions.request_sensitive_access`.

## Project

~~~bash
bash scripts/install.sh --profile project --scope /opt/my-app
~~~

## Whole Host

~~~bash
bash scripts/install.sh --profile whole-host
~~~

Fixe le plafond à `/` sans activer automatiquement Full, réseau, paquets, utilisateurs ou firewall.

## Utilisateur shell

~~~bash
bash scripts/install.sh --run-as deploy
~~~

Les jobs Scoped s’exécutent comme utilisateur réel non-root.

## OAuth public

Portico guide DNS et crée une identité OAuth dédiée :

~~~text
Username: vps-operator
Email:    operator@example.com
~~~

Ce ne sont pas des credentials Linux/SSH/root.

## Finalisation ChatGPT

`scripts/connect-chatgpt.sh` affiche le tutoriel complet. Configurez app/OAuth à votre rythme, envoyez `system.info`, revenez au terminal et appuyez sur Entrée.

## Test local seulement

~~~bash
bash scripts/install.sh --profile custom --scope /opt --local-only --yes
~~~

Ce n’est **pas** une installation publique terminée.

## Suppression

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Checkout également :

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

## Releases stables

Après `v0.1.0`, utilisez un tag stable pour la production :

~~~bash
git clone --branch v0.1.0 https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Le checkout Git permet mises à jour fast-forward vérifiées, backups, migrations et rollback automatique.
