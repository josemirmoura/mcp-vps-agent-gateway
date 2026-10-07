# Conector de nó do Portico Cloud

Status: primeira implementação pública de interoperabilidade do nó com o Cloud.

O conector Cloud é opcional. O Portico Community self-hosted continua funcionando sem ele.

Ele cria um canal HTTPS de saída entre um nó Linux autorizado e o control plane do Portico Cloud, mantendo o Broker local como fronteira de autorização da máquina.

## Modelo de autoridade

```text
tarefa do Portico Cloud
      |
      | lease por HTTPS de saída
      v
portico-cloud-node
      |
      | wire.Request tipado
      | subject = principal local configurado
      v
Broker local
      |
      | política / grants / aprovações locais
      v
operação no host
```

O usuário humano que originou a tarefa no Cloud é informação de proveniência. Ele não recebe autoridade local automaticamente.

O conector usa `PORTICO_CLOUD_BROKER_SUBJECT` como principal perante o Broker. No overlay Docker Compose, o padrão acompanha `VPS_AGENT_SUBJECT`.

Uma tarefa do Cloud não consegue transformar uma negação local do Broker em autorização.

## Identidade do nó

No primeiro enrollment o conector:

1. gera localmente um par de chaves Ed25519;
2. persiste a chave privada em arquivo com modo 0600;
3. envia ao Cloud somente a chave pública;
4. troca o token de enrollment de uso único;
5. salva a identidade de nó/workspace devolvida pelo Cloud;
6. remove o token de bootstrap do estado persistente.

O estado pendente é salvo antes da troca pela rede. Assim, se o Cloud confirmar o enrollment e a resposta se perder, a repetição usa a mesma chave e pode ser reconciliada com segurança.

A chave privada não sai do nó.

## Enrollment de uso único

Defina a origem da API Cloud e envie o token pela entrada padrão, evitando gravá-lo no histórico do shell:

~~~bash
export PORTICO_CLOUD_URL="https://<api-do-portico-cloud>"
read -rsp "Token de enrollment: " PORTICO_TOKEN
printf '%s' "$PORTICO_TOKEN" |   docker compose -f compose.yaml -f compose.cloud.yaml   run --rm -T cloud-node --enroll-only --token-stdin
unset PORTICO_TOKEN
~~~

O volume nomeado `cloud-node-state` preserva a identidade já cadastrada.

Depois, inicie o conector sem token de enrollment:

~~~bash
docker compose -f compose.yaml -f compose.cloud.yaml up -d cloud-node
~~~

## Protocolo em execução

O conector:

- envia heartbeats assinados;
- busca uma tarefa durável por vez;
- assina cada requisição de nó com a chave Ed25519;
- usa nonce e proteção contra replay no Cloud;
- converte a tarefa em uma requisição tipada do Broker;
- usa o ID da tarefa como `invocation_id` local;
- renova o lease enquanto a operação do Broker estiver ativa;
- envia a resposta limitada do Broker como resultado;
- pode repetir a conclusão com segurança graças à idempotência do protocolo Cloud.

Mapeamento:

| Tarefa Cloud | Requisição do Broker |
| --- | --- |
| principal local configurado | `subject` |
| `operation` | `tool` |
| `resource` | `resource` |
| `action` | `action` |
| ID da tarefa | `id` e `invocation_id` |
| `grant_id` | `grant_id` |
| `input` | `args` |

O conector nunca transporta o token administrativo do Broker pelo Cloud.

## Rede

O nó inicia conexões HTTPS de saída. Ambientes comuns atrás de NAT/firewall não precisam abrir uma nova porta administrativa de entrada.

Em produção, a URL do Cloud exige HTTPS. HTTP simples é aceito somente em loopback para desenvolvimento.

## Configuração

| Variável | Padrão | Função |
| --- | --- | --- |
| `PORTICO_CLOUD_URL` | obrigatória | origem da API Cloud |
| `PORTICO_CLOUD_NODE_NAME` | hostname | nome exibido/cadastrado |
| `PORTICO_CLOUD_STATE` | `/var/lib/portico-cloud-node/state.json` | identidade local |
| `PORTICO_CLOUD_BROKER_SOCKET` | `/run/vps-agent/broker.sock` | socket local do Broker |
| `PORTICO_CLOUD_BROKER_SUBJECT` | principal local do Portico | identidade autorizada no Broker |
| `PORTICO_CLOUD_HEARTBEAT_INTERVAL` | `30s` | frequência do heartbeat |
| `PORTICO_CLOUD_POLL_INTERVAL` | `2s` | intervalo ocioso de busca de tarefas |
| `PORTICO_CLOUD_RENEW_INTERVAL` | `10s` | frequência de renovação de lease |
| `PORTICO_CLOUD_BROKER_TIMEOUT` | `15m` | tempo máximo da chamada local |

Somente no primeiro enrollment também são aceitos:

- `--token-stdin`;
- `PORTICO_CLOUD_ENROLLMENT_TOKEN_FILE`, interpretado como nome de arquivo dentro de `/run/secrets`;
- `PORTICO_CLOUD_ENROLLMENT_TOKEN`, como alternativa menos recomendada.

## Limite atual

Este repositório público implementa o lado do nó, o transporte assinado e a entrega ao Broker local.

Cadastro de clientes, cobrança, criação de tarefas no Cloud, entitlements comerciais e interface de frota pertencem ao produto gerenciado Portico Cloud.
