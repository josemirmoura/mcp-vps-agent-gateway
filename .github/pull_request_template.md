## What changed

Describe the focused change and why it is needed.

## Product/security boundaries

- [ ] Gateway remains non-root.
- [ ] Broker remains the privileged authorization boundary.
- [ ] No new secret is committed or returned through normal tools.
- [ ] Whole Host, Full and unrestricted network remain separate decisions.
- [ ] User-facing installation does not request VPS/SSH credentials.
- [ ] Native/official mechanisms were preferred before custom glue.

## Validation

- [ ] Relevant tests added or updated.
- [ ] Negative/security tests added where authority changes.
- [ ] Documentation matches implementation.
- [ ] Migration/rollback impact considered.
- [ ] Installation-document contract passes when installation docs changed.

## Operational impact

State any migration, restart, compatibility or rollback implications.
