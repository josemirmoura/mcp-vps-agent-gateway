# Commercialization and SaaS plan

Status: planning baseline
Created: 2026-10-06

This document defines the commercial productization track for Portico. Architecture and security documents remain authoritative for runtime behavior.

## Product family

- Portico Community: self-hosted, source-available, free for permitted non-commercial use.
- Portico Cloud Starter: managed control plane for individuals and very small teams.
- Portico Cloud Pro: managed control plane for technical teams requiring stronger governance and longer audit.
- Portico Business Cloud: custom annual contract with enterprise identity, governance, SLA and deployment options.

A future paid Commercial Self-Hosted edition may be added after Cloud demand is validated. It is not required for the first SaaS launch.

## Licensing model

A license that forbids commercial use is not Open Source under the Open Source Definition. Public positioning must therefore use the terms source-available and Community edition, not OSI Open Source.

Recommended direction:

- keep Community source code public;
- permit personal, hobby, research and other explicitly permitted non-commercial uses;
- require a commercial license or Portico Cloud subscription for commercial exploitation;
- keep Cloud proprietary services outside the Community repository;
- define trademark rights separately from source-code rights.

Candidate standardized license: PolyForm Noncommercial 1.0.0. Final selection requires legal review before replacing Apache-2.0.

Existing Apache-2.0 releases remain usable under the terms under which they were published. A later license change cannot revoke those grants.

Before the first stable release, if commercial restriction is a core strategy:

1. review copyright/contributor ownership;
2. choose the source-available license;
3. define exactly what commercial use is prohibited;
4. decide treatment of universities, public research, nonprofits and government;
5. define commercial licensing terms;
6. create a trademark policy;
7. update LICENSE, NOTICE, README, site and installer notices together;
8. publish a new release candidate after the license change;
9. repeat release and operator-acceptance gates against that exact candidate.

Recommendation: do not publish stable v0.1.0 under Apache-2.0 if commercial-use restriction is strategic.

References:

- OSI Open Source Definition: https://opensource.org/osd
- OSI FAQ: https://opensource.org/faq
- PolyForm Noncommercial: https://polyformproject.org/licenses/noncommercial/1.0.0

## Competitive benchmark

Checked 2026-10-06. Revalidate official sources before public launch.

| Product | Packaging / pricing observed | Product lesson for Portico |
| --- | --- | --- |
| Tailscale | Personal $0; Standard $8/user/month; Premium $18/user/month; Enterprise custom | Free personal entry plus clear paid governance ladder works for secure infrastructure products |
| Twingate | Starter free; Teams about $5/user/month annual; Business about $10/user/month annual; Enterprise custom | Personal/team/business segmentation is easy to understand |
| Cloudflare Access | Free; $7/user/month annual; contract custom | Free entry, simple self-service and enterprise contract coexist |
| Teleport | Community plus Enterprise; usage based on active users and protected resources; Cloud and self-hosted | Strong precedent for monetizing infrastructure identity, protected resources, audit and MCP access |
| Arcade | Free; Team $25/month plus usage; Enterprise custom | Agent infrastructure can combine platform fee and usage |
| Composio | Free; Pro $29/month plus usage; Enterprise custom | MCP/tool infrastructure supports a low platform fee with metered expansion |
| Pipedream | Free development; Startup $99/month; Business custom | Production integration infrastructure can sustain a higher platform fee once operationally critical |

Official benchmark sources:

- https://tailscale.com/pricing
- https://www.twingate.com/pricing
- https://www.cloudflare.com/pt-br/sase/products/access/
- https://goteleport.com/pricing/
- https://goteleport.com/pricing/guide/
- https://www.arcade.dev/pricing/
- https://composio.dev/pricing/
- https://pipedream.com/pricing

## Market conclusions

Portico should not charge only per seat. Machines, agents, nodes and delegated jobs create value even with one human operator.

Recommended commercial metric:

