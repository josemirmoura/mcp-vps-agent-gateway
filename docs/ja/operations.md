# 運用

## 状態

~~~bash
bash scripts/diagnose.sh status
~~~

製品 version、Compose state、Broker/Gateway health、container restart count、Broker health snapshot、Audit chain status を表示します。

## Health

~~~bash
bash scripts/diagnose.sh health
~~~

Broker health snapshot のみ返します。

## Logs

~~~bash
bash scripts/diagnose.sh logs 200
~~~

設定済み secret 値を redact した最近のログを表示します。restart、auth failure、Gateway/Broker connectivity、timeout、unhealthy container の調査に使います。

## Audit

~~~bash
bash scripts/diagnose.sh audit 100
~~~

誰が、どの tool/resource/action を、allow/deny、結果、sequence、hash chain validity とともに確認できます。Operational logs と Security Audit は別の役割です。

## Diagnostic bundle

~~~bash
bash scripts/diagnose.sh bundle
~~~

Runtime/version、health、audit status/tail、logs、Docker/Compose version、redacted policy snapshot を含みます。0600 で作成されますが、共有前に必ず確認してください。

## Version

~~~bash
bash scripts/version.sh
bash scripts/version.sh --check
~~~

`--check` は可能なら release tag を更新し、current/ahead/divergent/update available を報告します。

## Update

~~~bash
bash scripts/update.sh
~~~

既定 target は最新の stable SemVer tag。clean working tree を確認し、変更概要を表示、non-fast-forward を拒否、package を停止、`.env`/policy/state/identity volumes を backup、DB コピーで migration を確認、target を build/verify し、失敗時は自動 rollback します。

RC 指定:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

## Safe remove

~~~bash
bash scripts/remove.sh safe
~~~

一時権限を revoke し、runtime を削除、local credential を rotate します。設定、policy、audit/state、integrated identity は保持されます。

## Purge

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Source checkout は二段階確認でのみ削除:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

委任された project、app、site、database、third-party image/container、service、file は削除しません。
