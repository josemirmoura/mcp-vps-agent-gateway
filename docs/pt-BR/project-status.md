# Status e maturidade do projeto

## Estágio atual

**PRÉ-RELEASE / PRODUTIZAÇÃO / GATE E2E CHATGPT CONCLUÍDO**

A implementação Go Docker-first possui validação repetível em runners limpos para runtime Scoped, lifecycle do pacote e caminho de operações no host.

Ainda não existe:

- release estável de produção;
- promessa de compatibilidade de produção;
- histórico prolongado em VPS real;
- claim de produção para Full/admin shell.

A evidência de laboratório cobre o pacote Docker ponta a ponta em Ubuntu limpo: CRUD de filesystem, negações de autorização, shell/jobs sandboxed, systemd do host, Docker/Compose do host, diagnósticos, lifecycle e auditoria. Em 2026-09-28, o caminho suportado de OAuth self-hosted também foi exercitado contra ChatGPT Web numa VPS real por uma chamada autenticada `system.info` observada pelo Broker e registrada na cadeia de auditoria. Isso fecha o Gate 0A do caminho suportado sem criar claim de produção estável.

## Escada de maturidade

### R0 — Somente arquitetura

Documentação existe. Sem implementação executável de referência.

### R1 — Caminho de produto + Gate 0B

- Gate 0A registra a rota real ChatGPT/cliente. **Concluído em 2026-09-28 para OAuth integrado + ChatGPT Web.**
- MCP Inspector passa.
- POC de leitura/escrita segura funciona apenas na raiz descartável de teste.

### R2 — Piloto privilegiado tipado

- Gateway e Broker separados.
- Um serviço não crítico pode ser inspecionado/reiniciado por tools tipadas.
- Recursos não autorizados falham fechados.
- Auditoria local existe.

### R3 — Piloto Scoped

- Um stack real roda sob policy Scoped explícita.
- Comportamento operacional durável testado por vários dias.
- Casos de recovery e denial exercitados.
- Trabalho rotineiro não exige Full.

### R4 — Produção Scoped endurecida

- autenticação do deployment validada;
- estado durável do Broker testado;
- jobs/retries/locks testados;
- entrega de segredos testada;
- backup/recovery testados;
- monitoramento operacional presente.

Somente após R4 o projeto pode ser descrito como production-capable para o uso Scoped documentado.

### R5 — Produção Elevated/Full

Além de R4:

- feature flag Full explicitamente habilitado;
- aprovação out-of-band funcionando;
- grants temporários/expiração funcionando;
- revoke-all funcionando;
- elevação de rede controlada separadamente;
- auditoria tamper-evident + checkpoint remoto;
- recovery de shell/admin testado.

## Default de Full

~~~yaml
features:
  full_mode_enabled: false
~~~

Full não é necessário para considerar o projeto bem-sucedido. Um Scoped forte é endpoint de produção válido.

## Evidência acima de popularidade

Stars/forks são sinais de comunidade, não prova de produção.

Prefira builds reproduzíveis, testes automatizados, releases, histórico de deployment, recovery tests documentados, aprendizados de incidentes, manutenção de dependências, security review e revisão externa.

## Próximo marco

O projeto possui evidência clean-room do toolbox Scoped amplo, pacote Docker e gate real ChatGPT Web OAuth concluído em 2026-09-28.

O marco atual é **aceitação clean-install pelo proprietário do `v0.1.0-rc.5`**. RC5 inclui MCP elicitation nativa para aprovação humana, inventário discovery-only do teto físico, enforcement de segredos protegidos aninhados e safety annotations explícitas no catálogo público.

O bloqueador para congelar `v0.1.0` é a instalação do tag RC5 exato pelo proprietário, incluindo OAuth real no ChatGPT, `system.info` auditado, UX nativa de aprovação, discovery sem vazamento de conteúdo, operações Scoped representativas, negação de `.env`/exceção temporária/revogação, revogação de root e checks de lifecycle. Confiabilidade de longa duração continua requisito posterior. Veja [operator-acceptance.md](operator-acceptance.md) e [implementation-validation.md](implementation-validation.md).