- workspace subscription;
- included users;
- included active nodes;
- included centralized audit/relay allowance;
- optional overages only after real COGS data exists.

Enterprise value should concentrate in SSO, SCIM, RBAC, JIT approvals, audit export, SIEM, SLA, dedicated deployment, data residency, advanced identity and support.

Portico differentiation versus network-access products:

- AI-native machine control;
- MCP/client neutrality;
- node-local reauthorization;
- AI-to-AI delegation;
- local RAG/database/compute access;
- durable tool/job execution.

Portico differentiation versus agent-tool platforms:

- owner-controlled machines remain authoritative;
- infrastructure operations are first-class;
- local execution and data locality;
- heterogeneous multi-node environments;
- the security boundary lives outside the LLM.

## Packaging hypothesis

These are validation hypotheses, not final prices.

### Portico Community

Price: $0.

Target: individual developers, homelabs, students, research and technical evaluation.

Includes local node runtime, Gateway, Broker, local policy/audit, install/update tools and public interoperability contracts. No managed Cloud control plane, SLA or commercial-use grant.

### Portico Cloud Starter

Initial pricing hypothesis: US$ 15/month billed annually or US$ 19 monthly.

Initial entitlement hypothesis:

- 1 workspace;
- up to 3 users;
- up to 5 active nodes;
- managed control plane;
- node enrollment and health;
- secure discovery/routing;
- basic agent routing when available;
- 30-day centralized audit;
- standard email support.

### Portico Cloud Pro

Initial pricing hypothesis: US$ 49/month billed annually or US$ 59 monthly.

Initial entitlement hypothesis:

- up to 10 users;
- up to 25 active nodes;
- multiple environments/workspaces;
- advanced policy templates;
- 90-day audit retention;
- team roles and approval policies;
- API and webhooks;
- scheduled exports/backups;
- priority support;
- higher relay/job limits;
- advanced agent registry/delegation controls when available.

### Portico Business Cloud

Pricing: custom annual contract.

Candidate capabilities:

- custom users, nodes and agents;
- SSO/SAML/OIDC;
- SCIM;
- custom RBAC and approval policies;
- dedicated tenant and optional dedicated region;
- private networking/VPC connectivity;
- BYOK/KMS options;
- SIEM/audit export;
- custom retention and data residency;
- SLA;
- onboarding and migration;
- named support;
- security review package;
- optional professional services;
- DPA/MSA;
- future customer-VPC or air-gapped deployment.

Do not publish a Business price floor until pilot sales establish willingness to pay and support cost.

## Unit economics

Measure COGS per tenant for control-plane compute, database, object storage/audit, relay bandwidth, transactional email, identity provider, observability, backups, support, payment processing and any hosted compute.

Initial targets:

- self-service gross margin at least 80%;
- Business gross margin target at least 70% after support/professional-service allocation;
- no unlimited feature with material unbounded COGS.

## Repository split

Use two primary repositories.

### Public Community repository

Keep the current public repository as the Community/runtime source of truth.

Public contents:

- Linux node runtime;
- Gateway and Broker;
- CLI and install/update/remove tooling;
- public protocol/API schemas;
- public SDK needed for interoperability;
- policy schema and security model;
- examples and Community docs;
- Community release artifacts;
- compatibility tests.

Do not place in the public repository:

- Cloud tenant control plane;
- billing implementation;
- commercial entitlement logic;
- internal admin console;
- abuse/fraud tooling;
- private production infrastructure;
- Business-only policy modules;
- production secrets.

### Private Cloud repository

Proposed private repository: josemirmoura/portico-cloud.

Suggested structure:

- apps/portal
- apps/admin
- services/control-plane
- services/identity-adapter
- services/billing
- services/entitlements
- services/node-registry
- services/routing
- services/audit
- services/notifications
- services/support
- packages/api-contracts
- packages/ui
- infra
- runbooks
- security
- docs

