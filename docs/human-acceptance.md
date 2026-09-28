# Final human acceptance gate

This gate is intentionally owner-operated. It is the last acceptance step before freezing the first stable `v0.1.0` release.

Do not run it casually on an important existing MCP installation. Prefer a clean supported VPS or a target environment the owner explicitly chooses for destructive lifecycle testing.

## Starting conditions

- use the release-candidate tag under test;
- use a supported Ubuntu 24.04-class Linux host;
- use Docker Engine 24+ with Docker Compose v2;
- provide at least 2 GB RAM for the bundled ZITADEL path;
- control DNS for the public MCP hostname;
- have public TCP 80/443 available through the supported edge path;
- use a ChatGPT Business or Enterprise/Edu web workspace for the supported full write-capable MCP path.

Record:

- release tag and commit;
- host OS/architecture;
- chosen authority profile;
- public hostname;
- start/end timestamps.

Do not record passwords, tokens, private keys or .env contents.

## 1. Clean installation

Clone the exact RC tag and enter the guided flow.

~~~bash
git clone --branch v0.1.0-rc.1 https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Confirm:

- product/version banner is correct;
- requirements are checked;
- Project / Custom / Whole Host choice is understandable;
- effective authority is shown before startup;
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
- invoke `system.info`.

The terminal must not print `INSTALLATION COMPLETE` until the authenticated call is observed through Gateway -> Broker -> policy -> execution -> audit.

## 4. Safe real operations

Within the chosen delegated scope, exercise representative non-destructive operations such as:

- read/write one disposable test file;
- inspect an allowed service or Docker resource;
- run one bounded scoped shell/job if enabled;
- verify an out-of-scope file/resource is denied.

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

The human gate passes only when every applicable step above succeeds without weakening the documented security model.

If a problem appears:

1. record the failing step without secrets;
2. fix the product;
3. rerun the affected automated tests;
4. repeat the necessary human section;
5. only then promote/freeze `v0.1.0`.

Full/R5 maturity and long-running R4 reliability remain separate from this first stable Scoped release decision unless explicitly claimed.
