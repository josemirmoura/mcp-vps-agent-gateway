# Build atau adopt

Diperiksa: 2026-09-26.

Sebelum membangun komponen besar, evaluasi solusi yang sudah ada.

## Gate -1

Tanyakan:

1. Apakah produk existing terhubung ke target client nyata?
2. Apakah mendukung operasi yang diperlukan?
3. Apakah permission ditegakkan server-side?
4. Apakah dapat self-hosted atau memenuhi control/privacy?
5. Apakah ada recovery/revocation?
6. Apakah adaptasi lebih murah daripada menjaga privileged control plane baru?

Hasil:

- Adopt
- Adapt/fork
- Build

## Pembanding

### VPS Guardian MCP

Server MCP fokus VPS dengan operasi terstruktur dan safety check, tanpa general-purpose shell.

Pada tanggal pengecekan, jalurnya adalah Python server di VPS + local npm launcher melalui SSH/stdIO. Benchmark untuk typed VPS operations, mutation surface sempit, confirmation, diagnostics/rollback, dan release discipline.

Ia menyelesaikan masalah transport client yang sedikit berbeda dari target remote ChatGPT Web.

Project:
https://github.com/murzirius/VPS-Guardian-MCP

### Remote Desktop Commander

Hosted remote MCP service untuk filesystem/terminal menggunakan Streamable HTTP dan OAuth dengan paired device agent.

Benchmark untuk remote MCP UX, pairing/revocation, OAuth, terminal/filesystem ergonomics, dan multi-client support.

Hosted implementation-nya bukan self-hosted Broker architecture yang dijelaskan di sini.

Project:
https://github.com/desktop-commander/remote-desktop-commander

## Mengapa build architecture ini

Build tetap masuk akal jika kombinasi yang diperlukan adalah:

- self-controlled VPS boundary;
- ChatGPT Web sebagai target;
- server-side Scoped policy;
- typed Linux/Docker/systemd operations;
- optional controlled shell;
- optional temporary elevation;
- explicit recovery semantics;
- operator-owned privileged Broker.

## Rule

Jangan build komponen hanya karena ada di north-star diagram.

Build hanya jika solusi existing tidak memenuhi requirement dan previous MVP gate membuktikan capability tersebut diperlukan.
