# Semântica de runtime e recovery

Comportamentos operacionais que precisam ser explícitos antes de produção.

## 1. Expiração de grant durante jobs longos

Job elevado deriva grant imutável no start.

~~~text
job_deadline <= grant_expiry
~~~

Ao expirar:

1. não aceitar novas mutações privilegiadas;
2. enviar terminação graciosa;
3. forçar após grace period limitado;
4. persistir estado final;
5. auditar.

Jobs Scoped comuns não precisam grant elevado. Se job admin precisar ultrapassar grant interativo, exija completion grant específico com deadline próprio.

## 2. Concorrência SQLite

SQLite single-writer é aceitável inicialmente.

- só Broker abre DB;
- transações curtas;
- nunca segurar transação durante shell/deploy/migration/Docker;
- persistir transição, commit, executar externo, persistir próxima;
- exclusão longa via locks lógicos com deadline;
- WAL;
- busy timeout/backoff.

Migre apenas se contenção medida justificar.

### Fencing de locks

Locks com TTL precisam fencing token monotônico. Job stale nunca libera recurso/commit protegido usando token antigo. Release compara resource + owner/action_id + token.

Evita ABA:

~~~text
A token 1
A expira
B token 2
A volta
A NÃO pode liberar/sobrescrever B
~~~

### Janela de crash da idempotência

~~~text
PENDING -> effect -> DONE
~~~

Crie journal antes do side effect. Crash após efeito e antes de DONE não reexecuta automaticamente; entra em reconciliação ou retorna indeterminate/reconcile-required.

## 3. Idempotência é metadata da infraestrutura

Não dependa do LLM para inventar key estável.

Gateway/integration gera/deriva identidade de retry e liga a subject + tool canônica + request hash normalizado + identidade da invocação. Sem identidade estável, não auto-retry operações que mudam estado.

Argumentos iguais podem ser dois pedidos legítimos.

## 4. Segredos

MVP não tem tools de leitura de segredo. Use arquivos root-owned fora do repo ou systemd credentials. Broker resolve `secret_ref` e injeta apenas no processo alvo.

Gateway não lê storage plaintext; tools não enumeram valores; secrets não entram em audit; redigir stdout/stderr/logs quando prático. Secret managers externos são adapters opcionais.

## 5. Fricção de aprovação

Trabalho rotineiro usa Scoped pré-autorizado. Aprovação out-of-band fica para elevação excepcional, evitando fadiga.

## 6. Recovery

### Broker reinicia durante job

Jobs em units/cgroups gerenciados, todos com deadline. No restart, Broker reconcilia state com processos vivos; jobs órfãos são adotados para monitoramento ou terminados conforme policy.

### SQLite corrompido

Fail closed:

1. parar novos writes;
2. invalidar approvals pendentes;
3. tratar grants elevados como revogados;
4. preservar DB danificado;
5. restaurar backup verificado;
6. reconciliar jobs;
7. reabilitar apenas após checks de integridade.

### Grant emitido erroneamente

Comando fora da superfície IA:

~~~text
vps-agent revoke-all
~~~

Revoga grants, bloqueia novos starts elevados, termina/quarentena jobs afetados conforme emergency policy e gera audit/checkpoint. Se houver suspeita de credencial comprometida, rotacione separadamente.

### Comprometimento da rota Gateway

Parar/remover rota MCP não pode parar workloads das aplicações.

## 7. Backups

Quando existirem, faça backup de policy, SQLite state, audit checkpoints e unit files. Não inclua plaintext secrets sem criptografia intencional.

## 8. Readiness

### Scoped production

Exige Gates 0A/0B e caminho privilegiado tipado, comportamento Scoped provado, recovery de state/jobs, secret path se usado, remoção Gateway/Broker sem parar workloads, limites de compatibilidade/suporte documentados e evidência de confiabilidade proporcional ao claim.

O RC satisfaz gates de implementação/E2E, mas ainda não declara histórico R4 prolongado.

### Full production

Além disso: semantics de expiry de grant/job, revoke-all, approval flow, remote audit checkpoint e separação de elevação de rede testados.
