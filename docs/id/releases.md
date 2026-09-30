# Kebijakan release dan version

Portico MCP mengikuti Semantic Versioning.

## Channels

### Stable

Release stable memakai tag seperti `v0.1.0`, `v0.1.1`, dan `v0.2.0` serta menjadi channel update normal. `main` bukan production.

### Release candidate

RC memakai tag SemVer pre-release seperti `v0.1.0-rc.1` untuk final acceptance.

### Development

`main` adalah branch development dan bukan target default `scripts/update.sh`.

## RC5

Candidate saat ini `0.1.0-rc.5`. RC5 menambah ceiling discovery tanpa membuka konten, secret protection di delegated root, native MCP elicitation, safety annotation pada public catalog, dan GitHub-native manual promotion. Tag RC sebelumnya menjadi immutable history.

Stable `v0.1.0` hanya dibuat setelah final clean-install acceptance gate berhasil.

## Publishing

Jalur pilihan adalah **GitHub Actions → release → Run workflow** di `main`. Workflow memvalidasi source, test, dan installation contract sebelum membuat tag tepat, publish multi-arch image, dan GitHub Release.

## Artefak

Image Gateway/Broker `linux/amd64` dan `linux/arm64`, Docker Compose bundle, SHA-256 checksum, release notes. Security CI menghasilkan report dan CycloneDX SBOM.

## Update

`scripts/update.sh` secara default memilih stable tag terbaru dari `origin`, menolak non-fast-forward, dan mempertahankan backup.

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~
