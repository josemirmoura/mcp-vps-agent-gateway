# MCP 2026-07-28: frozen required conformance evidence

Portico Community is an **MCP server** operating as an OAuth **resource server**. It is not being marketed as a native MCP client or OAuth authorization server. The protocol version is distinct from the Portico `VERSION` product SemVer.

## Source and test identity

- Official upstream implementation: `modelcontextprotocol/conformance`.
- Immutable upstream commit: `c37eec888e1c6ff140af79987a40008548b7cc5f` (`0.2.0-alpha.12`).
- Frozen requirements file: `requirements/2026-07-28.yaml` from that same commit.
- Required **server** scenarios: **37**. Required **client** scenarios: **32**, **NOT APPLICABLE** to Portico's server role.
- `not_scored` optional/pending scenarios do not count as required, and cannot be silently substituted for required tests.

## Execution and classification

Dedicated CI `.github/workflows/block2-mcp-requirements.yml` builds the **real Gateway** with a disposable local filesystem and `VPS_AGENT_AUTH_MODE=none` bound **only to loopback**. It builds the pinned official upstream runner from source via `npm ci` and executes:

```sh
node dist/index.js server --url http://127.0.0.1:18994/mcp --requirements 2026-07-28
```

The CI saves the official output and per-scenario `checks.json` artifacts. `scripts/mcp_conformance_matrix.py` emits a machine-readable report with:

| Classification | Meaning |
| --- | --- |
| **PASS** | All enforced checks succeeded; informational notes alone are not failures. |
| **FAIL** | A required check fails unexpectedly, or the report is corrupt. |
| **SKIPPED** | No required failure, but skipped or warning-only coverage remains. Not a pass. |
| **NOT_TESTED** | No evidence or a required **synthetic diagnostic fixture** is absent (for example a `test_*` tool/prompt or disposable resource). Not a pass and never reclassified as compliant. |
| **NOT_APPLICABLE** | Client-role requirements when evaluating Portico's server, or optional non-scored extensions. |

In particular, the original stateless smoke on `main` reported **24 success, 4 diagnostic-only NOT TESTED, 2 skipped** across 30 checks. That one scenario is **not the 37-scenario requirements suite** and does not prove complete conformance.

The evidence workflow is intentionally named `mcp-requirements-evidence-not-release-gate`: a green report-generation job only proves that the pinned test harness ran and classified evidence. **It cannot declare release readiness by itself.** The report has an explicit `release_gate_passed` Boolean, which is true only when all 37 scored scenarios PASS and the official runner exits zero. For an enforced gate, invoke the matrix parser with `--enforce`; failures then exit nonzero.

## Measured official run and uncovered capabilities (2026-10-09)

First full frozen requirements run on public Community PR #98, SHA `c4e18d9186aa2771285e45085caa40420ddf37a2`, [GitHub Actions run 38003928769](https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/38003928769), produced an immutable CI artifact `block2-mcp-2026-requirements-38003928769` (artifact ID `11650955192`). Upstream code was pinned to `c37eec888e1c6ff140af79987a40008548b7cc5f`. Runner exit `1`.

The initial machine classification was **7 PASS, 28 FAIL, 1 NOT_TESTED, 1 SKIPPED** out of 37. Inspection of **all actual `checks.json` records** found many supposed failures only say `unknown tool "test_*"`, `unknown prompt "test_*"`, or the disposable fixture resource does not exist. These are **missing test fixtures**, not evidence that unrelated Portico production tools failed. It also found an `INFO` about optional SSE responses and `WARNING` assertions that should not be treated as the same category as a hard protocol failure.

The classifier is now refined to distinguish fixture-unavailable `NOT_TESTED`, verified protocol `FAIL`, and `SKIPPED/WARNING`; **none of those count as PASS or release-ready**. Based on reanalysis of this same artifact, the expected revised classification is **8 PASS, 2 FAIL, 3 SKIPPED, 24 NOT_TESTED**. This revision still awaits confirmation by the CI running the updated classifier SHA. In particular the `completion/complete` absence and multi-round InputRequiredResult check remain reported as FAIL; further applicability/fixture review is required before treating either as a production defect or exception.

