# Arquitetura

## Objetivo

Dar a um cliente de IA acesso operacional útil a uma VPS Linux mantendo a VPS, e não o modelo, no controle de autorização e privilégio.

> **O LLM nunca é a fronteira de segurança.**

## Runtime canônico

O design de referência possui dois processos próprios do projeto, ambos em Go:

~~~text
ChatGPT / cliente MCP
        |
        | MCP Streamable HTTP
        v
+-----------------------------+
| vps-agent-gateway           |
| sem privilégios             |
| MCP + auth + schemas        |
| tools canônicas             |
| shaping de respostas        |
+-------------+---------------+
              |
              | Unix Domain Socket
              v
+-------------+---------------+
| vps-agent-broker            |
| privilegiado, local-only    |
| policy autoritativa         |
| SQLite + locks + jobs       |
| segredos + auditoria        |
| files + systemd + Docker    |
| execução sandboxed          |
+-------------+---------------+
              |
              v
     Linux / systemd / Docker
~~~

Proxy reverso existente ou túnel privado suportado é infraestrutura de ingresso, não terceiro serviço do projeto.

## Por que Go nos dois processos

O SDK Go oficial do MCP é Tier 1 e suporta MCP 2026-07-28. Uma linguagem reduz custo de packaging, dependências e manutenção sem remover a fronteira de privilégio entre processos.

Outra linguagem pode ser suportada, mas a referência não adiciona diversidade de runtime sem evidência de benefício.

## Gateway

O Gateway roda sem root. Pode expor MCP via Streamable HTTP, validar OAuth/OIDC, validar schemas, normalizar tools/resources, fazer preflight não autoritativo e chamar o Broker via socket Unix.

Não pode:

- rodar como root;
- acessar Docker socket;
- abrir o SQLite privilegiado;
- ler armazenamento plaintext de segredos;
- ser o ponto autoritativo de autorização;
- aprovar sua própria elevação.

## Broker

O Broker é a fronteira privilegiada de segurança.

Cada chamada privilegiada é reautorizada contra:

~~~text
subject
+ tool canônica
+ resource canônico
+ action
+ policy atual
+ grant/lease/job quando exigido
~~~

O Broker possui policy autoritativa, SQLite, idempotência, locks, jobs duráveis, operações seguras de filesystem, Docker/systemd, sandboxing, resolução de segredos e auditoria.

O Gateway é tratado como deputy não confiável.

## Modelo de capacidade

O produto entrega um catálogo amplo. Autoridade efetiva vem da policy, não de binários diferentes.

O operador pode autorizar uma raiz, várias raízes/recursos ou o host inteiro. Para multi-projeto, teto físico e raízes lógicas são separados: o teto é limite máximo, não grant de leitura.

No perfil Standard, uma operação discovery-only pode mostrar apenas nomes de diretórios imediatamente abaixo do teto para que o cliente peça o projeto correto sem abri-lo. Raízes extras são delegações dinâmicas do Broker ligadas ao subject, perfil `read`, `work` ou `compose`, e expiração opcional.

Criar pending request não é autorização. Em clientes com MCP elicitation, o Gateway retorna o pedido e o cliente renderiza sua confirmação humana nativa. O Broker valida subject, request pendente e token/nonce one-time antes de ativar. Tools de confirmação não aparecem no catálogo visível ao modelo. Clientes sem elicitation falham fechados para fallback de operador. Revogação entra em vigor pelo estado do Broker sem restart.

Arquivos com segredos são uma fronteira aninhada. `.env` exige segundo grant temporário, exact-path e aprovado. Templates comuns continuam conteúdo normal. Broker aplica isso em filesystem e shell sandbox para impedir que uma delegação ampla vire atalho de leitura de segredos.

Filesystem é apenas uma dimensão. systemd, Docker, shell, rede e ações administrativas têm escopos independentes.

Veja [product-model.md](product-model.md).

## Policy não é serviço separado

O Policy Engine vive dentro do Broker. Policies são deny-by-default.

Presets:

- Controlled — inspeção + writes estreitos;
- Scoped — autonomia dentro de perímetro explícito;
- Full — bundle opcional e temporário.

