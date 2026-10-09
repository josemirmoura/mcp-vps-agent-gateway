# Final operator acceptance gate

This gate is intentionally owner-operated. It is the last acceptance step before freezing the first stable `v0.1.0` release.

Do not run it casually on an important existing MCP installation. Prefer a clean supported VPS or a target environment the owner explicitly chooses for destructive lifecycle testing.

## Starting conditions

- use the release-candidate tag under test;
- use a supported Ubuntu 24.04-class Linux host;
- use Docker Engine 24+ with Docker Compose v2;
- provide at least 2 GB RAM for the bundled ZITADEL path;
- control DNS for the public MCP hostname;
- have public TCP 80/443 available through the supported edge path;
- use a ChatGPT Web account/workspace where Developer Mode / custom MCP app creation is actually available;
- for acceptance of write/modify capabilities, confirm that the current ChatGPT product surface actually exposes those actions. OpenAI's public documentation currently describes full MCP write/modify support for Business, Enterprise and Edu, while the operator's ChatGPT Plus environment independently completed a real scoped write/read/delete proof through Portico on 2026-10-06. Treat the observed target environment as acceptance evidence without turning it into a plan-wide guarantee.

Record:

- release tag and commit;
- host OS/architecture;
- chosen authority profile;
- public hostname;
- start/end timestamps.

Do not record passwords, tokens, private keys or .env contents.

## 1. Clean installation

Use the **exact immutable, published RC tag** approved for acceptance, not merely the development `VERSION` value. As of 2026-10-09, source identifies as RC7 while the latest published release is RC6; **`git clone --branch v0.1.0-rc.7` is not a valid published-RC procedure yet**.

Only after the final candidate has actually been released and its signatures verified:

~~~bash
# Supply the exact immutable tag recorded in the acceptance ticket.
PORTICO_ACCEPTANCE_RC=v0.1.0-rc.X

git ls-remote --exit-code --tags \
  https://github.com/josemirmoura/mcp-vps-agent-gateway.git \
  "refs/tags/$PORTICO_ACCEPTANCE_RC"

git clone --branch "$PORTICO_ACCEPTANCE_RC" \
  https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
test "$(git describe --tags --exact-match)" = "$PORTICO_ACCEPTANCE_RC"
bash scripts/install.sh
~~~

Replace the placeholder with the **real, verified release tag**; never run acceptance against an invented tag or floating `main`.

Confirm:

- product/version banner is correct;
- requirements are checked;
- Standard / Project / Whole Host choice is understandable;
- effective authority is shown before startup;
- Standard starts with /opt as the default physical ceiling and no project root authorized;
- a different absolute physical ceiling can be selected deliberately;
- the installer explains that the ceiling is a maximum boundary, not implicit read/write authority;
- no silent privilege escalation occurs;
- no VPS/SSH credential is requested by ChatGPT or documentation.

## 2. Public security and OAuth

Confirm:

- DNS resolves to the intended VPS;
- HTTPS is valid;
- protected-resource metadata is reachable;
- OIDC discovery is correct;
- DCR + PKCE work;
- dedicated operator login works;
- the operator subject is stable;
- public unauthenticated MCP fails closed.

## 3. ChatGPT connection and E2E completion

In ChatGPT web:

- enable/use Developer Mode/custom app according to current OpenAI workspace controls;
- create the app using the HTTPS `/mcp` endpoint;
- complete OAuth with the dedicated operator identity;
- invoke `system.info`;
- return to the terminal and press Enter to verify the audited call.

The terminal must not print `INSTALLATION COMPLETE` until the authenticated call is observed through Gateway -> Broker -> policy -> execution -> audit.

## 4. Native authority UX, discovery and safe real operations

Before granting a project root:

- call `permissions.discover_scope`;
- confirm it returns only immediate directory names below the physical ceiling;
- confirm no project file content is exposed by discovery.

Request access to one disposable project root:

