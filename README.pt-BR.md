# MCP VPS Agent Gateway

Arquitetura de referência orientada a segurança para conectar um assistente de IA, como o ChatGPT, a uma VPS Linux via MCP, mantendo autorização e privilégios sob controle do servidor.

## Ideia central

A experiência desejada é próxima à de um agente de desenvolvimento local:

- **Controlled** — o agente investiga e pede aprovação para mudanças relevantes.
- **Scoped** — autonomia dentro de um perímetro explicitamente autorizado.
- **Full** — acesso administrativo temporário, concedido por uma pessoa e com expiração automática.

Regra principal:

> **O LLM nunca é a fronteira de segurança.**

O modelo solicita ações. O servidor decide se elas podem acontecer.

## Arquitetura

```text
ChatGPT / Cliente MCP
        |
        v
MCP Gateway (sem root)
        |
        v
Policy Engine
        |
        v
Execution Broker (privilegiado e mínimo)
        |
        v
Linux / Docker / systemd / arquivos / jobs
```

## Princípios

- Gateway sem root.
- Gateway sem acesso direto ao Docker socket.
- Operações privilegiadas passam por broker local.
- Política aplicada server-side.
- Full somente com autorização humana e TTL.
- O agente não pode elevar a si próprio.
- Shell com timeout, limites de memória/processos e limite de saída.
- Filesystem protegido contra path traversal e escapes por symlink.
- Escritas idempotentes quando possível.
- Jobs longos persistentes.
- Auditoria de ações e decisões.
- Segredos referenciados/injetados, evitando exposição ao modelo.

Comece por [docs/architecture.md](docs/architecture.md) e depois siga [docs/implementation-runbook.md](docs/implementation-runbook.md).

Licença: Apache-2.0.
