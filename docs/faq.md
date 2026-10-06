# FAQ

## Does Whole Host mean Full?

No.

Whole Host sets the Broker's physical filesystem ceiling to /.

Full is an explicit capability bundle controlled by policy and feature gates. Full remains disabled by default and is not presented as production-ready.

## Does Full imply unrestricted Internet access?

No.

Unrestricted network is a separate capability/approval decision.

## Does ChatGPT receive my VPS password or SSH private key?

No.

The supported installation is run by the operator in the VPS terminal. ChatGPT receives the public MCP endpoint and completes OAuth. VPS/SSH credentials are not part of the supported connection flow.

## Why is the Broker privileged?

Host systemd, Docker and delegated host filesystem operations require a trusted privileged component.

The Broker is deliberately local-only and policy-authoritative. The remote Gateway stays non-root and does not receive the Docker socket or host root.

## Is Docker itself the security boundary?

No.

Docker is the packaging/lifecycle mechanism. The Broker is host-root-trusted code within the delegated perimeter; server-side authorization remains the effective authority boundary.

## Is there telemetry?

The project contains no first-party analytics/tracking client. Bundled ZITADEL telemetry is explicitly disabled.

See privacy.md.

## Can I run only one project?

Yes. Choose **Project** when Portico should be permanently limited to one project root.

## What is the recommended default?

**Standard**. It uses `/opt` as the physical ceiling, starts with no project root authorized, and lets the operator approve specific roots later from ChatGPT.

## Can I delegate several directories?

Yes. With Standard, approve the needed roots dynamically under the physical ceiling. Advanced operators can also manage static roots through the policy/helper tooling.

## Can I manage several VPS instances?

Yes. Each installation has its own instance identity, credentials, state and audit chain. The repository includes simultaneous multi-instance acceptance.

## Can another MCP client use the server?

The core is standards-based MCP Streamable HTTP. The supported public product path is validated with ChatGPT Web, but other compatible MCP clients can use the same server if they support the required authentication flow.

## When is installation considered complete?

Only after the real client call traverses authentication, Gateway, Broker, policy, execution and audit. Container health alone is not completion.

## Why does update.sh avoid main?

main is development. Stable SemVer tags are the normal update channel so production installations do not silently follow unreleased commits.

## Can I delete the MCP without deleting my apps?

Yes. Safe remove and purge are designed to remove MCP-owned runtime/configuration artifacts while preserving resources that the MCP administered.


## Is Portico only for ChatGPT?

No. ChatGPT Web is the operator's priority interactive client and the current validated public path, but the final product is vendor-neutral. Other compatible web AI clients, local orchestrators and agents are target clients.

## Will AIs be able to communicate through Portico?

Yes, in the planned multi-node architecture. Portico will support authorized AI-to-AI task delegation on the same machine and across different machines, with capability discovery, bounded task authority, provenance and audit.

## Does one AI inherit another AI's permissions?

No. Delegation never implies privilege inheritance. The destination node re-authorizes the delegated task and grants only the capabilities explicitly allowed for that task.

## Is multi-node already implemented?

Not yet. The current validated product path is one Linux host. Multi-node routing, Windows nodes and AI-to-AI communication are roadmap items built on the existing Gateway/Broker security model.
