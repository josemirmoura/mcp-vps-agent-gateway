# Project status and maturity

## Current stage

**PRE-RELEASE PRODUCTIZATION / CHATGPT E2E GATE COMPLETE**

Docker-first Go implementation は core Scoped runtime、package lifecycle、host-operation path で repeatable clean-runner validation を持ちます。

まだ stable production release、production compatibility promise、long-running real-VPS history、Full/admin shell production claim はありません。

Lab evidence は filesystem CRUD、authorization denial、sandboxed shell/jobs、host systemd、host Docker/Compose、diagnostics、lifecycle、audit をカバーします。2026-09-28 に integrated self-hosted OAuth path を real target VPS 上で ChatGPT Web に対して実行し、authenticated `system.info` が Broker と audit chain で確認されました。Gate 0A は閉じましたが stable production claim ではありません。

## Maturity ladder

### R0 — Architecture only

Documentation のみ。

### R1 — Product path + Gate 0B

Integrated OAuth + ChatGPT Web の Gate 0A 完了。MCP Inspector と disposable root の safe read/write POC。

### R2 — Typed privileged pilot

Gateway/Broker separate、non-critical service を typed tools で扱い、unauthorized resource は fail-closed、local audit。

### R3 — Scoped pilot

Explicit Scoped policy で real stack、数日の durable behavior、recovery/denial test、routine Full 不要。

### R4 — Hardened Scoped production

Deployment auth、durable Broker state、jobs/retries/locks、secret delivery、backup/recovery、monitoring を test。ここで初めて documented Scoped use case を production-capable と呼べます。

### R5 — Elevated/Full production

さらに Full flag、out-of-band approval、temporary grants/expiry、revoke-all、separate network elevation、tamper-evident audit + remote checkpoint、admin recovery test。

## Full default

~~~yaml
features:
  full_mode_enabled: false
~~~

Full は成功条件ではありません。Strong Scoped implementation は valid production endpoint です。

## Evidence over popularity

Stars/forks より reproducible builds、automated tests、releases、deployment history、recovery tests、incident learnings、dependency maintenance、security/external review を優先します。

## Current milestone

現在は **owner clean-install acceptance of `v0.1.0-rc.5`**。RC5 は native MCP elicitation、discovery-only ceiling inventory、nested protected-secret enforcement、public tool safety annotations を含みます。

Stable `v0.1.0` の blocker は exact RC5 tag の owner installation、real ChatGPT OAuth、audited `system.info`、native approval UX、content leakage のない discovery、representative Scoped ops、`.env` denial/temporary exception/revocation、root revocation、lifecycle checks。Long-running reliability は後段です。[operator-acceptance.md](operator-acceptance.md) と [implementation-validation.md](implementation-validation.md) を参照。
