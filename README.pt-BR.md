# MCP VPS Agent Gateway

Plano de controle MCP orientado a segurança para permitir que ChatGPT ou outro cliente MCP trabalhe numa VPS Linux com a autoridade escolhida explicitamente pelo dono da VPS.

> **Status: candidato a release no que depende de aceitação automatizada.** Os workflows em Ubuntu 24.04 validam o pacote Docker de ponta a ponta. Um release estável ainda depende do gate externo OAuth no ChatGPT Web e de uma chamada real auditada contra o deployment alvo.

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

Requisitos: VPS Linux, Docker Engine, plugin Docker Compose, Git, OpenSSL e Python 3.

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git &&
cd mcp-vps-agent-gateway &&
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/vps-agent-sandbox &&
bash scripts/init.sh --scope /opt/vps-agent-sandbox &&
docker compose up -d --build &&
bash scripts/verify.sh
~~~

O Começo rápido delega propositalmente apenas `/opt/vps-agent-sandbox`; troque esse caminho pelo diretório que você quer entregar ao MCP. A cadeia com `&&` para imediatamente se alguma etapa falhar.

O bootstrap cria segredos locais aleatórios, ID estável da instância, estado e uma policy do operador **fora do Git**, a partir de config/policy.example.yaml. `--scope` grava `VPS_AGENT_SCOPE_ROOT` no `.env` e migra os caminhos da policy padrão a partir do escopo anterior. O diretório escolhido precisa existir; agora o bootstrap falha antes de qualquer build Docker quando ele não existe. A autoridade continua sendo escolhida explicitamente pelo usuário.

A verificação local prova health, negação de token inválido, chamada MCP real de system.info e integridade do audit chain. **A instalação ainda não terminou.**

### Autoridade de filesystem sobre o host inteiro

Scoped é o padrão. Para expor deliberadamente o filesystem inteiro ao Broker, use o override explícito:

~~~bash
sed -i 's/^VPS_AGENT_WHOLE_HOST=.*/VPS_AGENT_WHOLE_HOST=1/' .env
docker compose -f compose.yaml -f compose.host.yaml up -d --build
~~~

`compose.host.yaml` é a chave deliberada de whole-host e define `/` como teto físico do Broker. A policy server-side continua controlando quais operações MCP são permitidas.
### HTTPS público + ChatGPT

Para uma conexão do ChatGPT com escrita, use um Authorization Server OAuth/OIDC aderente ao padrão MCP e configure endpoint público, issuer, audience/resource e subject esperado no .env. Bearer estático fica restrito a laboratório/acceptance local.

~~~dotenv
VPS_AGENT_DOMAIN=mcp.exemplo.com
VPS_AGENT_PUBLIC_URL=https://mcp.exemplo.com/mcp
VPS_AGENT_AUTH_MODE=oidc
VPS_AGENT_OIDC_ISSUER=https://auth.exemplo.com
VPS_AGENT_OIDC_AUDIENCE=https://mcp.exemplo.com/mcp
VPS_AGENT_SUBJECT=<subject-esperado-do-token>
~~~

Depois:

~~~bash
docker compose -f compose.yaml -f compose.https.yaml up -d --build
bash scripts/verify-public.sh
bash scripts/connect-chatgpt.sh
~~~

O verificador público exige HTTPS, discovery OAuth e negação fail-closed sem autenticação. O script de conexão mostra o fluxo atual do ChatGPT Web e espera uma **nova chamada auditada de system.info feita pelo ChatGPT**.

~~~text
tutorial
 -> app OAuth conectado no ChatGPT
 -> ChatGPT chama o MCP
 -> Gateway autentica
 -> Broker valida subject + policy
 -> operação autorizada chega à VPS
 -> audit registra
 -> INSTALAÇÃO CONCLUÍDA
~~~

Sem a chamada real, a instalação permanece incompleta.

Orientação oficial verificada em 27/09/2026: Full MCP com escrita/modificação está documentado no ChatGPT Web para Business, Enterprise e Edu. Pro fica limitado a read/fetch. MCP autenticado com escrita usa OAuth 2.1; o ChatGPT não apresenta API keys customizadas.

Fontes oficiais:
- https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt
- https://developers.openai.com/plugins/build/auth
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

- [Modelo do produto](docs/product-model.md)
- [Fluxo Docker e primeira execução](docs/installer-flow.md)
- [Arquitetura](docs/architecture.md)
- [Autenticação](docs/authentication.md)
- [Múltiplas instâncias](docs/multi-instance.md)
- [Integração ChatGPT](docs/chatgpt-integration.md)
- [Threat model](docs/threat-model.md)
- [Hardening](docs/security-hardening-v2.md)

## Licença

Apache-2.0. Veja [LICENSE](LICENSE).
