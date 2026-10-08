# Cross-repository Cloud-to-Broker interoperability test

The public connector's `TestIntegrationActualCloudAndLocalBroker` runs only when `PORTICO_E2E_CLOUD_URL`, `PORTICO_E2E_NODE_STATE`, `PORTICO_E2E_TASK_IDS`, and the explicit `PORTICO_E2E_ACK=YES_EPHEMERAL` are all supplied.

These values originate exclusively from a **disposable private Cloud PostgreSQL CI job**. The Cloud base URL must use loopback, and the node state file is mode 0600. The test creates a real locally scoped Broker with its own SQLite audit chain and Unix socket; it executes the public `Runner` against Cloud's real signed HTTP transport and expects one `system.info` allow plus one `file.read /etc/shadow` deny.

It verifies:
- the lease -> local Broker path uses the locally configured principal rather than the Cloud requester's identity;
- each Cloud task ID is present in the local tamper-evident Broker audit;
- no remote Cloud request expands the Broker's filesystem scope;
- the connector is given time to send signed completions back to the real Cloud API.

The private Cloud side then verifies PostgreSQL task states and tenant-scoped audit. The test skips when the private CI opt-in is absent; no private Cloud credentials or code are required to compile/test the Community package.

This test does **not** involve a live ChatGPT account, real WorkOS/Paddle merchant credentials, production VPS or Android UI.
