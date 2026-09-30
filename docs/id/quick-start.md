# Mulai cepat Portico MCP

Portico MCP dipasang dari terminal menggunakan Docker Compose transparan dan script repository.

## Persyaratan

- VPS Linux; Ubuntu 24.04 LTS adalah target RC yang tervalidasi;
- Docker Engine 24+ dan Docker Compose v2;
- minimal 2 GB RAM untuk ZITADEL bawaan;
- Git, OpenSSL, Python 3, curl;
- DNS publik, TCP 80/443, HTTPS valid;
- account/workspace ChatGPT yang benar-benar menyediakan Developer Mode dan custom MCP app.

## Mulai

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Bahasa resmi:

~~~text
en · pt-BR · es · de · fr · ja · id
~~~

Pilih eksplisit:

~~~bash
bash scripts/install.sh --lang id
~~~

## Alur terpandu

~~~text
Prasyarat
 -> Scope
 -> Otoritas efektif
 -> Kontainer
 -> Verifikasi lokal
 -> Akses publik aman
 -> Hubungkan ChatGPT
 -> panggilan MCP nyata dan diaudit
 -> INSTALLATION COMPLETE
~~~

Preflight memeriksa host, Docker/Compose, tools, systemd, RAM, dan edge. Traefik existing yang aman digunakan ulang; jika tidak ada dan 80/443 bebas, Traefik bawaan dapat digunakan.

## Profil rekomendasi: Standard

~~~text
physical ceiling: /opt
static roots:     none
project access:   approved later
~~~

Batas fisik hanya menentukan maksimum dan **tidak mengotorisasi /opt**.

Batas lain:

~~~bash
bash scripts/install.sh --profile custom --scope /srv/apps
~~~

`permissions.discover_scope` hanya menampilkan nama folder langsung. Akses diminta melalui native MCP confirmation:

~~~text
permissions.request_root_access
 -> native MCP confirmation
 -> Broker activates read / work / compose
~~~

AI tidak dapat menyetujui perluasan otoritasnya sendiri.

### Secret

`.env` dan `.env.*` tetap terkunci di proyek terotorisasi. Template seperti `.env.example` tetap terbaca. Secret nyata memerlukan persetujuan sementara terpisah melalui `permissions.request_sensitive_access`.

## Project

~~~bash
bash scripts/install.sh --profile project --scope /opt/my-app
~~~

## Whole Host

~~~bash
bash scripts/install.sh --profile whole-host
~~~

Mengubah batas menjadi `/` tanpa otomatis mengaktifkan Full, network, packages, users, atau firewall.

## Pengguna shell

~~~bash
bash scripts/install.sh --run-as deploy
~~~

Job Scoped dijalankan sebagai pengguna nyata non-root.

## OAuth publik

Portico memandu DNS dan membuat identity OAuth khusus:

~~~text
Username: vps-operator
Email:    operator@example.com
~~~

Ini bukan kredensial Linux/SSH/root.

## Penyelesaian ChatGPT

`scripts/connect-chatgpt.sh` menampilkan tutorial lengkap. Atur app/OAuth tanpa terburu-buru, kirim `system.info`, kembali ke terminal, dan tekan Enter.

## Local-only

~~~bash
bash scripts/install.sh --profile custom --scope /opt --local-only --yes
~~~

Ini **bukan** instalasi publik yang selesai.

## Removal

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Hapus checkout juga:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

## Stable release

Setelah `v0.1.0`, production sebaiknya menggunakan stable tag:

~~~bash
git clone --branch v0.1.0 https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Git checkout memungkinkan verified fast-forward update, backup, migration, dan automatic rollback.
