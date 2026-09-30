# Política de suporte

Portico MCP é um projeto open source.

## Linha de release suportada

Durante o período inicial de produtização:

- a release estável `0.x` mais recente é a linha pública suportada;
- a release candidate atual é suportada para aceitação/testes;
- `main` é desenvolvimento e não é canal estável de suporte.

Correções de segurança podem exigir atualização para o patch mais recente.

## Como obter ajuda

Use uma issue do GitHub para bugs reproduzíveis, falhas de instalação e problemas de documentação que não contenham segredos.

Antes de abrir uma issue, gere um bundle de diagnóstico sanitizado quando for prático:

~~~bash
bash scripts/diagnose.sh bundle
~~~

Nunca anexe `.env`, credenciais brutas, chaves SSH privadas ou material secreto sem redação.

## Problemas de segurança

Não publique detalhes de exploração em uma issue comum. Siga `SECURITY.md`.

## Nível de serviço

Não há SLA garantido de tempo de resposta ou disponibilidade para o projeto open source.

Compromissos de suporte de uma futura distribuição comercial, se existirem, devem ser documentados separadamente e não podem ser inferidos deste repositório.
