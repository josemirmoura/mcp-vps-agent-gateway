# Modelo de produto: toolbox completo, autoridade limitada

## Decisão central

Distribuir um conjunto completo de capacidades e controlar a autoridade por policy server-side.

Não existem binários separados "limited", "project" ou "full".

~~~text
mesmos binários
+ mesmo catálogo MCP
+ policy diferente
= autoridade efetiva diferente
~~~

O LLM nunca escolhe o escopo.

## Escopo do usuário

O operador decide exatamente quanto da VPS é delegado ao MCP. Exemplos de policy ajudam, mas nunca definem autoridade por si. A configuração autoritativa é a policy explícita escolhida e confirmada.

~~~text
uma pasta:
/opt/my-app

várias:
/opt/app-a
/var/www/site
/srv/data

filesystem inteiro:
/
~~~

O mesmo princípio vale independentemente para systemd, Docker, rede, pacotes, usuários/grupos e recursos administrativos.

## Perfis de conveniência

### Standard

Padrão recomendado para hosts com vários projetos. O operador escolhe um teto físico, padrão `/opt`. Nenhuma raiz de projeto é autorizada na instalação.

Portico pode descobrir apenas nomes das pastas imediatamente abaixo do teto. Conteúdo fica bloqueado até aprovação de uma raiz com `read`, `work` ou `compose`.

Arquivos secretos protegidos continuam como fronteira interna separada.

### Project

Acesso autônomo somente dentro de uma raiz, por exemplo `/opt/my-app`.

Capacidades típicas:

- CRUD de filesystem dentro da raiz;
- shell Scoped com cwd na raiz;
- operações Docker selecionadas;
- unidades systemd selecionadas;
- destinos de rede selecionados.

### Policy avançada/estática

Operadores avançados podem definir várias raízes e grupos diretamente na policy. Continua disponível, mas o perfil Standard guiado prefere delegação runtime em vez de exigir edição YAML inicial.

### Whole host

O operador autoriza explicitamente recursos host-wide. É escolha de policy, não build diferente.

Pode incluir acesso a `/`, systemd amplo, Docker, pacotes, usuários/grupos, firewall/rede e shell administrativo temporário. Capacidades perigosas continuam explícitas e auditáveis.

## Catálogo alvo

### Filesystem

`file.list`, `file.stat`, `file.read`, `file.mkdir`, `file.write`, `file.patch`, `file.copy`, `file.move`, `file.remove`, `file.remove_recursive`, `file.hash`, `file.chmod`, `file.chown`.

As capacidades existem mesmo quando a policy as desabilita. Remoção recursiva e alterações amplas de permissão são capabilities separadas.

### Comandos e jobs

`shell.exec`, `job.start`, `job.status`, `job.tail`, `job.cancel`.

`shell.exec` executa em sandbox imposta pelo servidor. Escopo apenas escrito em YAML não basta para comandos. A sandbox traduz policy em restrições do SO: cwd permitidos, `ProtectSystem`, `ReadWritePaths`/`ReadOnlyPaths`, `ProtectHome`, `PrivateTmp`, cgroups, `MemoryMax`, `TasksMax`, timeout, limites de saída e policy de rede.

### systemd

`service.list`, `service.status`, `service.logs`, `service.start`, `service.stop`, `service.restart`, `service.reload`, `service.enable`, `service.disable`.

Policy limita nomes/padrões canônicos e ações.

### Docker / Compose

`docker.list`, `docker.inspect`, `docker.logs`, `docker.start`, `docker.stop`, `docker.restart`, `compose.config`, `compose.pull`, `compose.up`, `compose.down`.

Prefira operações tipadas. Docker socket nunca vai ao Gateway.

### Diagnóstico

`system.info`, `system.health`, `system.disk`, `system.memory`, `process.list`, `process.inspect`, `network.listen`, `network.check`, `journal.read`.

### Administração

Disponível no produto, normalmente desabilitado até seleção explícita:

