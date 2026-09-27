# Ephemeral VPS proof harness

This directory turns a human test request into reproducible evidence on a fresh GitHub-hosted Ubuntu runner.

## Flow

~~~text
proof/request.json
      |
      v
GitHub-hosted Ubuntu runner
      |
      +-- build Gateway/Broker/proof client
      +-- install real systemd units
      +-- create scoped policy
      +-- start disposable test service
      +-- call MCP over Streamable HTTP
      +-- execute through Unix-socket Broker
      +-- verify audit and host-side effect
      +-- upload evidence artifact
~~~

## Request types

### File write/read

~~~json
{
  "id": "example-file",
  "subject": "josemir-proof",
  "type": "file_write_read",
  "path": "/var/lib/vps-agent/proof/example.txt",
  "content": "hello"
}
~~~

### Expected denial

~~~json
{
  "id": "example-deny",
  "subject": "josemir-proof",
  "type": "file_read_denied",
  "path": "/etc/shadow"
}
~~~

### Real systemd restart

~~~json
{
  "id": "example-restart",
  "subject": "josemir-proof",
  "type": "service_restart",
  "service": "vps-agent-test.service"
}
~~~

The workflow fails if the expected operation, denial, Linux side effect or audit integrity check does not match the request.
