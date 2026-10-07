# Portico MCP Community

**Conecte um chat web de IA compatível a um computador Linux com policy e auditoria aplicadas no servidor.**

Este repositório contém o runtime público do Portico Community.

O caminho validado atualmente conecta o ChatGPT Web ou outro cliente MCP compatível a um único host Linux por meio de um Gateway sem privilégios e de um Broker local privilegiado. O proprietário da máquina define a autoridade; o Broker aplica essa autoridade.

> **Status: candidato Community em pré-release.** Os workflows em Ubuntu 24.04 validam o pacote de ponta a ponta. A release estável, a confiabilidade prolongada em produção e o licenciamento futuro ainda dependem de gates específicos.

## Limite do produto Community

O Portico Community tem uma promessa deliberadamente simples:

```text
um chat web de IA compatível
        |
        | HTTPS + MCP Streamable HTTP + OAuth/OIDC
        v
Portico Gateway
        |
        | Unix socket local protegido
        v
Portico Broker
        |
        v
um computador Linux
```

Esse computador pode ser uma VPS, workstation ou servidor local.

O Community é self-hosted e não depende de um control plane gerenciado do Portico Cloud.

## O problema que resolve

Sem um caminho confiável de execução, trabalhar com uma máquina usando IA vira um revezamento manual: gerar comando, trocar para o terminal, executar, copiar o resultado de volta e repetir.

O Portico fecha esse ciclo mantendo a autorização na máquina controlada pelo proprietário. Conforme a policy local, o cliente de IA pode inspecionar arquivos, diagnosticar serviços, trabalhar com Docker/Compose, executar shell/jobs limitados e usar outras operações tipadas.

O caminho suportado não exige entregar à IA senha da VPS, chave SSH privada ou Docker socket.

## Modelo de segurança

**O LLM nunca é a fronteira de segurança. A máquina decide.**

Propriedades centrais:

- Gateway roda sem root;
- Gateway não recebe o filesystem raiz do host nem Docker socket;
- Broker só é alcançado por Unix socket local protegido;
- Broker reautoriza chamadas privilegiadas pela policy local;
- traversal e symlink escape são bloqueados;
- writes usam journal/idempotência quando aplicável;
- capacidades destrutivas ou administrativas exigem policy explícita;
- segredos protegidos não aparecem em tools genéricas;
- operações privilegiadas são auditadas.

## Requisitos

Para o caminho suportado atual:

- host Linux; Ubuntu 24.04 LTS é o alvo validado no release candidate;
- Docker Engine 24+ e Docker Compose v2;
- Git, OpenSSL, Python 3 e curl;
- DNS público + HTTPS válido para o endpoint MCP;
- um chat web/cliente MCP compatível em que a capability necessária de MCP personalizado esteja disponível.

A capability do cliente pode variar por conta, workspace, plano e rollout. As alegações de compatibilidade são baseadas em evidência, e não apenas no nome da assinatura. Veja [Compatibilidade](docs/compatibility.md).

## Começo rápido

```bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
```

O fluxo guiado cobre pré-requisitos, escopo de autoridade, containers, verificação local, HTTPS/OAuth, conexão com o cliente de IA e uma chamada MCP real auditada. Ele só imprime `INSTALAÇÃO CONCLUÍDA` / `INSTALLATION COMPLETE` depois que o gate ponta a ponta passa.

Veja:

- [Quick Start](docs/quick-start.md)
- [Contrato de instalação](docs/installation-contract.md)
- [Fluxo de instalação](docs/installer-flow.md)
- [Integração ChatGPT](docs/chatgpt-integration.md)

## Autoridade

O pacote oferece uma caixa de ferramentas ampla, enquanto o proprietário escolhe o que a IA pode operar.

```text
um projeto:         /opt/meu-app
várias raízes:      /opt/app + /var/www/site + /srv/dados
filesystem inteiro: /
```

Filesystem é apenas uma dimensão. A policy controla separadamente shell, units systemd, recursos Docker/Compose, rede, pacotes, usuários/grupos, firewall e administração temporária.

## Operação

```bash
bash scripts/diagnose.sh status
bash scripts/diagnose.sh health
bash scripts/diagnose.sh logs 200
bash scripts/diagnose.sh audit 100

bash scripts/update.sh
bash scripts/remove.sh safe
```

Após a primeira versão estável, tags SemVer estáveis serão o canal normal de atualização. O atualizador preserva configuração/estado do operador, valida o runtime novo e faz rollback quando a verificação falha e o fluxo suporta recuperação segura.

## Roadmap público

Este repositório pode evoluir o runtime Community e contratos públicos de interoperabilidade. Produtos Portico mais amplos podem adicionar capacidades gerenciadas multi-node separadamente.

Compromissos do roadmap público são limitados ao que já foi lançado, ao que é necessário para interoperabilidade ou ao que foi explicitamente aprovado para anúncio público. Veja [Roadmap público](docs/roadmap-multinode-control-plane.md).

## Documentação

- [Mapa da documentação](docs/README.md)
- [Modelo do produto](docs/product-model.md)
- [Arquitetura](docs/architecture.md)
- [Matriz de segurança](docs/security-release.md)
- [Privacidade e telemetria](docs/privacy.md)
- [Operação](docs/operations.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Política de releases](docs/releases.md)
- [Suporte](docs/support.md)

## Licença

A linha pública atual de pré-release é distribuída sob **Apache-2.0**. Veja [LICENSE](LICENSE).

O modelo de licenciamento das futuras releases estáveis Community está em revisão. As permissões históricas Apache-2.0 continuam regidas pelos termos aplicáveis às versões em que foram publicadas.
