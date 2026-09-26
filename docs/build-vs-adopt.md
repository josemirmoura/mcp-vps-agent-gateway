# Build vs adopt

Checked: 2026-09-26.

Before building a major component, evaluate existing solutions.

## Gate -1

Ask:

1. Does an existing product connect to the actual target client?
2. Does it support the required operations?
3. Does it enforce permissions server-side?
4. Can it be self-hosted or meet control/privacy requirements?
5. Does it provide recovery/revocation?
6. Does adapting it cost less than maintaining a new privileged control plane?

Outcome:

- Adopt
- Adapt/fork
- Build

## Useful comparison points

### VPS Guardian MCP

Published VPS-focused MCP server with structured, safety-checked operations and no general-purpose shell tool.

As of the checked date its documented path is a VPS-side Python server plus a local npm launcher over SSH/stdIO. This is a strong benchmark for:

- typed VPS operations
- narrow mutation surface
- confirmations
- diagnostics and rollback workflows
- packaged release discipline

It solves a somewhat different client-transport problem from the target remote ChatGPT Web architecture.

Project:
https://github.com/murzirius/VPS-Guardian-MCP

### Remote Desktop Commander

Hosted remote MCP service for filesystem and terminal access, using Streamable HTTP and OAuth with a paired device agent.

It is a strong benchmark for:

- remote MCP UX
- device pairing/revocation
- OAuth flow
- terminal/filesystem ergonomics
- multi-client support

Its hosted service implementation is not the self-hosted Broker architecture described here.

Project:
https://github.com/desktop-commander/remote-desktop-commander

## Why build this architecture

Building remains justified if the required combination is:

- self-controlled VPS boundary
- ChatGPT Web as target
- server-side Scoped policy
- typed Linux/Docker/systemd operations
- optional controlled shell
- optional temporary elevation
- explicit recovery semantics
- operator-owned privileged Broker

## Rule

Do not build a component merely because it exists in the north-star diagram.

Build only when an existing solution does not satisfy the requirement and the previous MVP gate demonstrates the capability is needed.
