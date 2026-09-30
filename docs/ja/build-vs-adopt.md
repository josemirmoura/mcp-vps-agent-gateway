# Build vs adopt

確認日: 2026-09-26。

大きな component を build する前に existing solution を評価します。

## Gate -1

確認:

1. Existing product は actual target client に接続できるか?
2. Required operations を support するか?
3. Permission を server-side で enforce するか?
4. Self-hosted または control/privacy requirement を満たすか?
5. Recovery/revocation があるか?
6. Adapt する方が new privileged control plane を保守するより安いか?

結果:

- Adopt
- Adapt/fork
- Build

## Comparison

### VPS Guardian MCP

Structured safety-checked operation を持ち general-purpose shell を持たない VPS-focused MCP server。

確認時点では VPS-side Python server + SSH/stdIO 上の local npm launcher。Typed VPS operation、narrow mutation surface、confirmation、diagnostic/rollback、release discipline の benchmark です。

Target remote ChatGPT Web architecture とは client transport problem が少し異なります。

Project:
https://github.com/murzirius/VPS-Guardian-MCP

### Remote Desktop Commander

Filesystem/terminal 向け hosted remote MCP service。Streamable HTTP と OAuth、paired device agent を使用。

Remote MCP UX、pairing/revocation、OAuth、terminal/filesystem ergonomics、multi-client support の benchmark。

Hosted implementation はここで定義する self-hosted Broker architecture とは異なります。

Project:
https://github.com/desktop-commander/remote-desktop-commander

## Why build this architecture

必要な組み合わせが次の場合は build が正当化されます:

- self-controlled VPS boundary;
- ChatGPT Web target;
- server-side Scoped policy;
- typed Linux/Docker/systemd operations;
- optional controlled shell;
- optional temporary elevation;
- explicit recovery semantics;
- operator-owned privileged Broker。

## Rule

North-star diagram にあるだけで component を build しません。

Existing solution が requirement を満たさず、previous MVP gate が必要性を実証した場合だけ build します。
