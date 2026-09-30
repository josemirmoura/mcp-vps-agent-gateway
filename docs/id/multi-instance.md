# Beberapa instance VPS dalam satu workspace ChatGPT

Ini skenario advanced. Onboarding normal tetap satu instance Portico MCP per instalasi.

## Model identity

Nama tool tetap stabil antar instalasi. Jangan rename tool per server.

Setiap instalasi memiliki:

- endpoint MCP HTTPS sendiri;
- relasi client/resource OAuth dan kredensial sendiri;
- `VPS_AGENT_INSTANCE_ID` stabil dari bootstrap;
- `VPS_AGENT_INSTANCE_NAME` yang terlihat operator;
- identity instance di `system.info`, audit Broker, dan structured log.

Contoh:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Loja"
VPS_AGENT_PUBLIC_URL=https://mcp-loja.example.com/mcp
~~~

VPS lain:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Blog"
VPS_AGENT_PUBLIC_URL=https://mcp-blog.example.com/mcp
~~~

Daftarkan sebagai app ChatGPT terpisah dengan nama terlihat yang cocok. Pilih atau @mention app target, jangan bergantung pada silent disambiguation nama tool yang sama.

## Verifikasi

Untuk setiap app, panggil `system.info` dan bandingkan `instance_id` serta `instance_name` dengan VPS yang diharapkan. Hostname hanya informasi pendukung karena provider dapat menggunakan hostname sama pada mesin disposable berbeda.

Lalu periksa audit Broker:

~~~bash
bash scripts/diagnose.sh audit 20
~~~

Identity instance yang sama harus muncul pada event audit dan structured log baru.

## Penggunaan dalam conversation sama

Jika UI ChatGPT mengizinkan beberapa custom MCP app pada satu conversation, pilih atau mention app secara eksplisit sebelum operasi. Perilaku UI dapat berubah tanpa perubahan server.

## Automated simultaneous-use gate

Repository memiliki workflow tiga mesin. Paket Docker nyata dijalankan bersamaan pada tiga runner Ubuntu independen, sengaja memakai subject, tool names, logical path, dan operation id yang sama.

Gate lolos jika:

- semua installation punya ID/nama berbeda;
- credential independen;
- lifetime runner overlap;
- operation id sama berhasil independen di tiga Broker state store;
- tiap audit chain hanya merekam identity sendiri;
- hostname dan audit head tetap berbeda;
- setiap instance selesai dengan state yang diharapkan.

Ini adalah server-side collision test tanpa manusia.

## ChatGPT product surface

Menghubungkan beberapa live app ke workspace yang sama adalah eksperimen UI terpisah. Bukan syarat onboarding single-VPS atau final E2E single-instance. Jika diuji, beri nama tiap VPS sebagai app terpisah dan pilih secara eksplisit.
