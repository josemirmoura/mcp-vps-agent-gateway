# Build vs adopt

Before implementing this architecture from scratch, evaluate whether an existing MCP server or remote-control product already satisfies the real use case.

## Gate -1: decide intentionally

Ask:

1. Does an existing product connect to the actual target client?
2. Does it support the required operations?
3. Does its security model match the risk tolerance?
4. Can it enforce server-side scopes rather than relying on model behavior?
5. Can it be self-hosted or otherwise meet privacy/control requirements?
6. Does it provide a usable recovery path?

Possible outcomes:

- **Adopt**: existing solution already meets requirements.
- **Adapt**: existing open-source solution is close enough to extend/fork.
- **Build**: requirements justify the reference architecture in this repository.

## Examples worth evaluating

These projects/products solve adjacent problems and should be evaluated rather than ignored:

### VPS Guardian MCP

Published MCP server focused on structured, safety-checked VPS operations. It deliberately avoids a general-purpose shell tool and uses explicit confirmations for mutations.

Useful comparison points:

- typed operations
- narrow mutation model
- SSH/stdIO deployment path
- packaged releases
- practical operator workflow

### Remote Desktop Commander

Hosted remote MCP service that exposes filesystem and terminal access to supported AI clients using a remote MCP endpoint with OAuth.

Useful comparison points:

- remote-client UX
- authentication
- terminal/filesystem ergonomics
- operational maturity

### Generic MCP gateways

Projects that proxy/aggregate remote MCP servers can solve transport and aggregation, but they generally do not replace the privileged Linux Broker or server-side VPS authorization model described here.

## Why this project may still be justified

This architecture is specifically optimized for:

- a self-controlled VPS security boundary
- Controlled / Scoped / temporarily elevated capabilities
- server-side authorization independent of the model
- typed Linux/Docker/systemd operations plus optional shell
- explicit recovery semantics
- direct remote MCP integration as the client ecosystem permits

## Rule

Do not build a component merely because it appears in the north-star architecture.

Build it only when:

- an existing solution does not satisfy the requirement, and
- a completed MVP gate demonstrates the missing capability is actually needed.