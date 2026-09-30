# Mapa da documentação

A documentação canônica fica em `docs/`. Esta pasta contém a tradução oficial em Português (Brasil). Se houver divergência com o arquivo em inglês da mesma versão, o inglês continua sendo a referência técnica até a correção da tradução.

## Operador / usuário

1. [quick-start.md](quick-start.md) — caminho mais curto para a instalação guiada.
2. [installation-contract.md](installation-contract.md) — fronteira normativa da instalação suportada.
3. [installer-flow.md](installer-flow.md) — fases completas da instalação.
4. [product-model.md](product-model.md) — Project, Standard, Whole Host e modelo de capacidades.
5. [chatgpt-integration.md](chatgpt-integration.md) — integração com ChatGPT e gate de conclusão.
6. [authentication.md](authentication.md) — OAuth/OIDC integrado.
7. [operations.md](operations.md) — status, saúde, logs, auditoria, atualização e remoção.
8. [troubleshooting.md](troubleshooting.md) — falhas comuns.
9. [faq.md](faq.md) — perguntas frequentes.
10. [compatibility.md](compatibility.md) — ambientes suportados, testados e não testados.
11. [privacy.md](privacy.md)
12. [releases.md](releases.md)
13. [support.md](support.md)
14. [operator-acceptance.md](operator-acceptance.md)

## Desenvolvedor / segurança

- [project-status.md](project-status.md)
- [architecture.md](architecture.md)
- [policy-schema.md](policy-schema.md)
- [threat-model.md](threat-model.md)
- [security-hardening-v2.md](security-hardening-v2.md)
- [security-release.md](security-release.md)
- [runtime-semantics-and-recovery.md](runtime-semantics-and-recovery.md)
- [tool-trust-and-confused-deputy.md](tool-trust-and-confused-deputy.md)
- [transport-and-aggregation.md](transport-and-aggregation.md)
- [implementation-validation.md](implementation-validation.md)
- [simulation-validation.md](simulation-validation.md)
- [mvp-first.md](mvp-first.md)
- [implementation-runbook.md](implementation-runbook.md)
- [build-vs-adopt.md](build-vs-adopt.md)
- [multi-instance.md](multi-instance.md)
- [productization-status.md](productization-status.md)
- [references.md](references.md)

Em caso de conflito, use esta precedência: arquitetura, contrato de instalação, status atual do projeto, schema de policy, hardening de segurança, semântica/recovery do runtime, documentos de apoio e documentos históricos.

**Arquitetura em uma frase:** dois processos Go, um Gateway MCP sem privilégios e um Broker local privilegiado ligados por socket Unix; o Broker controla autorização, estado e execução privilegiada.
