# ドキュメントマップ

正規のドキュメントは `docs/` にあります。このフォルダは公式の日本語訳です。同一バージョンの英語版と差異がある場合、翻訳が修正されるまで英語版を技術的な基準とします。

## オペレーター / ユーザー

1. [quick-start.md](quick-start.md) — 最短のサポート済みインストール手順。
2. [installation-contract.md](installation-contract.md) — サポート範囲の規範。
3. [installer-flow.md](installer-flow.md) — インストール全体の流れ。
4. [product-model.md](product-model.md) — Project / Standard / Whole Host。
5. [chatgpt-integration.md](chatgpt-integration.md) — ChatGPT 接続と完了ゲート。
6. [authentication.md](authentication.md) — 統合 OAuth/OIDC。
7. [operations.md](operations.md) — 状態、ログ、監査、更新、削除。
8. [troubleshooting.md](troubleshooting.md) — トラブルシューティング。
9. [faq.md](faq.md) — FAQ。
10. [compatibility.md](compatibility.md) — 検証済み/未検証環境。
11. [privacy.md](privacy.md)
12. [releases.md](releases.md)
13. [support.md](support.md)
14. [operator-acceptance.md](operator-acceptance.md)

## 開発 / セキュリティ

- [project-status.md](project-status.md)
- [architecture.md](architecture.md)
- [policy-schema.md](policy-schema.md)
- [threat-model.md](threat-model.md)
- [security-hardening-v2.md](security-hardening-v2.md)
- [security-release.md](security-release.md)
- [runtime-semantics-and-recovery.md](runtime-semantics-and-recovery.md)
- [tool-trust-and-confused-deputy.md](tool-trust-and-confused-deputy.md)
- [transport-and-aggregation.md](transport-and-aggregation.md)
- [implementation-validation.md](implementation-validation.md)
- [simulation-validation.md](simulation-validation.md)
- [mvp-first.md](mvp-first.md)
- [implementation-runbook.md](implementation-runbook.md)
- [build-vs-adopt.md](build-vs-adopt.md)
- [multi-instance.md](multi-instance.md)
- [productization-status.md](productization-status.md)
- [references.md](references.md)

競合時の優先順位は、アーキテクチャ、インストール契約、現在のプロジェクト状態、Policy Schema、Security Hardening、Runtime Semantics、補助文書、履歴文書です。

**アーキテクチャを一文で:** 非特権 MCP Gateway と、Unix socket で接続された特権ローカル Broker の2つの Go プロセス。Broker が認可、状態、特権実行を管理します。
