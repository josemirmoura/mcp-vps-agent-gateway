#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

export VPS_AGENT_ADMIN_TOKEN="${VPS_AGENT_ADMIN_TOKEN:-compose-validation-admin-token}"
export VPS_AGENT_STATIC_TOKEN="${VPS_AGENT_STATIC_TOKEN:-compose-validation-static-token}"
export VPS_AGENT_SUBJECT="${VPS_AGENT_SUBJECT:-compose-validation-operator}"
export VPS_AGENT_SCOPE_ROOT="${VPS_AGENT_SCOPE_ROOT:-/tmp/vps-agent-compose-validation}"
export VPS_AGENT_AUTH_MODE="${VPS_AGENT_AUTH_MODE:-integrated}"
export VPS_AGENT_DOMAIN="${VPS_AGENT_DOMAIN:-mcp.example.invalid}"
export VPS_AGENT_PUBLIC_URL="${VPS_AGENT_PUBLIC_URL:-https://mcp.example.invalid/mcp}"
export VPS_AGENT_OIDC_ISSUER="${VPS_AGENT_OIDC_ISSUER:-https://mcp.example.invalid}"
export VPS_AGENT_OAUTH_RESOURCE="${VPS_AGENT_OAUTH_RESOURCE:-https://mcp.example.invalid/mcp}"
export VPS_AGENT_EDGE_NETWORK="${VPS_AGENT_EDGE_NETWORK:-compose-validation-edge}"
export VPS_AGENT_TRAEFIK_CERTRESOLVER="${VPS_AGENT_TRAEFIK_CERTRESOLVER:-letsencrypt}"
export VPS_AGENT_LETSENCRYPT_EMAIL="${VPS_AGENT_LETSENCRYPT_EMAIL:-ops@example.invalid}"
export PORTICO_CLOUD_URL="${PORTICO_CLOUD_URL:-https://api.portico.example.invalid}"
export ZITADEL_BOOTSTRAP_PAT_EXPIRATION="${ZITADEL_BOOTSTRAP_PAT_EXPIRATION:-2099-01-01T00:00:00Z}"
export ZITADEL_LOGIN_PAT_EXPIRATION="${ZITADEL_LOGIN_PAT_EXPIRATION:-2099-01-01T00:00:00Z}"
export ZITADEL_MASTERKEY="${ZITADEL_MASTERKEY:-$(openssl rand -hex 16)}"
export ZITADEL_POSTGRES_PASSWORD="${ZITADEL_POSTGRES_PASSWORD:-$(openssl rand -hex 24)}"
export ZITADEL_BOOTSTRAP_ADMIN_PASSWORD="${ZITADEL_BOOTSTRAP_ADMIN_PASSWORD:-T1!$(openssl rand -hex 18)}"

docker compose \
  -f compose.yaml \
  -f compose.integrated-auth.yaml \
  -f compose.integrated-auth.proxy.yaml \
  config -q

docker compose \
  -f compose.yaml \
  -f compose.cloud.yaml \
  config -q

echo "INTEGRATED + CLOUD COMPOSE MODELS: PASS"
