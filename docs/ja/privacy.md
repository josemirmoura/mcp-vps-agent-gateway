# プライバシー、データ、テレメトリ

## データの保存場所

パッケージはオペレーターの VPS にセルフホストされます。運用状態、Policy、監査履歴、統合 ID データは、意図的に export しない限り VPS に残ります。

## Audit

Broker Audit は、誰がどの tool を呼び、どの resource/action を要求し、許可されたか、結果の sequence/integrity state を記録します。

Audit は通常の service log とは別です。

## Logs

Gateway、Broker、Docker、Identity Service の logs には timestamp、instance ID、tool 名、error、request context が含まれる場合があります。Diagnostic bundle は secret を redact し CI で leakage を検査しますが、potentially sensitive data として扱います。

## ChatGPT / MCP

MCP request に必要な data は ChatGPT または他の MCP client と Gateway の間を通過することがあります。Server-side authority は Policy が制御し、model output や remote content は authorization ではありません。Generic file、shell、tool result に secret を出さないでください。

## Secrets

Local credential は root/operator 管理 configuration または private volume に保存します。VPS password、private SSH key、root password、無関係な secret の送付は求めません。OAuth credential は VPS/SSH と別です。

## Telemetry

First-party analytics/tracking client はありません。ZITADEL telemetry は `ZITADEL_TELEMETRY_ENABLED=false` で無効です。Marketing tracking もありません。

## Removal

Safe remove は configuration、state、audit を保持し runtime を削除して credential を rotate します。Full purge は MCP 所有 artefact を削除し、application、site、database、third-party container、service、managed file を保持します。

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~
