# Pemecahan masalah

Mulai dengan:

~~~bash
bash scripts/diagnose.sh status
~~~

Jika perlu:

~~~bash
bash scripts/diagnose.sh bundle
~~~

Tinjau bundle sebelum membagikannya.

## Direktori scope tidak ada

Portico tidak membuat direktori arbitrer secara diam-diam:

~~~bash
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/my-app
~~~

Atau jalankan installer kembali dan setujui langkah sudo yang terlihat.

## Path non-kanonis / symlink

Gunakan absolute canonical path tanpa dot segment atau symlink ancestor. Ini hardening batas filesystem yang disengaja.

## Broker/Gateway tidak healthy

~~~bash
docker compose ps -a
docker compose logs --no-color --tail 100 broker gateway
bash scripts/diagnose.sh status
~~~

Jangan melewati health gate.

## Port 80/443 digunakan

Portico hanya menggunakan ulang Traefik yang dapat diidentifikasi dengan aman dan tidak mengganti web server yang tidak dikenal.

## Banyak Traefik

~~~bash
bash scripts/setup-integrated-auth.sh --edge-network YOUR_NETWORK
~~~

Jika ACME resolver ambigu:

~~~bash
--certresolver YOUR_RESOLVER
~~~

## DNS tidak resolve

Perbaiki A/AAAA, tunggu propagasi, lalu jalankan kembali.

## Public OAuth verification gagal

~~~bash
bash scripts/verify-public.sh
~~~

Periksa DNS, certificate, hostname/issuer, protected metadata, OIDC DCR/PKCE, dan proxy route. Jangan melemahkan validasi issuer/audience.

## ChatGPT tidak terhubung

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Periksa MCP app, OAuth, pilihan Portico, lalu minta ChatGPT memanggil `system.info`. Timeout tidak berarti instalasi selesai.

## update.sh tidak menemukan stable release

Sebelum stable pertama, tidak ada target otomatis secara sengaja:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

`main` bukan kanal produksi otomatis.

## Update rollback

Jangan hapus backup. Periksa output, `backups/<timestamp>/migration-check.json`, `state/update.log`, dan diagnosis. Rollback adalah fitur keselamatan.

## Secret muncul di diagnosis

Jangan bagikan artefak. Simpan lokal, rotasi kredensial terdampak bila perlu, dan ikuti `SECURITY.md`.
