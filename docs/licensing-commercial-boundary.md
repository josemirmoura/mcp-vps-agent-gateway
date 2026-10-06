# Portico licensing and public/private product boundary

**Status:** planned for the post-v0.1 commercial line  
**Decision owner:** project owner  
**Legal review required before changing the repository license.**

## Important current fact

The repository and already published release candidates have been distributed under Apache-2.0.

Those historical grants cannot be treated as if they never existed. A future licensing change can govern future code/releases to the extent the project has the legal right to relicense them, but it does not retroactively remove permissions already granted under Apache-2.0 for prior versions.

Therefore:

- keep historical tags immutable;
- document the license applying to each release;
- do not claim that older Apache-2.0 releases have become non-commercial;
- review copyright ownership/contributor rights before relicensing future code.

## Terminology

If future public Portico code forbids commercial use, call it:

- **source-available**;
- **community source**;
- **non-commercial source license**.

Do not market that edition as OSI open source. OSI's Open Source Definition requires free redistribution and prohibits discrimination against fields of endeavor, including business/commercial use.

## Recommended future model

Use a **dual commercial model**:

1. a public source-available Community edition for non-commercial use;
2. commercial licenses attached to Portico Cloud Pro, Team and Business;
3. proprietary/private cloud services and SaaS implementation.

### Candidate community license

Preferred candidate for legal review:

**PolyForm Noncommercial 1.0.0**

Why it fits the desired strategy:

- source remains inspectable;
- modifications and redistribution are permitted for allowed non-commercial purposes;
- commercial use is excluded;
- it is standardized rather than an ad-hoc home-made license.

Caveat: PolyForm Noncommercial also permits specified non-commercial organizations and education/research uses. If the desired boundary is literally personal use only, legal counsel should evaluate PolyForm Strict or a narrowly drafted custom license. Avoid improvising license text without review.

## Relicensing gate

Before replacing Apache-2.0 for a future release:

1. inventory copyright holders and external contributions;
2. confirm the project owner has authority to relicense every relevant file;
3. obtain contributor permission where required;
4. establish a Contributor License Agreement (CLA) for future external contributions that preserves the project's ability to dual-license;
5. retain third-party dependency licenses and notices independently;
6. add a release-level license matrix;
7. add a clear `COMMERCIAL-LICENSE.md` explaining how businesses obtain rights;
8. publish a trademark policy for the Portico name/logo;
9. update README, website, package metadata and installer notices;
10. have counsel review the final licensing package.

A DCO alone proves contributor provenance but does not automatically provide broad relicensing rights. If dual licensing is a strategic requirement, the contribution process must explicitly support it.

## Repository split

### Public repository

Current repository, potentially renamed later to a cleaner product name.

Keep public:

- Portico Node;
- Broker;
- local Gateway;
- CLI;
- installer/updater;
- local policy engine;
- local audit implementation;
- MCP schemas/tool definitions;
- node/agent protocol specifications;
- SDKs;
- examples;
- local development environment;
- security model and threat model;
- community documentation.

Goal: users can inspect the security-critical code that runs on their machines.

### Private cloud repository

Target private repository: `portico-cloud`.

Keep private:

- multi-tenant SaaS control plane implementation;
- account/workspace tenancy;
- subscription/entitlement engine;
- cloud node registry;
- hosted routing/relay services;
- cloud audit aggregation/indexing;
- commercial policy/template service;
- cloud agent registry;
- organization/team administration;
- billing/webhook logic;
- SaaS web portal;
- internal backoffice;
- product analytics;
- abuse/risk systems;
- enterprise SSO/SCIM glue;
- commercial feature flags;
- managed update orchestration;
- support/admin tooling.

Public protocol contracts should remain stable enough that private cloud services do not become opaque security authority over the node.

### Private infrastructure repository

Recommended separate private repository: `portico-infra`.

Keep private:

- Terraform/OpenTofu;
- cloud account topology;
- production Kubernetes/ECS/Nomad manifests as applicable;
- secrets references;
- database/network topology;
- WAF/CDN configuration;
- observability stack deployment;
- backup/DR definitions;
- production CI/CD;
- incident runbooks containing sensitive operational topology.

### Optional enterprise deployment repository

If Business develops dedicated/VPC/on-prem overlays, use a private `portico-enterprise` repository or private packages rather than contaminating the public Node code with customer-specific deployment logic.

## Security boundary rule

Commercial differentiation must not depend on hiding code that is necessary for the owner to understand the local privilege boundary.

The Broker and node-side authorization logic should remain inspectable. Proprietary value can live in:

- managed control plane;
- fleet UX;
- team governance;
- cloud availability;
- enterprise identity;
- aggregated observability;
- policy management;
- hosted relay;
- billing/entitlements;
- managed operations.

This preserves trust while retaining defensible SaaS IP.

## Commercial license behavior

A paid subscription should grant commercial-use rights only for the subscription/customer scope defined by the commercial terms.

Entitlements and license enforcement should be explicit and server-side for cloud features. Avoid brittle "phone home to keep local Broker alive" behavior. A subscription outage should not unexpectedly destroy local workloads or prevent the owner from safely inspecting/exporting their configuration.

Design grace periods for billing failures and cloud outages.

## Trademark

Even if code is source-available, reserve product identity:

- Portico;
- Portico MCP;
- logos;
- hosted service names.

Forks may comply with the code license while being prohibited from presenting themselves as the official Portico service. This requires a clear trademark policy separate from copyright licensing.

## Legal/compliance documents required for commercial launch

At minimum:

- Community source license;
- commercial software/service terms;
- Terms of Service;
- Privacy Policy;
- Acceptable Use Policy;
- DPA;
- subprocessor list;
- cookie policy if applicable;
- SLA for Business;
- security overview;
- vulnerability disclosure/security policy;
- trademark policy;
- commercial license/order form template;
- refund/cancellation policy;
- data-retention/deletion policy.

For Brazil, include LGPD obligations in the privacy/DPA design. GDPR and other regions become relevant as global customers are accepted.
