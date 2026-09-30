# Status de produtização — v0.1.0-rc.3

Este documento acompanha o checklist de produtização pública contra o branch atual de release candidate. É evidência/status, não substituto da arquitetura canônica ou documentos de segurança.

## Concluído no branch de produtização

- [x] implementação/status reconciliados com a evidência real de E2E no ChatGPT Web;
- [x] README EN/PT-BR reescrito em torno do ponto de entrada do produto;
- [x] guarda de sanitização do repositório ampliada além da documentação suportada;
- [x] nome canônico do produto, tagline e versão visível;
- [x] VERSION SemVer + CHANGELOG;
- [x] instalação guiada e transparente pelo terminal;
- [x] UX de autoridade Project / Custom / Whole Host;
- [x] resumo da autoridade efetiva antes de iniciar o runtime;
- [x] Whole Host separado de Full;
- [x] rede irrestrita separada de Full;
- [x] OAuth/OIDC integrado preservado e documentado;
- [x] auditoria e observabilidade operacional documentadas separadamente;
- [x] workflows de status/health/log/audit/bundle de diagnóstico;
- [x] redação do bundle de diagnóstico preservada;
- [x] canal de atualização estável por padrão;
- [x] verificação explícita de atualização via `scripts/version.sh --check`;
- [x] backup, validação de migração e rollback automático preservados;
- [x] safe remove e purge preservados;
- [x] landing page produtizada;
- [x] documentação de privacidade e telemetria;
- [x] matriz de compatibilidade/suporte;
- [x] templates de issue e PR;
- [x] validação release tag ↔ VERSION;
- [x] checksum de release;
- [x] SBOM/provenance de contêiner solicitados pelo workflow;
- [x] regressões OAuth para token inválido/inativo/expirado/issuer errado/audience errada;
- [x] rejeição explícita de token OAuth revogado/inativo na introspecção;
- [x] rate limiting na borda OAuth/DCR;
- [x] checker de links da documentação;
- [x] job de aceitação do ciclo de vida do instalador guiado;
- [x] Full/R5 explicitamente excluído de claims de produção.

## Gates de validação deste branch

- [x] PR aberto;
- [x] CI da implementação de referência verde;
- [x] aceitação dos componentes de auth integrada verde;
- [x] aceitação do pacote Docker verde;
- [x] aceitação do lifecycle incluindo instalação guiada verde;
- [x] aceitação efêmera Scoped completa verde;
- [x] prova efêmera de VPS verde;
- [x] aceitação simultânea de instâncias independentes verde;
- [x] cross-review final após evidências de CI.

## Deliberadamente não concluído aqui

- [ ] tag/release estável `v0.1.0`; depende do gate final do proprietário;
- [ ] aceitação final clean-install no ambiente alvo do proprietário;
- [ ] evidência de confiabilidade R4 de longa duração;
- [ ] maturidade de produção Full/R5;
- [ ] chave criptográfica de assinatura de release gerida pelo projeto.

## Metadados da vitrine GitHub

README, Pages, política de release, badges/links e conteúdo do repositório são tratados neste branch.

Descrição, topics e homepage do repositório são metadados do GitHub, não arquivos. Devem ser revisados no passe final de storefront; a superfície conectada atual não expõe mutação desses metadados, portanto nada disso deve ser marcado como concluído sem evidência.
