#!/usr/bin/env bash

# Canonical product identity and human-facing localization helpers.
# Machine-readable APIs and JSON-producing commands must stay language-neutral.

VPS_AGENT_PRODUCT_NAME="Portico MCP"
VPS_AGENT_PRODUCT_SLUG="portico-mcp"
VPS_AGENT_AUTHOR_NAME="Josemir Moura"
VPS_AGENT_AUTHOR_GITHUB="github.com/josemirmoura"
VPS_AGENT_OFFICIAL_LANGS="en pt-BR es de fr ja id"

vps_agent_repo_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")/../.." >/dev/null 2>&1 && pwd
}

vps_agent_version() {
  local root
  root="$(vps_agent_repo_root)"
  if [ -r "$root/VERSION" ]; then
    tr -d '[:space:]' <"$root/VERSION"
    return
  fi
  if command -v git >/dev/null 2>&1 && git -C "$root" describe --tags --always --dirty >/dev/null 2>&1; then
    git -C "$root" describe --tags --always --dirty
    return
  fi
  printf 'unknown'
}

vps_agent_normalize_lang() {
  local raw="${1:-}"
  raw="${raw%%.*}"
  raw="${raw//@/-}"
  raw="${raw//_/-}"
  local lower
  lower="$(printf '%s' "$raw" | tr '[:upper:]' '[:lower:]')"
  case "$lower" in
    pt|pt-br|pt-br-*) printf 'pt-BR' ;;
    en|en-*) printf 'en' ;;
    es|es-*) printf 'es' ;;
    de|de-*) printf 'de' ;;
    fr|fr-*) printf 'fr' ;;
    ja|ja-*) printf 'ja' ;;
    id|id-*) printf 'id' ;;
    *) printf 'en' ;;
  esac
}

vps_agent_init_language() {
  local explicit="${1:-}"
  local detected
  if [ -n "$explicit" ]; then
    detected="$explicit"
  elif [ -n "${VPS_AGENT_LANG:-}" ]; then
    detected="$VPS_AGENT_LANG"
  elif [ -n "${LC_ALL:-}" ]; then
    detected="$LC_ALL"
  elif [ -n "${LC_MESSAGES:-}" ]; then
    detected="$LC_MESSAGES"
  else
    detected="${LANG:-en}"
  fi
  VPS_AGENT_LANG="$(vps_agent_normalize_lang "$detected")"
  export VPS_AGENT_LANG
}

vps_agent_is_pt_br() {
  [ "${VPS_AGENT_LANG:-en}" = "pt-BR" ]
}

vps_agent_i18n_python() {
  local root
  root="$(vps_agent_repo_root)"
  python3 "$root/scripts/lib/i18n.py" "$@"
}

vps_agent_text() {
  local en="$1"
  local pt="$2"
  case "${VPS_AGENT_LANG:-en}" in
    en) printf '%s' "$en" ;;
    pt-BR) printf '%s' "$pt" ;;
    *)
      if command -v python3 >/dev/null 2>&1; then
        vps_agent_i18n_python text "$en" "$pt"
      else
        printf '%s' "$en"
      fi
      ;;
  esac
}

vps_agent_block() {
  local key="$1"
  local en="$2"
  local pt="$3"
  case "${VPS_AGENT_LANG:-en}" in
    en) printf '%s' "$en" ;;
    pt-BR) printf '%s' "$pt" ;;
    *)
      if command -v python3 >/dev/null 2>&1; then
        vps_agent_i18n_python block "$key" "$en" "$pt"
      else
        printf '%s' "$en"
      fi
      ;;
  esac
}

vps_agent_msg() {
  local key="$1"
  local en="$2"
  local pt="$3"
  shift 3
  case "${VPS_AGENT_LANG:-en}" in
    en)
      printf '%s' "$en"
      ;;
    pt-BR)
      printf '%s' "$pt"
      ;;
    *)
      if command -v python3 >/dev/null 2>&1; then
        vps_agent_i18n_python message "$key" "$en" "$pt" "$@"
      else
        printf '%s' "$en"
      fi
      ;;
  esac
}

vps_agent_language_label() {
  case "${VPS_AGENT_LANG:-en}" in
    en) printf 'English' ;;
    pt-BR) printf 'Português (Brasil)' ;;
    es) printf 'Español' ;;
    de) printf 'Deutsch' ;;
    fr) printf 'Français' ;;
    ja) printf '日本語' ;;
    id) printf 'Bahasa Indonesia' ;;
    *) printf 'English' ;;
  esac
}

vps_agent_choose_language() {
  [ -t 0 ] || return 0
  local answer=""
  cat <<'EOF'
1) English              (en)
2) Português (Brasil)   (pt-BR)
3) Español              (es)
4) Deutsch              (de)
5) Français             (fr)
6) 日本語                (ja)
7) Bahasa Indonesia     (id)
EOF
  printf 'Language / Idioma / Sprache / Langue / 言語 / Bahasa [1]: '
  read -r answer
  case "${answer:-1}" in
    1|en|EN|English|english) VPS_AGENT_LANG="en" ;;
    2|pt|PT|pt-BR|pt-br) VPS_AGENT_LANG="pt-BR" ;;
    3|es|ES|Español|espanol) VPS_AGENT_LANG="es" ;;
    4|de|DE|Deutsch|deutsch) VPS_AGENT_LANG="de" ;;
    5|fr|FR|Français|francais) VPS_AGENT_LANG="fr" ;;
    6|ja|JA|日本語) VPS_AGENT_LANG="ja" ;;
    7|id|ID|Indonesia|indonesia) VPS_AGENT_LANG="id" ;;
    *) printf 'Invalid language selection.\n' >&2; return 2 ;;
  esac
  export VPS_AGENT_LANG
}

vps_agent_confirm_language() {
  [ -t 0 ] || return 0
  [ "${VPS_AGENT_LANG_EXPLICIT:-0}" = "1" ] && return 0

  local answer=""
  printf '%s: %s\n' "$(vps_agent_text 'Detected language' 'Idioma detectado')" "$(vps_agent_language_label)"
  printf '%s' "$(vps_agent_text '[Enter] continue | [L] choose another language: ' '[Enter] continuar | [L] escolher outro idioma: ')"
  read -r answer
  case "$answer" in
    l|L) vps_agent_choose_language ;;
  esac
  printf '\n'
}

vps_agent_tagline() {
  vps_agent_text "Secure MCP Gateway for Linux Hosts" "Gateway MCP seguro para hosts Linux"
}

vps_agent_author_line() {
  vps_agent_text "Made by $VPS_AGENT_AUTHOR_NAME | $VPS_AGENT_AUTHOR_GITHUB" "Feito por $VPS_AGENT_AUTHOR_NAME | $VPS_AGENT_AUTHOR_GITHUB"
}

vps_agent_banner() {
  local version tagline author
  version="$(vps_agent_version)"
  tagline="$(vps_agent_tagline)"
  author="$(vps_agent_author_line)"

  printf '\n+------------------------------------------------------------+\n'
  printf '| %-58s |\n' "$VPS_AGENT_PRODUCT_NAME"
  printf '| %-58s |\n' "$tagline"
  printf '| %-58s |\n' "$author"
  printf '| %-58s |\n' "$(vps_agent_text "Version: $version" "Versão: $version")"
  printf '+------------------------------------------------------------+\n\n'
}

vps_agent_step() {
  printf '\n==> %s\n' "$*"
}

vps_agent_note() {
  printf '    %s\n' "$*"
}