The private repository consumes tagged public node releases and versioned public protocol contracts. Avoid copying the whole public tree into the private repository.

## Compatibility contract

Cloud and public nodes require a versioned protocol with:

- node protocol version;
- minimum/maximum supported Cloud version;
- capability negotiation;
- migration compatibility;
- deprecation windows;
- signed node identity;
- explicit entitlement reporting.

Cloud must never grant a node a capability that local Broker policy denies.

## SaaS architecture

Core principle: Cloud routes identity, discovery, policy context and tasks; every destination node remains authoritative for local execution.

Initial components:

- Browser portal and public API;
- Identity;
- Billing and Entitlements;
- Tenant/Workspace service;
- Control Plane;
- Node Registry;
- Capability Discovery;
- Routing;
- Agent Registry;
- Durable Tasks;
- Approval Context;
- Audit/Event service;
- Notifications;
- Support/Admin.

Implementation bias:

- backend/control plane: Go;
- portal: React/Next.js or equivalent mature web stack;
- primary database: managed PostgreSQL;
- cache/coordination: Redis/Valkey only where needed;
- durable queue/event service: adopt proven managed infrastructure before building one;
- object storage: S3-compatible;
- secrets/KMS: managed secret manager and KMS;
- infrastructure: Terraform/OpenTofu;
- billing: Stripe Billing;
- transactional email: managed provider;
- observability: OpenTelemetry-compatible stack.

Do not introduce Kubernetes in the first SaaS milestone unless measured operational requirements justify it.

## Build versus buy rule

Portico should build the differentiating control plane and buy mature commodity infrastructure.

Build internally:

- node identity/enrollment semantics;
- multi-node routing;
- policy context;
- agent/task delegation;
- Portico audit/provenance;
- entitlements and product-specific authorization;
- Portico portal UX.

Prefer mature providers/components for:

- password authentication and credential storage;
- email verification/recovery;
- MFA/passkeys;
- transactional email;
- payment processing/subscriptions;
- tax calculation where appropriate;
- managed PostgreSQL/object storage;
- KMS/secrets;
- queue/event infrastructure;
- observability plumbing.

Do not implement password hashing/storage, card handling or email delivery infrastructure merely to avoid a dependency. Those are security-sensitive commodity layers.

The existing ZITADEL experience makes it a serious identity candidate, but the Cloud identity provider should be selected through a short architecture decision comparing operational cost, B2B organization support, MFA/passkeys, SSO/SCIM path, exportability and vendor lock-in.

## Identity and account lifecycle

A complete SaaS must include:

- email signup;
- email verification;
- login/logout;
- secure session management;
- password reset with single-use time-limited tokens;
- no account-existence leakage in recovery;
- brute-force/rate-limit protection;
- session revocation;
- device/session list;
- MFA with TOTP;
- passkeys/WebAuthn;
- recovery codes;
- invitation lifecycle;
- ownership transfer;
- account export/deletion;
- organization/workspace suspension.

Core data model:

- user;
- organization;
- workspace;
- membership;
- role;
- subscription;
- entitlement;
- node;
- agent;
- policy;
- approval;
- task;
- audit_event;
- api_credential.

Initial roles:

- Owner;
- Admin;
- Operator;
- Auditor;
- Billing.

Business later adds custom roles, resource-scoped RBAC, approval policies and separation of duties.

## Node enrollment

Target onboarding flow:

1. user creates workspace;
2. portal creates short-lived one-time enrollment token;
3. user runs one install/enroll command on the machine;
4. node generates or activates its own identity;
5. token exchanges for durable node credentials;
6. Cloud shows node online and capabilities;
7. operator grants local authority explicitly;
8. first audited operation proves end-to-end success.

Security requirements:

- enrollment token expires quickly and is one-time;
- no root credential leaves the node;
- node credentials rotate;
- node can be revoked;
- cloned credentials are limited/detectable;
- every task is re-authorized locally.

