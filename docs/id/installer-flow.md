# Alur pertama Portico MCP

## Tujuan produk

Instalasi bersifat deklaratif dan terminal-first. State operator utama:

~~~text
.env                   # berisi VPS_AGENT_SCOPE_ROOT
config/policy.yaml     # policy logis capability/resource
~~~

Docker Compose memulai paket. Tidak ada installer kedua yang menyembunyikan konfigurasi.

## Batas yang didukung

Pengguna menjalankan perintah langsung di VPS. Tidak diperlukan developer, shell remote yang dikontrol ChatGPT, password VPS, atau private SSH key.

Password operator OAuth adalah kredensial layanan yang diperlukan dan dimasukkan lokal tanpa echo melalui `setup-integrated-auth.sh`.

## Titik masuk terpandu

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

`install.sh` mengorkestrasi komponen transparan yang sama, menampilkan otoritas efektif sebelum runtime, dan dapat dijalankan ulang setelah interupsi.

## Fase 1 — Bootstrap

Profil yang direkomendasikan adalah **Standard**: batas fisik `/opt` dan tidak ada root proyek statis. Batas hanya merupakan limit maksimum filesystem, bukan izin baca/tulis. Path absolut lain dapat dipilih.

~~~bash
bash scripts/init.sh --scope /opt --dynamic-baseline
~~~

Sebelum mengubah state, `scripts/preflight.py` memeriksa host, Docker/Compose, tools, systemd, RAM, dan edge. Dependency wajib yang hilang menghentikan proses sebelum state dibuat.

Bootstrap membuat secret lokal acak, instance ID stabil, `config/policy.yaml`, batas pilihan, pengguna non-root nyata untuk shell terbatas, direktori state, dan validasi Compose.

## Batas fisik filesystem

`compose.yaml` hanya bind-mount `VPS_AGENT_SCOPE_ROOT` ke Broker sebagai `/host`. Broker memastikan path filesystem, shell cwd, dan Compose tetap di dalam batas; escape ditolak fail-closed.

Standard dengan `--dynamic-baseline` tidak mengotorisasi `/opt`. Whole Host memerlukan `VPS_AGENT_WHOLE_HOST=1` dan `compose.host.yaml`, sehingga batas menjadi `/`, tetapi Full tetap tidak aktif.

## Fase 2 — Otoritas MCP

Standard mulai tanpa root proyek terotorisasi. `permissions.discover_scope` hanya boleh menampilkan nama direktori langsung di bawah batas, bukan isi.

~~~text
permissions.request_root_access
 -> pending request di Broker
 -> native MCP elicitation / konfirmasi klien
 -> delegasi terikat subject
~~~

- `read`: baca filesystem;
- `work`: baca/tulis + cwd shell terbatas;
- `compose`: menambahkan aksi Compose yang sudah dibolehkan static policy.

### File terlindungi

`.env` dan `.env.*` tetap terkunci di dalam proyek terotorisasi. `.env.example`, `.env.sample`, dan `.env.template` tetap menjadi template biasa.

`permissions.request_sensitive_access` membuat pengecualian terpisah yang temporary, exact-path, subject-bound, audited, expiring, dan independently revocable. Job shell Scoped mem-mask path tersebut dan hardlink alias yang terdeteksi kecuali grant sementara yang tepat aktif.

Delegasi dapat menargetkan batas fisik itu sendiri setelah peringatan lebih kuat. Delegasi tidak dapat keluar dari batas dan tetap tidak membuka secret terlindungi.

systemd, Docker/Compose, network, packages, users/groups, firewall, dan elevation tetap dibatasi secara terpisah.

## Fase 3 — Start

~~~bash
docker compose up -d --build
~~~

~~~text
Gateway: non-root, no host root
Broker: privileged, host mounted at /host, no remote control port
~~~

Docker adalah packaging; batas otorisasi adalah policy Broker.

## Fase 4 — Verifikasi lokal

~~~bash
bash scripts/verify.sh
~~~

Memeriksa Compose, health Broker/Gateway, integritas audit, autentikasi, dan panggilan aman `system.info`. Sukses lokal berarti runtime siap, **bukan instalasi selesai**.

## Fase 5 — OAuth terintegrasi + endpoint publik

DNS adalah tindakan manual eksternal. Portico memandu A/AAAA dan memvalidasi resolusi.

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Satu Traefik existing yang aman akan digunakan ulang. Jika tidak ada dan 80/443 bebas, Traefik bawaan dimulai. Lalu ZITADEL + PostgreSQL, operator khusus non-admin, username/email, resource audience, private introspection client, MCP-compatible DCR, binding Gateway/Broker, dan public verification disiapkan.

Sukses:

~~~text
INTEGRATED AUTH: READY
~~~

Verifikasi ulang:

~~~bash
bash scripts/verify-public.sh
~~~

Memeriksa HTTPS, OAuth/OIDC discovery, protected-resource metadata, DCR/PKCE, private introspection, dan fail-closed unauthenticated MCP.

## Fase 6 — Kapabilitas ChatGPT

Periksa bahwa account/workspace nyata benar-benar menyediakan Developer Mode dan custom MCP app creation. OpenAI mengontrol rollout dan UI.

## Fase 7 — Tutorial ChatGPT Web

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Pembuatan app dan OAuth login dilakukan di ChatGPT Web dengan identity OAuth khusus, bukan kredensial VPS/SSH.

## Fase 8 — Tes nyata

Script mencatat audit baseline. Pengguna menyelesaikan ChatGPT tanpa terburu-buru lalu kembali dan menekan Enter. ChatGPT diminta memanggil `system.info`.

Sukses memerlukan:

~~~text
real ChatGPT MCP call
+ expected authenticated subject
+ policy allow
+ successful execution
+ Broker audit
+ valid audit chain
~~~

Baru kemudian:

~~~text
CHATGPT WEB CONNECTION VERIFIED
INSTALLATION COMPLETE
~~~

## Update dan penghapusan

~~~bash
bash scripts/update.sh
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Hapus checkout juga:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

Proyek terdelegasi dan resource pihak ketiga tidak dihapus hanya karena Portico pernah diberi wewenang mengelolanya.
