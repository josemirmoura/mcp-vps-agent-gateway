# Compatibilité

## Validé pour le release candidate

### Système

- Ubuntu 24.04 LTS sur runners GitHub propres ;
- Linux avec systemd pour services/jobs du host.

### Architecture

- `linux/amd64` : acceptation runtime complète en CI ;
- `linux/arm64` : binaires/images construits, mais acceptation host équivalente pas encore exécutée sur matériel arm64 natif.

### Runtime

- Docker Engine 24+ ;
- Docker Compose v2 via `docker compose` ;
- au moins 2 Go RAM pour ZITADEL intégré ;
- Git, OpenSSL, Python 3 et curl.

2 Go est un minimum fonctionnel de dépendance, pas une recommandation de dimensionnement production.

Références :
- https://zitadel.com/docs/self-hosting/deploy/compose
- https://zitadel.com/docs/self-hosting/manage/requirements

## Chemin public ChatGPT

Nécessite DNS A/AAAA, TCP 80/443 public via Traefik compatible existant ou intégré, HTTPS valide et un compte/workspace ChatGPT exposant réellement Developer Mode et l’enregistrement d’app MCP personnalisée.

OpenAI contrôle disponibilité et rollout. Portico ne peut pas augmenter les permissions côté ChatGPT.

Référence projet :
https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

## Builds

- `linux/amd64`
- `linux/arm64`

## Pas de promesse actuelle

- hôtes non Linux ;
- Docker Desktop comme VPS de production ;
- Linux sans systemd pour host service/job ;
- Kubernetes ;
- Podman Compose ;
- Windows/macOS natifs ;
- Full/R5 en production.

D’autres distributions Linux peuvent fonctionner mais restent non validées.

## Ressources

Le minimum RAM est uniquement le seuil des dépendances. Le hashing des mots de passe et les charges MCP/Docker réelles peuvent demander davantage. Dimensionnez avec mesures réelles et `scripts/diagnose.sh health`.
