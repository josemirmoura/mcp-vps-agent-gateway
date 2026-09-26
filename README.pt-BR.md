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


## Início rápido

> Este repositório é, neste momento, uma arquitetura de referência e um guia de implementação, não um binário pronto. O caminho mais rápido é construir primeiro o **Gate 0** e provar o cliente MCP real antes de adicionar privilégios.

### 1. Clone

```bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
```

### 2. Leia o mínimo necessário

```text
docs/mvp-first.md
docs/security-hardening-v2.md
docs/transport-and-aggregation.md
AGENTS.md
```

### 3. Implemente somente o Gate 0

A primeira versão deve expor apenas:

```text
system.info
file.read_test
file.write_test
```

com leitura e escrita restritas a um diretório descartável, por exemplo:

```text
/tmp/vps-agent-poc/
```

Ainda **não** adicione root, controle de Docker, OAuth, modo Full, Approval Service ou shell administrativo genérico.

### 4. Teste com o cliente MCP real

O Gate 0 passa quando:

- o cliente descobre as ferramentas
- leitura funciona
- escrita funciona quando o produto/cliente permitir
- caminhos proibidos falham
- reconexões não corrompem estado

Se o cliente real não conseguir executar a ferramenta de escrita necessária, pare aí e resolva somente a camada de integração.

### 5. Entregue a um agente de código

Você pode passar esta instrução:

```text
Leia AGENTS.md e docs/mvp-first.md.
Implemente somente o Gate 0.
Não implemente execução privilegiada, acesso Docker, modo Full,
OAuth, Approval Service ou qualquer recurso além do Gate 0.
Adicione testes de confinamento de caminhos e erros claros.
```

Depois que o Gate 0 passar, avance para Gate 1 e Gate 2 em [docs/mvp-first.md](docs/mvp-first.md).


Comece por [docs/architecture.md](docs/architecture.md) e depois siga [docs/implementation-runbook.md](docs/implementation-runbook.md).

Licença: Apache-2.0.
