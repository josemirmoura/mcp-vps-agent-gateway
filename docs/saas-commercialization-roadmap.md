# Portico Cloud — SaaS end-to-end roadmap

**Status:** commercial product plan  
**Starts after:** stable v0.1.0 single-node release  
**Purpose:** build a complete, production-grade SaaS around the Portico control plane without weakening node-local authorization.

## 1. Product model

Portico Cloud is the managed control plane for Portico Nodes.

The SaaS must let a customer:

1. discover Portico;
2. understand plans and pricing;
3. create an account;
4. verify email;
5. sign in securely;
6. recover access;
7. create an organization/workspace;
8. subscribe;
9. enroll a node;
10. connect an AI client;
11. define who/what may access each node;
12. execute authorized work;
13. inspect audit/history;
14. invite teammates;
15. change plan/payment method;
16. cancel/export/delete the account safely;
17. obtain support without exposing machine secrets.

The SaaS is a control plane. The destination node remains the execution-policy authority.

## 2. Launch editions

### Community

Self-hosted, source-available, non-commercial.

### Cloud Pro

Self-serve commercial tier for solo professionals and small deployments.

Target launch hypothesis:

- US$19/month;
- US$190/year;
- 3 nodes;
- 3 total workspace members;
- 30-day audit index;
- standard support.

### Cloud Team

Self-serve commercial tier for small teams.

Target launch hypothesis:

- US$59/month;
- US$590/year;
- 10 nodes;
- 5 users;
- 90-day audit index;
- richer RBAC/policy templates;
- priority support.

### Business

Custom annual contract.

Potential options:

- custom fleet/seat limits;
- SSO/SCIM;
- SIEM export;
- custom retention;
- dedicated relay;
- private VPC/single-tenant;
- on-prem control plane;
- custom domain;
- SLA;
- data residency;
- professional services.

Prices are product hypotheses until unit economics and willingness-to-pay are measured.

## 3. Public/private architecture

~~~text
PUBLIC / CUSTOMER NODE
Portico Node
  -> Broker / policy
  -> local audit
  -> local tools
  -> node identity
       |
       | outbound authenticated channel
       v
PRIVATE PORTICO CLOUD
Edge/API
  -> Identity + tenancy
  -> Control Plane
  -> Node Registry
  -> Router/Relay
  -> Entitlements
  -> Billing/Metering
  -> Audit Index
  -> Agent Registry
  -> Notifications
       |
       v
PORTICO WEB APP / MCP ENDPOINTS
~~~

Never make the cloud control plane a substitute for local Broker authorization.

## 4. SaaS domains

Recommended product surfaces:

- marketing site;
- documentation;
- application portal;
- API;
- hosted MCP endpoint;
- node enrollment endpoint;
- optional relay endpoint;
- status page;
- trust/security center;
- support/help center.

Keep production, staging and development isolated.

## 5. Identity and account lifecycle

Do not build password cryptography from scratch.

Use a mature managed identity provider initially. Candidates should be evaluated on B2B organizations, MFA/passkeys, machine auth, SSO, SCIM, exportability and price. Clerk and WorkOS are strong current candidates.

Mandatory user flows:

- sign up with email;
- email verification;
- secure password login if enabled;
- password reset;
- magic link or passkey option;
- Google/GitHub social login if useful;
- MFA;
- recovery codes;
- session management;
- logout all sessions;
- account email change with verification;
- account deletion;
- security event notifications.

Organization/workspace flows:

- create organization;
- rename;
- invite member;
- accept/reject invite;
- remove member;
- transfer ownership;
- leave organization;
- roles;
- custom permissions when plan supports them;
- verified domain/auto-join for Business;
- SSO/SCIM for Business.

Initial roles:

- Owner;
- Admin;
- Operator;
- Auditor;
- Billing.

Separate human identity from node/machine identity.

## 5.1 Identity-provider decision gate

Portico already has working ZITADEL-based OAuth/OIDC for the MCP path. Do not casually add a second identity stack.

Evaluate three routes before S1 implementation:

### Route A — ZITADEL for both portal and MCP authorization

