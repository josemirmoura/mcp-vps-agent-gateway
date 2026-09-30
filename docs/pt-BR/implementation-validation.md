# Validação executável da implementação

Verificado: 2026-09-30.

Este documento registra evidências de runners Ubuntu limpos hospedados pelo GitHub. São máquinas descartáveis, portanto execuções bem-sucedidas provam comportamento reproduzível em Linux novo sem tocar produção.

## Evidência atual

### Write/read real MCP → Gateway → Broker → Linux

Workflow:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36283767952

Resultado:

- Ubuntu 24.04;
- Gateway non-root `vps-agent`;
- Broker root com socket Unix protegido por grupo;
- MCP Streamable HTTP local;
- `file.write_test` gravou arquivo real;
- `file.read_test` retornou os mesmos 56 bytes;
- host confirmou arquivo e SHA-256;
- cadeia de auditoria válida com 2 eventos.

~~~text
MCP VPS Agent Gateway passou pela VPS efemera do GitHub.
sha256:
572890f111b50050dd1c405e724a0cd2b31dee2df32bd52c30d3db6b1853eea1
~~~

### Prova negativa /etc/shadow

Workflow:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36283831850

`file.read_test` tentou `/etc/shadow`. O workflow só passa se houver denial/error e auditoria continuar válida. Resultado: **PASS**.

### Restart systemd real

Workflow:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36283884675

~~~text
service.status
service.restart
service.status
PID antes: 3124
PID depois: 3173
ActiveState: active
~~~

Audit com 3 eventos e cadeia válida. Resultado: **PASS**.

## Aceitação E2E do pacote Docker

Workflow:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36324349224

**PASS** em runner Ubuntu 24.04 limpo, usando Dockerfile + compose.yaml reais e superfície MCP pública.

Provado na mesma máquina descartável:

- health de Broker/Gateway;
- workflow Scoped de filesystem: mkdir/write/read/hash/patch/copy/move/chmod 0777/stat/list/delete/recursive delete;
- `/etc/shadow` negado;
- `shell.exec` cria arquivo real na raiz delegada e não lê `/etc/shadow`;
- status/restart systemd host;
- inspect/restart Docker host;
- Compose validate/up/down host;
- diagnósticos de disco, memória, processos, listeners, pacotes, usuários e grupos;
- integridade final da cadeia de auditoria.

Broker Docker usa namespaces do host para operações nativas. Shell Scoped usa namespace de mount systemd partindo de root vazio read-only e bindando apenas runtime/toolchain e raízes autorizadas.

Docker é packaging, não fronteira de autorização. Broker privilegiado é código host-root confiável; policy server-side é a autoridade efetiva.

## Evidência CI

A CI valida:

- `go vet ./...`;
- `go test -race -count=1 ./...`;
- builds nativos e linux/amd64 + arm64;
- simulação adversarial da arquitetura;
- smoke Docker contra daemon real;
- inspect sanitizado sem secrets de environment;
- verificação/hardening de units systemd;
- smoke de transient jobs;
- `govulncheck`, gosec, gitleaks;
- fuzzing bounded de filesystem/policy;
- limites de body e concorrência;
- prova negativa `SO_PEERCRED`;
- testes adversariais hardlink/symlink/rename-race;
- Trivy e SBOM CycloneDX.

O commit mais recente deve estar verde antes do merge.

## Prova da fronteira de privilégio

~~~text
/run/vps-agent              root:vps-agent 0750
/run/vps-agent/broker.sock  root:vps-agent 0660
/var/lib/vps-agent          root:vps-agent 0750
~~~

Policy, secrets de ambiente do Broker e SQLite são root-only. O workflow falha se a conta do Gateway puder ler `/etc/vps-agent/broker.env`, `/etc/vps-agent/policy.yaml` ou `/var/lib/vps-agent/state.db`.

## ChatGPT Web real + OAuth integrado

Em 2026-09-28 o caminho público suportado foi validado em VPS real:

~~~text
ChatGPT Web
 -> HTTPS MCP protected resource
 -> OAuth/OIDC discovery
 -> DCR + PKCE
 -> autenticação do operador dedicado
 -> validação Gateway
 -> autorização Broker subject + policy
 -> system.info
 -> registro de auditoria
~~~

O fluxo observou o evento autenticado do subject esperado e alcançou `CHATGPT WEB CONNECTION VERIFIED` / `INSTALLATION COMPLETE`. Hostname, subject e credenciais específicos não são publicados.

Isso fecha Gate 0A do caminho integrado. RC5 adiciona discovery-only do teto, proteção aninhada de segredos, elicitation MCP nativa, safety annotations, abuse limits e race tests ampliados. Ainda é necessária aceitação clean-install final pelo proprietário antes de `v0.1.0`; isso não prova confiabilidade prolongada ou maturidade Full/R5.

## O que prova

~~~text
MCP client
 -> Streamable HTTP
 -> non-root Gateway
 -> protected Unix socket
 -> privileged Broker
 -> server-side policy
 -> real Linux operation
 -> Broker audit
~~~

## O que ainda não prova

- confiabilidade longa de produção;
- compatibilidade formal ampla;
- remote audit anchoring;
- Full/admin shell em produção;
- recovery prolongado sob workloads reais.

Full/admin existem no código mas ficam desligados pela policy padrão.

## Prova interativa reproduzível

Pedido em `proof/request.json`.

Tipos:

- `file_write_read`;
- `file_read_denied`;
- `service_restart`.

Alterar a request em PR dispara `.github/workflows/ephemeral-proof.yml`, que envia evidence.json, audit-status, journals, permissões, estado systemd e conteúdo/hash aplicável.
