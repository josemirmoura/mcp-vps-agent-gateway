# ChatGPT 統合

確認日: 2026-09-30。

## 目標

~~~text
ChatGPT Web
 -> OAuth discovery + Dynamic Client Registration
 -> HTTPS /mcp
 -> Gateway
 -> Broker
 -> VPS
 -> tamper-evident audit
~~~

Container health だけでは installation complete ではありません。

## ChatGPT 前提条件

Public setup 前に、対象 account/workspace が Developer Mode と custom MCP app creation を実際に提供し、必要な permission が利用できることを確認します。Rollout と UI は OpenAI が管理します。

Project references:

- https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt
- https://developers.openai.com/plugins/build/app-quickstart

Read-only MCP は診断には役立ちますが、write-capable product path 完了とは異なります。

## Server prerequisites

1. Custom MCP creation が利用可能。
2. `bash scripts/verify.sh` 成功。
3. DNS が VPS を指している。
4. Integrated auth が ready:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

終了時:

~~~text
INTEGRATED AUTH: READY
~~~

Public boundary 再検証:

~~~bash
bash scripts/verify-public.sh
~~~

## Connection

~~~bash
bash scripts/connect-chatgpt.sh
~~~

期待 flow:

1. ChatGPT が RFC 9728 protected-resource metadata を読む;
2. ZITADEL を discover;
3. public OAuth client を dynamic register;
4. operator が専用 OAuth account で login;
5. Authorization Code + PKCE;
6. ChatGPT が MCP tools を discover;
7. `system.info` 呼び出し;
8. Gateway が access token 検証;
9. Broker が stable subject と policy を確認;
10. Audit が記録。

ChatGPT に VPS password、SSH private key、root、unrestricted remote shell を渡しません。OAuth password は identity-provider login にのみ入力します。

## Completion gate

`connect-chatgpt.sh` は Audit baseline を固定します。設定後 Enter を押すと、それ以降の authenticated `system.info` を確認します。

~~~text
tutorial shown
 != success

ChatGPT app connected
 + authenticated system.info
 + expected subject
 + Broker allow
 + successful execution
 + matching audit
 = INSTALLATION COMPLETE
~~~

## Transport

~~~text
https://<domain>/mcp
~~~

MCP Streamable HTTP over HTTPS。Custom WebSocket は使いません。

## Security boundary

ChatGPT confirmation は追加 UX control であり、server-side authorization の代替ではありません。

**Gateway authenticates. Broker authorizes. VPS owner chooses policy.**
