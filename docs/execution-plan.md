# Active public execution plan: Community v0.1.0

Status: active

## Immediate objective

Freeze a trustworthy first stable Community release for the documented single-Linux-node Scoped path.

## Baseline

The technical RC6 baseline has:

- repeatable automated validation;
- real ChatGPT Web/OAuth evidence;
- scoped write/read/delete evidence in the operator environment;
- local policy and audit;
- lifecycle/update/rollback validation;
- immutable RC6 prerelease artifacts.

## Gate 1: licensing decision

Before stable release, settle the license used by the future stable Community line.

The current RC6 was published under Apache-2.0. Historical grants remain under their original terms.

If the license changes before stable:

1. update LICENSE/NOTICE/public documentation consistently;
2. bump to a new release candidate;
3. run the full automated matrix;
4. publish an immutable candidate;
5. test that exact candidate in owner acceptance.

Final license selection requires owner/legal review and is intentionally not specified in detail in this public repository.

## Gate 2: owner clean-install acceptance

Against the exact immutable final RC:

- clean install;
- DNS + HTTPS;
- OAuth/OIDC;
- real compatible web AI connection;
- audited system.info;
- scoped read/write and representative bounded operations;
- out-of-scope denial;
- protected-secret denial/temporary exception/revocation;
- diagnostics/audit review;
- safe remove/reinstall;
- controlled update/rollback.

## Gate 3: stable v0.1.0

After Gate 2:

- freeze accepted commit;
- set VERSION;
- update changelog/status/release docs;
- run release validation;
- publish stable tag/release;
- verify checksums and provenance artifacts;
- point public documentation to stable instructions.

Stable v0.1.0 claims only the documented Community Linux/Scoped use case.

## After stable

Continue Community/runtime hardening, compatibility work and public interoperability contracts.

Detailed commercial Cloud planning is outside this public repository.
