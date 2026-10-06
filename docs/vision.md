# Portico vision

## Product definition

Portico is the secure control and communication layer that connects **AI clients, AI agents, computers and services** across one or many machines.

Its long-term job is to let authorized intelligences:

- discover machines and capabilities;
- inspect and operate local resources;
- access files, databases, RAGs, applications, services and compute;
- delegate work to specialist AIs;
- communicate AI-to-AI on the same machine;
- communicate AI-to-AI across different machines;
- move tasks between machines without moving authority with them;
- keep authorization, policy and audit under the owner's control.

ChatGPT Web is the operator's preferred and priority client, but **Portico is not architecturally tied to ChatGPT or OpenAI**. The product should remain open to other compatible web AI clients, local agents and MCP-compatible systems.

## North-star architecture

~~~text
                   AI CLIENTS
        ChatGPT Web / other web AIs
          local agents / automation
                       |
                       | MCP / supported protocols
                       v
              PORTICO CONTROL PLANE
       identity / routing / policy context
       discovery / audit aggregation
                       |
          +------------+------------+
          |            |            |
          v            v            v
       Node A        Node B        Node C
       Linux         Windows       Linux
          |            |            |
      local policy  local policy  local policy
          |            |            |
       files/RAG    apps/files    DB/GPU/jobs
       DB/agents    agents        agents
          |            |            |
          +------ AI <-> AI -------+
~~~

Every destination node remains authoritative for its own execution policy. A control-plane route is never sufficient authorization by itself.

## AI-to-AI communication

AI communication is a first-class target capability.

Portico should allow an authorized AI or orchestrator to:

1. discover available agents and their declared capabilities;
2. address an agent on the same node or another node;
3. submit a task with bounded context and authority;
4. receive progress/results through durable task handles;
5. preserve provenance of who delegated what to whom;
6. prevent delegated agents from inheriting authority they were not explicitly granted.

Examples:

~~~text
ChatGPT Web
  -> Portico
  -> srv-ia
  -> coding-agent
  -> build/test result
  -> ChatGPT Web
~~~

~~~text
research-agent@srv-ia
  -> Portico
  -> finance-agent@node-b
  -> structured result
  -> research-agent@srv-ia
~~~

~~~text
orchestrator@node-a
  -> specialist-1@node-a
  -> specialist-2@node-c
  -> merged result
~~~

The communication layer must treat agent output as untrusted data. Agent messages never alter policy, register trust, expose secrets or grant capabilities by themselves.

## Client neutrality

Portico should expose stable capabilities independently from the model vendor.

Priority order for the owner's environment:

1. ChatGPT Web as primary interactive cognitive client;
2. other compatible web AI clients;
3. local orchestrators and specialist agents;
4. automation and programmatic clients.

Vendor-specific adapters may improve UX, but the security and node model must remain portable.

## Local intelligence

A local LLM is optional.

Local models can be useful for:

- offline operation;
- private workloads;
- high-volume low-cost tasks;
- specialist agents;
- redundancy;
- preprocessing and filtering.

The Portico architecture must not require a sovereign local LLM. A cloud AI client may be the primary cognitive layer while data, execution and policy remain local.

## Data locality

RAGs, databases, vector indexes, files, logs and applications may remain on their owner-controlled nodes.

Portico should prefer executing retrieval and filtering near the data and returning only the minimum useful result to the requesting AI client.

## Current release versus final product

The current v0.1 productization path remains intentionally narrow:

~~~text
ChatGPT Web / compatible MCP client
        -> Gateway
        -> Broker
        -> one Linux host
~~~

That path is the proven foundation, not the final product boundary.

The final product target is:

~~~text
many AI clients
      <->
Portico
      <->
many heterogeneous machines
      <->
many local/remote AI agents and services
~~~

Expansion to multi-node routing, Windows, agent discovery and AI-to-AI delegation must be earned through explicit implementation gates without weakening the existing security model.

## Non-negotiable invariant

**The AI is never the security boundary.**

Humans define authority. Nodes enforce it. Portico routes it. Audit records it.