## Billing and entitlements

Stripe is the preferred first implementation unless a concrete regional/payment constraint proves otherwise.

Customer flows:

- select plan and checkout;
- monthly/annual subscription;
- payment method management;
- invoice/receipt history;
- upgrade/downgrade;
- cancel/resume;
- trial if adopted;
- coupons/promotions;
- failed-payment recovery/dunning;
- proration;
- taxes;
- refund/admin workflow.

Backend requirements:

- webhook signature verification;
- webhook idempotency;
- replay handling;
- subscription state machine;
- server-side entitlements;
- grace periods;
- delinquent-account policy;
- plan limits;
- usage metering;
- billing-admin audit.

Never gate security policy directly on an unverified payment webhook payload.

Centralize commercial feature decisions in an entitlement service. Example keys:

- cloud.nodes.max
- cloud.users.max
- audit.retention_days
- api.enabled
- webhooks.enabled
- sso.enabled
- scim.enabled
- business.sla
- business.dedicated_tenant

Do not scatter plan-name comparisons through application code.

## Customer portal

Minimum complete portal areas:

- Home: workspace health, nodes online/offline, recent tasks, alerts, onboarding;
- Nodes: enroll, search, health, OS/version, capabilities, last seen, revoke, update status;
- Agents: registry, capabilities, trust, node, version, enable/disable, task history;
- Policies/Approvals: effective authority, grants, requests, revoke, expiry, templates;
- Tasks/Jobs: running/completed/failed, logs/results, cancel, provenance;
- Audit: filters, subject, node, agent, tool, resource, decision, result, export;
- Team: members, invitations, roles, MFA status;
- Billing: plan, usage, invoices, payment method, upgrade/downgrade/cancel;
- API/Developer: API credentials, webhooks, SDK docs, delivery history;
- Support: docs, diagnostics workflow, ticket/contact, Business escalation.

## Internal admin portal

Private admin capabilities:

- tenant search;
- subscription/entitlement inspection;
- account suspension/reactivation;
- node/usage summaries;
- abuse signals;
- billing event inspection;
- feature flags;
- support-case linkage;
- status/incident tooling.

Any support impersonation must be explicit, time-limited and heavily audited; preferably omit it from the first version.

## Notifications

Support verification, password reset, invitation, payment failure, subscription changes, node-offline alerts, security events, approval expiry and Business incident communication.

## Multi-tenancy

Before paid launch prove:

- tenant IDs on every tenant-owned record;
- server-side tenant authorization;
- negative cross-tenant tests;
- no shared-cache key collisions;
- object-storage isolation;
- tenant-aware logs/metrics;
- tenant-specific encryption strategy where appropriate.

Business dedicated tenant is an option, not a substitute for correct shared-tenant isolation.

## Security program

Before public paid beta:

- Cloud threat model;
- secrets/KMS design;
- dependency/SAST/container scanning;
- DAST on public surface;
- WAF/edge controls;
- rate limits;
- CSP/secure headers;
- CSRF/XSS/SSRF controls;
- webhook verification;
- API abuse limits;
- backups and restore drill;
- incident-response runbook;
- vulnerability reporting;
- SBOM/dependency inventory;
- key rotation;
- break-glass procedure;
- audit of privileged operator actions.

Before Business GA:

- formal access reviews;
- vendor/subprocessor inventory;
- disaster-recovery objectives;
- penetration test;
- security questionnaire package;
- SOC 2 readiness roadmap if market pull justifies it.

## Legal, privacy and compliance

Before accepting paying customers:

- Terms of Service;
- Privacy Policy;
- commercial license/EULA;
- Community source license;
- trademark policy;
- Acceptable Use Policy;
- DPA template;
- subprocessors list;
- retention/deletion policy;
- account/data export;
- LGPD/GDPR operational workflow;
- cookie policy/consent where applicable;
- invoicing/tax treatment;
- Business support/SLA terms.

