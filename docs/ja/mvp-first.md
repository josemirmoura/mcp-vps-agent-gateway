# MVP-first implementation track

Full architecture は north star であり first milestone ではありません。

## Gate -1 — Adopt, adapt, build

Existing product / open-source MCP server を先に評価します。満たせば Adopt、近ければ Adapt、不足するときだけ Build。

Differentiating target:

~~~text
ChatGPT Web
+ self-controlled VPS
+ server-side policy
+ Scoped autonomy
+ optional temporary elevation
+ Broker under operator control
~~~

## Gate 0A — ChatGPT product surface

**2026-09-28 に integrated OAuth + ChatGPT Web path で completed。**

Privileged product path の前に actual target client を検証します。Private write-capable MCP がすべての plan に直接接続できると仮定しません。

Routes:

1. private full MCP write を support する workspace/plan;
2. eligible app/plugin remote write route;
3. distribution 未解決なら MCP Inspector。

記録:

~~~text
target_surface:
target_plan:
integration_route:
read_available:
write_available:
private_or_published:
tested_date:
~~~

Future account が custom MCP を提供しなければ、その boundary で止めるか distribution だけ変更します。Server security を弱めません。

## Gate 0B — Safe MCP POC

Unprivileged Go + official SDK。Expose:

~~~text
system.info
file.read_test
file.write_test
~~~

Filesystem は `/tmp/vps-agent-poc/` のみ。root、Docker、systemd write、SQLite、external secrets、Full、approval、generic shell は入れません。

Success: Inspector、tool discovery、client が許す read/write、forbidden path fail、clear error、safe retry/reconnect。

## Gate 1 — One typed privileged action

Unix socket で minimal Broker。Non-critical unit に `service.status`/`service.restart`。Generic admin shell、Full、approval UI、broad Docker admin、remote audit はまだ除外。

## Gate 2 — Scoped real-stack pilot

Explicit policy で real stack。Need-driven capability のみ追加: file.read、service status/restart、docker logs/restart、job.status。数日使い frequency、success/failure、operator correction、false denial、missing capability、recovery time を測定。Routine elevation なしで useful work ができれば pass。

## Gate 3 — Durability

必要時に Broker SQLite、durable jobs、idempotency identity、logical locks、systemd credentials/secret refs、stronger audit。

## Gate 4 — Broader writes

Need に基づき `file.write`/`file.patch`、validated config updates、typed Docker/systemd action、Scoped root 内 sandboxed `shell.exec`。Generic shell は必須ではありません。

## Gate 5 — Temporary elevation

Scoped が不足した場合のみ elevation requests、out-of-band human approval、temporary leases、revoke-all、remote audit anchoring、Full flag。これらを通過後に `shell.exec_admin` を検討。

## Rule

新 component は previous gate の evidence を必要とします。

> Actual target client は real server で valuable work を safely/reliably 完了できるか?
