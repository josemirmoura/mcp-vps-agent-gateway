# MCP competitive landscape and interoperable AI clients

Reference date: **2026-10-16** · Status: **public technical research and future direction**, not a supported-feature or security-certification statement.

Portico Community is a self-hosted runtime connecting **one compatible MCP client to one Linux host** through an unprivileged Gateway and a privileged local Broker. The target Linux host enforces owner-approved policy and writes an audit trail regardless of the AI model/client chosen.

## Related open-source projects

These repositories address overlapping use cases through different trust, execution and packaging designs:

| Project | Public repo | Relevant documented approach | What Portico may learn |
| --- | --- | --- | --- |
| Open Machine | https://github.com/OpenMachineLabs/openmachine | Local-first MCP runtime, workspaces, SSH workers, resumable structured tasks | Original operation IDs, explicit evidence, guarded edits and task recovery |
| DeskMCP | https://github.com/edmen12/deskmcp | Windows desktop MCP gateway, permission profiles, Task Rooms and UI controls | User-friendly setup, guarded concurrent file edits and verified update flow |
| chatgpt-mcp | https://github.com/alexcodeplace/chatgpt-mcp | Linux tools and durable `exec.*` jobs | Recovery after MCP disconnection, rollback, safe job outcomes |
| SSH MCP Bridge | https://github.com/shashikanth-gs/mcp-ssh-bridge | SSH multi-host access, HTTP OAuth and SFTP | Discovery, integrity-checked transfers and capability metadata |
| Aegis SSH MCP | https://github.com/nextgridit-team/Aegis-SSH-MCP | SSH command validation by host-specific profiles | Small, testable operation profiles and least-privilege defaults |

**Evidence caveat:** descriptions are based on public documentation and selected implementation files, not on independent exploit testing, comparative benchmarks or complete E2E executions by the Portico team. A different architecture is not proof that another project is unsafe.

## Where Portico differs

Portico separates the **Gateway** (remote transport, identity, tool schema) from the **machine-local privileged Broker** (authoritative policy, reauthorization, resource access and audit). The Gateway cannot grant itself elevated authority, and an optional future multi-node control plane cannot override local deny decisions.

This is a design advantage to validate empirically; privileged operations, container access, filesystem mounts, symlinks/hardlinks and external side effects still require thorough testing.

References: [Architecture](architecture.md) · [Policy schema](policy-schema.md) · [Security hardening](security-hardening-v2.md) · [Recovery semantics](runtime-semantics-and-recovery.md).

## AI/model/vendor neutrality

Portico's interface is MCP, **not a ChatGPT-only private API**. An AI model needs a host/client that can actually call MCP tools. A compatible integration also depends on the client's transport, authentication, tool schemas, user interaction and allowed account features.

Clients for which documentation or exploration exists include ChatGPT Web, Claude, Cursor and VS Code/Copilot. These are **not all verified Portico integrations**. Local runtimes and agent orchestrators, including MCP-capable hosts in ecosystems such as Hermes, are possible future integration targets; no automatic support is implied.

Do not claim a client is supported without:
1. exact client version and compatible transport (HTTPS Streamable HTTP or an explicitly approved adapter);
2. OAuth/OIDC and subject identification functioning;
3. tool discovery and a real authenticated `system.info` (plus a read-only call) functioning;
4. corresponding Broker audit event and a negative authorization test;
5. evidence reference in the [compatibility matrix](compatibility.md).

MCP Apps or native approval UI support is optional client-specific presentation. Unsupported approval UIs require the existing separately authenticated operator channel, never a silent permission grant. Secrets are never pasted into a chat to work around login.

See [AI client connection guide](connect-ai-clients.md) for current client-specific instructions and explicitly unverified combinations.

## Public candidate improvements

The following are **proposals, not release promises**, subject to threat review, CI, E2E and versioned documentation:

- large-file and large-tree operations with bounded resource use; integrity checks, explicit partial-state handling, resumable transfer where applicable;
- durable tasks, original operation receipts, explicit unknown outcomes and no blind duplicate side effects;
- verified installers, reproducible and signed releases, preserved policy across update and safe rollback;
- operator-friendly capability profiles and meaningful policy/denial diagnostics;
- interoperable typed tools and gradual conformance testing across multiple MCP clients.

For binary assets such as GGUF model weights and datasets, future tools should transfer data in chunks between authorized endpoints rather than placing large blobs in a chat response. Deletion of large trees must preserve protected-file constraints and reject unsafe mutations before harmful partial effects.

## Scope and governance

This public document covers the Community runtime and public interoperability contracts only. Any separate Infinity experimental deployment or managed Cloud offering has its own authority, release status and documentation. The Community baseline remains one client/one Linux computer.

External code may be considered only after review of the exact revision, LICENSE/NOTICE/dependencies, tests, security consequences and maintenance burden. Comparing ideas does not imply incorporating third-party source code.

The active [Public roadmap](roadmap-multinode-control-plane.md) and [Project status](project-status.md) remain authoritative for what has actually shipped. Commercial strategy and private release sequencing are deliberately not included here.
