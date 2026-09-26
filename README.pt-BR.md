# MCP VPS Agent Gateway

Arquitetura de referência orientada a segurança para conectar ChatGPT ou outro cliente MCP a uma VPS Linux sem transformar o modelo em fronteira de segurança.

> **Status: PRE-ALPHA / DOCS-FIRST.** Ainda não existe binário pronto para produção nem release estável.

## O que é

O alvo é:

~~~text
ChatGPT Web / cliente MCP
        |
        | Streamable HTTP
        v
vps-agent-gateway      sem root
        |
        | Unix socket
        v
vps-agent-broker       privilegiado, somente local
        |
        v
Linux / systemd / Docker
~~~

O Gateway cuida de MCP, autenticação e schemas.

O Broker é a fronteira real de segurança: policy, estado, jobs, filesystem, Docker/systemd, secrets e execução privilegiada.

> **O LLM nunca é a fronteira de segurança.**

## O que ainda não é

Hoje este repositório não é:

- software plug-and-play
- imagem Docker pronta
- release de produção
- shell root genérico para IA
- garantia de que todo plano do ChatGPT aceite MCP privado com escrita

A arquitetura completa é o norte. A implementação começa pequena.

## Modos

- **Controlled** — inspeção com mudanças estreitamente controladas.
- **Scoped** — autonomia dentro de um perímetro explícito.
- **Full** — pacote temporário de capabilities, opcional e desligado por padrão.

O trabalho rotineiro deve acontecer em Scoped. Full é excepcional.

## Stack de referência

- **Gateway:** Go + SDK MCP oficial para Go
- **Broker:** Go
- **IPC:** Unix Domain Socket
- **Transporte:** MCP Streamable HTTP
- **Estado:** SQLite, aberto somente pelo Broker
- **Isolamento:** systemd transient units + cgroups
- **Segredos:** systemd credentials e/ou arquivos root-owned
- **Sandbox adicional:** Landlock quando disponível
- **Entrada:** reverse proxy existente ou túnel privado suportado

Sem Kubernetes, Redis, service mesh ou daemon separado de policy.

## Início rápido

### 1. Clone

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
~~~

### 2. Leia primeiro

~~~text
docs/README.md
docs/project-status.md
docs/mvp-first.md
docs/architecture.md
AGENTS.md
~~~

### 3. Faça o Gate 0A antes de código privilegiado

Valide qual caminho real do ChatGPT será usado.

Em 2026-09-26, a OpenAI documenta MCP privado completo com write/modify em Developer Mode para Business, Enterprise e Edu. Não assuma que Plus Web aceita um MCP privado customizado com escrita.

Veja [docs/chatgpt-integration.md](docs/chatgpt-integration.md).

### 4. Implemente somente o Gate 0B

Exponha apenas:

~~~text
system.info
file.read_test
file.write_test
~~~

Restrinja arquivos a:

~~~text
/tmp/vps-agent-poc/
~~~

Ainda não adicione root, Docker, SQLite, Full, approval ou shell genérico.

### 5. Entregue isto a um agente de código

~~~text
Leia AGENTS.md, docs/README.md e docs/mvp-first.md.
Implemente somente o Gate 0B usando Go e o SDK MCP oficial para Go.
Não implemente execução privilegiada, Docker, Full, approval,
SQLite ou shell genérico.
Adicione testes de confinamento de caminhos, entradas inválidas e erros claros.
~~~

## Escada de implementação

~~~text
Gate -1   Adotar / adaptar / construir
Gate 0A   Provar a superfície real do ChatGPT
Gate 0B   POC MCP seguro de leitura/escrita
Gate 1    Uma ação privilegiada tipada
Gate 2    Um stack real em Scoped
Gate 3    Estado/jobs/secrets duráveis
Gate 4    Escritas validadas mais amplas
Gate 5    Elevação temporária opcional
~~~

Cada gate precisa justificar a próxima camada.

## Segurança central

- Gateway nunca roda como root.
- Gateway nunca recebe Docker socket.
- Broker reautoriza toda chamada privilegiada.
- Policy é deny-by-default e autoritativa dentro do Broker.
- Só o Broker abre o SQLite.
- Writes replay-safe usam idempotência gerada pela infraestrutura.
- Writes non-replay-safe nunca recebem retry cego.
- Filesystem resiste a traversal e symlink escape.
- Resultados de tools são dados não confiáveis.
- O modelo não registra MCP downstream dinamicamente.
- Full nasce desligado e não implica rede irrestrita.
- Secrets não são expostos por ferramenta genérica de leitura.

## Documentação

A ordem canônica e as regras de precedência estão em [docs/README.md](docs/README.md).

## Licença

Apache-2.0. Veja [LICENSE](LICENSE).
