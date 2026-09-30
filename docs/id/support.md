# Kebijakan support

Portico MCP adalah project open source.

## Release line yang didukung

Selama fase awal productization:

- stable release `0.x` terbaru adalah public supported line;
- release candidate saat ini didukung untuk acceptance/testing;
- `main` adalah development dan bukan stable support channel.

Security fix dapat mengharuskan upgrade ke patch release terbaru.

## Mendapatkan bantuan

Gunakan GitHub issue untuk bug yang dapat direproduksi, kegagalan instalasi, dan masalah dokumentasi yang tidak mengandung secret.

Sebelum membuka issue, kumpulkan sanitized diagnostic bundle bila praktis:

~~~bash
bash scripts/diagnose.sh bundle
~~~

Jangan pernah lampirkan `.env`, raw credential, private SSH key, atau secret material tanpa redaksi.

## Security issue

Jangan publikasikan detail exploit di issue biasa. Ikuti `SECURITY.md`.

## Service level

Tidak ada jaminan response-time atau uptime SLA untuk project open source.

Komitmen support untuk distribusi komersial masa depan, jika ada, harus didokumentasikan terpisah dan tidak boleh diasumsikan dari repository ini.
