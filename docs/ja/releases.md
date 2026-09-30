# Release / version policy

Portico MCP は Semantic Versioning に従います。

## Channels

### Stable

Stable release は `v0.1.0`, `v0.1.1`, `v0.2.0` のような tag を使い、通常の update channel です。`main` は production ではありません。

### Release candidate

RC は `v0.1.0-rc.1` のような SemVer pre-release tag を使い final acceptance に利用します。

### Development

`main` は development branch で `scripts/update.sh` の default target ではありません。

## RC5

Current candidate は `0.1.0-rc.5`。Ceiling discovery without content access、delegated root の secret protection、native MCP elicitation、public catalog safety annotations、GitHub-native manual promotion を追加します。過去 RC tag は immutable history です。

Stable `v0.1.0` は clean-install final acceptance gate 成功後にだけ作成します。

## Publishing

推奨は `main` の **GitHub Actions → release → Run workflow**。Source、test、installation contract を検証してから exact tag、multi-arch images、GitHub Release を作ります。

## Artefacts

Gateway/Broker images for `linux/amd64` and `linux/arm64`, Docker Compose bundle, SHA-256 checksums, release notes。Security CI は report と CycloneDX SBOM を生成します。

## Update

`scripts/update.sh` は既定で `origin` の最新 stable tag を選び、non-fast-forward を拒否し backup を保持します。

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~
