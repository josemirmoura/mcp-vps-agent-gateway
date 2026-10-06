# Active execution plan — stable v0.1.0 to multi-node Portico

**Status:** active  
**Started:** 2026-10-06  
**Immediate objective:** freeze a trustworthy stable `v0.1.0` before implementing the multi-node control plane.  
**North star:** one vendor-neutral Portico platform connecting multiple policy-authoritative nodes and AI agents, with audited AI-to-AI delegation, a non-commercial Community edition and a complete commercial Portico Cloud SaaS.

This plan is the execution sequence. It does not replace the architecture, security model or multi-node roadmap.

Commercial planning is detailed in:

- [competitive-commercial-benchmark.md](competitive-commercial-benchmark.md);
- [licensing-commercial-boundary.md](licensing-commercial-boundary.md);
- [saas-commercialization-roadmap.md](saas-commercialization-roadmap.md).

## Operating rule

Work advances gate by gate.

A gate is complete only when its code/documentation, automated validation and required real-environment evidence exist. Disposable GitHub runners may prove reproducibility, but they do not replace owner-operated checks that depend on the real target ChatGPT account, DNS, OAuth/browser flow or native human-confirmation UI.

No multi-node production code is merged before the stable v0.1 release gate passes.

## Phase 0 — baseline reconciliation

Status: **complete**

Evidence:

- `main` is on `v0.1.0-rc.6`;
- latest reference implementation CI on 2026-10-06 is green;
- real ChatGPT Web + integrated OAuth path has prior audited evidence;
- the operator's ChatGPT Plus environment completed a real scoped write/read/delete proof on 2026-10-06;
- multi-node/multi-AI vision and roadmap are recorded;
- current live VPS Portico reports healthy Broker/Gateway state, `scoped` mode and valid audit chain.

## Phase 1 — RC6 closeout branch

Status: **complete**

Work:

1. reconcile stale RC3/product-surface documentation with RC6;
2. make ChatGPT compatibility evidence-based rather than plan-name based;
3. keep official OpenAI documentation and operator-specific observed capability clearly separated;
4. publish this active execution plan;
5. open a dedicated closeout PR;
6. let all pull-request CI/acceptance workflows run;
7. inspect failures and fix them without weakening security;
8. merge only when every required automated gate is green.

Success criteria:

- documentation link/installation-contract checks pass;
- reference implementation CI passes;
- Docker package acceptance passes;
- lifecycle/guided-install acceptance passes;
- integrated-auth acceptance passes;
- full Scoped acceptance passes;
- architecture/security checks that apply to the changed files pass;
- no secret or environment-specific credential is committed.

## Phase 1.5 — publish immutable RC6 prerelease

Status: **in progress / automated release workflow**

After the closeout PR is green and merged:

1. publish the exact merged commit as `v0.1.0-rc.6` through the validated release workflow;
2. verify the immutable tag and GitHub prerelease;
3. verify release artifacts, checksums and the release validation result;
4. use that exact tag for the owner clean-install acceptance.

The operator acceptance must never test a moving branch or a different commit from the release candidate intended for stable promotion.

## Phase 2 — owner clean-install acceptance of RC6

Status: **blocked on owner interaction after Phase 1**

Use exactly `v0.1.0-rc.6` on a clean supported target chosen for acceptance.

Required real-world evidence:

1. clean guided install;
2. DNS + HTTPS;
3. integrated OAuth/OIDC;
4. real ChatGPT connection;
5. audited `system.info`;
6. discovery-only scope listing;
7. native MCP elicitation for a disposable root;
8. scoped read/write plus representative bounded operation;
9. out-of-scope denial;
10. protected `.env` denial;
11. temporary protected-file approval and revocation;
12. diagnostics/audit bundle review;
13. safe remove + reinstall;
14. controlled update/rollback evidence;
15. purge only after evidence is captured.

This phase intentionally requires a human for browser/OAuth/native confirmation and any destructive lifecycle choice on the selected target. Automation must not simulate those decisions and call the gate complete.

## Phase 3 — stable v0.1.0 promotion

