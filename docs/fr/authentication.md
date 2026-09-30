# Authentification

Vérifié avec le modèle OAuth MCP/OpenAI courant le 2026-09-27.

## Chemin supporté

Le chemin public ChatGPT supporté est **OAuth/OIDC intégré auto-hébergé**.

Le paquet exécute une instance ZITADEL dédiée + PostgreSQL à côté du Gateway. Aucun service d’identité tiers ni tunnel séparé n’est requis.

La vérification locale/labo utilise encore le bearer statique généré par `scripts/init.sh`, jamais comme credential public ChatGPT.

## Topologie

~~~text
ChatGPT
   |
   | HTTPS + OAuth 2.x / OIDC
   v
domaine public
   |-------------------------------|
   |                               |
   v                               v
Gateway                         ZITADEL
ressource protégée             authorization server
   |                               |
   v                               v
Broker                       PostgreSQL identity state
   |
   v
VPS
~~~

Le même hostname peut servir les deux rôles. `/mcp`, `/healthz` et metadata RFC 9728 vont au Gateway ; discovery, login, token, user-info et DCR à ZITADEL.

## Bootstrap

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Le script génère les secrets identity, réutilise Traefik si sûr, démarre l’edge intégré uniquement sur host propre, lance ZITADEL/PostgreSQL, active DCR ouvert requis par MCP, crée un opérateur dédié non-admin, stocke son subject stable dans `.env` et recrée Gateway/Broker en mode intégré.

Le mot de passe est lu sans écho et jamais écrit dans `.env` ou le marker.

Owner IAM bootstrap humain et PAT machine temporaire sont supprimés après création de l’opérateur. Le PAT interne du login-client reste dans un volume privé car ZITADEL Login en a besoin.

## Metadata de ressource protégée

~~~text
/.well-known/oauth-protected-resource
~~~

Annonce normalement :

~~~text
resource: https://mcp.example.com/mcp
authorization_servers:
  - https://mcp.example.com
scopes_supported:
  - openid
bearer_methods_supported:
  - header
~~~

`scripts/verify-public.sh` vérifie metadata, scopes, discovery OIDC, DCR, PKCE S256, refresh token, introspection privée, HTTPS et refus non authentifié.

## Binding de ressource et introspection

Un projet ZITADEL dédié représente l’audience MCP :

~~~text
MCP VPS Agent Resource
  └── API application: MCP VPS Agent Introspector
~~~

Project ID comme scope obligatoire :

~~~text
urn:zitadel:iam:org:project:id:<resource-project-id>:aud
~~~

Gateway valide chaque bearer opaque via RFC 7662 sur le réseau Docker privé.

Accepté seulement si `active: true`, `sub` présent, `iss` exact, `exp` futur, `aud` contient le projet et tous les scopes requis sont présents.

Le client d’introspection appartient au même projet ; ZITADEL applique aussi son contrôle d’audience et Gateway le répète.

L’introspection n’est pas une surface de gestion publique. Le secret reste local root-readable et redigé des bundles.

## Dynamic Client Registration

Les clients MCP s’enregistrent avant login ; DCR non authentifié est donc activé et rate-limited au Traefik. Les clients dynamiques vivent dans un projet dédié. L’opérateur doit quand même s’authentifier pour obtenir un token utile.

## Subject binding

~~~dotenv
VPS_AGENT_SUBJECT=<operator-user-id>
~~~

L’owner bootstrap est supprimé et n’est jamais l’identité Broker.

## Fail-closed

- static auth local seulement pour acceptance ;
- URL publique exige integrated mode ;
- HTTPS obligatoire ;
- resource/issuer metadata doivent correspondre ;
- discovery annonce DCR, S256 et refresh token ;
- audience scope dédié obligatoire ;
- token actif, non expiré, issuer/audience corrects ;
- `/mcp` sans auth renvoie 401 + challenge Bearer ;
- Broker revérifie subject + policy ;
- Gateway reste sans host root ni Docker socket.

## Edge proxy

Un seul Traefik identifiable est réutilisé. Sinon, avec 80/443 libres, démarrage du Traefik intégré. Jamais de remplacement d’un web server inconnu.

## Bearer static local/labo

~~~dotenv
VPS_AGENT_AUTH_MODE=static
~~~

Uniquement CI/local, pas le chemin public.
