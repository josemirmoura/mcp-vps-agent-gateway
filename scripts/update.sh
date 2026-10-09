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
    latest_rc="$(git tag -l 'v[0-9]*-rc.[0-9]*' --sort=-v:refname | head -n 1 || true)"
    if [ -n "$latest_rc" ]; then
      echo "  VPS_AGENT_UPDATE_REF=$latest_rc bash scripts/update.sh" >&2
    else
      echo "  VPS_AGENT_UPDATE_REF=<published-rc-tag> bash scripts/update.sh" >&2
    fi
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
  echo "$(vps_agent_text "Refusing non-fast-forward update: target $target_sha is not a descendant of $current." "Atualização non-fast-forward recusada: o alvo $target_sha não descende de $current.")" >&2
  exit 1
fi

# At this point the live package has not been modified. An offline backup
# requires stopping the services, but backup creation can fail (full disk,
# missing volume, interrupted shell). Never leave the original package down
# merely because its backup could not be completed. Do not restore partial
# archives or reset Git while the original runtime and state are unchanged.
resume_after_backup_failure() {
  local failed_status="$?"
  trap - ERR INT TERM
  echo "$(vps_agent_text 'Backup interrupted or failed; restarting the unchanged package...' 'Backup interrompido ou com falha; reiniciando o pacote original sem alterações...')" >&2
  if ! "${compose[@]}" up -d; then
    echo "$(vps_agent_text 'Automatic restart failed. Use the existing Compose configuration for operator recovery.' 'Falha ao reiniciar automaticamente. Use a configuração Compose existente para recuperação pelo operador.')" >&2
  fi
  exit "$(( failed_status == 0 ? 1 : failed_status ))"
}

trap resume_after_backup_failure ERR INT TERM
echo "$(vps_agent_text 'Stopping package for a consistent state backup...' 'Parando o pacote para criar um backup consistente do estado...')"
"${compose[@]}" stop

tar -czf "$backup_dir/operator-state.tar.gz" .env config/policy.yaml state

if [ "${VPS_AGENT_AUTH_MODE:-}" = "integrated" ]; then
  echo "$(vps_agent_text 'Snapshotting integrated identity volumes...' 'Criando snapshot dos volumes de identidade integrada...')"
  docker run --rm     -v mcp-vps-agent_zitadel-postgres-data:/source:ro     -v "$PWD/$backup_dir:/backup"     alpine:3.22 sh -c 'cd /source && tar -czf /backup/zitadel-postgres-volume.tar.gz .'
  docker run --rm     -v mcp-vps-agent_zitadel-bootstrap:/source:ro     -v "$PWD/$backup_dir:/backup"     alpine:3.22 sh -c 'cd /source && tar -czf /backup/zitadel-bootstrap-volume.tar.gz .'
fi
# Refuse to leave the pre-backup recovery window until all archives are
# readable. A successful tar/docker exit alone does not prove the snapshot
# file was actually created or can be decompressed.
tar -tzf "$backup_dir/operator-state.tar.gz" >/dev/null
if [ "${VPS_AGENT_AUTH_MODE:-}" = "integrated" ]; then
  tar -tzf "$backup_dir/zitadel-postgres-volume.tar.gz" >/dev/null
  tar -tzf "$backup_dir/zitadel-bootstrap-volume.tar.gz" >/dev/null
fi
printf '%s\n' "$current" >"$backup_dir/previous-commit"
printf '%s\n' "$target_sha" >"$backup_dir/target-commit"

# Full, usable backup now exists. From this point on, migration/rollout
# errors follow the normal state-restoring rollback path below.
trap - ERR INT TERM

