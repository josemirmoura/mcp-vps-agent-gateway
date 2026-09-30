# Hardening de segurança

Estas regras definem a postura além da arquitetura base.

## 1. Autorização pertence ao Broker

Gateway não é confiável para autorizar trabalho privilegiado.

Cada operação liga:

~~~text
subject
+ canonical tool
+ resource
+ action
+ policy
+ grant quando exigido
~~~

Nenhuma autoridade ambiente é herdada de chamada ou etapa anterior da conversa.

## 2. Expansão de permissão: pedido in-band, aprovação fora

Trabalho rotineiro usa Scoped.

Cliente MCP pode pedir nova raiz dentro do teto físico. O request não concede nada. Broker armazena pending approval tipado com subject, root, access profile e lifetime. Apenas fronteira humana separada ativa.

Delegações dinâmicas são estado do Broker: não editam `policy.yaml`, não aumentam teto e não habilitam ações ausentes da policy estática. Revogação pode ser in-band por reduzir autoridade. Expiração é server-side e jobs de shell não podem ultrapassar delegação temporária.

Anotações MCP e confirmações do host são sinais UX, não fronteira de segurança.

Com MCP elicitation, root e protected-file grants usam UI nativa. Gateway devolve request multi-round-trip com estado opaco; modelo não recebe self-approval tool nem token utilizável. Broker vincula decisão a request, subject, recurso, perfil e expiração. Sem elicitation, pending fica para fallback do operador.

### Elevação administrativa

Se habilitada no futuro:

- MCP pede; não aprova;
- humano usa fluxo autenticado separado;
- step-up auth e preferencialmente MFA/passkey;
- nonce one-time;
- rate limit/cooldown/coalescing;
- URLs expiram;
- GET params não codificam decisão.

Full só após critérios R5.

## 3. Full é capability-based

Exemplos:

~~~text
shell.admin
filesystem.read:any
filesystem.write:any
docker.admin
systemd.admin
~~~

`network.unrestricted` é separado e nunca implícito.

## 4. Replay safety

Tools state-changing são:

### replay-safe
- identidade de idempotência da infraestrutura;
- mesma identidade + mesma request retorna resultado armazenado;
- mesma identidade + request diferente conflita.

### non-replay-safe
- nunca retry cego;
- action IDs;
- resource locks/state;
- confirmação/autorização adequada ao risco.

LLM não é responsável pela key.

## 5. SQLite

Somente Broker abre DB privilegiado. Gateway e rotas humanas usam APIs estreitas. Transações curtas; nenhum comando externo dentro de transação.

## 6. Filesystem

Nunca autorizar com prefix string. Preferir `openat2`; fallback directory-FD seguro com `openat/fstatat/O_NOFOLLOW`. Se não houver implementação segura, writes privilegiados falham fechados.

## 7. Kernel capabilities

Broker health reporta capacidades. Sem Landlock, mantenha systemd/cgroup e reporte degradação. Sem `openat2`, use fallback documentado ou fail closed. Nunca fallback inseguro silencioso.

## 8. Confiança em tools/resultados

Tool output, logs, web content e downstream MCP são dados não confiáveis.

Nunca podem:

- mutar policy;
- criar/estender grants;
- mudar trust tier;
- registrar novo MCP server;
- expor secrets;
- bypassar network policy.

Downstreams configurados out-of-band; mudanças materiais de tool são fingerprinted/reviewed.

## 9. Segredos

Arquivos root-owned fora do Git e systemd credentials. Gateway não lê plaintext. Sem `secret.read_plaintext` genérico. Redigir credenciais de logs/output quando prático.

## 10. Auditoria cresce com privilégio

R1/R2: audit local estruturado.
R3/R4: ordering/integrity durável + retention testada.
R5/Full: sequence monotônica, hash chain, checkpoint/signature/autenticado, anchoring remoto e detecção de gaps/quebras.

Não chame logs locais root de tamper-proof.

## 11. Testes negativos obrigatórios

Testar conforme maturidade:

- path traversal/symlink escape negados;
- service/stack não autorizado negado;
- subject errado negado;
- grant stale/revogado negado;
- retry replay-safe executa uma vez;
- non-replay-safe não é repetido cegamente;
- tool result malicioso não concede capability;
- URL downstream arbitrária rejeitada;
- Gateway não abre Docker socket/SQLite privilegiado;
- ausência de kernel features não causa fallback inseguro;
- owner stale de lock não libera/commita após novo fencing token;
- crash pós-efeito/pré-DONE não duplica efeito.

Antes de R5 também: elevation spam rate-limited, self-approval impossível, Full sem network não tem egress irrestrito, tampering detectável e revoke-all funciona.