- `package.update/install/remove`;
- administração de usuário/grupo;
- firewall;
- chmod/chown além de faixas seguras;
- `shell.exec_admin`.

## Escopo multidimensional

Filesystem é apenas um eixo. A policy controla separadamente raízes, unidades systemd, recursos Docker, cwd de shell, destinos de rede, ações de package manager, usuários/grupos, firewall e capabilities admin temporárias.

Um perfil Project pode ter CRUD total em `/opt/my-app` e zero autoridade sobre nginx, Docker, apt ou Internet.

## chmod e operações destrutivas

`file.chmod`, inclusive 0777, pode existir quando a policy permitir. Perfis padrão não habilitam world-writable silenciosamente.

`file.remove` trata remoção comum; `file.remove_recursive` é destrutiva separada; chmod/chown host-wide é decisão administrativa distinta.

O produto suporta a operação; a policy decide se a instalação atual pode usá-la.

## UX de configuração

Operador ou IA assistente edita a policy declarativa para escolher:

1. raízes/recursos delegados;
2. preset Standard, Project ou Whole Host;
3. raízes de filesystem;
4. capabilities de arquivo;
5. shell e sandbox;
6. systemd;
7. Docker/Compose;
8. rede;
9. administração;
10. approval/elevation;
11. autenticação e exposição MCP.

`config/policy.yaml` é a configuração autoritativa legível por humano/IA. `docker compose config -q` e validação runtime rejeitam configuração inválida. A policy pode ser editada depois sem reinstalar binários.

## Packaging

O produto oficial usa Docker Compose.

~~~text
Gateway container
  non-root
  no /host
  no Docker socket
        |
        | Unix socket
        v
Broker container
  privileged host-control boundary
  host mounted at /host
        |
        v
VPS
~~~

O Broker container não é sandbox de segurança em volta da administração do host; ele é a fronteira privilegiada server-side empacotada em Docker.

Portanto:

- Docker fornece packaging/lifecycle;
- Gateway permanece separado e não privilegiado;
- Broker recebe host root porque implementa ações explicitamente autorizadas;
- Broker não expõe API TCP remota de controle;
- policy decide quais partes/recursos podem ser usados;
- montar `/host` não concede autoridade ao LLM;
- o mesmo pacote serve um projeto, vários recursos ou host inteiro.

Fluxo primário:

~~~bash
bash scripts/install.sh
~~~

É uma orquestração transparente sobre bootstrap, Compose, verificação, OAuth integrado e conexão ChatGPT. Oferece Standard, Project e Whole Host, mostra autoridade efetiva e pode retomar após interrupção.

Os comandos individuais continuam disponíveis. Não existe instalador proprietário paralelo. A fronteira suportada é definida por [installation-contract.md](installation-contract.md).

## Instalação só termina após verificar o ChatGPT

O tutorial ChatGPT Web é etapa anterior, mas não encerra a instalação.

Depois do runtime e policy, deve mostrar:

~~~text
MCP endpoint:
https://<host>/mcp

Authentication:
<configured method>

Effective scope:
<human-readable policy summary>

Next:
Connect this MCP to ChatGPT Web
~~~

A primeira execução deve:

1. verificar endpoint HTTPS;
2. verificar autenticação;
3. executar teste server-side inofensivo;
4. mostrar autoridade efetiva;
5. fornecer tutorial ChatGPT atual;
6. esperar a conexão;
7. exigir chamada inofensiva feita pelo próprio ChatGPT;
8. verificar subject, policy, execução e auditoria;
9. somente então marcar instalação completa.

Como a superfície do ChatGPT muda independentemente, o tutorial é versionado e revalidado em cada release.

Experiência final:

~~~text
clone / release bundle
 -> escolher o que MCP pode controlar
 -> iniciar Gateway + Broker via Compose
 -> validar policy e segurança
 -> bootstrap OAuth + HTTPS
 -> mostrar tutorial ChatGPT
 -> usuário conecta ChatGPT
 -> verificar chamada E2E real
 -> verificar auditoria
 -> instalação completa
~~~
