# Integrasi ChatGPT

Diperiksa: 2026-09-30.

## Target

~~~text
ChatGPT Web
 -> OAuth discovery + Dynamic Client Registration
 -> HTTPS /mcp
 -> Gateway
 -> Broker
 -> VPS
 -> tamper-evident audit
~~~

Health container saja tidak menyelesaikan instalasi.

## Prasyarat ChatGPT

Sebelum setup publik, pastikan account/workspace nyata benar-benar menyediakan Developer Mode, custom MCP app creation, dan permission yang dibutuhkan. OpenAI mengontrol rollout dan UI.

Referensi proyek:

- https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt
- https://developers.openai.com/plugins/build/app-quickstart

Koneksi read-only dapat berguna untuk diagnosis tetapi bukan penyelesaian jalur produk write-capable.

## Prasyarat server

1. Custom MCP creation tersedia.
2. `bash scripts/verify.sh` berhasil.
3. DNS menunjuk ke VPS.
4. Integrated auth siap:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Harus berakhir dengan:

~~~text
INTEGRATED AUTH: READY
~~~

Verifikasi ulang batas publik:

~~~bash
bash scripts/verify-public.sh
~~~

## Koneksi

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Alur yang diharapkan:

1. ChatGPT membaca RFC 9728 protected-resource metadata;
2. menemukan ZITADEL;
3. mendaftarkan public OAuth client secara dinamis;
4. operator login memakai account OAuth khusus;
5. Authorization Code + PKCE;
6. ChatGPT menemukan MCP tools;
7. memanggil `system.info`;
8. Gateway memvalidasi access token;
9. Broker mencocokkan stable subject dan policy;
10. audit mencatat panggilan.

ChatGPT tidak pernah menerima password VPS, private SSH key, root, atau unrestricted remote shell. Password OAuth hanya dimasukkan pada login identity provider.

## Gate penyelesaian

`connect-chatgpt.sh` menetapkan audit baseline. Setelah setup, Enter membuat Portico mencari panggilan `system.info` terautentikasi yang lebih baru.

~~~text
tutorial shown
 != success

ChatGPT app connected
 + authenticated system.info
 + expected subject
 + Broker allow
 + successful execution
 + matching audit
 = INSTALLATION COMPLETE
~~~

## Transport

~~~text
https://<domain>/mcp
~~~

MCP Streamable HTTP over HTTPS, tanpa custom WebSocket.

## Security boundary

Konfirmasi ChatGPT adalah kontrol UX tambahan, bukan pengganti otorisasi server-side.

**Gateway mengautentikasi. Broker mengotorisasi. Pemilik VPS memilih policy.**
