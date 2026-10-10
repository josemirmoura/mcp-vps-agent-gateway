# MCP 2026-07-28: scope and conformance evidence

The official frozen `modelcontextprotocol/conformance` requirements at commit `c37eec888e1c6ff140af79987a40008548b7cc5f` contain **37 server tests** and **32 client tests**. Pórtico is an MCP server, not an MCP client.

## Frozen score remains unchanged

On the staging code, the normal Gateway has **8 PASS, 2 FAIL, 3 SKIPPED, 24 NOT_TESTED**. The build-tagged test-only fixture has **35 PASS, 1 FAIL, 1 SKIPPED, 0 NOT_TESTED**. Neither passes the full frozen suite. Evidence: [GitHub Actions 38009010764](https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/38009010764).

### Distinguishing failure from capability selection

**completion-complete**: the upstream frozen file requires this scenario. The Go MCP SDK `v1.8.0` announces `completions` only when `ServerOptions.CompletionHandler` is configured. Pórtico does not install this handler or advertise that optional capability. The real-Gateway HTTP test `TestMCPCompletionCapabilityIsAbsentAndMethodIsClosed` proves that valid 2026 discovery does not announce completions and the unsupported method returns JSON-RPC `-32601` / HTTP 404. This does not convert the official FAIL to PASS.

**input-required-result-multi-round**: the upstream scenario calls the synthetic `test_input_required_result_multi_round` tool, which is deliberately unavailable in the normal Gateway. It remains a recorded official FAIL, requiring a fixture and separate applicability review. No missing test tool should be installed for a real ChatGPT user solely to raise a test score.

The official `server-stateless` scenario exercises real discovery, version negotiation, HTTP boundaries and notifications, but its capability-error and streaming checks also require named synthetic diagnostic tools. A fixture-only pass does not prove production behavior.

## Reproducible 37-scenario release review

The new `scripts/mcp_release_scope_audit.py` checks each frozen scenario name and upstream SHA against the original machine-readable report, retains raw status and runner exit, applies **zero** waivers and refuses altered/missing results. The report annotates scenario context separately from official pass/fail. The dedicated CI produces the original official Gateway report, original fixture report and their two companion assessments on the **same commit**.

The OAuth test `TestIntegratedOAuthResourceServerBoundaryE2E` exercises resource-server metadata, protected HTTP transport, Bearer challenge and claims/scope/subject constraints with disposable introspection. Third-round regressions include missing identity claims, missing expiry, malformed audience and scope-prefix mismatch. This is not a personal-browser PKCE acceptance test.

## Release decision

The **full frozen MCP conformance gate remains BLOCKED**. Neither a green evidence-upload job nor fixture tests can imply complete official conformance. A narrower release claim must explicitly identify advertised protocol features, unfinished verification and the operator's sign-off, without misrepresenting the frozen suite. Product authorization, signatures, legal approval and real client acceptance remain separate gates.
