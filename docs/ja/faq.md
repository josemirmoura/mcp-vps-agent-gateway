# FAQ

## Whole Host は Full ですか？

いいえ。Whole Host は Broker の物理 filesystem 上限を `/` にします。Full は Policy/Feature Gate で制御される別の capability bundle で、既定では無効です。

## Full は無制限 Internet ですか？

いいえ。Network は別の capability/approval です。

## ChatGPT に VPS password や SSH key が渡りますか？

いいえ。インストールは VPS terminal で実行し、ChatGPT は public MCP endpoint と OAuth を使います。

## Broker はなぜ privileged ですか？

Host systemd、Docker、委任された host filesystem 操作には trusted privileged component が必要です。Broker は local-only で policy-authoritative、Gateway は non-root のままです。

## Docker が security boundary ですか？

いいえ。Docker は packaging/lifecycle。実際の認可境界は Broker の server-side authorization です。

## Telemetry はありますか？

First-party analytics/tracking client はありません。Bundled ZITADEL telemetry も無効です。`privacy.md` を参照してください。

## 1つの project だけにできますか？

はい。**Project** profile を使います。

## 推奨 default は？

**Standard**。`/opt` を物理上限にし、開始時は project root 無許可、必要時に明示承認します。

## 複数 directory を委任できますか？

はい。物理上限内で動的承認、または advanced operator 向け static root が使えます。

## 複数 VPS は？

はい。各 instance は独立した identity、credentials、state、audit chain を持ちます。

## 他の MCP client は？

Core は standards-based MCP Streamable HTTP。Public product path は ChatGPT Web で検証され、必要な auth flow を支える互換 client も利用可能です。

## インストール完了条件は？

実際の client call が authentication、Gateway、Broker、policy、execution、audit を通過したときです。Container health だけでは完了しません。

## なぜ update.sh は main を使わない？

`main` は development。Production は stable SemVer tag を使います。

## Portico を削除して app を残せますか？

はい。Safe remove/purge は Portico 所有物だけを削除し、管理対象 resource を保持します。
