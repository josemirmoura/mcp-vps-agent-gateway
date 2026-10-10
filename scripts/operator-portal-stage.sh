#!/usr/bin/env bash
# Operator portal staging: no live container or operational file changes.
set -euo pipefail
ROOT="$(pwd -P)"
test -d "$ROOT/.git" || { echo "Execute na raiz do repositório Git operacional." >&2; exit 2; }
test -f "$ROOT/.env" && test -f "$ROOT/compose.yaml" && test -d "$ROOT/state" || {
 echo "Instalação incompleta; staging cancelado." >&2; exit 2;
}
if [[ -n "$(git status --porcelain)" ]]; then
 echo "Checkout operacional contém alterações locais; staging cancelado." >&2; exit 2
fi
CURRENT="$(git rev-parse HEAD)"
# The integration branch moves as other blocks are merged. Never validate an
# unpinned or old authorization branch in place of the reviewed candidate.
CANDIDATE_SHA="${PORTICO_STAGE_CANDIDATE_SHA:-}"
if [[ ! "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]]; then
 echo "Informe PORTICO_STAGE_CANDIDATE_SHA com os 40 caracteres do commit revisado." >&2
 exit 2
fi
CANDIDATE_REF="${PORTICO_STAGE_CANDIDATE_REF:-refs/heads/integration/community-stable-block5-20261009}"
case "$CANDIDATE_REF" in
 refs/heads/*|refs/tags/*) ;;
 *) echo "PORTICO_STAGE_CANDIDATE_REF deve ser refs/heads/... ou refs/tags/..." >&2; exit 2 ;;
esac
if ! git check-ref-format "$CANDIDATE_REF"; then
 echo "Referência Git do candidato inválida; staging cancelado." >&2
 exit 2
fi
git fetch --quiet --no-tags origin "$CANDIDATE_REF"
CANDIDATE="$(git rev-parse --verify 'FETCH_HEAD^{commit}')"
if [[ "$CANDIDATE" != "$CANDIDATE_SHA" ]]; then
 echo "Candidato divergiu do SHA revisado; staging cancelado. Atualize a aprovação antes de prosseguir." >&2
 exit 2
fi
git merge-base --is-ancestor "$CURRENT" "$CANDIDATE" || {
 echo "Candidato não descende do commit operacional; abortado." >&2; exit 2
}
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
PARENT="${PORTICO_STAGE_PARENT:-$HOME/portico-operator-validation}"
case "$PARENT" in
 "$ROOT"|"$ROOT"/*) echo "Staging deve ficar fora da instalação operacional." >&2; exit 2;;
esac
umask 077
mkdir -p "$PARENT"
chmod 700 "$PARENT"
STAGE="$PARENT/stage-$STAMP"
BACKUP="$PARENT/backup-$STAMP"
if [[ -e "$STAGE" || -e "$BACKUP" ]]; then
 echo "Diretório existente; nenhuma alteração." >&2; exit 2
fi
git worktree add --quiet --detach "$STAGE" "$CANDIDATE"
python3 "$STAGE/scripts/operator-portal-snapshot.py" --output "$BACKUP"
if docker info >/dev/null 2>&1; then
 DOCKER=(docker)
else
 echo "Docker exige autenticação sudo apenas para validar a configuração."
 DOCKER=(sudo docker)
fi
"${DOCKER[@]}" compose --env-file "$ROOT/.env" -f "$ROOT/compose.yaml" config --quiet
"${DOCKER[@]}" compose --env-file "$ROOT/.env" -f "$STAGE/compose.yaml" --profile operator-portal config --quiet
if [[ -f "$STAGE/compose.operator-portal.edge.yaml" && -f "$STAGE/compose.integrated-auth.yaml" ]]; then
 "${DOCKER[@]}" compose --env-file "$ROOT/.env" \
  -f "$STAGE/compose.yaml" -f "$STAGE/compose.integrated-auth.yaml" \
  -f "$STAGE/compose.operator-portal.edge.yaml" \
  --profile operator-portal config --quiet
 echo "TRAEFIK OPERATOR ROUTE MODEL: PASS (no deployment)"
fi
python3 -m unittest discover -s "$STAGE/web/operator-approval" -p 'test_*.py' -q
echo "STAGING READY, NO DEPLOY"
printf 'Installed SHA: %s\nCandidate SHA: %s\nStage: %s\nBackup: %s\n' "$CURRENT" "$CANDIDATE" "$STAGE" "$BACKUP"
echo "Contêineres em produção e checkout main permanecem sem alterações."
