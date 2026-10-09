# Community: MCP Apps in-chat operator approvals (design gate)

**Status:** owner-approved direction; NOT implemented or verified in ChatGPT conversation. HTTPS `/operator` remains the functional fallback.

## Target experience

When a compatible MCP client supports interactive MCP Apps and a verifiable owner/operator identity can be bound to the local node, display the responsive operator approval interface **within the conversation**. Show request ID, node, authenticated MCP subject, canonical path, profile, lifetime, expiration and root-wide warnings. Provide explicit, accessible **Approve** and **Deny**. Successful decisions remain visibly final instead of refetching the pending-only queue.

Reduce friction: do not request the owner's password repeatedly while a trusted, scoped operator session remains valid. **Being embedded in ChatGPT is not proof of owner authorization.** An OAuth-authenticated MCP client does not automatically carry the operator's right to grant filesystem authority. Without a verified binding, fail closed and use native MCP elicitation (if supported securely) or the existing HTTPS/SSH fallback.

## Implementation constraints

- Verify current official MCP Apps client capability and resource/UI contract with the actual SDK and target ChatGPT Web/mobile runtime. No claim of a modal/embedded surface before a real end-to-end test.
- Keep local Broker authoritative for policy, node/subject/expiry/one-shot scope, audit and revocation; no model-visible tool may authorize its own request.
- Design an identity and session proof for UI-only operator decisions: bind verified operator, node, request ID, action and short expiration; defend against spoofing, CSRF, XSS, replay, clickjacking, prompt injection, and concurrent decisions. The UI-only capability boundary alone is insufficient for authentication.
- Do not expose Broker admin tokens, approval tokens, secrets, Docker socket or a broad administrative API to browser resources, model context or public MCP tools. Keep `operator-run` socket isolated and current scoped permission limits enforced.
- Preserve current HTTPS portal requiring authentication, CLI/SSH fallback, native elicitation when appropriate, and separate step-up for sensitive/elevated/permanent grants.
- Reuse `web/operator-approval/` CSS/layout and backend response contracts where practical, without coupling external portal credentials to UI assets. Support localization, mobile, accessibility and untruncated resource descriptions.

## Engineering steps

1. Read `internal/gateway/native_approval.go`, `web/operator-approval/`, `cmd/vps-agent-broker/operator_bridge.go`, Compose overlays and this repository's release gates.
2. Write a concise capability/identity ADR grounded in official MCP Apps and the target client's actual support.
3. Prototype in-chat rendering with no decision authority; then add verified operator binding and operator-only decision channel; revalidate at Broker.
4. Add tests for supported/unsupported clients, login/session fallback, altered or expired IDs, deny/approve/replay/concurrency, source/user spoofing, broader-than-requested grant attempts, audit and revocation.
5. Run full CI on the exact release SHA and real end-to-end browser/chat acceptance. No unattended production upgrade or removal of password fallback.

## Current reference status (2026-10-09)

- PR [#88](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/88), branch `feat/operator-approval-web-backend`; HTTPS portal activated on one owner-operated Community node with scoped Broker, login, loopback+Traefik routing.
- A real temporary read approval was recorded and confirmed at Broker; the earlier UI false error was fixed on branch commit `63e924ff5dbc45ae49d9b518196efd4e296a7cdd` and verified via browser-flow tests. The operator confirmed the live static JS had updated.
- All 10 GitHub workflow checks succeeded on `63e924f`. **No real in-chat MCP App has been tested; live negative decision and revocation acceptance remain to be completed.**
- Private Cloud strategy and future multi-node adaptation are tracked in the private canonical and must not be copied into public Community docs.
