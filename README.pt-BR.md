# MCP VPS Agent Gateway

Plano de controle MCP orientado a segurança para permitir que ChatGPT ou outro cliente MCP trabalhe numa VPS Linux com a autoridade escolhida explicitamente pelo dono da VPS.

> **Status: candidato em productização pré-release.** Os workflows em Ubuntu 24.04 validam o pacote de ponta a ponta, e o caminho OAuth integrado e auto-hospedado foi verificado por uma chamada real auditada de `system.info` feita pelo ChatGPT Web em 28/09/2026. Ainda faltam a primeira release pública estável, uma promessa formal de compatibilidade e evidência de confiabilidade prolongada em produção.

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

Requisitos: VPS Linux suportada, Docker Engine + Docker Compose v2, Git, OpenSSL, Python 3 e curl. Veja [compatibilidade](docs/compatibility.md).

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

O terminal conduz o fluxo suportado:

~~~text
Ambiente
 -> Escopo: Project / Custom / Whole Host
 -> revisão da autoridade efetiva
 -> Containers
 -> Verificação local
 -> HTTPS + OAuth integrado
 -> Conectar ChatGPT
 -> system.info real e auditado
 -> INSTALAÇÃO CONCLUÍDA
~~~

**Project** é o padrão recomendado. **Whole Host** muda o teto físico do filesystem para `/`, mas não habilita Full nem rede irrestrita. Filesystem e capabilities continuam dimensões separadas da policy.

A orquestração é fina e transparente. Os scripts individuais `init.sh`, Compose, `verify.sh`, OAuth e ChatGPT continuam disponíveis e documentados. Veja o [Quick Start](docs/quick-start.md) e o [fluxo de instalação](docs/installer-flow.md).

O fluxo suportado é autônomo na própria VPS do usuário. Ele nunca pede senha da VPS, chave SSH privada, acesso administrativo remoto irrestrito ou segredos não relacionados ao serviço para ChatGPT ou mantenedor.

Durante o release candidate, `main` continua sendo desenvolvimento. Depois que o gate humano final congelar `v0.1.0`, instalações normais devem usar o checkout da tag estável.

## Caixa de ferramentas

A implementação atual inclui filesystem completo em escopo, shell/jobs sandboxed, systemd tipado, Docker/Compose, diagnósticos, pacotes, usuários/grupos, UFW, elevação fora do canal MCP, SQLite exclusivo do Broker, journal de operações, fencing locks e audit hash-chain.

Uma capability existir no pacote não significa que esteja habilitada. Quem decide é config/policy.yaml.

Delete recursivo, chmod/chown, pacotes, usuários, firewall, rede irrestrita e shell administrativo são escolhas explícitas. file.chmod pode aplicar 0777 se o dono da VPS habilitar essa capability.

## Segurança

- Gateway roda sem root e não recebe /host.
- No modo Scoped, o pacote monta fisicamente apenas `VPS_AGENT_SCOPE_ROOT`; acesso ao filesystem inteiro exige o override explícito `compose.host.yaml`.
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

## Operação

~~~bash
bash scripts/diagnose.sh status
bash scripts/diagnose.sh health
bash scripts/diagnose.sh logs 200
bash scripts/diagnose.sh audit 100

bash scripts/update.sh
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

A remoção segura preserva configuração e auditoria. O purge exige confirmação explícita e remove apenas artefatos do MCP. O update faz backup da configuração/estado, usa Git fast-forward, verifica o runtime novo e restaura código/estado anterior se a verificação falhar.
## Documentação

- [Quick Start](docs/quick-start.md)
- [Contrato de instalação](docs/installation-contract.md)
- [Modelo do produto](docs/product-model.md)
- [Operação](docs/operations.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Compatibilidade](docs/compatibility.md)
- [Privacidade e telemetria](docs/privacy.md)
- [Política de releases](docs/releases.md)
- [Suporte](docs/support.md)
- [Arquitetura](docs/architecture.md)
- [Autenticação](docs/authentication.md)
- [Integração ChatGPT](docs/chatgpt-integration.md)
- [Matriz de segurança do release](docs/security-release.md)

## Licença

Apache-2.0. Veja [LICENSE](LICENSE).
