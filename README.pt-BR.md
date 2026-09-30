# Portico MCP

**Conecte o ChatGPT Web diretamente à sua VPS Linux via MCP, com a autoridade definida por você e aplicada pelo servidor.**

O Portico MCP é uma ponte open source e auto-hospedada para permitir que o ChatGPT Web ou outro cliente MCP inspecione e opere uma VPS Linux pelo Model Context Protocol. O OAuth autentica o cliente, um Gateway sem root expõe a superfície MCP e um Broker local privilegiado reautoriza as operações no host conforme a policy definida pelo operador, registrando evidências de auditoria.

> **Status: candidato em productização pré-release.** Os workflows em Ubuntu 24.04 validam o pacote de ponta a ponta, e o caminho OAuth integrado e auto-hospedado foi verificado por uma chamada real auditada de `system.info` feita pelo ChatGPT Web em 28/09/2026. Ainda faltam a primeira release pública estável, uma promessa formal de compatibilidade e evidência de confiabilidade prolongada em produção.

## O problema que resolve

Sem um caminho confiável de execução, trabalhar com uma VPS usando IA vira um revezamento manual: o ChatGPT sugere um comando, você troca para o terminal, executa, copia os logs de volta, reconstrói o contexto e repete. Dar acesso amplo no estilo SSH reduz parte dessa fricção, mas cria um problema de confiança muito maior.

O Portico MCP fecha esse ciclo mantendo a autorização dentro da VPS. Pela própria conversa, o ChatGPT pode inspecionar arquivos, diagnosticar serviços, analisar logs, trabalhar com Docker e executar outras operações que o dono da VPS habilitar explicitamente.

~~~text
ChatGPT Web
    -> OAuth
    -> MCP Gateway
    -> Policy Broker
    -> Sua VPS Linux
~~~

O GitHub hospeda o código-fonte, a documentação, as releases e o canal de atualização. **Depois da instalação, o GitHub não fica no caminho operacional entre o ChatGPT Web e a VPS.**

O fluxo suportado não exige entregar ao ChatGPT a senha da VPS, chave SSH privada ou Docker socket.

## A ideia

Um único pacote traz a caixa de ferramentas ampla. **Quem decide a porção da VPS entregue ao MCP é o usuário.**

~~~text
um projeto:         /opt/meu-app
várias raízes:      /opt/app + /var/www/site + /srv/dados
filesystem inteiro: /
~~~

Filesystem é só uma dimensão. A policy controla separadamente shell, units systemd, recursos Docker/Compose, rede, pacotes, usuários/grupos, firewall e administração temporária.

Em hosts com vários projetos, um teto físico mais amplo como `/opt` pode conter várias raízes delegadas explicitamente. Ampliar o teto não autoriza o teto inteiro. O Portico pode descobrir apenas os nomes das pastas imediatamente abaixo desse teto para que o assistente peça o projeto correto sem abri-lo. As raízes estáticas continuam em `config/policy.yaml`; em produção, novas raízes podem ser solicitadas via MCP e, quando o cliente suporta elicitation, o próprio ChatGPT/MCP client apresenta sua interface nativa de confirmação humana. O Broker continua sendo a autoridade, e as raízes podem ser listadas e revogadas pelo chat sem reescrever a policy nem reiniciar o Broker. O `scripts/delegate-root.sh` continua disponível para bootstrap e raízes estáticas administradas pelo operador.

Arquivos com segredos formam uma segunda fronteira dentro dos projetos autorizados. `.env` e `.env.*` continuam bloqueados por padrão e exigem uma autorização temporária separada para o caminho exato; templates comuns como `.env.example` continuam legíveis. A mesma proteção vale para tools de arquivo e shell confinado.

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

## Requisitos

Antes de iniciar a instalação guiada, tenha:

- uma VPS Linux; Ubuntu 24.04 LTS é o alvo validado no release candidate;
- Docker Engine 24+ com Docker Compose v2;
- pelo menos 2 GB de RAM para o caminho OAuth integrado com ZITADEL;
- Git, OpenSSL, Python 3 e curl;
- um domínio/DNS público apontando para a VPS, portas TCP 80/443 disponíveis e HTTPS válido;
- uma conta/workspace do ChatGPT Web em que o Modo de Desenvolvedor e a criação de app MCP personalizado estejam realmente disponíveis.

A OpenAI controla disponibilidade e permissões por plano, workspace e rollout. A documentação oficial atual lista **suporte MCP completo, incluindo escrita/alteração, para Business, Enterprise e Edu**; usuários Pro podem conectar MCPs personalizados com permissões de leitura/busca no modo de desenvolvedor. O Portico MCP não pode elevar permissões que o próprio ChatGPT não exponha. Antes de depender de um rótulo de plano, confira a interface atual da conta e a documentação oficial da OpenAI.

Veja [compatibilidade](docs/compatibility.md) para a matriz suportada/testada.

## Começo rápido

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

O terminal conduz o fluxo suportado:

~~~text
Pré-requisitos
 -> Escopo: Standard / Project / Whole Host
 -> revisão da autoridade efetiva
 -> Containers
 -> Verificação local
 -> HTTPS + OAuth integrado
 -> Conectar ChatGPT
 -> system.info real e auditado
 -> INSTALAÇÃO CONCLUÍDA
~~~

**Standard** é o padrão recomendado: `/opt` é o teto físico, enquanto as raízes de projeto começam sem autorização e são liberadas depois por aprovação explícita. **Whole Host** muda o teto físico do filesystem para `/`, mas não habilita Full nem rede irrestrita. Filesystem e capabilities continuam dimensões separadas da policy.

A orquestração é fina e transparente. Os scripts individuais `init.sh`, Compose, `verify.sh`, OAuth e ChatGPT continuam disponíveis e documentados. Veja o [Quick Start](docs/quick-start.md) e o [fluxo de instalação](docs/installer-flow.md).

O fluxo suportado é autônomo na própria VPS do usuário. Ele nunca pede senha da VPS, chave SSH privada, acesso administrativo remoto irrestrito ou segredos não relacionados ao serviço para ChatGPT ou mantenedor.

Durante o release candidate, `main` continua sendo desenvolvimento. Depois que a aceitação final pelo operador congelar `v0.1.0`, instalações normais devem usar o checkout da tag estável.

## Caixa de ferramentas

A implementação atual inclui filesystem completo em escopo, shell/jobs sandboxed, systemd tipado, Docker/Compose, diagnósticos, pacotes, usuários/grupos, UFW, elevação fora do canal MCP, SQLite exclusivo do Broker, journal de operações, fencing locks e audit hash-chain.

Uma capability existir no pacote não significa que esteja habilitada. `config/policy.yaml` define a base estática. Delegações dinâmicas de raízes armazenadas pelo Broker só acrescentam autoridade de recurso após pedido e aprovação explícitos; elas nunca ampliam o teto físico nem habilitam ações ausentes da policy estática.

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
- segredos protegidos como `.env` continuam trancados mesmo dentro de uma raiz autorizada, salvo autorização temporária separada.
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

## Contribuição

Veja [CONTRIBUTING.md](CONTRIBUTING.md) para princípios de engenharia e validação e [SECURITY.md](SECURITY.md) para orientação de reporte privado de vulnerabilidades.

## Licença

Apache-2.0. Veja [LICENSE](LICENSE).
