# Multiple VPS instances in one ChatGPT workspace

This is an advanced scenario. The normal onboarding remains one VPS Agent instance per installation.

## Identity model

Tool names stay stable across installations. Do not rename tools per server.

Each installation instead gets:

- its own HTTPS MCP endpoint;
- its own OAuth client/resource relationship and credentials;
- a stable VPS_AGENT_INSTANCE_ID generated at bootstrap;
- an operator-visible VPS_AGENT_INSTANCE_NAME;
- instance identity in system.info, Broker audit events and structured logs.

Example:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Loja"
VPS_AGENT_PUBLIC_URL=https://mcp-loja.example.com/mcp
~~~

On another VPS:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Blog"
VPS_AGENT_PUBLIC_URL=https://mcp-blog.example.com/mcp
~~~

Register them as separate ChatGPT apps with matching visible names. Select or @mention the intended app rather than relying on ChatGPT to silently disambiguate identical tool names.

## Verification

For each app, call system.info and compare instance_id and instance_name with the expected VPS. Treat hostname as supporting information only: infrastructure providers can legitimately reuse the same hostname across separate disposable machines. Then inspect the Broker audit on that VPS:

~~~bash
bash scripts/diagnose.sh audit 20
~~~

The same instance_id/name should appear in new audit events and structured logs.

## Same-conversation use

If the current ChatGPT interface allows multiple draft/custom MCP apps in the same conversation, explicitly select or mention the intended app before an operation. UI behavior is product-surface behavior and may change independently of this server.

## Automated simultaneous-use gate

The repository includes a dedicated three-machine acceptance workflow. It starts the real Docker package on three independent Ubuntu runners at the same time and deliberately reuses the same subject, tool names, logical filesystem path and operation id on all three machines.

The gate passes only when:

- all three installations have distinct instance ids and instance names;
- all three use independent generated credentials;
- the three runner lifetimes overlap, proving simultaneous execution;
- the same operation id succeeds independently on all three Broker state stores;
- each audit chain records only its own instance identity;
- hostnames and audit chain heads remain distinct;
- each instance finishes with the expected local state.

This is the server-side collision test and can run without operator involvement.

## ChatGPT product-surface behavior

Connecting multiple live apps to the same ChatGPT workspace is a separate UI/product-surface experiment. It is not required for the normal single-VPS onboarding or for the final single-instance ChatGPT -> VPS E2E gate. If tested later, register each VPS as a separately named app and explicitly select the intended app rather than depending on silent disambiguation of identical tool names.