Advantages:

- reuses the protocol already proven by Portico;
- organizations/multi-tenancy;
- username/password;
- passkeys;
- MFA;
- external identity providers;
- enterprise SSO;
- audit;
- machine/service identities;
- avoids synchronizing two independent identity sources.

Trade-off:

- either operate ZITADEL ourselves or accept its cloud pricing/operational dependency.

Current ZITADEL Cloud Pro documentation indicates a US$100 monthly base plus usage-based components; Enterprise adds stronger SLA/support/compliance options.

### Route B — managed SaaS auth such as Clerk/WorkOS plus a separate MCP authorization layer

Advantages:

- extremely fast polished SaaS account UX;
- strong B2B organization features.

Trade-off:

- creates identity synchronization and two authorization surfaces unless the architecture is carefully unified.

Current benchmarks:

- Clerk Pro starts around US$20/month billed annually and includes B2B organization primitives; higher Business/enterprise features cost more.
- WorkOS AuthKit currently offers a large free user allowance, while Enterprise SSO/Directory Sync connections are separately charged.

### Route C — self-hosted ZITADEL initially, migrate operational model later

Advantages:

- minimum change from current proven Portico;
- full control;
- low vendor cost during engineering.

Trade-off:

- identity becomes production infrastructure we must patch, back up, monitor and secure ourselves.

**Recommended engineering default for the first cloud prototype:** keep ZITADEL as the canonical identity/OAuth model until an explicit ADR proves a replacement materially improves total cost, UX or enterprise sales. Avoid building password storage/authentication code inside Portico.

## 6. Node identity and enrollment

Each Portico Node needs its own cryptographic identity.

Enrollment flow:

1. user logs into Portico Cloud;
2. selects workspace;
3. creates one-time node enrollment token;
4. installer uses that token once;
5. node generates or receives long-term credentials;
6. cloud records node identity/fingerprint;
7. enrollment token becomes unusable;
8. node establishes outbound authenticated connection;
9. local policy remains independent;
10. owner can revoke/rotate node credentials.

Required controls:

- one-time bootstrap tokens;
- short expiration;
- mTLS or equivalent strong machine authentication;
- credential rotation;
- revocation;
- node fingerprints;
- replay protection;
- duplicate-node detection;
- last-seen/health;
- software version;
- capability version;
- quarantine state;
- key compromise recovery.

## 7. Tenancy model

Every cloud record must be tenant-scoped.

Core entities:

- User;
- Organization;
- Membership;
- Workspace;
- Role;
- Subscription;
- Entitlement;
- Node;
- NodeCredential;
- Agent;
- AgentCapability;
- PolicyTemplate;
- Connection;
- Operation;
- Task;
- AuditEvent;
- ApiKey;
- WebhookEndpoint;
- UsageMeter;
- InvoiceReference;
- SupportCase reference.

Use opaque immutable IDs. Human-readable slugs are aliases, not authorization keys.

All database queries involving tenant data must be scoped by organization/workspace. Add negative cross-tenant tests from day one.

## 8. Billing and subscription system

Do not lock the billing provider before deciding who carries global tax/compliance responsibility.

### Billing-provider ADR

Evaluate at least:

**Stripe Billing / Payments**

- stronger direct control over customer/payment data and subscription mechanics;
- current Brazilian card pricing and Billing fees are transparent;
- good fit when Portico is ready to own tax registrations/filing or use separate tax services.

**Paddle Merchant of Record**

- currently advertises 5% + US$0.50 per checkout transaction;
- handles payments, subscription billing, global sales-tax/VAT compliance, fraud/chargebacks and buyer billing support as Merchant of Record;
- attractive for an early global SaaS when operational simplicity is worth the higher percentage fee.

Lemon Squeezy/Stripe Managed Payments may remain a third benchmark, but avoid supporting multiple billing providers in the MVP.

**Decision rule:** choose the provider that minimizes total operational/legal cost for the first 100 paying customers, not merely the lowest payment-processing percentage. Hide the provider behind a narrow internal billing interface so a future migration does not contaminate entitlement logic.