Status: **blocked on Phase 2**

After owner acceptance passes:

1. freeze the accepted commit;
2. set `VERSION` to `0.1.0`;
3. update CHANGELOG/status/release docs;
4. run release validation;
5. create stable `v0.1.0` tag/release;
6. verify checksums, SBOM/provenance and published artifacts;
7. verify Pages/Wiki/storefront point to stable instructions;
8. close or rewrite issue #4 so its remaining state matches reality.

Stable v0.1.0 claims only the documented Scoped Linux use case. It does not claim R4 long-running production maturity or R5 Full/admin maturity.

## Phase 4 — Gate B: second Linux node

Status: **planned; starts only after stable v0.1.0**

Goal: prove one control plane can operate the existing VPS and SRV-IA while each node remains locally authoritative.

Minimum implementation:

- introduce stable `node_id`;
- create node registry and authenticated heartbeat;
- expose `node.list`, `node.info`, `node.health`, `node.capabilities`;
- add node-aware routing without duplicating the tool catalog per machine;
- re-authorize every routed operation at the destination node;
- preserve local Broker policy and local audit;
- aggregate only the audit metadata required for cross-node provenance;
- define offline/reconnect behavior.

Required acceptance:

- VPS and SRV-IA both registered;
- independent policies;
- authorized read/write on each;
- bounded shell/job on each;
- wrong-node resource request denied;
- revoking one node does not alter the other;
- one node offline does not break the other;
- audit clearly identifies subject, node, tool, resource and result.

GitHub-hosted runners should first simulate multiple independent nodes. Real SRV-IA acceptance follows only after the simulation gate is green.

## Phase 5 — Gate C/D: stable multi-node catalog and local data

Status: **planned**

After Gate B:

- stabilize node-aware tool schemas;
- add RAG and database adapters on SRV-IA;
- keep retrieval close to data;
- enforce result-size and secret boundaries;
- add negative tests for cross-node authorization and data exfiltration.

## Phase 6 — Gate E/F: Windows and compute

Status: **planned**

- implement native Windows Node for cockpit-i7;
- filesystem, PowerShell/jobs, services, processes and local policy;
- add GPU/compute discovery and durable jobs;
- enforce quotas, timeout, cancellation and audit.

Windows is not represented as WSL-only.

## Phase 7 — Gate G: AI-to-AI delegation

Status: **planned**

Implement only after multi-node routing and policy isolation are proven:

- agent registry and explicit trust;
- versioned capability declaration;
- durable task submit/status/result/cancel;
- same-node delegation;
- cross-node delegation;
- minimum necessary context;
- explicit task authority;
- provenance across the full delegation chain;
- no privilege inheritance;
- agent output remains untrusted data.

Required proofs include:

- ChatGPT -> local specialist -> result;
- local orchestrator -> remote specialist -> result;
- multi-hop delegation without authority escalation.

## Phase 8 — Gate H/I: second AI client and hardening

Status: **planned**

- validate at least one non-OpenAI compatible client;
- verify vendor-specific adapters do not become the security boundary;
- node credential rotation and revocation;
- replay and spoofing tests;
- compromised-node containment;
- control-plane backup/restore;
- audit recovery and integrity checks.

## Commercial product track after stable v0.1.0

The commercial SaaS track starts only after the exact stable v0.1.0 gate is passed. It then runs in parallel with the multi-node platform track.

### Commercial Gate S0 — licensing and repository split

- legally review the move from future Apache releases to a source-available non-commercial Community license;
- preserve historical Apache-2.0 grants/tags;
- inventory contributor rights before relicensing;
- establish CLA terms that preserve dual-licensing rights;
- keep node-side security-critical code public/inspectable;
- create private `portico-cloud` repository;
- create private `portico-infra` repository;
- define optional private enterprise deployment overlays;
- publish trademark and commercial-license policy.

The preferred candidate for legal review is PolyForm Noncommercial 1.0.0. If commercial use is prohibited, the Community edition must be described as source-available/community source rather than OSI open source.

### Commercial Gate S1 — SaaS skeleton

