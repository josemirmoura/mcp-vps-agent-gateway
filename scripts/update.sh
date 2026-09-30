#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh
vps_agent_init_language ""

if [ ! -f .env ] || [ ! -f config/policy.yaml ]; then
  echo "$(vps_agent_text 'Missing .env or config/policy.yaml. Run scripts/init.sh first.' 'Arquivo .env ou config/policy.yaml ausente. Execute scripts/init.sh primeiro.')" >&2
  exit 1
fi
if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "$(vps_agent_text 'Tracked working tree has local changes. Commit/stash them before update.' 'A árvore de trabalho possui alterações rastreadas. Faça commit/stash antes da atualização.')" >&2
  exit 1
fi

set -a
. ./.env
set +a

current="$(git rev-parse HEAD)"
current_label="$(git describe --tags --exact-match 2>/dev/null || true)"
[ -n "$current_label" ] || current_label="$(vps_agent_version)+${current:0:12}"

git fetch --tags origin

if [ -n "${VPS_AGENT_UPDATE_REF:-}" ]; then
  target="$VPS_AGENT_UPDATE_REF"
  channel="explicit"
else
  target="$(git tag -l 'v[0-9]*' --sort=-v:refname | grep -Ev -- '-' | head -n 1 || true)"
  channel="stable"
  if [ -z "$target" ]; then
    echo "$(vps_agent_text 'No stable SemVer release tag is available yet.' 'Ainda não há uma tag SemVer estável disponível.')" >&2
    echo "$(vps_agent_text 'main is intentionally not an automatic production update channel.' 'A branch main não é, intencionalmente, um canal automático de atualização de produção.')" >&2
    echo "$(vps_agent_text 'For release-candidate testing, select a target explicitly, for example:' 'Para testar release candidates, selecione explicitamente um alvo, por exemplo:')" >&2
    echo "  VPS_AGENT_UPDATE_REF=v0.1.0-rc.3 bash scripts/update.sh" >&2
    exit 2
  fi
fi

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup_dir="backups/$stamp"
mkdir -p "$backup_dir"
chmod 700 backups "$backup_dir"

compose=(docker compose -f compose.yaml)
if [ "${VPS_AGENT_WHOLE_HOST:-0}" = "1" ]; then
  compose+=(-f compose.host.yaml)
fi
if [ "${VPS_AGENT_AUTH_MODE:-}" = "integrated" ] && [ -f compose.integrated-auth.yaml ]; then
  compose+=(-f compose.integrated-auth.yaml)
  if [ "${VPS_AGENT_BUNDLED_PROXY:-0}" = "1" ] && [ -f compose.integrated-auth.proxy.yaml ]; then
    compose+=(-f compose.integrated-auth.proxy.yaml)
  fi
fi

vps_agent_banner
echo "$(vps_agent_text 'Update channel' 'Canal de atualização'):  $channel"
echo "$(vps_agent_text 'Current version' 'Versão atual'): $current_label"
echo "$(vps_agent_text 'Current commit' 'Commit atual'):  $current"
target_sha="$(git rev-parse "$target")"
target_label="$target"
target_product_version="$(git show "$target_sha:VERSION" 2>/dev/null | tr -d '[:space:]' || true)"
[ -n "$target_product_version" ] && target_label="$target_product_version ($target)"
echo "$(vps_agent_text 'Target version' 'Versão alvo'):  $target_label"
echo "$(vps_agent_text 'Target commit' 'Commit alvo'):   $target_sha"
echo
echo "$(vps_agent_text 'Changes:' 'Alterações:')"
git log --oneline --no-decorate "$current..$target_sha" | head -n 12 || true
if [ "$current" = "$target_sha" ]; then
  echo "$(vps_agent_text 'Already at target version.' 'Já está na versão alvo.')"
  exit 0
fi
if ! git merge-base --is-ancestor "$current" "$target_sha"; then
  echo "Refusing non-fast-forward update: target $target_sha is not a descendant of $current." >&2
  exit 1
fi

echo "$(vps_agent_text 'Stopping package for a consistent state backup...' 'Parando o pacote para criar um backup consistente do estado...')"
"${compose[@]}" stop

tar -czf "$backup_dir/operator-state.tar.gz" .env config/policy.yaml state