Required flows:

- pricing page;
- free/community explanation;
- checkout;
- monthly/annual billing;
- trial if used;
- subscription activation;
- plan upgrade;
- downgrade;
- proration;
- renewal;
- cancellation at period end;
- immediate cancellation where policy allows;
- payment method update;
- invoice history;
- tax/VAT fields;
- coupons/promotions if used;
- failed-payment recovery/dunning;
- grace period;
- refund workflow;
- customer billing portal;
- webhook reconciliation.

Never trust the browser to decide entitlement.

Webhook processing must be:

- signature verified;
- idempotent;
- durable;
- replay-safe;
- auditable.

Entitlements must be derived from authoritative subscription state and persisted separately enough to survive transient provider outages.

Suggested entitlement keys:

- max_nodes;
- max_members;
- max_workspaces;
- audit_retention_days;
- api_access;
- shared_agents;
- policy_templates;
- webhook_count;
- sso;
- scim;
- siem_export;
- custom_domain;
- dedicated_relay;
- private_deployment;
- support_tier.

## 9. Usage metering

Meter only what materially drives cost or value.

Recommended launch meters:

- active nodes;
- optional relay bytes;
- optional retained audit volume;
- optional cloud task executions if expensive.

Avoid billing normal MCP tool calls individually at launch unless economics force it. Infrastructure buyers prefer predictable cost.

Record usage with:

- tenant ID;
- meter type;
- quantity;
- timestamp window;
- source event ID;
- idempotency key.

Provide in-app usage dashboards before charging overages.

## 10. Web application

Recommended initial frontend:

- TypeScript;
- Next.js or equivalent mature framework;
- accessible component system;
- responsive desktop/mobile layout.

Core screens:

### Public

- home;
- product;
- security;
- pricing;
- docs;
- changelog;
- download/install;
- contact sales;
- status;
- sign in;
- sign up.

### Authenticated

- onboarding;
- organization/workspace switcher;
- overview dashboard;
- nodes;
- node details;
- node health;
- capabilities;
- agents;
- tasks/jobs;
- policies/delegations;
- audit;
- integrations;
- API keys;
- webhooks;
- team;
- usage;
- plan/billing;
- invoices;
- security/session settings;
- account/profile;
- support.

### Business/admin

- SSO setup;
- SCIM setup;
- SIEM destinations;
- retention;
- domains;
- dedicated deployment status;
- security contacts.

## 11. SaaS API

Version public APIs.

Suggested namespaces:

~~~text
/api/v1/account
/api/v1/organizations
/api/v1/workspaces
/api/v1/nodes
/api/v1/agents
/api/v1/tasks
/api/v1/audit
/api/v1/policies
/api/v1/api-keys
/api/v1/webhooks
/api/v1/usage
/api/v1/billing
~~~

Use:

- OpenAPI specification;
- typed errors;
- request IDs;
- idempotency keys for mutations;
- pagination;
- rate limits;
- audit of privileged SaaS actions.

The hosted MCP surface should map to the same underlying authorization domain rather than becoming a parallel permissions system.

## 12. Cloud control-plane services

Initial service boundaries should remain few.

A practical first deployment:

### Web/API service

- portal backend;
- organizations;
- billing;
- entitlements;
- user-facing APIs.

### Control Plane service

- node registry;
- capability registry;
- routing;
- heartbeat;
- node status;
- operation dispatch.

### Worker service

- webhooks;
- email;
- audit indexing;
- metering;
- scheduled lifecycle jobs.

### Relay service

Only when direct/outbound control-plane transport requires relaying high-volume or bidirectional data.

Do not split into many microservices before load or isolation requires it.

## 13. Data layer

Recommended baseline:

- PostgreSQL as system of record;
- object storage for larger immutable exports/bundles;
- no extra cache/queue until measured need.

Use the transactional outbox pattern for important asynchronous events.

Add Redis/NATS/SQS only when concrete throughput/latency requirements justify them.

Key database requirements:

- row/tenant isolation enforced in application and tests;
- encryption at rest;
- PITR;
- regular backups;
- migration discipline;
- migration rollback/recovery plans;
- separate production credentials;
- read replicas only when measured need exists.

