# Compatibilidade

Este documento separa suporte validado, cobertura de build e ambientes não testados.

## Validado para o release candidate

### Sistema operacional

- Ubuntu 24.04 LTS em runners GitHub limpos;
- host Linux com systemd para funcionalidades de serviço/job do host.

### Arquitetura

- `linux/amd64`: aceitação de runtime executada em CI limpa;
- `linux/arm64`: binários e imagens são construídos, mas aceitação completa equivalente em hardware arm64 nativo ainda não foi executada.

### Runtime

Mínimos da rota OAuth embutida:

- Docker Engine 24+;
- Docker Compose v2 como `docker compose`;
- pelo menos 2 GB RAM para ZITADEL;
- Git;
- OpenSSL;
- Python 3;
- curl.

Esse piso segue os requisitos oficiais do ZITADEL Compose e é mínimo funcional, não recomendação de sizing de produção.

Referências:

- https://zitadel.com/docs/self-hosting/deploy/compose
- https://zitadel.com/docs/self-hosting/manage/requirements

## Caminho público ChatGPT

Uma instalação pública completa exige:

- DNS A/AAAA controlado pelo operador;
- TCP público 80/443 por Traefik existente compatível ou embutido;
- certificado HTTPS válido;
- conta/workspace do ChatGPT cuja superfície atual realmente exponha Developer Mode e registro de app MCP customizado.

A OpenAI controla disponibilidade, controles de workspace e rollout. O projeto não trata o nome do plano como garantia. O Portico não consegue ampliar permissões do lado do ChatGPT.

Referência:
https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

## Builds

- `linux/amd64`
- `linux/arm64`

## Sem promessa atual de suporte

- hosts não Linux;
- Docker Desktop como VPS de produção;
- Linux sem systemd para host service/job;
- Kubernetes;
- Podman Compose;
- Windows/macOS nativos;
- Full/R5 em produção.

Outras distribuições Linux podem funcionar, mas permanecem não testadas até evidência explícita.

## Recursos

2 GB RAM é apenas o piso da dependência de identidade. Hash de senhas, workloads MCP/Docker existentes e retenção de logs/auditoria alteram os requisitos reais. Use `scripts/diagnose.sh health` e métricas do host.
