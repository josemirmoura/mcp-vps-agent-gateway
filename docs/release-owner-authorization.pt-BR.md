# Autorização de publicação do Pórtico Community (para iniciantes)

Este roteiro é para **o último momento da preparação de uma versão**, quando o Bloco 5 apresentar uma candidata revisada e todos os critérios estiverem satisfeitos. **Não execute agora.**

## Primeiro: confirme que a versão pode ser publicada

O coordenador da integração deve mostrar:
- a versão exata proposta (`v0.1.0-rc.N` ou, depois, `v0.1.0`);
- a identificação completa de 40 caracteres do código revisado (**SHA**);
- os testes automáticos desse mesmo código e, separadamente, as verificações do protocolo MCP;
- o aceite real do computador/celular e um plano de recuperação;
- a decisão jurídica sobre licença, autores e titularidade;
- as evidências de assinaturas para a candidata já publicada, quando cabíveis.

Se faltar algum desses itens, **não inicie a publicação**. A tela do GitHub não verifica sozinha todas as decisões jurídicas e de aceite.

## Onde clicar quando tudo estiver aprovado

1. Acesse [o projeto Community no GitHub](https://github.com/josemirmoura/mcp-vps-agent-gateway).
2. Clique na aba **Actions** (Ações).
3. Na coluna esquerda, selecione o procedimento chamado **release**.
4. Clique em **Run workflow** (Executar fluxo de trabalho).
5. Em **Use workflow from**, escolha a branch **main**. Nunca uma branch de testes.
6. No campo **tag**, cole o número da versão exata que o Bloco 5 informou, começando com `v`.
7. No campo **approved_sha**, cole os **40 caracteres completos** da identificação do código fornecida na homologação. Não invente nem abrevie.
8. No campo **publish_confirmation**, digite **PUBLICAR** seguido de um espaço e da mesma versão. Exemplo: `PUBLICAR v0.1.0-rc.7` (só se essa for realmente a versão aprovada).
9. Confira os três campos, então clique para executar. A publicação só prosseguirá se você estiver autenticado no GitHub como proprietário, o código ainda for o atual da `main` e as verificações exigidas passarem.

## O que deve acontecer

O GitHub mostrará etapas técnicas de validação, segurança, imagens e assinatura. A versão só aparecerá em **Releases** após a conclusão do procedimento.

**A tela verde do lançamento não substitui o aceite das assinaturas reais.** O Bloco 5 precisa verificar os arquivos e as três imagens geradas usando os programas `scripts/verify-release-source.py` e `scripts/verify-release-images.py`, registrando os resultados.

## Se der erro

- **Não tente publicar com outro número de versão nem usando código diferente.**
- Não exclua etiquetas ou arquivos publicados por conta própria.
- Informe ao Bloco 5 qual etapa do GitHub falhou, usando apenas o endereço público da execução. Ele verificará se foi criada uma etiqueta parcial ou imagem e indicará o procedimento seguro.
- Não envie senha, código de autenticação, chave SSH, token de acesso ou conteúdo do arquivo `.env` pelo chat.

**Este roteiro autoriza somente uma publicação específica no GitHub.** Não libera alterações na sua VPS, no domínio ou nos serviços existentes.
