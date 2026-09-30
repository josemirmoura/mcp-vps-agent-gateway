# トラブルシューティング

最初に:

~~~bash
bash scripts/diagnose.sh status
~~~

必要なら:

~~~bash
bash scripts/diagnose.sh bundle
~~~

共有前に bundle を確認してください。

## Scope directory が存在しない

Portico は任意ディレクトリを黙って作りません。

~~~bash
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/my-app
~~~

または installer を再実行し、表示された sudo 作成を承認します。

## Path が non-canonical / symlink

Dot segment や symlink ancestor のない absolute canonical path を使います。これは意図的な filesystem boundary hardening です。

## Broker/Gateway が healthy にならない

~~~bash
docker compose ps -a
docker compose logs --no-color --tail 100 broker gateway
bash scripts/diagnose.sh status
~~~

Health gate を迂回せず、原因を修正してください。

## 80/443 が使用中

安全に識別できる既存 Traefik のみ再利用します。不明な Web server を置き換えません。

## Traefik が複数

~~~bash
bash scripts/setup-integrated-auth.sh --edge-network YOUR_NETWORK
~~~

ACME resolver も曖昧なら:

~~~bash
--certresolver YOUR_RESOLVER
~~~

## DNS が解決しない

A/AAAA を修正し、反映を待って再実行します。

## Public OAuth verification が失敗

~~~bash
bash scripts/verify-public.sh
~~~

DNS、certificate、hostname/issuer、protected metadata、OIDC DCR/PKCE、proxy route を確認します。Issuer/Audience validation を弱めないでください。

## ChatGPT connection が来ない

~~~bash
bash scripts/connect-chatgpt.sh
~~~

MCP app、OAuth、Portico selection を確認し、ChatGPT に `system.info` を呼ばせます。Timeout は完了扱いではありません。

## update.sh に stable release がない

Stable 前は自動 target がないのが仕様です。

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

`main` は自動 production channel ではありません。

## Update rollback

Backup を削除せず、output、`backups/<timestamp>/migration-check.json`、`state/update.log`、diagnose を確認します。Rollback は安全機能です。

## 診断に secret が見える

共有しないでください。ローカル保存し、必要に応じて credential を rotate し、`SECURITY.md` に従います。
