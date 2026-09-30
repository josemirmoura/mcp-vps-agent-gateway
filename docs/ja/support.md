# Support policy

Portico MCP は open-source project です。

## Supported release line

初期 productization 期間:

- 最新 stable `0.x` release が public supported line;
- current release candidate は acceptance/testing 向けに support;
- `main` は development であり stable support channel ではありません。

Security fix のため最新 patch release への upgrade が必要になる場合があります。

## Help

Secret を含まない reproducible bug、installation failure、documentation issue は GitHub issue を使用してください。

可能なら issue 作成前に sanitized diagnostic bundle を収集します。

~~~bash
bash scripts/diagnose.sh bundle
~~~

`.env`、raw credential、private SSH key、unredacted secret material を添付しないでください。

## Security issues

通常の issue に exploit detail を公開しないでください。`SECURITY.md` に従います。

## Service level

Open-source project に guaranteed response-time / uptime SLA はありません。

将来 commercial distribution に support commitment がある場合は別途文書化し、この repository から推測しないでください。
