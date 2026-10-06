# Portico commercial benchmark and positioning

**Status:** product-planning draft  
**Checked:** 2026-10-06  
**Scope:** adjacent competitors in secure infrastructure access, managed control planes, AI tool layers and AI gateways.

Portico does not yet have one exact like-for-like competitor. The relevant market is the intersection of:

1. zero-trust infrastructure access;
2. machine/workload identity;
3. hosted control planes;
4. MCP/tool execution for AI agents;
5. multi-node AI orchestration.

## Benchmark

| Product | Current public pricing signal | Strongest overlap with Portico | Gap Portico can exploit |
|---|---:|---|---|
| Tailscale | Personal $0; Standard $8/user/mo; Premium $18/user/mo; Enterprise custom | secure device/service connectivity, ACLs, identity, audit, machine access | networking-first rather than typed AI execution and AI-to-AI delegation |
| Pomerium | Personal free; Business $7/user/mo annual ($9 monthly); Enterprise custom | managed control plane + self-hosted data plane, policy, identity, audit | application-access-first rather than owner-authorized AI operation of heterogeneous nodes |
| Cloudflare Access | Free; pay-as-you-go $7/user/mo annual; contract custom | ZTNA, policy, scale, identity-aware access | broad SSE/SASE platform, not an AI-native machine-control layer |
| Teleport | enterprise-oriented pricing/custom guide | infrastructure identity, JIT access, machine/workload identities, MCP/server access | heavy enterprise identity/PAM footprint; Portico can be much simpler for AI-first node operations |
| Arcade | Free; Team $25/mo + usage; Enterprise custom | managed auth/action runtime for AI agents, usage-based actions | SaaS/app action layer rather than local machine/node authority |
| Composio | Free; Pro $29/mo + usage; Enterprise custom | large AI tool catalog, OAuth management, MCP, usage metering | external SaaS integrations rather than secure owner-controlled computers/servers |
| Portkey | open-source self-host; Production $49/mo; Enterprise custom | AI gateway, observability, RBAC, production controls | model/API traffic gateway rather than machine execution and cross-node delegation |

Primary-source URLs checked:

- https://tailscale.com/pricing
- https://www.pomerium.com/pricing
- https://www.cloudflare.com/pt-br/sase/products/access/
- https://goteleport.com/pricing/guide/
- https://www.arcade.dev/pricing/
- https://composio.dev/pricing
- https://portkey.ai/pricing

## Market pattern

The adjacent market converges on three commercial patterns:

1. **free personal/community entry** to accelerate adoption;
2. **self-serve paid tier** in roughly the low tens of dollars per month or single-digit/low-double-digit per-user pricing;
3. **enterprise/custom** for SSO, governance, compliance, deployment flexibility, SLA and support.

Portico should follow the shape, not copy the billing unit.

Portico is node-centric and authority-centric. Pricing only by seat would underprice high-value machine fleets and overprice solo operators. Pricing only by tool call would make infrastructure control feel unpredictable. The launch model should therefore be a **workspace subscription with included nodes/users and simple node overage**, while expensive relay/compute usage remains separately meterable if it becomes material.

## Proposed launch SKUs

These prices are hypotheses to validate with real users and unit economics before public GA.

### Portico Community

**$0, self-hosted, source-available, non-commercial use.**

Target:

- personal labs;
- education/research;
- evaluation;
- open community adoption.

Public components:

- Portico Node/Broker;
- local Gateway;
- CLI and installer;
- public MCP/tool schemas;
- node/agent protocol specifications;
- SDKs and examples;
- local single-owner operation.

No hosted SLA, no commercial-use license, no managed fleet/control-plane service.

### Portico Cloud Pro

**Target list price: US$19/month or US$190/year.**

For solo professionals, developers and small commercial deployments.

Initial entitlement hypothesis:

- 1 workspace;
- 3 nodes included;
- 1 owner + 2 collaborators;
- hosted control plane;
- managed identity for nodes;
- remote node health;
- cloud routing;
- 30-day cloud audit index;
- standard email support;
- API/MCP tokens;
- automatic product updates/compatibility notices;
- commercial-use license for covered deployment.

Candidate overage: per additional active node, with the exact amount determined from cost telemetry before GA.

### Portico Cloud Team

**Target list price: US$59/month or US$590/year.**

For small teams operating multiple machines/agents.

Initial entitlement hypothesis:

- 10 nodes included;
- 5 users included;
- multiple workspaces/projects;
- team RBAC;
- shared agent registry;
- policy templates;
- 90-day audit retention;
- webhook integrations;
- higher limits;
- priority support;
- commercial-use license.

Candidate overages:

- additional active nodes;
- additional seats only if support/security cost makes them material;
- optional relay/egress above a generous included allowance.

### Portico Business

**Custom annual contract.**

For organizations needing governance, compliance or dedicated architecture.

Candidate capabilities:

- custom node/user limits;
- SAML/OIDC SSO;
- SCIM;
- custom roles/policies;
- long audit retention and SIEM export;
- custom domain;
- private relay or dedicated control plane;
- customer VPC / single-tenant cloud deployment;
- optional on-prem control plane;
- data-residency options;
- SLA;
- priority security support;
- onboarding/migration/professional services;
- commercial licensing and negotiated terms;
- DPA/security review support.

Do not publish a low artificial Business floor until support and infrastructure costs are measured.

## Differentiation to defend

Portico should position around the combination competitors currently split across products:

- AI-native typed operations;
- policy enforced on the destination node;
- machine owner remains the authority;
- multi-node routing without exposing raw root credentials;
- cloud AI + local AI + local tools in one fabric;
- AI-to-AI delegation with explicit bounded authority;
- data/RAG can stay local;
- vendor-neutral AI clients;
- cloud convenience without moving execution authority into the cloud.

The product should avoid becoming merely another tunnel, VPN, generic MCP registry or generic agent framework.

## Benchmark cadence

Recheck this benchmark:

- before final public pricing;
- every quarter during the first year;
- whenever a major adjacent vendor launches AI-agent governance, MCP fleet management or machine-control pricing.

Track:

- public plan names/prices;
- billing unit;
- free limits;
- node/device limits;
- seats;
- audit retention;
- SSO/SCIM;
- private deployment;
- AI/MCP support;
- SLA/support;
- data residency;
- enterprise security claims.
