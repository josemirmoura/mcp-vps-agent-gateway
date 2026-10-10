# Autorizações adaptativas: validação, ativação e rollback

Estado: candidato derivado da PR [#88](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/88). O novo fluxo precisa de CI no SHA exato e aceitação em staging. Este documento é um procedimento de release; nenhuma implantação é executada por sua leitura.

O [contrato de autorização](adaptive-operator-approvals.md), a [matriz de compatibilidade](approval-compatibility.md) e o [fallback HTTPS/SSH](operator-approval-fallback.md) definem o comportamento esperado. Produção somente após autorização explícita do proprietário para a versão e configuração propostas.

## Pré-condições de release

Registrar SHA completo da versão anterior e candidata, digests de imagens, referências do portal/scripts, versão de política, ID estável do nó e modelo Compose efetivamente usado. O portal serve arquivos do checkout por bind mount; seu código também faz parte da versão de release.

- Todos os workflows exigidos devem estar verdes no mesmo SHA candidato: Go/autorização, scanner, portal, fallback, transporte MCP, autenticação, pacote e aceitação operacional. Um teste focado ou CI de commit anterior não conclui esse gate.
- Aprovação, negação, ausência de grant quando negado, replay, concorrência, expiração, revogação, cancelamento e troca de identidade/máquina devem ter evidência auditável no Broker.
- A matriz deve separar evidência documental de fornecedor, prova de SDK/harness e prova do cliente real. Não declarar funcionamento em ChatGPT ou outro produto apenas porque anuncia MCP Apps/elicitation.
- Preservar portal HTTPS externo, CLI/SSH, política local, dados, workloads e rotas MCP/OAuth. Grants continuam sujeitos à política e à reautorização do Broker.
- Planejar backup consistente e rollback antes de qualquer mudança autorizada. Guardar segredos e backups somente em ambiente protegido do operador.

## Revisão isolada do candidato

Depois de obter o SHA revisado, preparar um checkout independente. Exemplo de revisão de código; não inicia serviços:

```bash
PORTICO_APPROVAL_CANDIDATE_SHA=COLE_O_SHA_COMPLETO_APROVADO
git fetch origin feat/adaptive-multiclient-approvals
git cat-file -e "$PORTICO_APPROVAL_CANDIDATE_SHA^{commit}"
git worktree add --detach ../portico-approval-review "$PORTICO_APPROVAL_CANDIDATE_SHA"
cd ../portico-approval-review
git status --short
python3 -m unittest discover -s web/operator-approval -p 'test_*.py' -v
python3 -m unittest discover -s tests/scripts -p 'test_operator*.py' -v
node --test web/operator-approval/test_app.mjs
go test ./...
```

Usar as versões fixadas pelo repositório/workflows. Se o ambiente não permite toolchain Go, sockets Unix ou browser, executar a mesma verificação em runner adequado e registrar a limitação; não eliminar testes para produzir um resultado verde.

O preparador `scripts/operator-portal-stage.sh` exige **SHA completo e imutável** da revisão já conferida pelo integrador; não aceita uma branch antiga ou um candidato implícito. Por padrão consulta a branch de integração atual, mas só prossegue se o HEAD obtido coincidir exatamente com `PORTICO_STAGE_CANDIDATE_SHA`. Se o Bloco 5 atualizar staging, solicite a nova SHA verificada antes de executar. Para uma tag já aprovada, indique `PORTICO_STAGE_CANDIDATE_REF=refs/tags/<tag>`. Exemplo de preparo (sem iniciar contêineres):

```bash
PORTICO_STAGE_CANDIDATE_SHA=COLE_O_SHA_EXATO_DE_40_CARACTERES \
PORTICO_STAGE_CANDIDATE_REF=refs/heads/integration/community-stable-block5-20261009 \
bash scripts/operator-portal-stage.sh
```

O script aborta antes de criar novo worktree/backup se a referência não coincidir, não for descendente da instalação atual ou estiver inválida. Ainda é necessário validar a proveniência do SHA, os workflows e a configuração de isolamento antes de qualquer implantação. `STAGING READY, NO DEPLOY` significa somente preparação e verificação local, não aceite de cliente real.

Para execução de serviços, preferir VM/nó descartável. Usar banco, raízes, sockets, credenciais, projeto Compose, portas, DNS/TLS e OAuth de teste próprios. Um worktree sozinho não isola serviços: o modelo possui nome padrão de projeto, Broker privilegiado e recursos do host. Nunca executar um `docker compose up` genérico no checkout de revisão supondo que ele é independente da instalação ativa.

## Configuração do ambiente de teste

Apps vem desabilitado por padrão. Manter inicialmente:

```dotenv
VPS_AGENT_MCP_APPS=0
PORTICO_OPERATOR_FRAME_ANCESTORS=
```

Conferir o contrato já existente de portal: `PORTICO_OPERATOR_PUBLIC_ORIGIN` deve ser uma origem HTTPS sem caminho e `VPS_AGENT_OPERATOR_PORTAL_URL` deve apontar para `<origem>/operator`. A senha scrypt e a credencial restrita do socket são provisionadas no terminal confiável. O utilitário `scripts/operator-portal-credentials.py` é para primeira configuração; ele recusa sobrescrever configuração existente e não é ferramenta de rotação. Nenhuma senha/token deve ser enviado a chat, elicitation, recurso UI ou ferramenta MCP.

No Compose padrão, `PORTICO_OPERATOR_ID` alimenta a identidade configurada do Broker e do portal; `VPS_AGENT_INSTANCE_ID` alimenta o ID do nó em Broker/Gateway e `PORTICO_OPERATOR_NODE_ID` no portal. Em execução separada, manter esses vínculos consistentes. O sujeito MCP permitido pelo Broker continua sendo configuração distinta da identidade do aprovador. Exigir um ID de instalação estável e não vazio antes do release.

Conferir o modelo efetivo com `docker compose ... config --quiet`, usando os mesmos arquivos `-f`, perfil e projeto previstos na proposta de staging. Evitar imprimir o modelo expandido, que pode conter segredos. O portal precisa somente do socket restrito `operator-run`, permanece não-root e não recebe Docker nem estado administrativo do Broker.

## Aceitação dos quatro canais

| Caso | Ação e evidência esperada |
|---|---|
| HTTPS externo | Abrir link de pedido sem login, observar bloqueio; autenticar, revisar sujeito/máquina/recurso/perfil/TTL, negar e provar ausência de grant; repetir com aprovação de pasta read curta e conferir auditoria |
| Sessão reutilizada | Segundo pedido temporário de pasta read usa sessão válida sem nova senha e ainda exige confirmação explícita; sessão expirada/logout/troca de configuração exige login |
| Verificação por pedido | Arquivo protegido e work/compose exigem senha fresca vinculada ao nonce; prova de outro pedido/sessão, expirada ou repetida falha; negação funciona sem essa redigitação |
| MCP Apps | Cliente anuncia MIME correto; handshake e `frameDomains` permitem a origem; iframe autenticado mostra escopo real; operador aprova e nega; consultar Broker comprova ambos |
| Framing/cookies/bridge bloqueados | Link externo e orientação SSH continuam disponíveis; nenhum bloqueio ou timeout concede permissão |
| Elicitation URL/form | `accept`, `decline`, `cancel` e continuação adulterada não criam grants; somente decisão autenticada no portal/SSH altera autoridade |
| Somente texto/link | Capability ausente/inválida entrega HTTPS quando elegível; operações posteriores continuam negadas enquanto pedido está pendente |
| Headless/SSH | Sem portal, CLI lista pedido, mostra escopo integral e exige frase exata em TTY; falta de terminal/credencial, EOF ou expiração não aprova |
| Pedido fora da bridge | Permanência, todo o teto físico e elevação administrativa não podem ser aprovados pelo serviço web; continuar pelo fluxo administrativo local apropriado |
| Concorrência/replay/rede | Duas janelas/duplo clique geram uma só decisão e concessão; nonce usado/expirado/snapshot trocado é recusado; perda de resposta exige consulta antes de repetir |
| Isolamento e ciclo de vida | Outro sujeito/nó/sessão não consulta ou decide pedido indevido; cancelamento, expiração e revogação encerram autoridade corretamente |

Em cliente real, registrar versão/data, conta ou tipo de workspace sem dados sensíveis, capabilities recebidas, protocolo negociado, modalidade selecionada, origens ancestrais observadas, browser/OS e resultado da auditoria. Emulação mobile ajuda a validar layout; aceitação em Android/iOS reais exige evidência separada.

Para habilitar o ensaio Apps, configurar `PORTICO_OPERATOR_FRAME_ANCESTORS` somente com origens HTTPS exatas observadas e aprovadas e então `VPS_AGENT_MCP_APPS=1`. Nunca usar wildcard ou adivinhar domínios sandbox de fornecedores. Todos os ancestrais efetivos precisam ser compatíveis com a política; o MIME anunciado pelo cliente não prova que o navegador aceita iframe/cookies.

O portal externo mantém `X-Frame-Options: DENY` e `frame-ancestors 'none'`; somente `/operator/embed` recebe a allowlist. A sessão embutida usa cookie separado Secure/HttpOnly/SameSite=None/Partitioned, com até 15 minutos. Se o browser não oferecer isolamento útil, usar o link externo. Não prometer SSO entre clientes ou máquinas.

## Promoção autorizada

A proposta concreta para o proprietário deve conter SHA/digests, configuração não secreta, resultados de CI/cliente, serviços que precisam ser recriados, backup e rollback. Confirmar que o checkout ativo não mudou desde a revisão. Usar o modelo Compose real da instalação, incluindo overlays OAuth/edge necessários; não resetar OAuth, remover volumes nem recriar toda a stack por conveniência.

Após autorização, aplicar somente a promoção acordada e verificar saúde, cadeia de auditoria, HTTPS, MCP/OAuth e SSH. Inicialmente manter Apps desabilitado se a prova do cliente ainda estiver pendente. Ativar framing/Apps somente para clientes e origens comprovados. Reinício seletivo deve fazer parte da janela aprovada, pois pode interromper sessões/conexões.

## Rollback

**Desativação da modalidade embutida:** voltar à configuração abaixo e recriar seletivamente Gateway/serviço operador segundo o modelo efetivo e a janela aprovada:

```dotenv
VPS_AGENT_MCP_APPS=0
PORTICO_OPERATOR_FRAME_ANCESTORS=
```

Isso desliga anúncio de Apps e framing da Central. Portal externo e SSH permanecem; o Broker corrigido continua autoritativo. Reiniciar o serviço operador invalida sessões e nonces em memória. Alterar modalidade/reiniciar não revoga automaticamente grants já emitidos; revogação exige ação própria no Broker.

**Retorno de código:** restaurar imagens/digests revisados do Broker/Gateway e arquivos compatíveis de portal/scripts/Compose. Como o portal usa bind mounts, reverter apenas as imagens não restaura sua UI/API. Preservar configuração de identidade, política e estado. Evitar voltar a uma versão que permita confirmar pedido nativo sem prova administrativa independente. Preferir o Broker corrigido com Apps desligado; se não houver retorno de código seguro, desabilitar o serviço/bridge web e operar por SSH até correção.

Não há migração de esquema nesta alteração; normalmente o banco vivo deve permanecer. Grants anteriores continuam sendo aplicados/revogados pelo Broker. Status de pedido antigo de arquivo protegido sem correlação segura a grant exige inspeção do operador; erro/estado desconhecido nunca equivale a aprovação.

**Recuperação de banco:** somente como procedimento separado e supervisionado. Restaurar snapshot pode recuperar grants revogados e perder decisões/auditoria posteriores. Parar escritores, confirmar a necessidade, preservar estado atual, restaurar backup consistente, validar auditoria e revisar autoridade antes de reiniciar. Essa ação requer autorização explícita própria.

Depois de qualquer rollback, comprovar saúde, auditoria, pedido negado sem grant, ausência de autoaprovação por IA, escopo e revogação, portal externo, SSH e rotas MCP/OAuth. Registrar SHA/digests anterior/final, motivo e resultado. Manter evidência da versão falha sem divulgar credenciais.
