# Kontrak instalasi

Status: **keputusan normatif proyek**.

Dokumen ini membedakan prosedur sementara untuk pengembangan/validasi dari pengalaman instalasi yang didukung untuk pengguna akhir. Implementasi, test, README, tutorial, dan review rilis harus mematuhinya.

## Pengalaman yang didukung

Instalasi resmi:

- terminal-first;
- Docker Compose first;
- memakai perintah dan skrip yang transparan serta dapat diperiksa;
- dapat direproduksi di VPS milik pengguna;
- dapat dipakai tanpa bantuan developer proyek;
- **native first**: mekanisme resmi platform, Docker, MCP, dan OAuth/OIDC berbasis standar sebelum glue code khusus.

Pengguna memasang dan mengoperasikan Portico langsung di VPS. ChatGPT baru dihubungkan setelah sisi server dan autentikasi publik siap.

## Secret dan akses remote

Tutorial **tidak boleh pernah** meminta pengguna memberikan kepada ChatGPT atau maintainer:

- kata sandi VPS;
- private SSH key;
- SSH atau akses admin remote tanpa batas;
- kredensial root;
- kredensial admin cloud;
- output terminal yang berisi secret;
- secret yang tidak benar-benar diperlukan layanan.

Secret yang memang diperlukan dimasukkan secara lokal di VPS atau melalui UI/API native layanan terkait. Kata sandi operator OAuth khusus dimasukkan lokal dan tidak diberikan ke ChatGPT.

Dalam pengembangan, operator manusia mungkin menjalankan perintah sementara dan mengembalikan diagnosis yang telah disanitasi. Itu adalah keterbatasan lingkungan pengembangan, **bukan persyaratan produk**.

## Tanggung jawab otomasi

Tugas deterministik yang wajar untuk diotomasi harus diserap oleh paket, termasuk:

- deteksi dependency dan error yang dapat ditindaklanjuti;
- deteksi port dan edge proxy;
- validasi Compose;
- startup dan health check;
- setup/validasi HTTPS;
- bootstrap/discovery OAuth/OIDC;
- pemeriksaan endpoint MCP;
- test fail-closed;
- diagnosis dengan redaksi secret;
- instruksi recovery ketika pilihan otomatis tidak aman.

Perintah investigasi satu kali selama development tidak boleh menjadi langkah wajib pengguna.

## Tindakan manual yang disengaja

Hanya dibolehkan bila platform membutuhkan keputusan operator atau aksi browser/UI. Dokumentasi harus menjelaskan:

1. apa yang dilakukan;
2. mengapa tidak aman untuk diotomasi;
3. bagaimana memvalidasinya.

Contoh: memilih batas fisik, mengatur DNS eksternal, memasukkan password OAuth secara lokal, dan menghubungkan app MCP di ChatGPT Web.

## Gate penyelesaian instalasi

Startup container dan verifikasi lokal adalah gate sementara.

~~~text
VPS configured
 -> MCP public over valid HTTPS
 -> OAuth/OIDC works
 -> ChatGPT Web connected
 -> real ChatGPT MCP call succeeds
 -> expected subject/policy/audit confirmed
 -> INSTALLATION COMPLETE
~~~

`scripts/connect-chatgpt.sh` adalah bagian dari alur resmi. Instalasi belum selesai sampai Broker melihat panggilan ChatGPT terautentikasi yang diharapkan dan audit chain tetap valid.

## Hanya untuk development

Diagnosis ad hoc, Actions probe sementara, runner khusus, CI ephemeral, hostname/IP/path/branch development, curl/openssl investigasi, dan SSH developer harus berada di issue/PR evidence, bukan tutorial publik.

## Review final rilis

1. Tinjau README dan semua terjemahan resmi.
2. Tinjau `installer-flow.md` dan `chatgpt-integration.md` untuk setiap bahasa resmi.
3. Tinjau landing page.
4. Hapus artefak development.
5. Pastikan tutorial tidak meminta kredensial VPS/SSH.
6. Pastikan setiap aksi manual menjelaskan alasan dan validasi.
7. Jalankan check installation contract dan coverage i18n di CI.
8. Jalankan instalasi penuh dari VPS bersih.
9. Akhiri dengan panggilan MCP nyata dari ChatGPT dan bukti audit Broker.

## Aturan engineering

**NATIVE FIRST.** Utamakan API resmi, konfigurasi yang didukung, Docker/Compose, MCP, OAuth/OIDC, dan mekanisme native. Kode khusus hanya jika perlu, harus minimal, terpusat, terdokumentasi, dan reversible.
