# Références techniques

Vérifié : 2026-09-26. Revalidez versions et disponibilité au moment de l’implémentation.

## OpenAI / ChatGPT

- Concepts de serveur: https://developers.openai.com/plugins/concepts/mcp-server
- Construire un serveur: https://developers.openai.com/plugins/build/mcp-server
- Authentification: https://developers.openai.com/plugins/build/auth
- Sécurité et confidentialité: https://developers.openai.com/plugins/guides/security-privacy
- Disponibilité des Plugins: https://help.openai.com/en/articles/20001256-plugins-in-chatgpt-and-codex
- Developer Mode / MCP complet: https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

## MCP

- Spécification MCP: https://modelcontextprotocol.io/specification/
- Vue d’ensemble de la release 2026-07-28: https://blog.modelcontextprotocol.io/posts/2026-07-28/
- Extension Tasks: https://tasks.extensions.modelcontextprotocol.io/specification/draft/tasks
- SDK Go officiel: https://github.com/modelcontextprotocol/go-sdk

Le SDK Go est Tier 1 et supporte MCP 2026-07-28.

## Linux

- Manuels systemd: https://www.freedesktop.org/software/systemd/man/
- Landlock: https://docs.kernel.org/userspace-api/landlock.html
- openat2: https://man7.org/linux/man-pages/man2/openat2.2.html

## Docker

- Sécurité Docker Engine: https://docs.docker.com/engine/security/

## Auth

Tout fournisseur OAuth/OIDC conforme aux standards peut être utilisé si nécessaire.

## Principle

Annotations UX, instructions au modèle et confirmations host aident le comportement mais ne remplacent pas l’autorisation du Broker.
