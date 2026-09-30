# Kompatibilitas

## Tervalidasi untuk release candidate

### Sistem operasi

- Ubuntu 24.04 LTS pada runner GitHub bersih;
- Linux dengan systemd untuk fungsi host service/job.

### Arsitektur

- `linux/amd64`: runtime acceptance lengkap di CI;
- `linux/arm64`: binary/image dibangun, tetapi full host acceptance setara belum dijalankan pada hardware arm64 native.

### Runtime

- Docker Engine 24+;
- Docker Compose v2 sebagai `docker compose`;
- minimal 2 GB RAM untuk ZITADEL bawaan;
- Git, OpenSSL, Python 3, curl.

2 GB adalah functional dependency floor, bukan rekomendasi sizing production.

Referensi:
- https://zitadel.com/docs/self-hosting/deploy/compose
- https://zitadel.com/docs/self-hosting/manage/requirements

## Jalur publik ChatGPT

Membutuhkan DNS A/AAAA, TCP 80/443 publik melalui Traefik compatible yang ada atau bawaan, HTTPS valid, dan account/workspace ChatGPT yang benar-benar menyediakan Developer Mode dan custom MCP app registration.

OpenAI mengontrol availability dan rollout. Portico tidak dapat meningkatkan permission sisi ChatGPT.

Referensi proyek:
https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

## Build

- `linux/amd64`
- `linux/arm64`

## Belum diklaim didukung

- host non-Linux;
- Docker Desktop sebagai VPS production;
- Linux tanpa systemd untuk host service/job;
- Kubernetes;
- Podman Compose;
- Windows/macOS native;
- Full/R5 production.

Distribusi Linux lain mungkin bekerja tetapi belum tervalidasi.

## Resource sizing

RAM minimum hanyalah floor dependency. Password hashing, workload MCP/Docker, dan retensi audit/log dapat meningkatkan kebutuhan nyata. Gunakan `scripts/diagnose.sh health` dan metric host.
