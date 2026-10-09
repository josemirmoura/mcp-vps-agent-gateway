# Contrato de instalação

Status: **decisão normativa do projeto**.

Este documento delimita os procedimentos temporários de desenvolvimento/validação e a experiência de instalação oferecida ao usuário. A implementação, os testes, os READMEs, os tutoriais e a revisão de release precisam respeitar este contrato.

Versão em inglês: [Installation contract](installation-contract.md).

## Experiência de instalação suportada

A instalação é:

- iniciada e acompanhada pelo terminal;
- baseada em Docker Compose;
- executada por comandos e scripts transparentes, passíveis de inspeção;
- reproduzível na VPS do próprio operador;
- utilizável sem assistência dos desenvolvedores do projeto;
- *native-first*: usa mecanismos oficiais, Docker, MCP e OAuth/OIDC padronizados antes de adaptações próprias.

O operador instala e administra o pacote diretamente na VPS. O ChatGPT só é conectado depois que a instalação local e a autenticação pública estão prontas.

## Segredos e acesso remoto

Os tutoriais **nunca** exigem a entrega dos seguintes dados à IA ou à equipe do projeto:

- senha da VPS;
- chave SSH privada;
- acesso SSH ou administrativo remoto irrestrito;
- credenciais de root;
- credenciais administrativas do provedor da nuvem;
- saídas de terminal que contenham segredos;
- qualquer outro segredo que não seja estritamente necessário ao serviço.

Os segredos indispensáveis à operação são informados **localmente na VPS** ou na interface/API nativa do serviço correspondente. Exemplo: a senha da identidade OAuth dedicada é digitada no configurador local e utilizada posteriormente na tela de login do provedor de identidade, sem passar por mensagens do chat.

Durante o desenvolvimento, um agente sem acesso à máquina pode excepcionalmente pedir que o operador execute diagnósticos e compartilhe resultados previamente sanitizados. Isso é uma limitação da sessão de desenvolvimento, **não é requisito do produto** e não deve entrar no tutorial de instalação.

## Responsabilidade da automação

Quando uma etapa determinística pode ser automatizada com segurança, o pacote deve executá-la. Conforme aplicável, isso inclui:

- detecção de dependências e mensagens com correções práticas;
- análise das portas disponíveis e do proxy de borda;
- validação do Docker Compose;
- inicialização e checagem de saúde dos contêineres;
- configuração e validação de HTTPS;
- inicialização e descoberta OAuth/OIDC;
- checagem do endpoint MCP;
- testes de autenticação que falham de maneira segura (*fail closed*);
- diagnósticos com remoção de dados sensíveis;
- orientação de recuperação quando o software não pode escolher pelo operador.

Comandos investigativos usados uma única vez durante desenvolvimento não viram etapas obrigatórias de instalação.

## Ações manuais deliberadas

Uma ação manual é aceitável somente quando a plataforma exige decisão do operador ou interação com navegador/interface. Para cada ação, a documentação explica:

1. exatamente o que fazer;
2. por que a automação não pode tomar essa decisão com segurança;
3. como confirmar o resultado.

Exemplos: escolher o teto físico do filesystem, apontar DNS quando o instalador não controla o provedor, informar localmente a senha da conta OAuth dedicada e cadastrar o servidor MCP no ChatGPT Web.

## Critério de conclusão

Contêineres iniciados e verificação local são etapas intermediárias, **não** a conclusão.

~~~text
VPS configurada
 -> MCP público com HTTPS válido
 -> autenticação OAuth/OIDC funcionando
 -> conexão no ChatGPT Web configurada
 -> chamada MCP real feita pelo ChatGPT
 -> sujeito/política/auditoria confirmados
 -> INSTALLATION COMPLETE
~~~

O script `scripts/connect-chatgpt.sh` integra a jornada oficial. O processo só se encerra como concluído quando o Broker registra a chamada autenticada esperada e a cadeia de auditoria permanece válida.

Uma conexão somente de leitura pode servir para diagnóstico, mas não comprova uma instalação com ferramentas de escrita quando esse for o objetivo.

## Procedimentos exclusivos de desenvolvimento

Estes procedimentos pertencem à engenharia e ao aceite, salvo promoção explícita para uma funcionalidade pública segura:

- pedir ao proprietário que execute comandos investigativos e cole diagnósticos;
- probes temporários do GitHub Actions;
- runners autogerenciados específicos do projeto;
- máquinas efêmeras de CI;
- hostnames e IPs de VPS de desenvolvimento, caminhos de checkout e nomes de branch;
- ramificações temporárias de implementação;
- comandos avulsos curl/openssl usados apenas para depurar falhas;
- acesso SSH por desenvolvedores ou manutenção remota improvisada.

Essas informações pertencem aos registros de issues/PRs e às notas técnicas de desenvolvimento, não aos READMEs, tutoriais, site público ou fluxo suportado de conexão ao ChatGPT.

## Revisão final antes de release

Antes de declarar um candidato pronto:

1. revisar `README.md` e `README.pt-BR.md`;
2. revisar `docs/installer-flow.md` / `docs/installer-flow.pt-BR.md` e os guias de integração do ChatGPT;
3. conferir as páginas públicas EN/PT-BR;
4. retirar hostnames, IPs, nomes de branches e diagnósticos temporários da experiência de instalação;
5. verificar que não há instruções de compartilhamento de senhas VPS ou chaves privadas;
6. explicar motivo e validação de cada ação manual indispensável;
7. executar `python3 scripts/check-installation-contract.py` na CI;
8. realizar a instalação integral em uma VPS limpa e suportada;
9. finalizar com chamada MCP real no ChatGPT Web, sujeito autorizado e prova na auditoria do Broker.

## Regra de engenharia

**NATIVE FIRST.** Priorizar APIs oficiais, configuração suportada, Docker/Compose, MCP, OAuth/OIDC e mecanismos nativos. Código próprio só quando não houver caminho nativo suficiente, mantendo-o mínimo, centralizado, documentado e reversível.

## Estado de aprovação de clientes

O uso de MCP Apps/elicitation depende de recursos anunciados e efetivamente testados pelo cliente. Nunca presumir que uma janela integrada autentica o proprietário. A autoridade pertence ao Broker. Integrações HTTPS/CLI/SSH são alternativas operacionais quando realmente presentes e configuradas no runtime utilizado. Não anunciar instalação estável nem validação móvel antes do aceite específico.

**Versão em inglês:** [Installation contract](installation-contract.md).
