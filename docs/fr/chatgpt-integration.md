# Intégration ChatGPT

Vérifié : 2026-09-30.

## Cible

~~~text
ChatGPT Web
 -> OAuth discovery + Dynamic Client Registration
 -> HTTPS /mcp
 -> Gateway
 -> Broker
 -> VPS
 -> audit détectant les altérations
~~~

La santé des conteneurs ne suffit pas à terminer l’installation.

## Prérequis ChatGPT

Avant le setup public, vérifiez que le compte/workspace réel expose Developer Mode et la création d’app MCP personnalisée, ainsi que les permissions réellement disponibles. OpenAI contrôle rollout et interface.

Références du projet :

- https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt
- https://developers.openai.com/plugins/build/app-quickstart

Une connexion read-only peut aider au diagnostic mais ne représente pas le chemin produit complet avec écriture.

## Prérequis serveur

1. Création MCP disponible.
2. `bash scripts/verify.sh` réussi.
3. DNS vers la VPS.
4. Auth intégrée prête :

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Doit terminer par :

~~~text
INTEGRATED AUTH: READY
~~~

Vérifier la frontière publique :

~~~bash
bash scripts/verify-public.sh
~~~

## Connexion

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Flux attendu :

1. ChatGPT lit les metadata RFC 9728 ;
2. découvre ZITADEL ;
3. enregistre dynamiquement un client OAuth public ;
4. l’opérateur se connecte avec le compte dédié ;
5. Authorization Code + PKCE ;
6. ChatGPT découvre les tools ;
7. `system.info` est appelé ;
8. Gateway valide le token ;
9. Broker vérifie subject stable et policy ;
10. l’audit enregistre l’appel.

ChatGPT ne reçoit jamais mot de passe VPS, clé SSH privée, root ou shell distant sans restriction. Le mot de passe OAuth est saisi uniquement dans le login du fournisseur d’identité.

## Porte de finalisation

`connect-chatgpt.sh` fixe une base d’audit. Après configuration, Entrée déclenche la recherche d’un nouvel appel authentifié `system.info`.

~~~text
tutoriel affiché
 != succès

app ChatGPT connectée
 + system.info authentifié
 + subject attendu
 + Broker allow
 + exécution réussie
 + audit correspondant
 = INSTALLATION COMPLETE
~~~

## Transport

~~~text
https://<domain>/mcp
~~~

MCP Streamable HTTP sur HTTPS, sans transport WebSocket propriétaire.

## Frontière de sécurité

Les confirmations ChatGPT sont des contrôles UX supplémentaires, pas l’autorisation.

**Gateway authentifie. Broker autorise. Le propriétaire de la VPS choisit la policy.**