- record whether the connected client actually advertises MCP elicitation. If it does, verify its **native confirmation UI** (not the legacy iframe); if it does not, verify pending/fail-closed status and the **separately authenticated operator approval fallback** supported by that released candidate. Do not assume the current ChatGPT host supplies native elicitation;
- verify path, access profile, duration and physical ceiling are all visible for both narrow and ceiling-wide root requests;
- confirm that requests containing invisible Unicode formatting characters (bidirectional overrides, isolate marks, zero-width joiners), Unicode line/paragraph separators or invalid UTF-8 are rejected before the native approval dialog, with no delegated access granted;
- confirm a request covering the full physical ceiling cannot proceed without the stronger current-and-future-descendants disclosure;
- for an exact-ceiling request, confirm the stronger current-and-future-descendants warning appears; denying that broad request is sufficient for this UI check;
- approve a narrower disposable root and verify the Broker activates only the requested `read`, `work` or `compose` profile;
- revoke the root and verify access fails immediately.

Within the approved disposable scope, exercise representative operations such as:

- read/write one disposable test file;
- inspect an allowed service or Docker resource;
- run one bounded scoped shell/job if enabled;
- verify an out-of-scope file/resource is denied.

Then validate the nested secret boundary:

- create a disposable `.env.example` and confirm it remains readable;
- create a disposable `.env` and confirm normal file read/hash and confined shell access are denied despite the parent project being authorized;
- request temporary protected-file access to that exact `.env`;
- confirm the supported client or independent operator approval surface clearly identifies the protected file, access profile and expiration, with approve/deny both usable on desktop and mobile;
- approve the temporary exception, perform only the intended test, then revoke it;
- confirm the `.env` becomes inaccessible again immediately;
- confirm no secret content appears in audit/log output.

Do not widen policy merely to make a test pass.

## 5. Audit and observability

Run:

~~~bash
bash scripts/diagnose.sh status
bash scripts/diagnose.sh health
bash scripts/diagnose.sh logs 100
bash scripts/diagnose.sh audit 100
bash scripts/diagnose.sh bundle
~~~

Confirm:

- health and operational logs answer whether the system is healthy;
- audit independently shows subject/tool/resource/decision/result/sequence;
- audit chain is valid;
- diagnostic bundle is created with restrictive permissions;
- no configured secret appears in the reviewed bundle.

## 6. Update and rollback

Before stable release, an RC-to-RC update may be selected explicitly.

~~~bash
bash scripts/version.sh --check
VPS_AGENT_UPDATE_REF=<next-rc-tag> bash scripts/update.sh
~~~

For the final stable promotion, repeat this section using the actual release transition being accepted.

Confirm:

- current/target versions are visible;
- change summary is visible;
- backup is created;
- policy/state/identity data survive;
- migration validation runs;
- successful update verifies the runtime;
- a deliberately failing acceptance update rolls back automatically when using the project's controlled test procedure.

## 7. Safe remove and reinstall

~~~bash
bash scripts/remove.sh safe
~~~

Confirm:

- Gateway/Broker runtime disappears;
- active delegated grants are revoked;
- local active credentials are rotated/invalidated;
- .env, policy, state/audit and integrated identity state are preserved;
- applications, sites, databases, third-party containers, services and managed files remain.

Reinstall:

~~~bash
bash scripts/install.sh
~~~

Confirm the preserved installation can be resumed successfully.

## 8. Purge

Only after all previous evidence is captured:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Confirm:

- MCP-owned local configuration/state/identity artifacts are removed as documented;
- resources merely administered by the MCP remain intact.

## Pass criteria

The operator acceptance gate passes only when every applicable step above succeeds without weakening the documented security model.

If a problem appears:

1. record the failing step without secrets;
2. fix the product;
3. rerun the affected automated tests;
4. repeat the necessary operator acceptance section;
5. only then promote/freeze `v0.1.0`.

Full/R5 maturity and long-running R4 reliability remain separate from this first stable Scoped release decision unless explicitly claimed.
