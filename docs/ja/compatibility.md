# 互換性

## Release Candidate で検証済み

### OS

- clean GitHub runner 上の Ubuntu 24.04 LTS;
- host service/job 機能には systemd を持つ Linux。

### Architecture

- `linux/amd64`: clean CI で runtime acceptance 実施;
- `linux/arm64`: binaries/images は build 済みだが、native arm64 hardware 上の同等 full host acceptance は未実施。

### Runtime

- Docker Engine 24+;
- `docker compose` として使える Docker Compose v2;
- bundled ZITADEL 用に最低 2 GB RAM;
- Git, OpenSSL, Python 3, curl。

2 GB は dependency の機能最低値であり、production sizing 推奨ではありません。

参照:
- https://zitadel.com/docs/self-hosting/deploy/compose
- https://zitadel.com/docs/self-hosting/manage/requirements

## Public ChatGPT path

必要条件:

- operator 管理の DNS A/AAAA;
- compatible existing Traefik または bundled Traefik 経由の public TCP 80/443;
- valid HTTPS certificate;
- Developer Mode と custom MCP app registration を実際に提供する ChatGPT account/workspace。

提供状況は OpenAI が管理します。Portico は ChatGPT 側の permission を拡張できません。

参照:
https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

## Build

- `linux/amd64`
- `linux/arm64`

## 現在サポートを約束しないもの

- non-Linux host;
- production VPS としての Docker Desktop;
- host service/job 用 systemd がない Linux;
- Kubernetes;
- Podman Compose;
- native Windows/macOS;
- production Full/R5。

他の Linux distro は動く可能性がありますが、明示的な acceptance evidence がない限り未検証です。

## Resource sizing

2 GB は identity stack の依存最低値です。Password hashing や既存 Docker workload、Audit/log retention により実際の要件は増えます。`scripts/diagnose.sh health` と host metrics で測定してください。
