# Política de releases e versões

Portico MCP segue Semantic Versioning.

## Channels

### Estável

Releases estáveis usam tags como `v0.1.0`, `v0.1.1` e `v0.2.0`. Uma release estável é o canal normal de atualização. `main` não é canal de produção.

### Release candidate

RCs usam tags SemVer de pré-release, como `v0.1.0-rc.1`, e servem para aceitação final.

### Desenvolvimento

`main` é o branch de desenvolvimento e não é o alvo padrão de `scripts/update.sh`.

## RC5

O candidato atual é `0.1.0-rc.5`. RC5 inclui descoberta do teto sem abrir conteúdo, proteção de segredos em raízes delegadas, MCP elicitation nativa, anotações de segurança no catálogo público e promoção manual de release pelo GitHub. Tags RC anteriores são histórico imutável.

Stable `v0.1.0` só é criado após o gate final de aceitação em instalação limpa.

## Publicação

O caminho preferido é **GitHub Actions → release → Run workflow** em `main`. O workflow valida código, testes e contrato de instalação antes de criar a tag exata, publicar imagens multi-arquitetura e criar a GitHub Release.

## Artefatos

Imagens Gateway/Broker para `linux/amd64` e `linux/arm64`, bundle Docker Compose, checksums SHA-256 e notas da release. A CI de segurança produz relatórios de vulnerabilidade e SBOM CycloneDX.

## Atualização

Por padrão, `scripts/update.sh` escolhe a tag SemVer estável mais recente de `origin`, recusa non-fast-forward e preserva backup após sucesso ou rollback.

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~
