# Verificação independente da assinatura de uma release Community

**Estado:** procedimento de pré-instalação, sem modificar a VPS. Verifica os artefatos de **código-fonte** de uma release futura criada pelo workflow `release.yml`. Não é evidência de que RC7, ou qualquer release histórica, esteja assinada: versões publicadas antes da adoção do Sigstore podem não possuir bundles.

## Uso

No terminal de uma máquina de validação confiável, com GitHub CLI `gh`, Python 3 e Sigstore `cosign` instalados:

```bash
RELEASE_TAG=v0.1.0-rc.7
mkdir -p "./release-assets/$RELEASE_TAG"
gh release download "$RELEASE_TAG" --repo josemirmoura/mcp-vps-agent-gateway \
  --dir "./release-assets/$RELEASE_TAG" \
  --pattern 'mcp-vps-agent-source-package.tar.gz*'
python3 scripts/verify-release-source.py \
  --directory "./release-assets/$RELEASE_TAG" \
  --tag "$RELEASE_TAG"
```

**Se o download falhar, a release ainda não estiver publicada ou qualquer bundle estiver ausente, pare.** Não contorne o verificador, não aceite somente checksum SHA-256 não assinado e não pressuponha que um workflow de assinatura sintética prove uma release publicada.

O verificador exige:
1. Todos os quatro artefatos, sem links simbólicos: tar.gz, SHA-256, bundle Sigstore do tar.gz e bundle Sigstore do arquivo SHA-256.
2. Certificado OIDC associado ao GitHub Actions workflow `josemirmoura/mcp-vps-agent-gateway/.github/workflows/release.yml` e emissor `https://token.actions.githubusercontent.com`. Não aceita somente assinatura válida por outra identidade.
3. SHA-256 do tar.gz idêntico ao checksum cujo material foi assinado; nome de arquivo único e exatamente esperado.
4. O arquivo `mcp-vps-agent/VERSION`, lido **sem extrair o pacote**, deve corresponder à tag solicitada.
5. Falhar sem aceitar pacotes se `cosign` não estiver presente ou falhar.

**Não verifica:** identidade do commit da tag no repositório remoto, assinatura e digest das três imagens GHCR, SBOM/proveniência, licença ou aceitação operacional com ChatGPT. Esses controles continuam gates separados; ver `docs/releases.md` e `docs/security-release.md`.

## Testes da implementação

`python3 -m unittest discover -s tests/scripts -p 'test_release_source_verifier.py' -v`.

Os testes usam um executável **cosign simulado** para confirmar chamadas e cenários adversos. Ele não prova a criptografia do Sigstore; a criptografia real precisa ser testada em uma release legítima, publicada e baixada. Não executar instalador nem subir contêineres durante estes testes.
