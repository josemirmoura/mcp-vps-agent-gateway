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
