# Trilha de implementação MVP-first

A arquitetura completa é o norte, não o primeiro marco.

## Gate -1 — Adotar, adaptar ou construir

Avalie produtos existentes e servidores MCP open source primeiro.

Escolha deliberadamente:

- Adotar se uma solução já satisfaz os requisitos.
- Adaptar se uma implementação está próxima.
- Construir somente quando a combinação necessária estiver ausente.

O alvo diferenciador é:

~~~text
ChatGPT Web
+ VPS sob controle próprio
+ policy server-side
+ autonomia Scoped
+ elevação temporária opcional
+ Broker sob controle do operador
~~~

## Gate 0A — Provar a superfície do produto ChatGPT

**Status: concluído em 2026-09-28 para OAuth integrado + ChatGPT Web.**

Esse gate era obrigatório antes de considerar validado o caminho privilegiado.

Alvo primário: ChatGPT Web.

Em 2026-09-26, a OpenAI documentava full private MCP write/modify em Developer Mode para Business, Enterprise e Edu. A disponibilidade de plugins/apps varia por plano, superfície, região e capacidades incluídas.

Não presuma que MCP privado customizado com escrita pode ser anexado diretamente ao Plus Web.

Valide uma rota suportada:

1. rota privada de desenvolvimento — workspace/plano com full MCP write;
2. rota Plus Web — plugin/app elegível cuja escrita MCP remota esteja disponível;
3. rota protocol-only — MCP Inspector enquanto distribuição não estiver resolvida.

### Sucesso do Gate 0A

Registre:

~~~text
target_surface:
target_plan:
integration_route:
read_available:
write_available:
private_or_published:
tested_date:
~~~

A rota deve ser revalidada quando a superfície do ChatGPT mudar. Se a conta alvo não expuser MCP customizado, pare nesse boundary ou mude somente a distribuição. Não enfraqueça o servidor.

## Gate 0B — POC MCP seguro

Construa o menor servidor como usuário sem privilégios.

Referência: Go + SDK MCP Go oficial.

Exponha somente:

~~~text
system.info
file.read_test
file.write_test
~~~

Restrinja filesystem a:

~~~text
/tmp/vps-agent-poc/
~~~

Não adicione root, Docker, writes systemd, SQLite, segredos externos, Full, approval ou shell genérico.

### Sucesso do Gate 0B

- MCP Inspector passa;
- a rota ChatGPT escolhida descobre as tools quando disponível;
- leitura funciona;
- escrita funciona quando o produto permite;
- paths proibidos falham;
- erros são compreensíveis;
- retry/reconnect é seguro.

## Gate 1 — Uma ação privilegiada tipada

Adicione o Broker mínimo:

~~~text
vps-agent-gateway  (non-root)
        |
   Unix socket
        v
vps-agent-broker   (privileged)
~~~

Exponha:

~~~text
system.info
file.read_test
service.status
service.restart
~~~

`service.restart` fica limitado a uma unidade descartável/não crítica.

Ainda exclua shell admin genérico, Full, UI de aprovação, Docker amplo e auditoria remota.

### Sucesso do Gate 1

Cliente consegue inspecionar, reiniciar e verificar um serviço de teste, gerar auditoria e falhar com segurança em unidades não autorizadas.

## Gate 2 — Piloto real Scoped

Defina uma policy Scoped explícita para um stack real.

Adicione somente capacidades comprovadamente necessárias, como:

~~~text
system.info
file.read
service.status
service.restart
docker.logs
docker.action(restart)
job.status
~~~

Use por vários dias e meça frequência, sucesso/falha, correções do operador, false denials, capabilities ausentes e tempo de recovery.

Gate 2 passa quando Scoped resolve trabalho útil sem elevação rotineira.

## Gate 3 — Durabilidade

Adicione quando necessário:

- SQLite do Broker;
- jobs duráveis;
- identidade de idempotência gerada pela infraestrutura;
- locks lógicos;
- systemd credentials/referências de segredos;
- auditoria estruturada mais forte.

## Gate 4 — Writes mais amplos

Com necessidade demonstrada:

- `file.write` / `file.patch`;
- atualizações de configuração validadas;
- ações Docker/systemd tipadas adicionais;
- `shell.exec` sandboxed dentro das raízes Scoped.

Shell genérico não é pré-requisito para operação útil.

## Gate 5 — Elevação temporária

Somente se Scoped se provar insuficiente:

- pedidos de elevação;
- aprovação humana out-of-band;
- leases temporários;
- revoke-all;
- ancoragem remota de auditoria;
- feature flag Full.

Somente após esses controles passarem considere `shell.exec_admin`.

## Regra

Cada novo componente precisa de evidência do gate anterior.

Otimize primeiro para:

> O cliente alvo real consegue concluir trabalho valioso, seguro e confiável em um servidor real?
