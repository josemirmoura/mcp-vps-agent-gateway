# Referencias técnicas

Comprobado: 2026-09-26. Vuelve a validar versiones y disponibilidad al implementar.

## OpenAI / ChatGPT

- Conceptos de servidor: https://developers.openai.com/plugins/concepts/mcp-server
- Construir un servidor: https://developers.openai.com/plugins/build/mcp-server
- Autenticación: https://developers.openai.com/plugins/build/auth
- Seguridad y privacidad: https://developers.openai.com/plugins/guides/security-privacy
- Disponibilidad de Plugins: https://help.openai.com/en/articles/20001256-plugins-in-chatgpt-and-codex
- Developer Mode / MCP completo: https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

## MCP

- Especificación MCP: https://modelcontextprotocol.io/specification/
- Resumen de la release 2026-07-28: https://blog.modelcontextprotocol.io/posts/2026-07-28/
- Extensión Tasks: https://tasks.extensions.modelcontextprotocol.io/specification/draft/tasks
- SDK Go oficial: https://github.com/modelcontextprotocol/go-sdk

El SDK Go es Tier 1 y soporta MCP 2026-07-28.

## Linux

- Manuales systemd: https://www.freedesktop.org/software/systemd/man/
- Landlock: https://docs.kernel.org/userspace-api/landlock.html
- openat2: https://man7.org/linux/man-pages/man2/openat2.2.html

## Docker

- Seguridad de Docker Engine: https://docs.docker.com/engine/security/

## Auth

Puede usarse cualquier proveedor OAuth/OIDC conforme a estándares cuando sea necesario.

## Principle

Las anotaciones UX, instrucciones al modelo y confirmaciones del host ayudan, pero no sustituyen la autorización del Broker.