rollback() {
  local failed_status="$?"
  [ "$failed_status" -ne 0 ] || failed_status=1
  # Never recurse into the rollback handler while recovering from a failure.
  trap - ERR INT TERM
  echo "$(vps_agent_text "Update failed; staging rollback from $current..." "Atualização falhou; preparando retorno para $current...")" >&2

  local stage old_state helper_copy
  stage="$(mktemp -d "$PWD/$backup_dir/rollback-stage.XXXXXXXX")" || exit "$failed_status"

  # The original state stays untouched until the archive is read, validated
  # and extracted successfully to a separate directory. Invalid, incomplete,
  # or malicious archives must not trigger rm -rf state.
  if ! python3 scripts/lib/update-snapshot.py \
      --archive "$backup_dir/operator-state.tar.gz" --dest "$stage"; then
    echo "ROLLBACK BLOCKED: unusable operator snapshot; live state unchanged. Recover manually using $backup_dir." >&2
    exit "$failed_status"
  fi

  # This helper must remain available after git reset returns to the old
  # checkout, where it may not have existed yet.
  helper_copy="$PWD/$backup_dir/restore-identity-volume.sh"
  if ! cp scripts/lib/restore-identity-volume.sh "$helper_copy"; then
    echo "ROLLBACK BLOCKED: could not preserve volume recovery helper." >&2
    exit "$failed_status"
  fi
  chmod 700 "$helper_copy"

  if ! "${compose[@]}" stop; then
    echo "ROLLBACK BLOCKED: unable to stop upgraded services safely." >&2
    exit "$failed_status"
  fi
  if ! git reset --hard "$current"; then
    echo "ROLLBACK BLOCKED: unable to restore previous source revision." >&2
    exit "$failed_status"
  fi

  # Stage each volume within its Docker volume BEFORE moving any live entry.
  # On move errors, the isolated helper attempts to restore the previous
  # entries, leaving its work directories for manual recovery if needed.
  if [ -f "$backup_dir/zitadel-postgres-volume.tar.gz" ]; then
    for spec in \
      "mcp-vps-agent_zitadel-postgres-data:zitadel-postgres-volume.tar.gz" \
      "mcp-vps-agent_zitadel-bootstrap:zitadel-bootstrap-volume.tar.gz"
    do
      local volume archive_name
      volume="${spec%%:*}"
      archive_name="${spec#*:}"
      if ! docker run --rm --network none \
          -v "$volume:/target" \
          -v "$PWD/$backup_dir:/backup:ro" \
          alpine:3.22 sh /backup/restore-identity-volume.sh "/backup/$archive_name"; then
        echo "ROLLBACK BLOCKED: identity volume restore failed ($volume); snapshot retained." >&2
        exit "$failed_status"
      fi
    done
  fi

  # Do not delete the upgraded state directory. Rename it into the retained
  # backup first, then atomically place the prevalidated original state.
  old_state="$PWD/$backup_dir/failed-target-state"
  if [ -e "$old_state" ]; then
    echo "ROLLBACK BLOCKED: displaced state backup already exists." >&2
    exit "$failed_status"
  fi
  if [ -e state ] && ! mv state "$old_state"; then
    echo "ROLLBACK BLOCKED: could not preserve upgraded state." >&2
    exit "$failed_status"
  fi
  if ! mv "$stage/state" state; then
    [ ! -e "$old_state" ] || mv "$old_state" state || true
    echo "ROLLBACK BLOCKED: could not install staged original state." >&2
    exit "$failed_status"
  fi
  # The source archive is retained; these same-filesystem renames avoid
  # partially written .env/policy files during rollback.
  if ! mv -f "$stage/.env" .env || \
     ! mv -f "$stage/config/policy.yaml" config/policy.yaml; then
    echo "ROLLBACK BLOCKED: operator configuration was not fully restored." >&2
    exit "$failed_status"
  fi

  if ! "${compose[@]}" up -d --build || \
     ! bash ./scripts/verify.sh; then
    echo "ROLLBACK FAILED: original runtime did not pass health verification." >&2
    exit "$failed_status"
  fi
  if [ "${VPS_AGENT_AUTH_MODE:-}" = "integrated" ] && ! bash ./scripts/verify-public.sh; then
    echo "ROLLBACK FAILED: original public authentication did not pass verification." >&2
    exit "$failed_status"
  fi
  printf '{"time":"%s","from":"%s","to":"%s","result":"rolled_back"}\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$current" "$target_sha" >> state/update.log
  echo "ROLLBACK COMPLETE: original runtime recovered; upgrade remains failed." >&2
  exit "$failed_status"
}
trap rollback ERR INT TERM

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