Full é desabilitado por padrão.

## Aprovação é fluxo

Trabalho rotineiro em Scoped não deve interromper o humano.

Elevação temporária pode criar request, mas aprovação ocorre fora do canal de ação MCP. Para grants de root e arquivos protegidos, prefere-se MCP elicitation nativa. Broker continua dono da decisão, vinculando aprovação ao subject e token one-time, impedindo replay e autoaprovação pelo modelo. Rota admin separada continua como fallback e para MFA/passkeys.

## Full

Full não é um bit root. Expande para capabilities como:

~~~text
shell.admin
filesystem.read:any
filesystem.write:any
docker.admin
systemd.admin
~~~

Rede irrestrita não é implícita e exige aprovação separada. Full é temporário, revogável e indisponível até gates de MVP/recovery.

## Transporte

Use MCP Streamable HTTP em endpoint HTTPS estável. No MCP 2026-07-28 o core é stateless. Não criar WebSocket próprio ou sessão de transporte customizada. Jobs e leases usam handles explícitos. O SDK oficial cuida de negociação/backward compatibility.

## Filesystem

Nunca autorize paths por prefixo string.

Preferência:

- `openat2` com flags restritivas.

Fallback:

- walk seguro com directory FDs + `openat/fstatat/O_NOFOLLOW`; ou
- fail closed em writes privilegiados.

Nunca cair silenciosamente para autorização por string.

## Isolamento de processos

Baseline:

- transient units systemd;
- cgroups;
- `NoNewPrivileges`;
- `PrivateTmp`;
- restrições de filesystem;
- `MemoryMax`;
- `TasksMax`;
- deadline;
- limite de saída;
- cancelamento.

Landlock é defesa em profundidade quando disponível; ausência é reportada, não remove controles baseline.

## Docker

Gateway nunca recebe `/var/run/docker.sock`. Operações Docker são typed Broker operations autorizadas contra stacks/actions canônicos.

## Jobs

Modelo interno autoritativo:

~~~text
job.start
job.status
job.tail
job.cancel
~~~

Jobs não dependem de HTTP permanecer aberto. MCP Tasks pode ser adaptado no futuro sem mudar a semântica do Broker.

## Estado

SQLite inicial, aberto somente pelo Broker. Transações curtas; nunca manter transação enquanto comando/deploy/migration/Docker externo roda.

Estado inclui approvals, leases, jobs, idempotência, locks e metadata de auditoria.

### Fencing de locks

Use pelo menos:

~~~text
resource
owner/action_id
monotonic fencing_token
expires_at
~~~

Nova aquisição após expiração incrementa token. Owner antigo não pode liberar/commit contra token mais novo, evitando ABA race.

### Janela de crash da idempotência

Não faça efeito externo e só depois crie registro. Use journal:

~~~text
PENDING -> external effect -> DONE
~~~

Commit `PENDING` antes do efeito. Se crash ocorrer entre efeito e `DONE`, retry entra em reconciliação em vez de repetir cegamente.

## Segredos

Preferir mecanismos Linux nativos:

- arquivos root-owned fora do repo;
- systemd credentials.

Broker resolve referências e injeta valores apenas no processo alvo. Gateway e tools visíveis ao modelo não expõem plaintext.

## Auditoria por maturidade

Gate 1: audit local estruturado.
Scoped production: sequência/integridade duráveis.
Antes de Full production: hash chain tamper-evident, checkpoint/forward remoto e recovery testado.

## Confiança em tools downstream

Se MCPs downstream forem agregados:

- upstreams allowlisted out-of-band;
- nomes determinísticos/namespaced;
- mudanças de schema/description fingerprinted e revisadas;
- resultados tratados como dados não confiáveis;
- resultados nunca alteram policy, criam leases, registram servidores ou expõem segredos.

## Não objetivos da primeira versão

Não é gateway MCP universal, control plane multi-tenant, scheduler distribuído, serviço root-shell genérico, projeto Kubernetes, substituto de SSH nem tentativa de suportar todo cliente MCP.

Primeiro objetivo:

> Provar com segurança que o cliente de IA alvo consegue executar uma operação pequena, útil e auditável em uma VPS.
