# Contrato de instalação

Status: **decisão normativa do projeto**.

Este documento define a fronteira entre procedimentos temporários de desenvolvimento/validação e a experiência de instalação suportada para o usuário final. Implementação, testes, README, tutoriais e revisão de release devem respeitá-lo.

## Experiência suportada

A instalação oficial é:

- orientada primeiro ao terminal;
- Docker Compose primeiro;
- baseada em comandos e scripts transparentes e inspecionáveis;
- reproduzível na VPS do próprio usuário;
- utilizável sem assistência dos desenvolvedores;
- **NATIVO PRIMEIRO**: mecanismos oficiais da plataforma, Docker, MCP e OAuth/OIDC baseado em padrões antes de customizações.

O usuário instala e opera o pacote diretamente na VPS. O ChatGPT só é conectado depois que a instalação server-side e a autenticação pública estiverem prontas.

## Segredos e acesso remoto

O tutorial suportado **nunca deve instruir o usuário a fornecer ao ChatGPT ou a um mantenedor**:

- senha da VPS;
- chave SSH privada;
- SSH irrestrito ou outro acesso administrativo remoto;
- credenciais root;
- credenciais administrativas do provedor cloud;
- saída de terminal contendo segredos;
- qualquer segredo que não seja estritamente necessário para o serviço.

Segredos necessários ao serviço são digitados localmente na VPS ou por UI/API nativa correspondente. A senha dedicada do operador OAuth, por exemplo, é digitada localmente e não é fornecida ao ChatGPT.

Sessões de desenvolvimento podem exigir temporariamente que um humano execute comandos e devolva diagnósticos sanitizados quando o agente não tiver canal de execução. Isso é limitação do ambiente de desenvolvimento, **não requisito do produto**, e não deve entrar nas instruções do usuário.

## Responsabilidade de automação

Toda tarefa determinística que possa ser razoavelmente automatizada deve ser absorvida pelo pacote, incluindo:

- detecção de dependências e erros acionáveis;
- detecção de portas e proxy de borda;
- validação do Compose;
- inicialização e health checks;
- configuração e validação HTTPS;
- bootstrap e discovery OAuth/OIDC;
- testes do endpoint MCP;
- testes fail-closed;
- coleta de diagnóstico com redação de segredos;
- instruções claras de recuperação quando a automação não puder decidir com segurança.

Comandos pontuais de investigação usados durante desenvolvimento não viram passos obrigatórios do tutorial.

## Ações manuais deliberadas

Ações manuais só são aceitáveis quando a plataforma exige decisão do operador ou ação em navegador/UI. A documentação deve dizer:

1. o que o usuário precisa fazer;
2. por que isso não pode ser automatizado com segurança;
3. como validar o resultado.

Exemplos: escolher o teto de autoridade da VPS, configurar DNS fora do controle do pacote, digitar localmente a senha do operador OAuth e conectar o app MCP no ChatGPT Web.

## Gate de conclusão

Inicialização dos contêineres e verificação local são gates intermediários.

~~~text
VPS configurada
 -> MCP público com HTTPS válido
 -> OAuth/OIDC funcionando
 -> conexão no ChatGPT Web configurada
 -> chamada MCP real do ChatGPT bem-sucedida
 -> subject/policy/auditoria esperados confirmados
 -> INSTALAÇÃO CONCLUÍDA
~~~

`scripts/connect-chatgpt.sh` faz parte do fluxo oficial. A instalação não termina até o Broker observar a chamada autenticada esperada do ChatGPT e a cadeia de auditoria permanecer válida.

## Procedimentos somente de desenvolvimento

Pertencem a desenvolvimento/aceitação, salvo promoção explícita para um recurso suportado:

- pedir ao proprietário comandos ad hoc e saída;
- probes temporários no GitHub Actions;
- runners self-hosted específicos;
- máquinas efêmeras de CI;
- hostnames, IPs, paths ou branches de VPS de teste;
- branches temporários;
- probes curl/openssl de investigação;
- SSH ou shell remoto de desenvolvedor.

Esse material fica em issues/PRs ou notas de desenvolvimento, nunca no tutorial suportado.

## Revisão final de release

Antes de declarar uma release pronta:

1. revisar README e todas as traduções oficiais;
2. revisar `installer-flow.md` e `chatgpt-integration.md` em todos os idiomas oficiais;
3. revisar a landing pública;
4. remover artefatos específicos do desenvolvimento;
5. confirmar que o tutorial não pede credenciais VPS/SSH;
6. confirmar que toda ação manual inevitável tem motivo e etapa de validação;
7. executar os checks de contrato de instalação e cobertura i18n na CI;
8. rodar a instalação completa numa VPS suportada limpa;
9. terminar com chamada MCP real do ChatGPT Web e evidência da auditoria do Broker.

## Regra de engenharia

**NATIVO PRIMEIRO.** Preferir APIs oficiais, configuração suportada, Docker/Compose, MCP, OAuth/OIDC e mecanismos nativos. Customização só quando necessária, e então mínima, centralizada, documentada e reversível.