## 14. Node-to-cloud transport

The existing public MCP transport remains MCP Streamable HTTP.

Node-to-control-plane transport is a separate design decision. Requirements:

- outbound-friendly from customer networks;
- TLS;
- strong node authentication;
- reconnect;
- backoff;
- replay protection;
- bounded message size;
- durable operation/task IDs;
- offline state;
- cancellation;
- no ambient privilege.

Evaluate HTTP/2 streaming, gRPC or another standards-based transport before inventing a custom protocol.

## 15. Audit architecture

Two layers:

### Node-local audit

Authoritative evidence of local execution/policy decision.

### Cloud audit index

Searchable fleet-level metadata.

Cloud audit event should include:

- tenant;
- human/service subject;
- client;
- node ID;
- tool;
- action;
- canonical resource;
- policy decision;
- grant/delegation;
- operation/task ID;
- result;
- timestamps;
- integrity/provenance references.

Do not automatically upload secret-bearing stdout/stderr. Default cloud audit to metadata and bounded redacted outputs.

Retention by plan can be commercial differentiation.

## 16. Notifications

Transactional email provider required.

Email classes:

- verify email;
- password/security recovery;
- invite;
- node enrolled/revoked;
- suspicious sign-in;
- billing failure;
- invoice/receipt;
- plan change;
- security advisory;
- Business SLA incident communications.

Later:

- Slack/Teams;
- generic webhooks;
- SMS only for high-value security needs.

## 17. API keys and service accounts

Users/organizations need non-human access.

Capabilities:

- create;
- name;
- scope;
- expiration;
- last-used;
- rotate;
- revoke.

Never show secret values again after creation.

Business should support service accounts and workload identity with tighter scopes than human users.

## 18. Admin/backoffice

A complete SaaS needs a private operator console.

Functions:

- locate account/org/subscription/node;
- view health without seeing customer secrets;
- suspend abusive tenant;
- revoke compromised tokens/nodes;
- inspect webhook failures;
- retry safe jobs;
- entitlement override with expiry;
- plan correction;
- support impersonation only when explicitly authorized and fully audited;
- feature flags;
- account deletion/export workflows;
- incident annotations.

Every support/admin action must be audited.

## 19. Support system

At launch:

- documentation;
- troubleshooting;
- contact form/email;
- ticket IDs;
- product diagnostics export with redaction.

Team:

- priority queue.

Business:

- onboarding;
- named support channel;
- response SLA;
- optional dedicated Slack/Teams;
- escalation procedure.

## 20. Product analytics

Track product behavior, never secrets.

Core funnel:

~~~text
visit pricing
 -> sign up
 -> verify
 -> create workspace
 -> install first node
 -> node online
 -> first successful MCP call
 -> second successful session
 -> second node/team invite
 -> paid conversion
~~~

North-star operational metrics:

- signup-to-first-node time;
- first-node-to-first-successful-tool-call time;
- activation rate;
- paid conversion;
- MRR/ARR;
- churn;
- expansion;
- active nodes;
- nodes per paid workspace;
- successful operation rate;
- p50/p95 control-plane latency;
- node reconnection success;
- support tickets per 100 accounts;
- cloud infrastructure cost per paid workspace/node.

Do not collect command/file contents for analytics.

## 21. Observability and operations

Production must have:

- structured logs;
- metrics;
- traces;
- alerting;
- uptime checks;
- error tracking;
- audit for internal admin actions;
- status page;
- incident management;
- SLOs;
- backup monitoring;
- restore drills.

Minimum service SLO targets should be set only after beta measurements. Business SLA must be lower-risk than the measured SLO.

## 22. Security program

Before paid GA:

- threat model for cloud;
- tenant-isolation tests;
- dependency/SAST/secret scans;
- image scans;
- WAF/rate limiting;
- credential rotation;
- KMS-backed secrets;
- encryption in transit and at rest;
- MFA mandatory for internal admins;
- least-privilege cloud IAM;
- production access logging;
- break-glass procedure;
- vulnerability disclosure;
- incident response runbook;
- backup/restore drills;
- external penetration test before serious Business selling.

