# 技術リファレンス

確認日: 2026-09-26。実装時に version と product availability を再確認してください。

## OpenAI / ChatGPT

- MCP server concepts: https://developers.openai.com/plugins/concepts/mcp-server
- Build an MCP server: https://developers.openai.com/plugins/build/mcp-server
- Authentication: https://developers.openai.com/plugins/build/auth
- Security and privacy: https://developers.openai.com/plugins/guides/security-privacy
- Plugins availability: https://help.openai.com/en/articles/20001256-plugins-in-chatgpt-and-codex
- Developer Mode / full MCP: https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

## MCP

- MCP specification: https://modelcontextprotocol.io/specification/
- 2026-07-28 release overview: https://blog.modelcontextprotocol.io/posts/2026-07-28/
- Tasks extension: https://tasks.extensions.modelcontextprotocol.io/specification/draft/tasks
- Official Go SDK: https://github.com/modelcontextprotocol/go-sdk

Go SDK は Tier 1 で MCP 2026-07-28 を support します。

## Linux

- systemd manuals: https://www.freedesktop.org/software/systemd/man/
- Landlock: https://docs.kernel.org/userspace-api/landlock.html
- openat2: https://man7.org/linux/man-pages/man2/openat2.2.html

## Docker

- Docker Engine security: https://docs.docker.com/engine/security/

## Auth

必要な deployment では standards-compliant OAuth/OIDC provider を利用できます。

## Principle

Client UX annotation、model instruction、host confirmation は挙動を助けますが Broker authorization の代替ではありません。