## Production operations

Required:

- dev/staging/prod;
- infrastructure as code;
- automated migrations;
- rollback;
- feature flags;
- progressive rollout where useful;
- status page;
- synthetic health checks;
- SLO/SLI;
- alerting;
- incident timeline;
- backup monitoring;
- restore tests;
- capacity and COGS dashboards.

## SaaS delivery gates

### SaaS Gate 0 — commercial specification

Deliver benchmark, licensing decision, packaging/pricing hypothesis, public/private boundary, Cloud threat-model draft and unit-economics model.

Exit: product can be explained in one page; free/paid boundary is explicit; no licensing contradiction remains.

### SaaS Gate 1 — private Cloud skeleton

Deliver private portico-cloud repository, IaC, staging, managed PostgreSQL, identity integration, tenant/workspace model, portal shell, CI/CD and observability baseline.

Exit: one internal user can register/login and create a workspace in staging.

### SaaS Gate 2 — managed node control plane

Deliver one-time node enrollment, node identity, registry/heartbeat, node list/health, stable routing, local reauthorization and centralized audit metadata.

Exit: two real nodes can be enrolled and controlled from one Cloud workspace with no cross-node authority leakage.

### SaaS Gate 3 — complete account lifecycle

Deliver signup, verification, login/logout, password reset, MFA/passkey, invitations, roles, session management and account deletion/export.

Exit: identity lifecycle passes automated and manual security acceptance.

### SaaS Gate 4 — billing and entitlements

Deliver Stripe Checkout, subscriptions, customer billing portal, webhook state machine, Starter/Pro entitlements, upgrades/downgrades/cancel, invoices, dunning and Billing role.

Exit: test customer completes the full paid lifecycle without manual database edits.

### SaaS Gate 5 — Cloud Starter private beta

Deliver onboarding, node enrollment, core dashboard, managed control plane, audit, billing and support.

Target 5–10 design partners. Measure time-to-first-node, time-to-first-authorized-action, weekly active workspaces, support burden, COGS and willingness to pay.

### SaaS Gate 6 — Cloud Pro

Deliver teams, roles, longer audit, API/webhooks, policy templates, multiple environments, advanced agent controls and priority support.

Exit: first recurring Pro customers with acceptable margin and support burden.

### SaaS Gate 7 — Business Cloud pilots

Deliver SSO, SCIM, custom retention, SIEM export, dedicated tenant option, SLA instrumentation, security review package, DPA/MSA process and onboarding playbook.

Exit: at least two pilot organizations complete security review and production deployment.

### SaaS Gate 8 — GA

Deliver production support process, status page, polished docs, migration/update policy, public pricing, legal docs and incident/DR exercises.

Exit: Starter/Pro can acquire, activate, bill, support and retain customers without operator database intervention.

## Metrics

North-star operational metric: successful policy-authorized tasks completed through Portico per active workspace.

Activation funnel:

- visit;
- signup;
- verified account;
- workspace created;
- node enrolled;
- authority granted;
- first successful tool/task;
- second active day;
- paid.

Commercial metrics:

- MRR/ARR;
- trial-to-paid;
- Community-to-Cloud conversion;
- Starter-to-Pro expansion;
- churn;
- ARPA;
- gross margin;
- support cost per tenant;
- CAC and payback when paid acquisition begins.

Reliability/security metrics:

- node online rate;
- routing success;
- task success;
- authorization correctness;
- cross-tenant isolation tests;
- audit completeness;
- incident count;
- recovery time.

## Focus rule

Do not build the entire SaaS before validating the critical path.

Minimum monetizable sequence:

1. stable Community runtime;
2. Cloud identity/workspace;
3. node enrollment;
4. managed routing/control;
5. centralized audit;
6. billing;
7. Starter paid beta.

SSO, SCIM, dedicated regions, deep compliance and large-enterprise operations follow paid self-service validation.
