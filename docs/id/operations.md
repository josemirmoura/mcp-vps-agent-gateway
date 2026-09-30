# Operasi

## Status

~~~bash
bash scripts/diagnose.sh status
~~~

Menampilkan versi produk, state Compose, health Broker/Gateway, jumlah restart container, snapshot health Broker, dan status audit chain.

## Health

~~~bash
bash scripts/diagnose.sh health
~~~

Mengembalikan snapshot health Broker.

## Log

~~~bash
bash scripts/diagnose.sh logs 200
~~~

Menampilkan log terbaru dengan nilai secret yang dikonfigurasi telah di-redact. Gunakan untuk restart, kegagalan auth, gangguan Gateway/Broker, timeout, dan container unhealthy.

## Audit

~~~bash
bash scripts/diagnose.sh audit 100
~~~

Menjawab siapa, tool/resource/action apa, allow/deny, hasil, sequence, dan validitas hash chain. Operational log dan security audit memiliki fungsi berbeda.

## Diagnostic bundle

~~~bash
bash scripts/diagnose.sh bundle
~~~

Berisi runtime/version, health, audit status/tail, log, versi Docker/Compose, dan snapshot policy yang di-redact. Dibuat dengan mode 0600; tetap tinjau sebelum dibagikan.

## Version

~~~bash
bash scripts/version.sh
bash scripts/version.sh --check
~~~

`--check` memperbarui release tag bila memungkinkan dan melaporkan current/ahead/divergent/update available.

## Update

~~~bash
bash scripts/update.sh
~~~

Target default adalah stable SemVer terbaru. Updater memeriksa working tree bersih, menampilkan perubahan, menolak non-fast-forward, menghentikan paket, membackup `.env`, policy, state, identity volume, menguji migrasi pada salinan DB, membangun/memverifikasi target, dan rollback otomatis saat gagal.

RC eksplisit:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

## Safe remove

~~~bash
bash scripts/remove.sh safe
~~~

Mencabut grant sementara, menghapus runtime, dan merotasi kredensial lokal aktif sambil mempertahankan konfigurasi, policy, audit/state, dan integrated identity.

## Purge

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Checkout source hanya dihapus dengan konfirmasi kedua:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

Proyek, aplikasi, situs, database, image/container pihak ketiga, service, dan file terdelegasi tetap dipertahankan.
