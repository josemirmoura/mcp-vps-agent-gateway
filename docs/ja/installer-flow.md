# Portico MCP 初回実行フロー

## 製品目標

インストールは宣言的でターミナル中心です。主なオペレーター状態は:

~~~text
.env                   # VPS_AGENT_SCOPE_ROOT を含む
config/policy.yaml     # 論理的 capability/resource policy
~~~

Docker Compose がパッケージを起動します。設定を隠す別のインストーラーはありません。

## サポート境界

ユーザーが VPS 上で直接コマンドを実行します。開発者アクセス、ChatGPT 制御のリモート shell、VPS パスワード、SSH 秘密鍵は不要です。

OAuth オペレーターパスワードはサービスに必要な credential であり、`setup-integrated-auth.sh` にローカルかつ非表示で入力します。

## ガイド付き入口

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

`install.sh` は既存の透明なコンポーネントをオーケストレーションし、runtime 起動前に実効権限を表示し、中断後も再実行できます。

## Phase 1 — Bootstrap

推奨は **Standard**: 物理上限 `/opt`、静的プロジェクトルートなし。上限は filesystem の最大境界であり、読み書き権限ではありません。別の絶対パスも選べます。

~~~bash
bash scripts/init.sh --scope /opt --dynamic-baseline
~~~

状態変更前に `scripts/preflight.py` が host、Docker/Compose、tools、systemd、RAM、edge を確認します。必須条件が欠ける場合、状態作成前に停止します。

Bootstrap はローカルランダム secret、安定した instance ID、`config/policy.yaml`、選択した上限、制限付き shell 用の実在 non-root user、state directory、Compose validation を準備します。

## 物理 filesystem 上限

`compose.yaml` は `VPS_AGENT_SCOPE_ROOT` のみを Broker の `/host` に bind mount します。filesystem、shell cwd、Compose path はこの境界内に限定され、escape は fail-closed です。

Standard の `--dynamic-baseline` は `/opt` 自体を許可しません。Whole Host は `VPS_AGENT_WHOLE_HOST=1` と `compose.host.yaml` が必要で、上限を `/` にしますが Full は有効にしません。

## Phase 2 — MCP 権限

Standard はプロジェクト権限なしで開始します。`permissions.discover_scope` は上限直下の directory 名だけを表示し、内容は開けません。

~~~text
permissions.request_root_access
 -> Broker pending request
 -> native MCP elicitation / client confirmation
 -> subject-bound delegation
~~~

- `read`: filesystem read;
- `work`: read/write + confined shell cwd;
- `compose`: static policy が許可済みの Compose action を追加。

### 保護ファイル

`.env` と `.env.*` は許可済みプロジェクト内でもロックされます。`.env.example`, `.env.sample`, `.env.template` は通常の template として読めます。

`permissions.request_sensitive_access` は別の temporary exact-path、subject-bound、audited、expiring、revocable な例外です。Scoped shell は該当ファイルと検出済み hardlink alias を mask します。

強い警告の後で物理上限自体を委任することもできますが、上限外へは出られず、protected secret も解除されません。

systemd、Docker/Compose、network、packages、users/groups、firewall、elevation は別に policy 制御されます。

## Phase 3 — 起動

~~~bash
docker compose up -d --build
~~~

~~~text
Gateway: non-root, no host root
Broker: privileged, host mounted at /host, no remote control port
~~~

Docker は packaging。認可境界は Broker policy です。

## Phase 4 — ローカル検証

~~~bash
bash scripts/verify.sh
~~~

Compose、Broker/Gateway health、Audit integrity、authentication、安全な `system.info` を検証します。成功しても **インストール完了ではありません**。

## Phase 5 — 統合 OAuth + 公開 endpoint

DNS は外部の手動操作です。Portico は A/AAAA 設定を案内して解決を確認します。

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

安全に特定できる Traefik が1つあれば再利用します。なければ 80/443 が空いている場合に bundled Traefik を起動します。ZITADEL + PostgreSQL、専用 non-admin operator、resource audience、private introspection client、MCP-compatible DCR、Gateway/Broker subject binding、public verification を構成します。

成功:

~~~text
INTEGRATED AUTH: READY
~~~

再検証:

~~~bash
bash scripts/verify-public.sh
~~~

HTTPS、OAuth/OIDC discovery、protected-resource metadata、DCR/PKCE、private introspection、unauthenticated MCP の fail-closed を確認します。

## Phase 6 — ChatGPT capability

実際の account/workspace が Developer Mode と custom MCP app creation を提供していることを確認します。提供状況は OpenAI が管理します。

## Phase 7 — ChatGPT Web tutorial

~~~bash
bash scripts/connect-chatgpt.sh
~~~

App 作成と OAuth login は ChatGPT Web で専用 OAuth identity を使います。VPS/SSH credential は使いません。

## Phase 8 — 実際の接続確認

Script は Audit baseline を記録します。ユーザーは自分のペースで設定し、戻って Enter を押します。ChatGPT に `system.info` を呼ばせます。

成功条件:

~~~text
real ChatGPT MCP call
+ expected authenticated subject
+ policy allow
+ successful execution
+ Broker audit
+ valid audit chain
~~~

その後だけ:

~~~text
CHATGPT WEB CONNECTION VERIFIED
INSTALLATION COMPLETE
~~~

## 更新と削除

~~~bash
bash scripts/update.sh
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Checkout も削除:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

委任されたプロジェクトや第三者リソースは、Portico が管理権限を持っていたという理由だけで削除されません。
