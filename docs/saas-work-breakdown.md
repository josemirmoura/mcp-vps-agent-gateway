# Portico Cloud SaaS work breakdown

Status: execution backlog  
Created: 2026-10-06

This document turns the commercial SaaS plan into implementable epics. Each epic has a dependency and an exit condition.

## Delivery principle

The smallest monetizable chain is:

Community node -> account/workspace -> node enrollment -> managed control plane -> audit -> billing -> paid Starter customer.

Anything outside this chain is deferred unless it is required for security, legal operation or reliability.

## Epic 0 — commercial and legal foundation

Dependencies: none.

Deliverables:

- final Community license;
- commercial-license template;
- trademark policy;
- pricing/packaging v1;
- Terms of Service;
- Privacy Policy;
- Acceptable Use Policy;
- DPA baseline;
- subprocessors register;
- tax/invoicing decision;
- support/SLA definitions.

Exit:

- Community/commercial boundary is legally coherent;
- paid checkout can link to final customer terms;
- stable Community release license is settled.

## Epic 1 — private repository and engineering foundation

Dependencies: Epic 0 can run in parallel, but no production launch before it finishes.

Deliverables:

- private portico-cloud repository;
- CODEOWNERS/review policy;
- CI;
- secret scanning;
- dependency scanning;
- staging and production deployment pipelines;
- dev/staging/prod configuration model;
- database migration tooling;
- feature flags;
- infrastructure as code;
- engineering ADR directory.

Exit:

- a trivial service can be deployed automatically to staging and promoted with auditable change history.

## Epic 2 — tenant data model

Dependencies: Epic 1.

Entities:

- user;
- organization;
- workspace;
- membership;
- role;
- subscription;
- entitlement;
- node;
- node_credential;
- agent;
- policy_reference;
- approval;
- task;
- audit_event;
- api_credential;
- webhook_endpoint;
- notification_preference.

Deliverables:

- PostgreSQL schema;
- migration history;
- tenant ownership rules;
- unique constraints;
- soft-delete/retention policy where appropriate;
- test fixtures.

Exit:

- automated cross-tenant tests prove that one tenant cannot read or mutate another tenant's data.

## Epic 3 — authentication and account lifecycle

Dependencies: Epics 1–2.

Deliverables:

- signup;
- email verification;
- login/logout;
- password policy;
- password reset;
- session list;
- revoke current/all sessions;
- brute-force protection;
- recovery token single-use and expiry;
- TOTP MFA;
- WebAuthn/passkeys;
- recovery codes;
- account deletion/export.

Security acceptance:

- reset flow does not leak account existence;
- sessions rotate correctly;
- MFA recovery is auditable;
- CSRF/session fixation tests pass;
- rate-limit tests pass.

Exit:

- a new user can create, secure, recover and delete an account without operator intervention.

## Epic 4 — organizations, workspaces and team lifecycle

Dependencies: Epics 2–3.

Deliverables:

- create workspace;
- invite member;
- accept/decline invitation;
- Owner/Admin/Operator/Auditor/Billing roles;
- remove/suspend member;
- ownership transfer;
- organization/workspace deletion workflow;
- tenant-scoped authorization middleware.

Exit:

- all authorization is server-side;
- role tests cover every protected route;
- invitations cannot be replayed after acceptance/expiry.

## Epic 5 — node identity and enrollment

Dependencies: Epic 4 plus stable public protocol contract.

Deliverables:

- one-time enrollment token;
- node key/identity generation;
- enrollment exchange;
- node credential rotation;
- heartbeat;
- last-seen state;
- revoke node;
- protocol/capability version negotiation;
- installer enrollment command.

Exit:

- a clean Linux node can join a workspace using one short-lived enrollment token without sending SSH/root credentials to Cloud.

## Epic 6 — managed control plane

Dependencies: Epic 5.

Deliverables:

- node registry;
- capability discovery;
- route selection;
- task envelope;
- destination-node authentication;
- local Broker reauthorization;
- timeout/cancel;
- reconnect semantics;
- offline-state handling;
- correlation IDs.

Exit:

- two nodes can execute authorized requests from one workspace;
- wrong-node/wrong-tenant requests fail closed;
- taking one node offline does not break the other.

## Epic 7 — durable tasks and AI delegation substrate

Dependencies: Epic 6.

Deliverables:

- task submit/status/result/cancel;
- durable queue;
- idempotency;
- retries with replay rules;
- task provenance;
- agent identity;
- agent capability declaration;
- same-node delegation;
- cross-node delegation.

Exit:

- task state survives service restart;
- delegation never inherits more authority than explicitly granted.

## Epic 8 — centralized audit and event pipeline

Dependencies: Epics 4–7.

Deliverables:

- immutable/correlation-friendly audit events;
- tenant/node/agent/task linkage;
- searchable storage;
- retention enforcement;
- export;
- tamper-evidence strategy;
- plan-specific retention;
- Business SIEM export interface.

Exit:

- every security-sensitive action can be reconstructed across client -> Cloud -> node -> agent chain.

## Epic 9 — customer portal

Dependencies: Epics 3–8.

Pages:

- onboarding;
- home/dashboard;
- nodes;
- node detail;
- agents;
- policies/approvals;
- tasks/jobs;
- audit;
- team;
- billing;
- API credentials;
- webhooks;
- support;
- account/security settings.

Exit:

- a Starter customer can onboard and operate the product without CLI use except the one node-install/enroll command.

## Epic 10 — billing engine

Dependencies: Epics 2–4 and initial pricing.

Provider: Stripe Billing unless a concrete market/payment constraint supersedes it.

Deliverables:

