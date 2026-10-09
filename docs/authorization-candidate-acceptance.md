# Authorization candidate: desktop/mobile acceptance

This checklist covers the authorization candidate derived from #93/#88, preserving #94. It is a scoped Community review, not a release or deployment authorization. The integration owner selects the final integrated SHA and records its CI separately. See [rollout and rollback](adaptive-approvals-rollout.md).

## Candidate and isolation

Record the complete candidate SHA from the reviewed PR, its base, CI URLs, Gateway/Broker version, image digests and portal checkout SHA. Binary version alone does not identify this patch. Keep a previous safe candidate/configuration and a consistent protected backup for any authorized promotion.

Use a disposable Linux node with its own Compose project, roots, database, Unix sockets, TLS origin and test credentials. A worktree does not isolate a privileged Broker from the host. Do not run a generic Compose startup on the installed node. Provision passwords and socket credentials locally; never paste them, cookies, CSRF values, decision nonces or raw network captures into a chat.

Before client acceptance, verify these commands on the candidate in a suitable Linux runner:

```sh
go test -race -count=1 ./internal/state ./internal/broker ./internal/gateway ./cmd/vps-agent-broker
python3 -m unittest discover -s web/operator-approval -p 'test_*.py' -v
node --test web/operator-approval/test_*.mjs
```

The dedicated browser workflow also runs the pinned Chromium harness. It emulates desktop/mobile; the real Broker test exercises HTTP BFF → restricted Unix IPC → SQLite decision/grant/audit. Neither is real ChatGPT acceptance.

## Real client sequence

Run the sequence independently on desktop and a physical mobile device. Record client/account surface, browser/OS/version, date UTC, candidate SHA and sanitized outcomes. Use only harmless fixture files, including a synthetic protected file; do not test with personal secrets.

1. Connect the test MCP endpoint and call `system.info`; confirm the expected node and subject in Broker audit. Record the capabilities actually received by Gateway. Do not assume elicitation or Apps from a subscription/client name.
2. Request a short temporary **read** root inside the physical ceiling, with no prior grant. The read must fail while pending. Open its HTTPS link without operator login; details/decisions must require authentication. Confirm full node, subject, path, profile and lifetime after local operator login.
3. **Deny** explicitly. Query `permissions.approval_status` and retry the harmless read: status denied, no grant, read denied, operator/requester/node provenance in audit. Repeat the used decision: no second decision/grant.
4. Request another temporary read and **approve** explicitly. A valid operator session may be reused, but a separate decision remains required. Status approved and the exact harmless read succeed. Another subject/root/node stays denied. Revoke explicitly through the requesting subject's revocation tool; status revoked and read denied again. A separate short grant must also stop permitting reads after its expiry.
5. Request temporary work/compose or protected-file access permitted by local policy. Approve must require fresh password verification per request; wrong password, proof from another request/session, expired nonce or logout during verification cannot approve. **Deny** must work without retyping the password. Do not enable Full/admin elevation for this gate.
6. Test pending request expiry, cancellation, duplicate clicks and two windows attempting approve/deny. Allow only one terminal decision and at most one grant. Let a decision nonce expire before clicking: it must fail, then consult state. Losing a reply must require consultation; do not blindly repeat an operation.
7. Logout and reload a terminal request; an old page/late response cannot restore approval controls. Session expiration (15 minutes) or operator/node/credential configuration change must require new authentication. Test CSRF/foreign Origin rejection in the automated suite; never loosen these controls to make a mobile client work.
8. Exercise the available channels below. Unsupported channels are recorded as **NOT TESTED** or **NOT APPLICABLE**, never PASS. Record actual approve/deny and fallback outcomes separately from rendering.

| Channel | Required observations |
|---|---|
| MCP Apps | MIME negotiated, UI handshake `2026-01-26`, observed exact frame ancestors, allowed operator `frameDomains`, independent login, explicit approve **and** deny; `serverTools` only when announced. Credentials never enter the bridge. |
| Restricted host | Missing/malformed `serverTools` produces no tools/call from the wrapper; HTTPS/SSH remains available. Missing `openLinks`, blocked iframe/cookies or bridge timeout leaves a visible URL and SSH instruction. |
| Elicitation, if actually announced | Accept/decline/cancel is navigation only; without an authenticated portal/CLI decision the Broker stays pending and the read stays denied. |
| HTTPS external | Responsive page, full scope legible, keyboard/touch login, approve/deny, repeated read-session use, work/protected step-up and logout. |
| CLI/SSH | Run `python3 scripts/operator-approvals.py --request apr_...` in the trusted interactive terminal of the test checkout. Exact approve/deny phrase; non-TTY, wrong ID, expiry or cancelled input grants nothing. |

## Evidence and result

For each step record PASS, FAIL, NOT TESTED or NOT APPLICABLE, the request ID, node, subject, channel, expected/observed status, grant count and sanitized audit sequence references. Do not collect secret values or raw authentication traffic. A screenshot may show only synthetic scope and public state.

**Developed** means the candidate exists. **Integrated** requires block 5's final commit and CI. **Published** requires the authorized immutable release and verified artifacts. **Accepted** requires observed owner decisions in the real target client/device. Do not conflate these stages.

## Owner-only gate and rollback

After the exact candidate and all automated evidence are prepared, Josemir's indispensable action is personal authentication and visual/touch observation in his real desktop/mobile client. Explain any missing capability and record the fallback instead of requesting a password/token in chat. Deployment, DNS or production changes require their own concrete authorization.

If Apps fails, leave Apps/framing disabled and use the authenticated external portal or trusted SSH. If an authorization invariant fails, stop acceptance, revoke test grants explicitly, preserve sanitized evidence and retain the previous safe Broker/configuration. Restoring UI/images or logging out does not revoke issued grants. Database restore is a separate supervised action: it can resurrect revoked grants and lose later audit; it is not the default rollback.
