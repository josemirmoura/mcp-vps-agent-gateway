# Technical references

Re-check versions, availability and product-plan constraints at implementation time.

## OpenAI / ChatGPT

- OpenAI Plugins: https://developers.openai.com/plugins/
- MCP server guidance: https://developers.openai.com/plugins/build/mcp-server
- Authentication guidance: https://developers.openai.com/plugins/build/auth
- Security and privacy: https://developers.openai.com/plugins/guides/security-privacy
- OpenAI Help Center: https://help.openai.com/

## Model Context Protocol

- MCP home: https://modelcontextprotocol.io/
- MCP specification: https://modelcontextprotocol.io/specification/

## Linux isolation and filesystem

- systemd documentation: https://www.freedesktop.org/software/systemd/man/
- Linux Landlock: https://docs.kernel.org/userspace-api/landlock.html
- openat2(2): https://man7.org/linux/man-pages/man2/openat2.2.html

## Containers

- Docker Engine security: https://docs.docker.com/engine/security/
- Podman documentation: https://docs.podman.io/

## Authentication

Any standards-compliant OAuth 2.0/OIDC provider may be used. Validate issuer, audience, signature, expiration, subject and scopes server-side.

## General rule

Client UX annotations and model instructions can improve behavior, but they are not authorization controls. The server-side policy and execution boundary remain authoritative.