if [ "${VPS_AGENT_AUTH_MODE:-}" = "integrated" ]; then
  echo "$(vps_agent_text 'Snapshotting integrated identity volumes...' 'Criando snapshot dos volumes de identidade integrada...')"
  docker run --rm     -v mcp-vps-agent_zitadel-postgres-data:/source:ro     -v "$PWD/$backup_dir:/backup"     alpine:3.22 sh -c 'cd /source && tar -czf /backup/zitadel-postgres-volume.tar.gz .'
  docker run --rm     -v mcp-vps-agent_zitadel-bootstrap:/source:ro     -v "$PWD/$backup_dir:/backup"     alpine:3.22 sh -c 'cd /source && tar -czf /backup/zitadel-bootstrap-volume.tar.gz .'
fi
printf '%s\n' "$current" >"$backup_dir/previous-commit"
printf '%s\n' "$target_sha" >"$backup_dir/target-commit"

rollback() {
  echo "$(vps_agent_text "Update failed; rolling back to $current..." "Atualização falhou; revertendo para $current...")" >&2
  "${compose[@]}" stop >/dev/null 2>&1 || true
  git reset --hard "$current"
  rm -rf state
  tar -xzf "$backup_dir/operator-state.tar.gz"

  if [ -f "$backup_dir/zitadel-postgres-volume.tar.gz" ]; then
    echo "$(vps_agent_text 'Restoring integrated identity volumes...' 'Restaurando volumes de identidade integrada...')" >&2
    docker run --rm       -v mcp-vps-agent_zitadel-postgres-data:/target       -v "$PWD/$backup_dir:/backup:ro"       alpine:3.22 sh -c 'find /target -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +; tar -xzf /backup/zitadel-postgres-volume.tar.gz -C /target'
    docker run --rm       -v mcp-vps-agent_zitadel-bootstrap:/target       -v "$PWD/$backup_dir:/backup:ro"       alpine:3.22 sh -c 'find /target -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +; tar -xzf /backup/zitadel-bootstrap-volume.tar.gz -C /target'
  fi

  "${compose[@]}" up -d --build
  bash ./scripts/verify.sh
  if [ "${VPS_AGENT_AUTH_MODE:-}" = "integrated" ]; then
    bash ./scripts/verify-public.sh
  fi
  printf '{"time":"%s","from":"%s","to":"%s","result":"rolled_back"}\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$current" "$target_sha" >> state/update.log
}
trap rollback ERR

git merge --ff-only "$target"
if [ "$(git rev-parse HEAD)" != "$target_sha" ]; then
  echo "$(vps_agent_text 'Update did not land on the requested target SHA.' 'A atualização não terminou no SHA alvo solicitado.')" >&2
  false
fi
"${compose[@]}" config -q

echo "$(vps_agent_text 'Building target images before touching the real state database...' 'Construindo as imagens alvo antes de tocar no banco real de estado...')"
"${compose[@]}" build

migration_dir="$backup_dir/migration-check"
mkdir -p "$migration_dir"
tar -xzf "$backup_dir/operator-state.tar.gz" -C "$migration_dir" state
if [ ! -f "$migration_dir/state/state.db" ]; then
  echo "$(vps_agent_text 'State backup does not contain state/state.db; refusing update.' 'O backup de estado não contém state/state.db; atualização recusada.')" >&2
  false
fi
broker_image="${VPS_AGENT_BROKER_IMAGE:-mcp-vps-agent-broker:local}"
echo "$(vps_agent_text 'Validating target schema migration and audit chain on a copied database...' 'Validando a migração de schema e a cadeia de auditoria em uma cópia do banco...')"
docker run --rm \
  --entrypoint /usr/local/bin/vps-agent \
  -v "$PWD/$migration_dir/state:/check" \
  "$broker_image" state-check --db /check/state.db | tee "$backup_dir/migration-check.json"

"${compose[@]}" up -d
bash ./scripts/verify.sh
if [ "${VPS_AGENT_AUTH_MODE:-}" = "integrated" ]; then
  bash ./scripts/verify-public.sh
fi

trap - ERR
printf '{"time":"%s","from":"%s","to":"%s","result":"success"}\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$current" "$target_sha" >> state/update.log

echo "$(vps_agent_text 'UPDATE COMPLETE' 'ATUALIZAÇÃO CONCLUÍDA'): $current -> $target_sha"
echo "$(vps_agent_text 'Backup retained at' 'Backup mantido em') $backup_dir"
