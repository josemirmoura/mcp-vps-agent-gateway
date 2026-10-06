# Active execution plan — stable v0.1.0 to multi-node Portico

**Status:** active  
**Started:** 2026-10-06  
**Immediate objective:** close the commercial licensing gate, cut the final licensed release candidate, pass owner acceptance, and freeze a trustworthy stable `v0.1.0` before implementing the multi-node control plane.  
**North star:** one vendor-neutral Portico control plane connecting multiple policy-authoritative nodes and AI agents, with audited AI-to-AI delegation.

This plan is the execution sequence. It does not replace the architecture, security model or multi-node roadmap.

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

Status: **complete**

Completed on 2026-10-06:

1. publish the exact merged commit as `v0.1.0-rc.6` through the validated release workflow;
2. verify the immutable tag and GitHub prerelease;
3. verify release artifacts, checksums and the release validation result;
4. use that exact tag for the owner clean-install acceptance.

The operator acceptance must never test a moving branch or a different commit from the release candidate intended for stable promotion.

## Phase 1.7 — commercial licensing and product-boundary gate

Status: **new blocker before stable v0.1.0**

Commercial strategy now requires a public Community edition that is source-available and free for permitted non-commercial use, plus proprietary Portico Cloud services.

Before stable:

1. obtain legal review of the intended source-available license;
2. confirm copyright/contributor authority for relicensing future versions;
3. define permitted non-commercial use and commercial-use boundary;
4. define trademark policy;
5. define the public Community repository versus private portico-cloud repository boundary;
6. update LICENSE, NOTICE, README, site and installer notices consistently;
7. bump to a new RC after any license change;
8. run the full automated matrix and publish that exact immutable RC.

RC6 remains the validated technical baseline. If the license changes, RC6 is not the stable-promotion candidate because it was published under Apache-2.0.

See commercial-saas-plan.md and licensing-decision.md.

## Phase 2 — owner clean-install acceptance of final licensed RC

Status: **blocked on Phase 1.7**

Use the exact immutable release-candidate tag produced after the licensing gate. If the only post-RC6 material change is licensing/product packaging, the next candidate is expected to be RC7, but the exact tag is determined by the actual release state.

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

After owner acceptance of the final licensed RC passes:

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

## Parallel commercial/SaaS productization track

Detailed plan: commercial-saas-plan.md. Implementation backlog: saas-work-breakdown.md.

The SaaS track starts with specification now, but implementation should not distract from the stable Community release gate.

Sequence:

1. SaaS Gate 0 — competitive benchmark, licensing, packaging/pricing, public/private boundary and unit economics;
2. SaaS Gate 1 — private portico-cloud repository, staging, identity, tenant/workspace model and portal shell;
3. SaaS Gate 2 — managed node enrollment, registry, routing and centralized audit metadata;
4. SaaS Gate 3 — full account lifecycle including password recovery, MFA/passkeys, invitations and sessions;
5. SaaS Gate 4 — Stripe billing, subscriptions, entitlements, invoices, dunning and upgrades/downgrades;
6. SaaS Gate 5 — Cloud Starter private beta;
7. SaaS Gate 6 — Cloud Pro;
8. SaaS Gate 7 — Business Cloud pilots with SSO/SCIM/SIEM/SLA/dedicated options;
9. SaaS Gate 8 — GA.

Commercial packaging hypothesis:

- Community: $0, self-hosted, source-available, non-commercial;
- Cloud Starter: initial hypothesis US$ 15 annual-equivalent / US$ 19 monthly;
- Cloud Pro: initial hypothesis US$ 49 annual-equivalent / US$ 59 monthly;
- Business Cloud: custom annual contract.

Prices remain validation hypotheses until COGS and willingness-to-pay are measured.

## Separate maturity tracks

These do not block the first stable Scoped release unless the project explicitly begins claiming them:

### R4 long-running production maturity

Requires measured real deployment history, recovery drills, monitoring and incident evidence.

### R5 Full/admin maturity

Requires production-tested temporary elevation, revoke-all, network separation, remote audit anchoring and administrative recovery.

## Current stop point requiring owner

The next unavoidable owner intervention is the **Phase 1.7 licensing decision/legal review**. After the final license instrument is approved, automation can update the repository, cut the next RC and rerun CI. The following unavoidable human gate is Phase 2, which requires the real browser/OAuth/native-confirmation acceptance.
