# MCP VPS Agent Gateway

Plano de controle MCP orientado a segurança para permitir que ChatGPT ou outro cliente MCP trabalhe numa VPS Linux com a autoridade escolhida explicitamente pelo dono da VPS.

> **Status: pre-alpha / pacote Docker de referência executável.** O pacote está sendo validado em máquinas Ubuntu limpas e efêmeras do GitHub. Ainda não é um release estável de produção.

## A ideia

Um único pacote traz a caixa de ferramentas ampla. **Quem decide a porção da VPS entregue ao MCP é o usuário.**

~~~text
um projeto:         /opt/meu-app
várias raízes:      /opt/app + /var/www/site + /srv/dados
filesystem inteiro: /
~~~

Filesystem é só uma dimensão. A policy controla separadamente shell, units systemd, recursos Docker/Compose, rede, pacotes, usuários/grupos, firewall e administração temporária.

**O LLM nunca é a fronteira de segurança. O servidor decide.**

## Runtime

~~~text
ChatGPT Web / cliente MCP
        |
        | HTTPS + MCP Streamable HTTP
        v
Gateway container
sem root, sem /host
        |
        | Unix socket protegido
        v
Broker container
fronteira privilegiada
        |
        | host montado em /host
        v
Linux / systemd / Docker / arquivos
~~~

Docker é o mecanismo de empacotamento. O Broker continua sendo um componente privilegiado e deve ser tratado como equivalente a root na porção delegada. A policy server-side é autoritativa.

## Começo rápido

Requisitos: VPS Linux, Docker Engine, plugin Docker Compose, Git e OpenSSL.

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway

bash scripts/init.sh
~~~

Depois edite config/policy.yaml e .env.

~~~bash
docker compose up -d --build
bash scripts/verify.sh
~~~

HTTPS automático opcional com Caddy:

~~~bash
docker compose -f compose.yaml -f compose.https.yaml up -d --build
~~~

Defina VPS_AGENT_PUBLIC_URL no .env e execute:

~~~bash
bash scripts/connect-chatgpt.sh
~~~

O script mostra o tutorial do ChatGPT Web e depois espera uma **chamada real e auditada de system.info vinda do ChatGPT**. Mostrar o tutorial não conclui a instalação.

~~~text
tutorial
 -> usuário conecta o ChatGPT
 -> ChatGPT chama o MCP
 -> Broker confirma subject
 -> policy autoriza
 -> audit registra
 -> INSTALAÇÃO CONCLUÍDA
~~~

Se a chamada real não chegar, a instalação continua incompleta.

## Caixa de ferramentas

A implementação atual inclui filesystem completo em escopo, shell/jobs sandboxed, systemd tipado, Docker/Compose, diagnósticos, pacotes, usuários/grupos, UFW, elevação fora do canal MCP, SQLite exclusivo do Broker, journal de operações, fencing locks e audit hash-chain.

Uma capability existir no pacote não significa que esteja habilitada. Quem decide é config/policy.yaml.

Delete recursivo, chmod/chown, pacotes, usuários, firewall, rede irrestrita e shell administrativo são escolhas explícitas. file.chmod pode aplicar 0777 se o dono da VPS habilitar essa capability.

## Segurança

- Gateway roda sem root e não recebe /host.
- Broker não expõe API TCP; o Gateway usa Unix socket.
- Broker reautoriza cada chamada privilegiada pela policy.
- Gateway não recebe Docker socket.
- filesystem bloqueia traversal e symlink escape.
- writes usam operation journal e idempotência.
- mutações usam locks/fencing quando necessário.
- resultados de tools são dados não confiáveis.
- Full não implica rede irrestrita.
- secrets não aparecem em tools genéricas.

## Validação

O GitHub Actions valida vet, race detector, govulncheck, simulação adversarial, systemd real, Docker real, acceptance Gateway -> Broker -> Linux, acceptance do pacote Docker e testes negativos.

Veja [validação da implementação](docs/implementation-validation.md).

## Documentação

- [Modelo do produto](docs/product-model.md)
- [Fluxo Docker e primeira execução](docs/installer-flow.md)
- [Arquitetura](docs/architecture.md)
- [Integração ChatGPT](docs/chatgpt-integration.md)
- [Threat model](docs/threat-model.md)
- [Hardening](docs/security-hardening-v2.md)

## Licença

Apache-2.0. Veja [LICENSE](LICENSE).