Later:

- SOC 2 readiness;
- formal vendor risk process;
- SIEM;
- data residency;
- compliance mappings.

## 23. Privacy and data lifecycle

Define per data class:

- what is collected;
- why;
- where stored;
- retention;
- who can access;
- export;
- deletion.

Customer controls must include:

- export organization data;
- delete workspace;
- revoke all nodes;
- delete account;
- configurable retention for eligible plans.

Design for LGPD from launch and avoid architecture that blocks GDPR compliance.

## 24. Abuse and acceptable use

Protect the service from being turned into a remote-control abuse platform.

Controls:

- verified accounts;
- rate limits;
- anomaly detection;
- node enrollment limits;
- abuse reporting;
- tenant suspension;
- malware/credential-stuffing restrictions in AUP;
- no arbitrary access to nodes without node-owner enrollment;
- bounded trial resources.

## 25. Commercial/legal package

Before charging:

- Terms of Service;
- Privacy Policy;
- Acceptable Use Policy;
- subscription terms;
- cancellation/refund policy;
- commercial code license;
- trademark policy.

Before Business:

- DPA;
- SLA;
- subprocessor list;
- security overview/questionnaire;
- order form/MSA template;
- support policy;
- data-processing/deletion commitments.

## 26. Email/domain/reputation setup

Production communication requires:

- dedicated sending domain;
- SPF;
- DKIM;
- DMARC;
- bounce handling;
- complaint handling;
- transactional/marketing separation;
- unsubscribe for marketing mail;
- security-email priority path.

## 27. Customer onboarding

Aim for:

~~~text
Sign up
 -> create workspace
 -> select plan
 -> install Node
 -> enrollment token
 -> node online
 -> connect ChatGPT/other AI
 -> first audited operation
~~~

Target: a technically competent new user reaches first successful operation without human support.

Provide:

- copy/paste installer;
- OS detection;
- health check;
- exact recovery messages;
- guided AI-client connection;
- in-product checklist;
- sample disposable test.

## 28. Release/update channel

Cloud and Node releases are separate.

Node:

- stable;
- release candidate;
- controlled update;
- rollback.

Cloud:

- continuous deployment;
- backward-compatible node API window;
- schema negotiation;
- minimum supported Node version;
- deprecation notices;
- staged rollout;
- canary;
- rollback.

Never force a cloud deploy that strands a healthy older Node without a documented compatibility window.

## 29. Feature flags and entitlements

Separate:

- release flag;
- tenant entitlement;
- security policy.

A feature flag cannot grant authorization to a node. It only controls product availability.

Use server-side entitlements for paid features.

## 30. Environment strategy

At least:

- local development;
- preview/ephemeral;
- staging;
- production.

Production must have separate:

- database;
- credentials;
- encryption keys;
- identity configuration;
- billing mode;
- email domain;
- telemetry.

Never test destructive billing/security migrations directly in production.

## 31. Infrastructure as code

Private `portico-infra` repository should define:

- network;
- database;
- runtime;
- DNS;
- TLS;
- object storage;
- secrets references;
- monitoring;
- backups;
- CI/CD roles.

Avoid click-only production infrastructure.

## 32. SaaS delivery gates

### S0 — commercial architecture freeze

- benchmark reviewed;
- public/private split approved;
- license strategy legally reviewed;
- plan hypotheses defined;
- cost model template created.

### S1 — SaaS skeleton

- production domains;
- web app;
- auth;
- organization/workspace;
- Postgres;
- staging/prod separation;
- CI/CD;
- basic observability.

No money yet.

### S2 — node cloud enrollment

- node enrollment token;
- machine identity;
- node registry;
- heartbeat;
- revoke/rotate;
- node dashboard;
- one real Linux node managed through cloud.

### S3 — hosted MCP/control plane alpha

- hosted MCP endpoint;
- authenticated user/client;
- route to authorized node;
- local reauthorization;
- durable operations;
- cloud audit metadata;
- no cross-tenant leakage.