- production/staging environments;
- web application;
- managed user authentication;
- email verification;
- password reset/recovery;
- MFA/passkeys;
- organization/workspace tenancy;
- Postgres system of record;
- CI/CD;
- observability;
- transactional email;
- admin/backoffice skeleton.

### Commercial Gate S2 — cloud node enrollment

- one-time enrollment token;
- node cryptographic identity;
- authenticated outbound connection;
- node registry;
- heartbeat;
- rotate/revoke;
- health and version dashboard;
- tenant isolation tests.

### Commercial Gate S3 — hosted Portico control plane alpha

- hosted MCP endpoint;
- workspace-aware routing;
- local Broker reauthorization on every operation;
- durable operation/task IDs;
- cloud audit metadata;
- node offline/reconnect behavior;
- no cross-tenant resource access.

This gate converges with Platform Gate B. The first cloud alpha may support Linux only.

### Commercial Gate S4 — billing and entitlements

- Stripe products/prices;
- Checkout;
- monthly/annual subscriptions;
- subscription lifecycle;
- upgrades/downgrades/proration;
- Customer Portal;
- invoices;
- payment-method updates;
- failed-payment recovery and grace periods;
- signed/idempotent webhooks;
- entitlement engine;
- usage meters and spend visibility.

Launch pricing hypotheses:

- **Cloud Pro:** US$19/month or US$190/year;
- **Cloud Team:** US$59/month or US$590/year;
- **Business:** custom annual contract.

Final prices require cost telemetry and willingness-to-pay validation.

### Commercial Gate S5 — first paid Cloud Pro

Minimum paid scope:

- secure account lifecycle;
- one workspace;
- commercial subscription;
- up to 3 Linux nodes;
- hosted MCP/control plane;
- current typed node operations subject to local policy;
- 30-day cloud audit index;
- API key management;
- billing portal;
- cancellation/export/delete;
- monitoring/backups/support.

First revenue must not wait for Windows, GPU scheduling, productized RAG or multi-hop AI-to-AI.

### Commercial Gate S6 — Cloud Team

- up to 10 included nodes;
- 5 users;
- invites;
- Owner/Admin/Operator/Auditor/Billing roles;
- multiple workspaces/projects;
- shared fleet;
- shared agents;
- policy templates;
- 90-day audit;
- webhooks;
- priority support.

### Commercial Gate S7 — Business pilot

- SSO;
- SCIM;
- SIEM/audit export;
- custom retention;
- custom domains;
- dedicated relay or single-tenant/VPC deployment;
- optional on-prem control plane;
- DPA;
- SLA;
- security questionnaire process;
- onboarding/professional services;
- commercial order form.

### Commercial Gate S8 — SaaS GA

Require:

- pricing/unit economics validated;
- subscription failure paths tested;
- tenant-isolation security review;
- backup/restore drills;
- incident response;
- status page;
- Terms/Privacy/AUP/commercial license/trademark package;
- customer export/deletion;
- support operations;
- product analytics without customer secrets;
- reliable Node/cloud compatibility policy;
- at least one external paying-customer cohort using the self-serve funnel.

### SaaS complete definition

A new customer must be able to discover Portico, register, verify/recover identity, create a workspace, pay, enroll nodes, connect an AI, operate within policy, invite teammates according to plan, inspect audit/usage, manage billing, get support, cancel and export/delete data without manual backend intervention.

The operator must be able to deploy safely, monitor, restore backups, reconcile billing, handle incidents, suspend abuse and support customers without reading node secrets.

## Separate maturity tracks

These do not block the first stable Scoped release unless the project explicitly begins claiming them:

### R4 long-running production maturity

Requires measured real deployment history, recovery drills, monitoring and incident evidence.

### R5 Full/admin maturity

Requires production-tested temporary elevation, revoke-all, network separation, remote audit anchoring and administrative recovery.

## Current stop point requiring owner

The next unavoidable owner interaction is **Phase 2**, after the RC6 closeout PR is green and merged.

Until then, development should continue without asking the owner to perform manual terminal/browser work.