A separate conformance fixture that is *actually attached to the Portico Handler* is needed to exercise content tool variants, prompts, resources and multi-round elicitation on the real transport. Do not add fake `test_*` operations to the real user-facing Gateway catalog and do not hide them behind a broad expected-failures baseline. Because the Gateway registration layer is reserved for **Block 1**, any necessary injection seam must go via a documented handoff to Block 5 rather than a conflicting edit in this Block.

## OAuth Resource Server

`internal/gateway/oauth_resource_smoke_test.go` tests the actual Gateway HTTP handler against disposable loopback introspection for protected-resource metadata, Bearer challenge, missing/expired/inactive tokens, issuer/audience/subject/scope denial and positive subject propagation. It is synthetic Resource Server evidence, **not** a live browser authorization-code+PKCE test and does not certify the external identity provider.

## Segunda rodada: comparação no mesmo SHA

A [PR #110](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/110) adiciona um executável de conformance isolado por build tag, com ferramentas, prompts e recursos sintéticos exigidos pelo teste oficial. Esse binário **não** faz parte do Gateway Community publicado; qualquer PASS adicional não é uma prova de suporte funcional no servidor de produção.

Na [execução oficial 38008396287](https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/38008396287), código Community **`04ce22b3fe4917c05f8158dcfdbb2dc5ffecb887`**, o mesmo upstream fixado (`c37eec888e1c6ff140af79987a40008548b7cc5f`) e os mesmos 37 cenários obrigatórios foram exercitados **em duas variantes no mesmo commit**:

| Variante | PASS | FAIL | SKIPPED | NOT_TESTED | Release gate |
| --- | ---: | ---: | ---: | ---: | ---: |
| Gateway normal, compilado **sem** fixtures | **8** | **2** | **3** | **24** | **BLOCKED** |
| Gateway MCP compartilhado + catálogo sintético exclusivo do laboratório | **35** | **1** | **1** | **0** | **BLOCKED** |

O artefato de evidência preservado pelo GitHub Actions, ID `11652840175`, contém os dois arquivos `evidence/production-matrix.json` e `evidence/fixture-matrix.json`, resultados originais da suíte e saída do runner. O workflow concluiu com sucesso **como coleta de evidência**, mas **ambos os runners oficiais retornaram código 1 e ambos registraram `release_gate_passed=false`**.

A única falha classificada no laboratório é `completion-complete`, porque a instância real `gateway.NewMCPServer` não registra `completion/complete`. A única classificação `SKIPPED` no laboratório é `server-stateless`, pois o runner também contém verificações opcionais/ignoradas; nenhum cenário recebeu falsamente PASS por exceção. O laboratório já registra `test_missing_capability` e verifica o erro oficial `-32021`/HTTP 400, mas isso **não comprova sozinho** aplicação dessa regra no Gateway normal com OAuth/Broker.

**Handoff:** configurar `mcp.ServerOptions.CompletionHandler` na inicialização do servidor exige revisar `internal/gateway/gateway.go`, arquivo sob responsabilidade do Bloco 1. A integração e a CI do SHA reunido pertencem ao Bloco 5. Nunca integrar ferramentas `test_*` ao catálogo real para elevar pontuação.

## Release blockers and coordination

- Fully reconcile and run the required scenario matrix for the **exact final integrated SHA**, identifying missing features/fixtures without exposing synthetic diagnostic tools on the production MCP catalog.
- Do not reinterpret tests requiring imaginary `test_*` diagnostic tools as production functional requirements without an explicit protocol/conformance review. They remain visible as NOT_TESTED until independently measured.
- Keep `gateway.go` and the authoritative Broker authorization changes with Block 1. Propose protocol changes via Block 5 handoff.
- Actual ChatGPT/MCP Apps compatibility, native elicitation and approved/denied permissions on mobile/desktop belong to Blocks 1 and 5.
- Promoting the release requires security scanners green, exact-RC signing, license approval and operator acceptance in addition to this evidence.
