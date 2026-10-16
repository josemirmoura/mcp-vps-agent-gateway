# Public roadmap

Status: high-level public direction

The detailed commercial and implementation roadmap is maintained privately. This document lists only public architectural directions.

## Current supported focus

- one compatible web AI/MCP client;
- one Linux computer;
- local server-enforced policy;
- local audit;
- typed filesystem, jobs, service/container and related operations where enabled;
- guided installation and lifecycle management.

## Public future directions

Subject to implementation, security validation and release evidence:

- stable public node identity and capability contracts;
- interoperable node-aware protocols;
- additional operating-system support;
- durable task interfaces;
- public agent capability/delegation contracts;
- stronger release-supply-chain verification;
- continued compatibility with multiple standards-based AI clients.

No item in this roadmap is a production promise until it is released and documented in the compatibility matrix.

## Invariant

Any future multi-node or AI-to-AI capability must preserve destination-node authorization. Routing a request never authorizes it by itself.


## Public interoperability and engineering watchlist (2026-10-16)

Portico will remain **AI-client-neutral by design**: support is based on interoperable MCP transports, identity and tools instead of hardcoding a model/vendor. Compatibility with ChatGPT, Claude, Cursor, VS Code/Copilot or local agent hosts must be verified and recorded per client/version; model names alone are not certifications.

Candidate engineering improvements include:
- reliable large-tree operations and future integrity-checked transfers of multi-gigabyte artifacts without loading them into chat output;
- task/job recovery after disconnects with original operation identity and no blind replay of uncertain side effects;
- reproducible and verifiable installation/update/rollback;
- portable typed tool contracts and stronger multi-client E2E conformance.

These are *watchlist directions only*, subject to security gates and release evidence. See the [public competitive benchmark](competitive-landscape-and-interoperability.md) and [client compatibility](compatibility.md). The current Community one-client/one-Linux-host boundary remains unchanged.
