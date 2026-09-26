# Security Policy

## Reporting a vulnerability

Please report security issues privately to the repository maintainer instead of opening a public issue with exploit details.

Include when possible:

- affected component
- reproduction steps
- expected vs actual behavior
- impact
- suggested mitigation

## Security posture

This project assumes AI-generated actions, remote clients and external content can be hostile.

Deliberate security requirements include:

- no root MCP Gateway
- no direct Docker socket for the Gateway
- deny-by-default authorization
- human-approved temporary elevation
- Unix-socket-only privileged Broker
- filesystem escape resistance
- sandboxed shell execution
- idempotent writes where practical
- redacted audit logs
- no secrets committed to the repository

A change that weakens one of these properties should be treated as a security-sensitive architectural change.
