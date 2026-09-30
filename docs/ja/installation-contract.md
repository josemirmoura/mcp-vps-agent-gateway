# インストール契約

状態: **プロジェクトの規範的決定**。

この文書は、一時的な開発/検証手順と、エンドユーザー向けにサポートされるインストール体験の境界を定義します。実装、テスト、README、チュートリアル、リリース審査はこれに従います。

## サポートされる体験

公式インストールは次の原則に従います。

- ターミナル中心;
- Docker Compose first;
- 透明で確認可能なコマンドとスクリプト;
- ユーザー自身の VPS で再現可能;
- プロジェクト開発者の支援なしで利用可能;
- **ネイティブ優先**: 独自 glue code より先に、公式プラットフォーム機構、Docker、MCP、標準 OAuth/OIDC を使う。

ユーザーは VPS 上で直接 Portico をインストール/運用し、サーバー側と公開認証が準備できた後にのみ ChatGPT を接続します。

## シークレットとリモートアクセス

チュートリアルは、ChatGPT やメンテナーに次を渡すよう **決して要求してはいけません**。

- VPS パスワード;
- SSH 秘密鍵;
- 無制限 SSH / 管理リモートアクセス;
- root 資格情報;
- クラウド管理資格情報;
- シークレットを含むターミナル出力;
- サービスに不要なシークレット。

必要なシークレットは VPS 上でローカル入力するか、対応するサービスのネイティブ UI/API を使います。専用 OAuth オペレーターパスワードはローカル入力され、ChatGPT には渡りません。

開発中に人間が一時的にコマンドを実行してサニタイズ済み診断を返すことがありますが、これは開発環境の制約であり **製品要件ではありません**。

## 自動化責任

合理的に自動化できる決定的な作業はパッケージが吸収します。

- 依存関係チェックと実行可能なエラー;
- ポート/Edge Proxy 検出;
- Compose 検証;
- 起動と Health Check;
- HTTPS 設定/検証;
- OAuth/OIDC Bootstrap/Discovery;
- MCP Endpoint チェック;
- fail-closed 認証テスト;
- Secret を redact した診断収集;
- 安全に自動判断できない場合の復旧案内。

開発中の一時的な調査コマンドをユーザーチュートリアルに持ち込みません。

## 意図的な手動操作

プラットフォーム上、人間の判断やブラウザ操作が不可避な場合だけ許容します。文書は次を明示します。

1. 何をするか;
2. なぜ安全に自動化できないか;
3. どう検証するか。

例: 物理上限の選択、外部 DNS 設定、OAuth パスワードのローカル入力、ChatGPT Web での MCP app 接続。

## 完了ゲート

コンテナ起動やローカル検証は中間ゲートです。

~~~text
VPS configured
 -> MCP public over valid HTTPS
 -> OAuth/OIDC works
 -> ChatGPT Web connected
 -> real ChatGPT MCP call succeeds
 -> expected subject/policy/audit confirmed
 -> INSTALLATION COMPLETE
~~~

`scripts/connect-chatgpt.sh` は公式フローの一部です。Broker が期待する認証済み ChatGPT 呼び出しを監査で確認するまで完了ではありません。

## 開発専用手順

Ad-hoc diagnostics、temporary Actions probes、専用 self-hosted runners、ephemeral CI、開発用 hostname/IP/path/branch、調査用 curl/openssl、開発者 SSH は issue/PR 証拠に置き、公開チュートリアルには含めません。

## リリース前レビュー

1. README と公式翻訳を確認;
2. 各公式言語の `installer-flow.md` と `chatgpt-integration.md` を確認;
3. 公開 landing を確認;
4. 開発固有情報を除去;
5. VPS/SSH credential を要求していないか確認;
6. 手動操作に理由と検証手順があるか確認;
7. Installation Contract と i18n Coverage を CI で実行;
8. clean VPS でフルインストール;
9. 実際の ChatGPT MCP 呼び出しと Broker Audit で終了。

## エンジニアリング規則

**NATIVE FIRST。** 公式 API、サポートされた設定、Docker/Compose、MCP、OAuth/OIDC を優先します。独自コードは必要最小限、中央集約、文書化、可逆にします。
