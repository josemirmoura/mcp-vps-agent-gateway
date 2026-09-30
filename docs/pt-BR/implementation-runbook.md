# Runbook de implementação

> Caminho histórico de construção. O runtime Scoped amplo, estado/jobs duráveis, OAuth integrado e código opcional de elevação já existem. Use este documento para entender a sequência de implementação, não como status atual. O status atual está em [project-status.md](project-status.md); trabalho de produtização não deve reabrir gates concluídos sem evidência concreta.

A arquitetura canônica está em `docs/architecture.md` e a sequência de gates em `docs/mvp-first.md`.

## 0. Regras básicas

Antes de codar:

1. leia `docs/README.md`;
2. complete Gate -1;
3. complete Gate 0A;
4. não construa além do gate atual;
5. use Go nos dois binários de referência;
6. mantenha Gateway sem privilégios e Broker local-only.

## 1. Layout do repositório

~~~text
cmd/
  vps-agent-gateway/
  vps-agent-broker/
internal/
  gateway/
  broker/
  shared/
policy/
deploy/
tests/
~~~

Runtime:

~~~text
/etc/vps-agent/       configuração/metadados de segredos
/run/vps-agent/       broker.sock
/var/lib/vps-agent/   state/jobs/audit
~~~

## 2. Gate 0B

Construa apenas o Gateway com SDK Go oficial e Streamable HTTP.

~~~text
system.info
file.read_test
file.write_test
~~~

Restrinja a `/tmp/vps-agent-poc/`. Exija usuário non-root, sem Docker socket/SQLite/shell genérico, schemas, limites de saída, testes de confinamento e MCP Inspector.

Use uma interface interna Executor para depois trocar o executor local por BrokerClient sem reescrever handlers.

## 3. Gate 1 — Broker

Adicione o Broker e IPC via Unix Domain Socket:

~~~text
/run/vps-agent/       root:vps-agent 0750
/run/vps-agent/broker.sock root:vps-agent 0660
~~~

Cheque peer credentials quando prático.

Primeiras operações: `ping`, `system.info`, `service.status`, `service.restart`, limitadas a uma unidade não crítica explicitamente configurada. Autorização deny-by-default.

Auditoria Gate 1 pode ser journald estruturado ou JSON append-oriented. Registre action ID, subject, tool/resource canônicos, decisão, resultado/exit e duração. Nunca segredos brutos.

## 4. Gate 2 — Piloto Scoped

Parser/validador de policy no Broker. Campos/capabilities desconhecidos falham fechados. Defina um stack real e adicione apenas tools necessárias, como file.read, service status/restart, docker logs/action e job.status.

Docker continua Broker-only e tipado. Rode por vários dias e registre capabilities faltantes reais.

## 5. Gate 3 — Estado durável

Só agora adicione SQLite se necessário. Apenas Broker abre `/var/lib/vps-agent/state.db`. Use WAL, transações curtas, busy timeout/backoff, locks lógicos com deadline e fencing token monotônico.

Estado sugerido: jobs, operation journal/idempotência, resource locks, metadata de auditoria e depois approvals/grants.

### Journal de operação

Antes de side effect protegido contra replay:

1. inserir/claim `PENDING`;
2. commit curto;
3. executar efeito externo;
4. marcar `DONE` e persistir resultado.

Crash após efeito e antes de DONE deve retornar `reconcile-required` ou reconciliar especificamente, nunca repetir cegamente.

Idempotência liga subject + tool canônica + hash normalizado + identidade de invocação. Argumentos iguais não provam retry.

## 6. Jobs

~~~text
job.start
job.status
job.tail
job.cancel
~~~

Quando apropriado, cada job roda em unidade/cgroup systemd própria. Persista owner, policy/grant, resource/action, unit/pid, criação/deadline, status e exit code. Perder conexão MCP não perde o estado do job.

## 7. Segredos

Prefira arquivos root-owned fora do repo e systemd credentials. Broker resolve referências. Não existe tool de leitura plaintext. Teste stdout/stderr/audit contra exposição de segredos conhecidos.

## 8. Gate 4 — Writes amplos

Implemente resolução segura e writes atômicos. Prefira `openat2`; fallback seguro ou fail closed.

Mudança de configuração:

~~~text
backup/write temporário
 -> validador nativo
 -> replace atômico
 -> reload/restart
 -> healthcheck
 -> rollback quando seguro
~~~

Exemplos: `nginx -t`, `docker compose config`, validação systemd.

`shell.exec` somente se typed tools forem insuficientes, em transient unit/cgroup com cwd roots, timeout, MemoryMax, TasksMax, output limit, cancellation, ambiente filtrado e network policy. Landlock é defense-in-depth.

## 9. Auth/Distribuição ChatGPT

O caminho suportado está implementado via OAuth self-hosted integrado. Siga [chatgpt-integration.md](chatgpt-integration.md) e [authentication.md](authentication.md). Gateway valida token ativo, issuer, audience/resource, expiração, subject e scopes por introspecção privada; Broker revalida subject e policy.

## 10. Gate 5 — Elevação

Existe machinery opcional e acceptance-tested, mas isso não é claim Full/R5. Full permanece off e Scoped é o caminho suportado.

Quando necessário: request de elevação, aprovação humana out-of-band, nonce one-time, capabilities temporárias explícitas, expiry, revoke-all, rede separada e auditoria reforçada. Broker valida identidade/assertion e nonce. Só depois considerar `shell.exec_admin`.

## 11. Semântica de grants/jobs

Default:

~~~text
job_deadline <= grant_expiry
~~~

Ao expirar grant: bloquear novas mutações, terminar graciosamente, forçar após grace bounded, persistir estado final e auditar. Job Scoped comum não precisa grant elevado. Completion grant especial exige aprovação explícita.

## 12. Recovery

Broker restart: reconciliar jobs persistidos com units/processes e adotar/terminar conforme policy.

SQLite corrompido: fail closed, invalidar approvals, tratar grants como revogados, preservar DB, restaurar backup verificado, reconciliar jobs e reabilitar só após integridade.

Comando de emergência fora da superfície IA:

~~~text
vps-agent revoke-all
~~~

Revoga grants elevados e bloqueia novos starts.

## 13. Hardening systemd

Gateway: User=vps-agent, NoNewPrivileges, ProtectSystem=strict, ProtectHome, PrivateTmp, ReadWritePaths mínimos, sem Docker socket.

Broker: root-owned, sem listener TCP, socket Unix apenas, API mínima, paths fixos, journald e restart-on-failure. Revise com `systemd-analyze security`.

## 14. Aceitação de produção

Scoped: gates iniciais, negative auth tests, filesystem safe, recovery, segredos, caminho do cliente, workloads sobrevivendo à remoção do control plane.

Full: além disso approval/expiry/revoke-all, separação de rede, remote audit checkpoint e contenção/recovery do admin shell.

## 15. Rollback

O Gateway/Broker é control plane opcional. Parar/remover sua rota não pode parar workloads existentes. Essa propriedade é obrigatória.
