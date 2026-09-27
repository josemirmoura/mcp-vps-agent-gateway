#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

if [ ! -f .env ] || [ ! -f config/policy.yaml ]; then
  echo "Missing .env or config/policy.yaml. Run ./scripts/init.sh first." >&2
  exit 1
fi
if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Tracked working tree has local changes. Commit/stash them before update." >&2
  exit 1
fi

set -a
. ./.env
set +a

current="$(git rev-parse HEAD)"
target="${VPS_AGENT_UPDATE_REF:-origin/main}"
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

echo "Current version: $current"
git fetch --tags origin
target_sha="$(git rev-parse "$target")"
echo "Target version:  $target_sha"
if [ "$current" = "$target_sha" ]; then
  echo "Already at target version."
  exit 0
fi
if ! git merge-base --is-ancestor "$current" "$target_sha"; then
  echo "Refusing non-fast-forward update: target $target_sha is not a descendant of $current." >&2
  exit 1
fi

echo "Stopping package for a consistent state backup..."
"${compose[@]}" stop

tar -czf "$backup_dir/operator-state.tar.gz" .env config/policy.yaml state

if [ "${VPS_AGENT_AUTH_MODE:-}" = "integrated" ]; then
  echo "Snapshotting integrated identity volumes..."
  docker run --rm     -v mcp-vps-agent_zitadel-postgres-data:/source:ro     -v "$PWD/$backup_dir:/backup"     alpine:3.22 sh -c 'cd /source && tar -czf /backup/zitadel-postgres-volume.tar.gz .'
  docker run --rm     -v mcp-vps-agent_zitadel-bootstrap:/source:ro     -v "$PWD/$backup_dir:/backup"     alpine:3.22 sh -c 'cd /source && tar -czf /backup/zitadel-bootstrap-volume.tar.gz .'
fi
printf '%s\n' "$current" >"$backup_dir/previous-commit"
printf '%s\n' "$target_sha" >"$backup_dir/target-commit"

rollback() {
  echo "Update failed; rolling back to $current..." >&2
  "${compose[@]}" stop >/dev/null 2>&1 || true
  git reset --hard "$current"
  rm -rf state
  tar -xzf "$backup_dir/operator-state.tar.gz"

  if [ -f "$backup_dir/zitadel-postgres-volume.tar.gz" ]; then
    echo "Restoring integrated identity volumes..." >&2
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
  echo "Update did not land on the requested target SHA." >&2
  false
fi
"${compose[@]}" config -q

echo "Building target images before touching the real state database..."
"${compose[@]}" build

migration_dir="$backup_dir/migration-check"
mkdir -p "$migration_dir"
tar -xzf "$backup_dir/operator-state.tar.gz" -C "$migration_dir" state
if [ ! -f "$migration_dir/state/state.db" ]; then
  echo "State backup does not contain state/state.db; refusing update." >&2
  false
fi
broker_image="${VPS_AGENT_BROKER_IMAGE:-mcp-vps-agent-broker:local}"
echo "Validating target schema migration and audit chain on a copied database..."
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

echo "UPDATE COMPLETE: $current -> $target_sha"
echo "Backup retained at $backup_dir"
