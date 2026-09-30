# Architecture

## Goal

AI client に Linux VPS の useful operational access を与えつつ、authorization/privilege の制御は model ではなく VPS に残します。

> **LLM は security boundary ではありません。**

## Canonical runtime

Project-owned Go processes 2つ:

~~~text
ChatGPT / MCP client
        |
        | MCP Streamable HTTP
        v
+-----------------------------+
| vps-agent-gateway           |
| unprivileged                |
| MCP + auth + schemas        |
+-------------+---------------+
              |
              | Unix Domain Socket
              v
+-------------+---------------+
| vps-agent-broker            |
| privileged, local-only      |
| authoritative policy        |
| SQLite + locks + jobs       |
| secrets + audit             |
| files + systemd + Docker    |
| sandboxed execution         |
+-------------+---------------+
              |
              v
     Linux / systemd / Docker
~~~

Reverse proxy/private tunnel は ingress infrastructure で、third project service ではありません。

## Why Go

Official MCP Go SDK は Tier 1 で MCP 2026-07-28 対応。一言語で packaging/dependency/maintenance cost を下げつつ process privilege boundary を維持。

## Gateway

Non-root。MCP expose、OAuth/OIDC/schema validation、tool/resource normalization、non-authoritative preflight、Unix socket Broker call を実行可能。

root、Docker socket、privileged SQLite、plaintext secret、authoritative authorization、自分の elevation approval は不可。

## Broker

Privileged security boundary。Every privileged call を再認可:

~~~text
subject
+ canonical tool
+ canonical resource
+ action
+ current policy
+ grant/lease/job when required
~~~

Broker owns policy evaluation、SQLite、idempotency、locks、durable jobs、safe filesystem、Docker/systemd、sandboxing、secret resolution、audit。Gateway は untrusted deputy。

## Capability model

One broad catalog、effective authority は policy。One root、multiple roots/resources、whole host を選択可能。

Physical ceiling と logical roots は分離。Ceiling は maximum boundary で read grant ではありません。Standard は immediate directory names の discovery-only を許せます。Dynamic delegation は Broker-owned、subject-bound、`read`/`work`/`compose`、optional expiry。

Pending request は authorization ではありません。MCP elicitation 対応 client は native human confirmation を表示。Broker が request/subject/one-time token を独立検証。Confirmation tools は model-visible ではありません。Elicitation 非対応は operator fallback へ fail-closed。Revocation は Broker state ですぐ有効。

Secret-bearing file は nested boundary。`.env` は second temporary exact-path grant。Template は通常 content。Shell sandbox も protected path を mask。

Filesystem 以外に systemd、Docker、shell、network、admin を別々に scope。See [product-model.md](product-model.md).

## Policy / approval

Policy Engine は Broker module、別 daemon ではありません。Policies deny-by-default。Controlled、Scoped、Full。Full default off。

Routine Scoped は human interruption なし。Temporary elevation approval は action channel の外。Native elicitation を優先し、Broker が subject + one-time nonce、anti-replay を enforce。Separate admin route は fallback/MFA。

## Full

Explicit capabilities に展開:

~~~text
shell.admin
filesystem.read:any
filesystem.write:any
docker.admin
systemd.admin
~~~

Unrestricted network は別 approval。Full は temporary/revocable/gated。

## Transport

Stable HTTPS の MCP Streamable HTTP。2026-07-28 core は stateless。Custom WebSocket/session machinery は作らない。Jobs/leases は explicit handles、protocol negotiation は SDK。

## Filesystem

String-prefix authorization 禁止。`openat2` restrictive flags を優先。Fallback は secure directory-FD walk または fail closed。String check への silent fallback はしない。

## Process isolation

systemd transient units、cgroups、NoNewPrivileges、PrivateTmp、FS restrictions、MemoryMax、TasksMax、deadline、output limit、cancellation。Landlock は defense-in-depth。

## Docker

Gateway に `/var/run/docker.sock` を渡さない。Typed Broker operations を canonical stack/action に対して認可。

## Jobs

~~~text
job.start
job.status
job.tail
job.cancel
~~~

Internal durable model が authoritative。HTTP connection に依存しない。MCP Tasks は将来 adapter 可能。

## State

SQLite は Broker のみ。Transaction は短く、external command/deploy/migration/Docker action 中に開きっぱなしにしない。

State: approvals、leases、jobs、idempotency journal、resource locks、audit metadata。

Lock は resource、owner/action_id、monotonic fencing_token、expires_at を持つ。Expired lock の再取得で token increment、stale owner は newer token を release/commit 不可。

Idempotency:

~~~text
PENDING -> external effect -> DONE
~~~

External effect 前に PENDING commit。Effect 後 DONE 前 crash は reconciliation、blind re-execute しない。

## Secrets

Native Linux first: repo 外 root-owned files、systemd credentials。Broker が refs を解決し target process にだけ inject。Gateway/model tools は plaintext retrieval しない。

## Audit maturity

Gate 1: structured local audit。
Scoped production: durable sequence/integrity。
Before Full: tamper-evident hash chain、remote checkpoint/forward、tested recovery。

## Downstream trust

Upstreams allowlisted out-of-band、tool names deterministic/namespaced、schema changes fingerprint/review、results untrusted data。Result が policy/lease/server registration/secret exposure を変更しない。

## First-version non-goals

Universal gateway、multi-tenant control plane、distributed scheduler、generic root shell、Kubernetes、SSH replacement、every MCP client support ではありません。

> First goal: actual target AI client が1台の VPS で small, valuable, auditable operation を安全に実行できることを証明する。
