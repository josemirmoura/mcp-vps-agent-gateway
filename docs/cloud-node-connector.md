# Portico Cloud node connector

Status: first public node-side Cloud interoperability implementation.

The Cloud connector is optional. Community self-hosting does not require it.

It gives an enrolled Linux node an outbound HTTPS path to a Portico Cloud control plane while preserving the local Broker as the machine authorization boundary.

## Authority model

The connector runs as the same unprivileged Portico runtime UID used by the Gateway and reaches the privileged Broker only through the protected Unix socket.

For every routed task:

```text
Portico Cloud task
      |
      | outbound HTTPS lease
      v
portico-cloud-node
      |
      | typed wire.Request
      | subject = locally configured Broker principal
      v
local Broker
      |
      | local policy / grants / approvals
      v
host operation
```

The Cloud requester is provenance, not automatic local authority.

The connector uses `PORTICO_CLOUD_BROKER_SUBJECT` as the local Broker principal. By default the Compose overlay maps it to `VPS_AGENT_SUBJECT`.

A Cloud task cannot override a local Broker deny.

## Node identity

On first enrollment the connector:

1. generates an Ed25519 key pair locally;
2. persists the private key in a mode-0600 state file;
3. sends only the public key to Cloud;
4. exchanges the one-time enrollment token;
5. stores the returned node/workspace identity;
6. removes the bootstrap token from persistent state.

Pending enrollment state is persisted before the network exchange. This allows safe retry with the same key if the server commits enrollment but the response is lost.

The private key does not leave the node.

## Safe one-shot enrollment

Set the Cloud API URL, then pipe the one-time token through stdin instead of placing it in shell history:

~~~bash
export PORTICO_CLOUD_URL="https://<your-portico-cloud-api>"
read -rsp "Enrollment token: " PORTICO_TOKEN
printf '%s' "$PORTICO_TOKEN" |   docker compose -f compose.yaml -f compose.cloud.yaml   run --rm -T cloud-node --enroll-only --token-stdin
unset PORTICO_TOKEN
~~~

The named `cloud-node-state` volume retains the enrolled node identity.

Then start the connector without an enrollment token:

~~~bash
docker compose -f compose.yaml -f compose.cloud.yaml up -d cloud-node
~~~

## Runtime protocol

The connector:

- sends signed heartbeats;
- polls for one durable task at a time;
- signs every Cloud node request with the node Ed25519 key;
- rejects/recovers through Cloud replay protection;
- maps a task to an exact Broker request;
- uses the Cloud task ID as the local Broker invocation ID;
- renews the Cloud lease while a long Broker operation is active;
- posts the bounded Broker response as the task result;
- retries completion safely when Cloud supports identical idempotent completion retries.

Task-to-Broker mapping:

| Cloud task | Broker request |
| --- | --- |
| local configured Broker subject | `subject` |
| `operation` | `tool` |
| `resource` | `resource` |
| `action` | `action` |
| task ID | `id` and `invocation_id` |
| `grant_id` | `grant_id` |
| `input` | `args` |

The connector never transports a Broker admin token through Cloud.

## Network model

The node initiates outbound HTTPS connections. Ordinary NAT/firewall environments do not need a new inbound administration port.

Production Cloud URLs require HTTPS. Plain HTTP is accepted only for loopback development.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORTICO_CLOUD_URL` | required | Cloud API origin |
| `PORTICO_CLOUD_NODE_NAME` | hostname | display/enrollment name |
| `PORTICO_CLOUD_STATE` | `/var/lib/portico-cloud-node/state.json` | local identity state |
| `PORTICO_CLOUD_BROKER_SOCKET` | `/run/vps-agent/broker.sock` | local Broker socket |
| `PORTICO_CLOUD_BROKER_SUBJECT` | local Portico subject | principal authorized by the Broker |
| `PORTICO_CLOUD_HEARTBEAT_INTERVAL` | `30s` | node heartbeat cadence |
| `PORTICO_CLOUD_POLL_INTERVAL` | `2s` | idle task polling cadence |
| `PORTICO_CLOUD_RENEW_INTERVAL` | `10s` | active task lease renewal cadence |
| `PORTICO_CLOUD_BROKER_TIMEOUT` | `15m` | maximum local Broker call time |

For first enrollment only, the binary also accepts:

- `--token-stdin`;
- `PORTICO_CLOUD_ENROLLMENT_TOKEN_FILE`, interpreted as a filename under `/run/secrets`;
- `PORTICO_CLOUD_ENROLLMENT_TOKEN` as a less-preferred convenience path.

## Current boundary

This connector implements the node-side transport and local Broker handoff.

Customer signup, billing, Cloud task creation, product entitlements and fleet UI belong to the managed Cloud product and are not implemented in this public Community repository.
