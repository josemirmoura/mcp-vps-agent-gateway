# Community licensing decision brief

Status: decision pending  
Created: 2026-10-06

This brief exists because the intended commercial model changed after RC6 was published under Apache-2.0.

## Goal

Keep the Community source publicly readable and useful for personal/homelab/research adoption while preventing unlicensed commercial exploitation, and reserve proprietary Cloud/Business code and managed services for commercial products.

## Non-negotiable consequence

A license that prohibits commercial use is source-available, not OSI Open Source.

The public product should therefore be named **Portico Community** and described as **source-available** if a non-commercial restriction is adopted.

## Candidate models

### Option A — PolyForm Noncommercial 1.0.0 + commercial license

Fit: strongest current match to the stated intent.

Allows non-commercial use, modification and redistribution while requiring a separate commercial license for commercial purposes.

Advantages:

- standardized license text;
- readable;
- preserves public source and community experimentation;
- supports dual licensing;
- includes a patent license;
- avoids inventing bespoke legal language.

Trade-offs:

- it is not OSI Open Source;
- it permits certain noncommercial organizations such as educational/public-research/government institutions;
- some companies may avoid evaluating software whose commercial use requires a separate license;
- commercial-use boundary still needs a clear commercial FAQ and sales path.

Recommended if the intended free audience includes personal users, homelabs, students and research.

### Option B — PolyForm Strict 1.0.0 + commercial license

Fit: only if the free edition should be much more restrictive.

Advantages:

- stronger control over modification/distribution;
- simpler separation between personal evaluation and commercial product.

Trade-offs:

- weaker community contribution/fork ecosystem;
- less attractive for technical adoption;
- conflicts with the desired public collaborative development model.

Not recommended for Portico unless preventing redistribution matters more than ecosystem growth.

### Option C — permissive/open-source core + proprietary Cloud

Examples of permissive/core licensing include Apache-2.0 or another OSI license for the node runtime while keeping Cloud proprietary.

Advantages:

- maximum developer adoption;
- easiest integrations, contributions and ecosystem growth;
- no ambiguity around open-source terminology.

Trade-offs:

- third parties may commercially package the public runtime;
- differentiation must come from Cloud, brand, hosted operations, enterprise governance and pace of execution.

This is the strongest ecosystem strategy but does not satisfy the stated requirement to forbid commercial exploitation of the public edition.

### Option D — custom Community license

Fit: only if standardized licenses cannot express the business boundary.

Advantages:

- exact custom restrictions.

Trade-offs:

- legal drafting and review cost;
- adoption friction;
- ambiguity for users;
- license scanners may not recognize it;
- higher maintenance burden.

Avoid unless necessary.

## Recommendation

Subject to legal review, use **PolyForm Noncommercial 1.0.0 for future Portico Community releases plus a separate commercial license**.

Use a commercial grant for:

- Portico Cloud Starter;
- Portico Cloud Pro;
- Portico Business Cloud;
- any future Commercial Self-Hosted offering;
- OEM/embedded redistribution;
- managed-service providers;
- commercial internal use if the selected Community license does not permit it.

## Questions that require owner/legal decision

1. Should universities and public research institutions remain free?
2. Should nonprofits and government agencies remain free?
3. Is evaluation inside a for-profit company permitted for a limited trial?
4. Is internal use by a commercial company prohibited unless licensed?
5. Are consulting companies allowed to use Community while delivering unpaid demonstrations?
6. Is redistribution of modified Community builds allowed for non-commercial purposes?
7. Should OEM/hosting/MSP use always require a commercial agreement?
8. Should there be a free commercial trial period?

## Commercial-license policy

The commercial license should define:

- legal entity/customer;
- licensed product/edition;
- users/nodes/agents or other commercial metrics;
- term;
- support/SLA if any;
- update entitlement;
- redistribution/OEM rights where applicable;
- warranty/liability;
- audit/compliance terms;
- termination;
- governing law/venue;
- data-processing terms when Cloud is used.

## Trademark policy

Source licensing should not automatically grant the right to present a fork or hosted service as official Portico.

Define:

- official Portico name/logo use;
- acceptable "compatible with Portico" language;
- fork naming;
- certification language;
- Cloud service naming.

## Release migration

Because RC6 is already Apache-2.0:

1. retain RC6 as immutable historical release;
2. approve the future Community license;
3. update source/license notices;
4. bump VERSION to the next RC;
5. update release documentation and website terminology;
6. run full CI/security/package matrix;
7. publish the next immutable RC;
8. perform owner clean-install acceptance on that exact RC;
9. only then promote stable v0.1.0.

## Evidence to preserve

Before relicensing future releases, archive:

- contributor/copyright review;
- selected license version;
- legal review outcome;
- date of change;
- exact last Apache-2.0 tag;
- exact first source-available tag;
- commercial-license template version.

## References

- Open Source Definition: https://opensource.org/osd
- OSI FAQ on commercial use: https://opensource.org/faq
- PolyForm licenses: https://polyformproject.org/licenses
- PolyForm Noncommercial 1.0.0: https://polyformproject.org/licenses/noncommercial/1.0.0
