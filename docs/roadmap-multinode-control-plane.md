# Roadmap — Portico Multi-Node Control Plane

**Status:** planejado  
**Decisão registrada:** 2026-10-06  
**Objetivo:** evoluir o Portico de ponte segura entre ChatGPT e uma única máquina Linux para um **control plane multi-node**, no qual o ChatGPT Web pode atuar como cérebro principal do ecossistema computacional do operador.

## 1. Decisão arquitetural

A direção planejada é:

~~~text
ChatGPT Web / ChatGPT Plus
        |
        | MCP
        v
Portico Control Plane
        |
        +-------------------+-------------------+-------------------+
        |                   |                   |                   |
        v                   v                   v                   v
     SRV-IA              VPSs Linux         cockpit-i7         futuros nós
     Linux               Linux              Windows            Linux/Windows
        |                   |                   |                   |
   Portico Node         Portico Node       Portico Node        Portico Node
        |                   |                   |                   |
 files / shell          Docker / n8n       files / apps       serviços locais
 RAG / bancos           sites / bancos     automação          bancos / RAG
 GPU / jobs             serviços           ferramentas        hardware
~~~

O ChatGPT é o **orquestrador cognitivo principal**. O Portico permanece como a camada de identidade, autorização, roteamento, auditoria e execução segura. Cada máquina continua dona de seus dados, serviços e políticas locais.

## 2. Evidência já obtida

Em 2026-10-06, no ambiente real do operador usando **ChatGPT Plus**, o MCP do Portico expôs ferramentas de leitura e escrita e foi validado com uma operação descartável de:

1. escrita de arquivo;
2. leitura do mesmo arquivo;
3. remoção do arquivo.

A prova foi executada dentro de uma raiz já delegada pelo Broker e respeitou o modo `scoped`.

Essa evidência demonstra que, para o ambiente atual do operador, o ChatGPT Plus consegue executar ações de escrita via Portico. Isso **não deve ser convertido em promessa genérica de compatibilidade para toda conta Plus**, pois disponibilidade de recursos do ChatGPT pode variar por conta, rollout e produto.

## 3. Princípios

### 3.1 Um MCP, vários nós

Evitar um MCP separado por computador.

O catálogo MCP deve permanecer estável e receber um identificador de nó, por exemplo:

~~~text
node.list()
node.info(node_id)

file.read(node_id, path)
file.write(node_id, path, content)

shell.exec(node_id, cwd, command)

docker.list(node_id)
docker.action(node_id, name, action)

service.status(node_id, name)
service.restart(node_id, name)

rag.search(node_id, collection, query)

db.query(node_id, database, query)

gpu.job(node_id, workload)
~~~

Novos computadores devem ser registrados no control plane sem exigir recriação do catálogo MCP apresentado ao ChatGPT.

### 3.2 O Portico continua sendo a fronteira de segurança

O LLM nunca decide autoridade.

Cada ação permanece sujeita a:

~~~text
subject
+ node_id
+ canonical tool
+ canonical resource
+ action
+ node policy
+ grant/lease when required
~~~

O nó de destino deve reautorizar a operação localmente. O control plane não pode transformar uma autorização global em acesso irrestrito ao host.

### 3.3 Dados e RAG permanecem próximos da máquina

RAGs, bancos vetoriais, bancos relacionais, arquivos e índices podem permanecer fisicamente no nó.

Exemplo:

~~~text
ChatGPT
   -> rag.search(node="srv-ia", collection="hidrotech", query="...")
   -> Portico
   -> SRV-IA
   -> Qdrant / pgvector / arquivos locais
   -> somente resultado relevante retorna ao ChatGPT
~~~

O mesmo princípio vale para consultas a bancos, logs, métricas e arquivos.

### 3.4 Serviços residentes continuam locais

O ChatGPT não substitui processos permanentes do sistema.

Continuam locais:

- watchdogs;
- cron/systemd timers;
- backups;
- filas;
- indexação de RAG;
- monitoramento;
- sincronização;
- coleta de métricas;
- serviços de negócio;
- jobs duráveis.

O ChatGPT entra quando há necessidade de raciocínio, decisão, investigação, planejamento ou execução orientada por linguagem.

### 3.5 LLM local passa a ser opcional

O Portico não deve depender de um LLM local para funcionar.

Modelos locais podem permanecer como:

- fallback offline;
- processamento privado;
- tarefas repetitivas de grande volume;
- redução de custo em workloads específicos;
- redundância;
- agentes especialistas locais.

O cérebro principal pode ser o ChatGPT Web quando disponível.

## 4. Componentes planejados

### 4.1 Portico Control Plane

Responsabilidades:

- registro de nós;
- identidade dos nós;
- descoberta de capacidades;
- roteamento de chamadas;
- estado online/offline;
- política global mínima;
- auditoria agregada;
- nomes estáveis de nós;
- versionamento de capabilities;
- revogação de nós;
- health/status;
- encaminhamento seguro para o Broker local.

O control plane não deve possuir autoridade implícita sobre todos os nós.

### 4.2 Portico Node

Agente instalado em cada máquina.

Responsabilidades:

- autenticar o control plane;
- anunciar capabilities;
- aplicar política local;
- executar operações;
- manter auditoria local;
- proteger segredos;
- isolar shell/jobs;
- operar filesystem;
- integrar serviços nativos;
- manter jobs duráveis mesmo se a conexão MCP cair.

### 4.3 Linux Node

Primeiro alvo porque o Broker atual já cobre Linux.

Capacidades reaproveitáveis:

- filesystem;
- shell/jobs;
- systemd;
- Docker/Compose;
- processos;
- rede;
- pacotes;
- usuários/grupos;
- firewall;
- auditoria;
- política.

O SRV-IA deve ser o primeiro nó adicional real além da VPS atual.

