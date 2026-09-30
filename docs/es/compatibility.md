# Compatibilidad

Este documento separa soporte validado, cobertura de build y entornos no probados.

## Validado para el release candidate

### Sistema operativo

- Ubuntu 24.04 LTS en runners limpios de GitHub;
- host Linux con systemd para servicios/jobs del host.

### Arquitectura

- `linux/amd64`: aceptación de runtime ejecutada en CI limpio;
- `linux/arm64`: binarios e imágenes se construyen, pero todavía no existe aceptación completa equivalente en hardware arm64 nativo.

### Runtime

Mínimos para la ruta OAuth incluida:

- Docker Engine 24+;
- Docker Compose v2 vía `docker compose`;
- al menos 2 GB RAM para ZITADEL incluido;
- Git, OpenSSL, Python 3 y curl.

El mínimo de Docker/RAM sigue los requisitos oficiales de ZITADEL Compose. Es un mínimo funcional, no dimensionamiento recomendado de producción.

Referencias:

- https://zitadel.com/docs/self-hosting/deploy/compose
- https://zitadel.com/docs/self-hosting/manage/requirements

## Ruta pública con ChatGPT

Una instalación pública completa necesita:

- registro DNS A/AAAA bajo control del operador;
- TCP 80/443 público mediante Traefik compatible existente o el incluido;
- certificado HTTPS válido;
- cuenta/workspace de ChatGPT cuya superficie actual exponga modo desarrollador y registro de app MCP personalizada.

OpenAI controla disponibilidad por plan/workspace y rollout. Una etiqueta de suscripción por sí sola no garantiza compatibilidad. Portico tampoco puede elevar permisos del lado de ChatGPT.

Referencia actual del proyecto:
https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

El paquete usa un hostname público para MCP y el issuer OAuth/OIDC integrado.

## Builds

Release automation construye:

- `linux/amd64`
- `linux/arm64`

## Sin promesa actual de soporte

- hosts no Linux;
- Docker Desktop como VPS de producción;
- distribuciones sin systemd para service/job;
- Kubernetes;
- Podman Compose;
- Windows/macOS nativos;
- Full/R5 en producción.

Otros Linux pueden funcionar, pero siguen no validados.

## Dimensionamiento

2 GB es solo el suelo de la dependencia de identidad. ZITADEL advierte que hashing de contraseñas puede generar picos de CPU y publica recomendaciones mayores para producción/HA.

La carga MCP, workloads Docker existentes y retención de logs/auditoría cambian los requisitos reales. Usa `scripts/diagnose.sh health` y métricas del host; dimensiona producción con datos medidos.
