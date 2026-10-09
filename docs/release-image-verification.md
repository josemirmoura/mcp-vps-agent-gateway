# Verificação das imagens assinadas de uma release Community

Este gate verifica, em modo somente leitura, as três imagens do Pórtico no GHCR. Não instala, executa ou altera contêineres. É independente do gate do pacote de código-fonte.

## Pré-requisitos

- Uma release genuína já publicada pelo workflow `.github/workflows/release.yml`.
- `crane` (go-containerregistry) e `cosign` (Sigstore), executados em terminal local confiável.
- Acesso de leitura ao GHCR; não colocar credenciais em argumentos ou logs.

## Comando

```bash
python3 scripts/verify-release-images.py --tag v0.1.0-rc.7
```

Substitua a tag por um candidato realmente publicado. Não suponha que RC7 já existe ou foi assinado.

O script percorre obrigatoriamente Gateway, Broker e conector Cloud:

1. Resolve o digest de `ghcr.io/josemirmoura/<imagem>:<tag>` com `crane digest`.
2. Verifica **o digest imutável** (`imagem@sha256:...`) com `cosign verify`, nunca apenas a tag móvel.
3. Restringe o certificado OIDC ao workflow de release deste repositório e ao emissor `https://token.actions.githubusercontent.com`.
4. Exige tag de assinatura exatamente igual à solicitada ou `refs/heads/main` no caso de execução manual do workflow.
5. Só imprime `RELEASE IMAGES VERIFIED` depois que todas as três imagens forem verificadas, e informa seus digest refs para auditoria.

**Limites:** esse gate não prova imutabilidade do ponteiro de tag, correspondência com o commit remoto, SBOM/proveniência, integridade do ambiente de execução, aceitação operacional nem assinatura do pacote de código-fonte. Os controles devem ser somados, não confundidos.

## Testes sem acesso à produção

```bash
python3 -m unittest discover -s tests/scripts -p 'test_release_image_verifier.py' -v
```

Esses testes simulam `crane` e `cosign` para validar fluxo e tratamento de falhas. Não provam criptografia real. O gate de release exige repetir com imagens autênticas assinadas após a publicação.