### 4.4 Windows Node

Criar implementação própria, sem depender de gambiarras frágeis.

Capacidades iniciais:

- filesystem;
- PowerShell/Process;
- serviços Windows;
- processos;
- aplicações permitidas;
- rede;
- jobs;
- auditoria;
- política local;
- integração opcional com WSL quando útil, sem tratar WSL como substituto do Node Windows.

O `cockpit-i7` será o alvo inicial de validação.

## 5. Tool catalog planejado

O catálogo deve privilegiar ferramentas tipadas e estáveis.

### Nodes

- node.list
- node.info
- node.health
- node.capabilities

### Filesystem

- file.list
- file.stat
- file.read
- file.write
- file.patch
- file.copy
- file.move
- file.remove

### Shell/jobs

- shell.exec
- job.start
- job.status
- job.tail
- job.cancel

### Serviços

- service.list
- service.status
- service.logs
- service.start
- service.stop
- service.restart

### Containers

- docker.list
- docker.inspect
- docker.logs
- docker.action
- compose.validate
- compose.pull
- compose.up
- compose.down

### Dados

- db.list
- db.query
- db.schema

### RAG

- rag.collections
- rag.search
- rag.ingest_status
- rag.reindex

### Compute

- gpu.info
- gpu.job
- compute.job

As ferramentas só aparecem como disponíveis quando o produto suporta a capability; a política do nó continua decidindo se o operador autorizou seu uso.

## 6. Roteamento

Chamadas multi-node devem usar `node_id` explícito.

Exemplo:

~~~text
shell.exec(
  node_id="srv-ia",
  cwd="/opt/projeto",
  command="git status"
)
~~~

O control plane resolve o nó e encaminha a solicitação. O Broker/Node do destino repete a autorização local antes de executar.

Nenhum nó pode aceitar uma operação apenas porque o control plane a encaminhou.

## 7. Identidade e confiança entre nós

Planejar:

- identidade criptográfica por nó;
- bootstrap explícito;
- rotação de credenciais;
- revogação;
- pinning/registro do nó;
- heartbeat autenticado;
- canal cifrado;
- proteção contra node spoofing;
- separação entre identidade humana e identidade de máquina.

## 8. Auditoria

A auditoria multi-node deve permitir responder:

- quem pediu;
- qual ChatGPT/MCP subject;
- qual nó;
- qual ferramenta;
- qual recurso;
- qual ação;
- qual política autorizou;
- qual grant foi usado;
- resultado;
- timestamp;
- operation_id;
- cadeia de integridade.

Cada nó mantém evidência local e o control plane pode manter um índice agregado.

## 9. Fases de implementação

### Gate A — preservar v0.1

Antes da expansão multi-node:

- manter a instalação Linux atual estável;
- não quebrar o caminho ChatGPT -> Gateway -> Broker -> VPS;
- manter testes e auditoria existentes.

### Gate B — SRV-IA como segundo nó Linux

Objetivo: provar duas máquinas reais sob um único control plane.

Critérios:

- registrar VPS atual e SRV-IA;
- `node.list` retorna ambos;
- leitura autorizada em ambos;
- escrita autorizada em ambos;
- shell/job em ambos;
- políticas independentes;
- logs/auditoria identificam corretamente o nó;
- indisponibilidade de um nó não derruba o outro.

### Gate C — catálogo multi-node estável

- adicionar `node_id` às ferramentas;
- garantir compatibilidade do catálogo;
- evitar ferramenta por máquina;
- validar roteamento;
- validar deny-by-default em nó errado;
- validar revogação.

### Gate D — RAG e banco no SRV-IA

- integrar Qdrant/pgvector ou solução escolhida;
- `rag.search`;
- consultas tipadas a banco;
- limites de saída;
- proteção de segredos;
- auditoria.

### Gate E — cockpit-i7 / Windows Node

- agente Windows nativo;
- filesystem;
- PowerShell/jobs;
- processos;
- serviços;
- política;
- auditoria;
- conexão ao control plane;
- operação real via ChatGPT.

### Gate F — compute/GPU

- descoberta de GPUs;
- submissão de jobs;
- filas;
- limites de recursos;
- timeout;
- logs;
- cancelamento;
- integração com workloads locais.

### Gate G — endurecimento

- rotação/revogação de identidade de nós;
- testes adversariais multi-node;
- perda/reconexão de nó;
- replay;
- conflito de operation_id;
- autorização cruzada;
- comprometimento de um nó sem propagação automática aos demais;
- backup/restore do control plane;
- recuperação de auditoria.

## 10. Critério de sucesso

A evolução estará comprovada quando o operador puder, em uma única conversa no ChatGPT:

1. listar os nós autorizados;
2. selecionar implicitamente ou explicitamente o nó adequado;
3. consultar RAG/dados locais;
4. ler e editar arquivos;
5. executar jobs;
6. operar serviços e containers autorizados;
7. usar recursos de compute;
8. atravessar Linux e Windows sem trocar de MCP;
9. manter políticas independentes por máquina;
10. auditar todas as ações.

## 11. Escopo de produto

Esta evolução transforma o Portico de uma ponte ChatGPT -> VPS em um **control plane seguro para IAs operarem múltiplos computadores e servidores do proprietário**.

O objetivo de longo prazo é permitir que um mesmo cliente de IA use um conjunto heterogêneo de máquinas como infraestrutura operacional, mantendo:

- autoridade no lado do proprietário;
- política server-side;
- dados localizados quando possível;
- ferramentas tipadas;
- auditoria;
- isolamento entre nós;
- independência do modelo de IA.

O Portico deve continuar compatível com outros clientes MCP. ChatGPT é o cliente prioritário do operador, não uma dependência arquitetural exclusiva.
