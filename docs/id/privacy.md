# Privasi, data, dan telemetry

## Lokasi data

Paket di-self-host pada VPS operator. State operasional, policy, riwayat audit, dan data identitas terintegrasi tetap di VPS kecuali operator sengaja mengekspornya.

## Audit

Catatan audit Broker merekam siapa memanggil tool, resource/action yang diminta, apakah diizinkan, serta state sequence/integrity hasilnya.

Audit berbeda dari log layanan biasa.

## Logs

Log Gateway, Broker, Docker, dan identity service dapat memuat timestamp, identifier instance, nama tool, error, dan request context. Diagnostic bundle meredaksi secret terkonfigurasi dan diuji di CI terhadap leakage, tetapi tetap harus dianggap berpotensi sensitif.

## ChatGPT / MCP

Data yang dibutuhkan untuk request MCP dapat lewat antara ChatGPT atau client MCP lain dan Gateway. Authority server-side dikontrol policy. Output model atau konten remote bukan authorization. Jangan tampilkan secret melalui file generik, shell, atau hasil tool.

## Secrets

Kredensial lokal disimpan dalam konfigurasi yang dikontrol root/operator atau private volume. Instalasi tidak pernah meminta password VPS, private SSH key, password root, atau secret yang tidak terkait. Kredensial OAuth khusus terpisah dari VPS/SSH.

## Telemetry

Tidak ada first-party analytics/tracking client. Telemetry ZITADEL dinonaktifkan dengan `ZITADEL_TELEMETRY_ENABLED=false`. Tidak ada marketing tracking.

## Removal

Safe remove mempertahankan konfigurasi, state, dan audit, menghapus runtime aktif serta merotasi kredensial lokal. Full purge menghapus artefak milik MCP dan mempertahankan aplikasi, situs, database, container pihak ketiga, service, serta file terkelola.

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~
