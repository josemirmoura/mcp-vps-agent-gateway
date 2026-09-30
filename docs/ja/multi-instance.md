# 1つの ChatGPT workspace で複数 VPS instance を使う

Advanced scenario です。通常 onboarding は installation ごとに Portico MCP instance 1つです。

## Identity model

Tool name は installation 間で安定させ、server ごとに rename しません。

各 installation は:

- 独自 HTTPS MCP endpoint;
- 独自 OAuth client/resource relationship と credentials;
- bootstrap 生成の安定した `VPS_AGENT_INSTANCE_ID`;
- operator-visible `VPS_AGENT_INSTANCE_NAME`;
- `system.info`, Broker audit, structured logs の instance identity。

例:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Loja"
VPS_AGENT_PUBLIC_URL=https://mcp-loja.example.com/mcp
~~~

別の VPS:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Blog"
VPS_AGENT_PUBLIC_URL=https://mcp-blog.example.com/mcp
~~~

ChatGPT では visible name を合わせた別々の app として登録し、同名 tool の silent disambiguation に頼らず対象 app を select/@mention します。

## Verification

各 app で `system.info` を呼び、`instance_id` と `instance_name` を期待する VPS と比較します。Hostname は補助情報だけにします。Provider が別の disposable machine に同じ hostname を使うことがあるためです。

Broker audit:

~~~bash
bash scripts/diagnose.sh audit 20
~~~

同じ instance identity が新しい audit event と structured log に現れる必要があります。

## Same-conversation use

ChatGPT UI が同じ conversation で複数 custom MCP app を許す場合、operation 前に対象 app を明示的に選択/mention します。UI behavior は server と独立して変わる可能性があります。

## Automated simultaneous-use gate

Repository には three-machine acceptance workflow があり、3つの independent Ubuntu runner で real Docker package を同時起動し、同じ subject、tool names、logical path、operation id を意図的に再利用します。

Pass 条件:

- 3 instance の ID/name が異なる;
- generated credential が独立;
- runner lifetime が overlap;
- 同じ operation id が各 Broker state store で独立に成功;
- 各 audit chain が自分の identity だけを記録;
- hostname と audit head が異なる;
- 各 instance が期待 state で終了。

これは server-side collision test で、人間操作なしで実行できます。

## ChatGPT product surface

同一 workspace に複数 live app を接続するのは別の UI experiment で、normal single-VPS onboarding や final single-instance E2E gate には不要です。Test する場合は VPS ごとに別名 app を登録し明示的に選びます。
