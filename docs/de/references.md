# Technische Referenzen

Geprüft: 2026-09-26. Versionen und Produktverfügbarkeit bei Implementierung erneut prüfen.

## OpenAI / ChatGPT

- MCP-Server-Konzepte: https://developers.openai.com/plugins/concepts/mcp-server
- MCP-Server bauen: https://developers.openai.com/plugins/build/mcp-server
- Authentifizierung: https://developers.openai.com/plugins/build/auth
- Sicherheit und Datenschutz: https://developers.openai.com/plugins/guides/security-privacy
- Plugins-Verfügbarkeit: https://help.openai.com/en/articles/20001256-plugins-in-chatgpt-and-codex
- Developer Mode / vollständiges MCP: https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

## MCP

- MCP-Spezifikation: https://modelcontextprotocol.io/specification/
- Release-Überblick 2026-07-28: https://blog.modelcontextprotocol.io/posts/2026-07-28/
- Tasks Extension: https://tasks.extensions.modelcontextprotocol.io/specification/draft/tasks
- Offizielles Go SDK: https://github.com/modelcontextprotocol/go-sdk

Das Go SDK ist Tier 1 und unterstützt MCP 2026-07-28.

## Linux

- systemd-Handbücher: https://www.freedesktop.org/software/systemd/man/
- Landlock: https://docs.kernel.org/userspace-api/landlock.html
- openat2: https://man7.org/linux/man-pages/man2/openat2.2.html

## Docker

- Docker-Engine-Sicherheit: https://docs.docker.com/engine/security/

## Auth

Jeder standardkonforme OAuth/OIDC-Provider kann bei Bedarf verwendet werden.

## Principle

Client-UX-Anmerkungen, Modellanweisungen und Host-Bestätigungen helfen, ersetzen aber nicht die Broker-Autorisierung.
