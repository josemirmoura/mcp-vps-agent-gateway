# FAQ

## Apakah Whole Host berarti Full?

Tidak. Whole Host mengatur batas fisik filesystem Broker menjadi `/`. Full adalah bundle capability terpisah yang dikontrol policy/feature gate dan default-nya nonaktif.

## Apakah Full berarti Internet tanpa batas?

Tidak. Network adalah capability/approval terpisah.

## Apakah ChatGPT menerima password VPS atau private SSH key saya?

Tidak. Instalasi dilakukan lokal di VPS; ChatGPT menerima endpoint MCP publik dan menjalankan OAuth.

## Mengapa Broker privileged?

Host systemd, Docker, dan operasi filesystem host terdelegasi memerlukan komponen privileged yang dipercaya. Broker tetap local-only dan policy-authoritative; Gateway tetap non-root.

## Apakah Docker security boundary?

Tidak. Docker adalah packaging/lifecycle. Otorisasi server-side Broker adalah boundary efektif.

## Telemetry?

Tidak ada first-party analytics/tracking client. Telemetry ZITADEL bawaan dinonaktifkan. Lihat `privacy.md`.

## Hanya satu proyek?

Ya. Gunakan profil **Project**.

## Default yang disarankan?

**Standard**: `/opt` sebagai batas fisik, tidak ada root proyek saat mulai, kemudian persetujuan eksplisit.

## Beberapa direktori?

Ya. Secara dinamis di bawah batas atau static root untuk operator advanced.

## Beberapa VPS?

Ya. Setiap instalasi memiliki identity, credentials, state, dan audit chain sendiri.

## Klien MCP lain?

Core menggunakan MCP Streamable HTTP berbasis standar. Jalur publik divalidasi dengan ChatGPT Web; klien kompatibel lain dapat bekerja jika mendukung alur autentikasi.

## Kapan instalasi selesai?

Hanya setelah panggilan klien nyata melewati authentication, Gateway, Broker, policy, execution, dan audit. Health container saja tidak cukup.

## Mengapa update.sh menghindari main?

`main` adalah development. Production mengikuti stable SemVer tag.

## Hapus Portico tanpa menghapus aplikasi?

Ya. Safe remove dan purge menghapus artefak milik Portico sambil mempertahankan resource yang pernah dikelolanya.
