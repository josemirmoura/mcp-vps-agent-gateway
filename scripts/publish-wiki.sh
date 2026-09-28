#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

usage() {
  cat <<'EOF'
Usage: bash scripts/publish-wiki.sh [--check]

Publishes only the versioned Wiki navigation files:
  wiki/Home.md
  wiki/Inicio.md
  wiki/_Sidebar.md

The script preserves any other pages already present in the GitHub Wiki.
It reuses the Git authentication already configured for this environment.

Options:
  --check   verify that the Wiki remote is reachable and show whether the
            three managed pages differ, without committing or pushing.

Optional environment override:
  VPS_AGENT_WIKI_REMOTE=<git-url>
EOF
}

check_only=0
case "${1:-}" in
  "") ;;
  --check) check_only=1 ;;
  -h|--help) usage; exit 0 ;;
  *) usage >&2; exit 2 ;;
esac

managed_files=(Home.md Inicio.md _Sidebar.md)
for file in "${managed_files[@]}"; do
  if [ ! -f "wiki/$file" ]; then
    echo "Missing managed Wiki source: wiki/$file" >&2
    exit 1
  fi
done

origin_url="$(git remote get-url origin 2>/dev/null || true)"
if [ -z "$origin_url" ] && [ -z "${VPS_AGENT_WIKI_REMOTE:-}" ]; then
  echo "No Git origin remote found. Set VPS_AGENT_WIKI_REMOTE explicitly." >&2
  exit 1
fi

wiki_remote="${VPS_AGENT_WIKI_REMOTE:-}"
if [ -z "$wiki_remote" ]; then
  case "$origin_url" in
    git@github.com:*.git)
      wiki_remote="${origin_url%.git}.wiki.git"
      ;;
    https://github.com/*.git)
      wiki_remote="${origin_url%.git}.wiki.git"
      ;;
    ssh://git@github.com/*.git)
      wiki_remote="${origin_url%.git}.wiki.git"
      ;;
    *)
      echo "Cannot derive the GitHub Wiki remote from origin: $origin_url" >&2
      echo "Set VPS_AGENT_WIKI_REMOTE to the Wiki Git URL explicitly." >&2
      exit 1
      ;;
  esac
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
wiki_dir="$tmp_dir/wiki"

if ! git clone --quiet "$wiki_remote" "$wiki_dir"; then
  cat >&2 <<EOF
Unable to clone the GitHub Wiki remote:
  $wiki_remote

Check that:
  1. the repository Wiki is enabled and initialized;
  2. this environment already has GitHub write authentication;
  3. the authenticated account can edit the Wiki.

No repository or Wiki content was changed.
EOF
  exit 1
fi

for file in "${managed_files[@]}"; do
  cp "wiki/$file" "$wiki_dir/$file"
done

git -C "$wiki_dir" add -- "${managed_files[@]}"

if git -C "$wiki_dir" diff --cached --quiet; then
  echo "Wiki navigation is already in sync."
  exit 0
fi

if [ "$check_only" -eq 1 ]; then
  echo "Wiki navigation differs from the versioned source."
  git -C "$wiki_dir" diff --cached --stat -- "${managed_files[@]}"
  exit 0
fi

if [ -z "$(git config user.name || true)" ] || [ -z "$(git config user.email || true)" ]; then
  cat >&2 <<'EOF'
Git author identity is not configured in this environment.
Configure it once, then rerun:
  git config --global user.name "Your Name"
  git config --global user.email "you@example.com"
EOF
  exit 1
fi

git -C "$wiki_dir" commit --quiet -m "docs: sync Wiki navigation"
git -C "$wiki_dir" push --quiet origin HEAD

echo "Wiki navigation published successfully."
echo "Managed pages: Home, Inicio, _Sidebar"