### S4 — billing-ready

- Stripe products/prices;
- checkout;
- subscriptions;
- entitlements;
- upgrade/downgrade;
- cancellation;
- billing portal;
- invoices;
- webhooks;
- grace/dunning;
- usage dashboard.

Run in test mode until all failure paths are exercised.

### S5 — Cloud Pro paid beta

- real payment;
- commercial license;
- 3-node entitlement;
- first external paying customers;
- support process;
- backup/restore test;
- security review.

Success criterion: repeated independent customers reach first value and pay without manual backend intervention.

### S6 — Cloud Team

- invitations;
- roles;
- five-user entitlement;
- shared fleet;
- policy templates;
- shared agents;
- 90-day audit;
- webhooks;
- priority support.

### S7 — Business pilot

- SSO;
- SCIM;
- SIEM/export;
- custom retention;
- private deployment option;
- SLA draft;
- DPA;
- onboarding process;
- contract/order form.

Success criterion: at least one design partner completes security/procurement review.

### S8 — public GA

- pricing final;
- unit economics validated;
- legal package complete;
- security assessment;
- incident drills;
- status page;
- documentation;
- support coverage;
- reliable subscription lifecycle;
- defined compatibility policy.

## 33. Platform/SaaS convergence gates

The SaaS cannot leap ahead of the core security model.

### Cloud Alpha requires

- stable v0.1 local path;
- multi-node node identity/routing;
- SaaS S1-S3.

### Paid Pro requires

- local Broker reauthorization proven;
- cloud tenant isolation;
- billing S4;
- recovery/backups;
- commercial license.

### Team requires

- multi-user RBAC;
- delegated authority provenance;
- shared audit.

### Business requires

- enterprise identity;
- stronger operational controls;
- support/SLA/legal package;
- deployment isolation options.

## 34. Minimum viable paid product

The first paid product should deliberately exclude many future ambitions.

Minimum paid Portico Cloud Pro:

- account/login/recovery;
- workspace;
- subscription;
- node enrollment;
- 1-3 Linux nodes;
- hosted MCP endpoint;
- node list/health;
- current typed filesystem/shell/service/Docker operations subject to local policy;
- cloud audit index;
- API key management;
- billing portal;
- cancellation;
- support;
- backups/monitoring.

Do not block first revenue on:

- Windows;
- GPU scheduler;
- RAG productization;
- AI-to-AI multi-hop;
- enterprise SSO;
- mobile apps;
- Kubernetes;
- global multi-region active-active.

Those become expansion features.

## 35. Cost model before final pricing

Measure:

- database per tenant/node;
- control-plane CPU/memory;
- relay bandwidth;
- audit storage;
- object storage;
- email;
- auth provider;
- monitoring;
- support time;
- payment fees;
- taxes;
- fraud/chargeback;
- backup;
- enterprise sales/support cost.

Calculate contribution margin by Pro and Team separately.

Target pricing must leave enough margin for support and future reliability investment, not merely cover raw cloud compute.

## 36. Go-to-market validation

Before locking pricing:

- 10-20 interviews across homelab/professional/SMB/AI-builder segments;
- 5 design partners;
- measure willingness to pay;
- test Pro vs Team packaging;
- test node-based pricing;
- identify which feature causes upgrade;
- track objections around cloud trust, privacy and licensing.

Prefer charging early paid-beta customers rather than collecting only verbal interest.

## 37. Definition of SaaS complete

Portico Cloud is commercially complete when an unknown customer can:

1. find the product;
2. understand the offer;
3. register securely;
4. recover access;
5. pay;
6. receive the right entitlements;
7. enroll nodes;
8. connect an AI;
9. operate within local policy;
10. collaborate according to plan;
11. inspect usage/audit;
12. update billing;
13. obtain support;
14. cancel;
15. export/delete data;

and the operator can:

16. observe the service;
17. recover backups;
18. handle incidents;
19. reconcile billing;
20. suspend abuse;
21. support customers without secret exposure;
22. deploy updates safely;
23. prove tenant isolation and audit provenance.
