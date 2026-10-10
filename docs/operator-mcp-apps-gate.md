# Community: MCP Apps in-chat operator approvals (acceptance gate)

**Status:** implemented as a review candidate in [PR #93](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/93), stacked on #88; not deployed or verified in a real ChatGPT conversation. HTTPS `/operator` and interactive SSH remain available. See the [adaptive architecture](adaptive-operator-approvals.md), [compatibility evidence](approval-compatibility.md) and [rollout/rollback](adaptive-approvals-rollout.md).

## Target experience

When a compatible MCP client supports interactive MCP Apps and a verifiable owner/operator identity can be bound to the local node, display the responsive operator approval interface **within the conversation**. Show request ID, node, authenticated MCP subject, canonical path, profile, lifetime, expiration and root-wide warnings. Provide explicit, accessible **Approve** and **Deny**. Successful decisions remain visibly final instead of refetching the pending-only queue.

Reduce friction: do not request the owner's password repeatedly while a trusted, scoped operator session remains valid. **Being embedded in ChatGPT is not proof of owner authorization.** An OAuth-authenticated MCP client does not automatically carry the operator's right to grant filesystem authority. Native elicitation presents navigation/intention only; its accept/decline/cancel response never grants access. Decisions require the independently authenticated operator origin or trusted CLI.

## Implementation constraints

- Verify current official MCP Apps client capability and resource/UI contract with the actual SDK and target ChatGPT Web/mobile runtime. No claim of a modal/embedded surface before a real end-to-end test.
- Keep local Broker authoritative for policy, node/subject/expiry/one-shot scope, audit and revocation; no model-visible tool may authorize its own request.
- Design an identity and session proof for UI-only operator decisions: bind verified operator, node, request ID, action and short expiration; defend against spoofing, CSRF, XSS, replay, clickjacking, prompt injection, and concurrent decisions. The UI-only capability boundary alone is insufficient for authentication.
- Do not expose Broker admin tokens, approval tokens, secrets, Docker socket or a broad administrative API to browser resources, model context or public MCP tools. Keep `operator-run` socket isolated and current scoped permission limits enforced.
- Preserve current HTTPS portal requiring authentication, CLI/SSH fallback, native elicitation when appropriate, and separate step-up for sensitive/elevated/permanent grants.
- Reuse `web/operator-approval/` CSS/layout and backend response contracts where practical, without coupling external portal credentials to UI assets. Support localization, mobile, accessibility and untruncated resource descriptions.

## Implementation and remaining acceptance

1. Candidate implements negotiated capability selection, credential-free Apps resource, restricted operator bridge, separate bound sessions, one-shot snapshot-bound decisions, critical password reauthentication and atomic Broker grant/audit.
2. Automated tests cover SDK negotiation, native replies that remain pending, real Broker/Unix IPC/BFF integration, replay/concurrency/expiry/identity changes and minimum sandbox operation. Browser profiles emulate desktop/mobile and are reference-host evidence only.
3. The minimum Apps sandbox suppresses native form submission and popups: login uses explicit click/Enter fetches; the external portal link uses advertised host `openLinks` and `ui/open-link`, with the HTTPS URL also visible for copying.
4. Run full CI on the exact reviewed SHA, then real ChatGPT and other available client acceptance with observed capabilities, exact frame ancestors, authenticated owner decisions and fallback. Do not assume reference-host results apply to vendor CSP/cookies.
5. Production promotion requires explicit owner authorization for a concrete version and configuration. No unattended upgrade or removal of HTTPS/SSH.

## Current reference status (2026-10-09)

- PR [#88](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/88), branch `feat/operator-approval-web-backend`; HTTPS portal activated on one owner-operated Community node with scoped Broker, login, loopback+Traefik routing.
- A real temporary read approval was recorded and confirmed at Broker; the earlier UI false error was fixed on branch commit `63e924ff5dbc45ae49d9b518196efd4e296a7cdd` and verified via browser-flow tests. The operator confirmed the live static JS had updated.
- All 10 GitHub workflow checks succeeded on `63e924f`. **No real in-chat MCP App has been tested; live negative decision and revocation acceptance remain to be completed.**
- Candidate `1cd4e0fb4b0a483cbf3fcd539a0d6cbe4acb5bce`: [browser run](https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/37911350591) passed **16/16**, including real cross-site CHIPS partition isolation, minimal sandbox click/Enter login and user-clicked external fallback. This does not establish vendor-client compatibility; later commits require their own CI evidence.
- Private Cloud strategy and future multi-node adaptation are tracked in the private canonical and must not be copied into public Community docs.