- products/prices;
- Checkout;
- monthly/annual subscriptions;
- customer billing portal;
- verified webhooks;
- idempotent webhook ingestion;
- subscription state machine;
- upgrade/downgrade;
- proration;
- cancel/resume;
- invoices/receipts;
- dunning;
- coupon support;
- refund/admin workflow;
- tax treatment integration.

Exit:

- sandbox customer can complete signup -> paid -> upgrade -> failed payment -> recovery -> cancel without database edits.

## Epic 11 — entitlement service

Dependencies: Epic 10.

Deliverables:

- plan-independent entitlement keys;
- limits for users/nodes/audit/API/webhooks;
- effective-entitlement endpoint;
- cached entitlement evaluation;
- billing event reconciliation;
- grace-period behavior;
- admin override with audit.

Exit:

- no application code needs to ask "is plan X?";
- it asks for explicit capabilities/limits.

## Epic 12 — API credentials and webhooks

Dependencies: Epics 4, 8, 11.

Deliverables:

- API key creation/revocation;
- scoped API credentials;
- secret shown once;
- webhook endpoint registration;
- signing secret;
- delivery retries;
- dead-letter/replay;
- delivery history.

Exit:

- Pro customer can integrate Portico without handing over a human session credential.

## Epic 13 — notifications

Dependencies: Epics 3, 4, 8, 10.

Deliverables:

- verification;
- reset;
- invitation;
- payment failure;
- plan changes;
- node offline;
- security events;
- approval expiration;
- incident communication;
- user preferences.

Exit:

- mandatory security/billing notifications cannot be disabled accidentally.

## Epic 14 — internal admin/support console

Dependencies: core SaaS.

Deliverables:

- tenant lookup;
- health/usage view;
- subscription and entitlement inspection;
- suspension/reactivation;
- feature flags;
- billing event view;
- support case linkage;
- incident controls;
- privileged action audit.

Exit:

- support staff can resolve normal operational cases without direct production-database access.

## Epic 15 — security hardening

Dependencies: continuous; mandatory before paid beta.

Deliverables:

- Cloud threat model;
- KMS/secrets architecture;
- SAST/dependency/image scans;
- DAST;
- WAF/edge protection;
- abuse/rate limits;
- CSP/security headers;
- CSRF/XSS/SSRF defenses;
- webhook signature/replay defenses;
- backup and restore test;
- incident runbook;
- key rotation;
- break-glass procedure;
- admin-action audit.

Exit for paid beta:

- critical/high findings are triaged and release policy is explicit;
- restore drill succeeds;
- cross-tenant negative suite is green.

Exit for Business GA:

- penetration test;
- formal access reviews;
- DR objectives;
- vendor/subprocessor review;
- customer security package.

## Epic 16 — observability and SRE

Dependencies: starts with Epic 1.

Deliverables:

- structured logs;
- metrics;
- traces;
- correlation IDs;
- tenant-safe telemetry;
- SLOs;
- alerts;
- synthetic checks;
- status page;
- capacity dashboards;
- COGS dashboards;
- incident timeline.

Initial SLO candidates to validate:

- control-plane API availability;
- node routing success;
- task acceptance latency;
- task completion reliability;
- audit ingestion completeness.

Exit:

- on-call can detect, diagnose and recover a representative failure without querying random containers manually.

## Epic 17 — Cloud Starter launch

Dependencies: Epics 0–16 at minimum acceptable depth.

Deliverables:

- public landing/pricing;
- self-service signup;
- node onboarding;
- paid billing;
- docs;
- support channel;
- analytics funnel;
- cancellation/export.

Private-beta target: 5–10 design partners.

Exit:

- first recurring paid customers;
- measured COGS;
- measured support burden;
- validated activation funnel.

## Epic 18 — Cloud Pro

Dependencies: Starter evidence.

Deliverables:

- multiple environments/workspaces;
- richer roles/approvals;
- 90-day or validated longer audit;
- API/webhooks;
- policy templates;
- advanced agent controls;
- priority support;
- higher quotas.

Exit:

- recurring Pro customers demonstrate expansion willingness and acceptable gross margin.

## Epic 19 — Business Cloud

Dependencies: Pro stability plus enterprise demand.

Deliverables:

- SSO/SAML/OIDC;
- SCIM;
- custom roles;
- SIEM export;
- custom retention;
- dedicated tenant;
- private networking/VPC options;
- BYOK/KMS option;
- SLA instrumentation;
- data-residency options;
- DPA/MSA workflow;
- security questionnaire package;
- named support/onboarding.

Exit:

- at least two production pilots complete security review and deploy successfully.

## Epic 20 — GA operating model

Dependencies: Starter/Pro production evidence.

Deliverables:

- release/support cadence;
- incident process;
- DR exercise;
- status communications;
- customer success process;
- billing reconciliation;
- churn/cancellation feedback;
- public roadmap rules;
- deprecation policy;
- protocol compatibility policy.

Exit:

- acquisition, activation, billing, support, upgrades, cancellation and recovery do not require founder-level database intervention.

## Dependency spine

Critical path:

Epic 0 -> 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 8 -> 9 -> 10 -> 11 -> 17.

Parallelizable after foundations:

- Epic 7 AI delegation;
- Epic 12 API/webhooks;
- Epic 13 notifications;
- Epic 14 admin;
- Epic 15 security;
- Epic 16 SRE.

Do not let Business features delay the Starter critical path.

## Definition of done for every SaaS epic

An epic is not complete when the UI looks finished.

It is complete when:

- code is reviewed;
- migrations are reversible or safely forward-only;
- authorization tests exist;
- negative tests exist;
- observability exists;
- runbook exists where operationally relevant;
- documentation exists;
- secrets are not committed;
- CI is green;
- staging evidence exists;
- security implications are reviewed.
