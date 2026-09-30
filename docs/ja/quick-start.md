# Portico MCP クイックスタート

Portico MCP は透明な Docker Compose と repository scripts を使って terminal からインストールします。

## 要件

- Linux VPS。RC の検証対象は Ubuntu 24.04 LTS;
- Docker Engine 24+ と Docker Compose v2;
- bundled ZITADEL 用に最低 2 GB RAM;
- Git, OpenSSL, Python 3, curl;
- public DNS、TCP 80/443、valid HTTPS;
- Developer Mode と custom MCP app を実際に利用できる ChatGPT account/workspace。

## 開始

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

公式言語:

~~~text
en · pt-BR · es · de · fr · ja · id
~~~

明示選択:

~~~bash
bash scripts/install.sh --lang ja
~~~

## ガイドフロー

~~~text
Prerequisites
 -> Scope
 -> Effective authority
 -> Containers
 -> Local verification
 -> Secure public access
 -> Connect ChatGPT
 -> real audited MCP call
 -> INSTALLATION COMPLETE
~~~

Preflight は host、Docker/Compose、tools、systemd、RAM、edge を確認します。既存 Traefik を安全に識別できれば再利用し、なければ 80/443 が空いている場合に bundled Traefik を使えます。

## 推奨: Standard

~~~text
physical ceiling: /opt
static roots:     none
project access:   approved later
~~~

物理上限は最大境界であり、**/opt 自体を許可しません**。

別の上限:

~~~bash
bash scripts/install.sh --profile custom --scope /srv/apps
~~~

`permissions.discover_scope` は上限直下の folder 名だけを表示します。アクセスは native MCP confirmation で要求します。

~~~text
permissions.request_root_access
 -> native MCP confirmation
 -> Broker activates read / work / compose
~~~

AI は自分の authority expansion を自己承認できません。

### Secrets

`.env` と `.env.*` は許可済み project 内でもロックされます。`.env.example` などの template は読めます。実際の secret には separate temporary `permissions.request_sensitive_access` が必要です。

## Project

~~~bash
bash scripts/install.sh --profile project --scope /opt/my-app
~~~

## Whole Host

~~~bash
bash scripts/install.sh --profile whole-host
~~~

物理上限を `/` にしますが、Full、network、packages、users、firewall を自動有効化しません。

## Shell user

~~~bash
bash scripts/install.sh --run-as deploy
~~~

Scoped job は実在する non-root user で実行します。

## Public OAuth

Portico は DNS を案内し、専用 OAuth identity を作ります。

~~~text
Username: vps-operator
Email:    operator@example.com
~~~

Linux/SSH/root credentials ではありません。

## ChatGPT 完了

`scripts/connect-chatgpt.sh` が完全な tutorial を表示します。App/OAuth を自分のペースで設定し、`system.info` を送信し、terminal に戻って Enter を押します。

## Local-only

~~~bash
bash scripts/install.sh --profile custom --scope /opt --local-only --yes
~~~

これは public installation 完了ではありません。

## Removal

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Checkout も削除:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

## Stable release

`v0.1.0` 以降、production は stable tag を使います。

~~~bash
git clone --branch v0.1.0 https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Git checkout により verified fast-forward update、backup、migration、automatic rollback が可能です。
