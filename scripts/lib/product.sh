#!/usr/bin/env bash

# Canonical product identity for human-facing terminal scripts.
# Do not source this file from machine-readable APIs or JSON-producing commands.

VPS_AGENT_PRODUCT_NAME="Portico MCP"
VPS_AGENT_PRODUCT_SLUG="portico-mcp"
VPS_AGENT_AUTHOR_NAME="Josemir Moura"
VPS_AGENT_AUTHOR_GITHUB="github.com/josemirmoura"

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
  case "$raw" in
    pt|pt_BR|pt-BR|pt_*) printf 'pt-BR' ;;
    en|en_US|en-US|en_GB|en-GB|en_*) printf 'en' ;;
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

vps_agent_text() {
  local en="$1"
  local pt="$2"
  if vps_agent_is_pt_br; then
    printf '%s' "$pt"
  else
    printf '%s' "$en"
  fi
}

vps_agent_language_label() {
  if vps_agent_is_pt_br; then
    printf 'Português (Brasil)'
  else
    printf 'English'
  fi
}

vps_agent_confirm_language() {
  [ -t 0 ] || return 0
  [ "${VPS_AGENT_LANG_EXPLICIT:-0}" = "1" ] && return 0

  local answer=""
  if vps_agent_is_pt_br; then
    printf 'Idioma detectado: Português (Brasil)\n'
    printf '[Enter] continuar | [E] English: '
    read -r answer
    case "$answer" in
      e|E|en|EN|english|English) VPS_AGENT_LANG="en"; export VPS_AGENT_LANG ;;
    esac
  else
    printf 'Detected language: English\n'
    printf '[Enter] continue | [P] Português (Brasil): '
    read -r answer
    case "$answer" in
      p|P|pt|PT|pt-br|pt-BR) VPS_AGENT_LANG="pt-BR"; export VPS_AGENT_LANG ;;
    esac
  fi
  printf '\n'
}

vps_agent_tagline() {
  vps_agent_text     "Secure MCP Gateway for Linux Hosts"     "Gateway MCP seguro para hosts Linux"
}

vps_agent_author_line() {
  vps_agent_text     "Made by $VPS_AGENT_AUTHOR_NAME | $VPS_AGENT_AUTHOR_GITHUB"     "Feito por $VPS_AGENT_AUTHOR_NAME | $VPS_AGENT_AUTHOR_GITHUB"
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
  printf '| %-58s |\n' "Version: $version"
  printf '+------------------------------------------------------------+\n\n'
}

vps_agent_step() {
  printf '\n==> %s\n' "$*"
}

vps_agent_note() {
  printf '    %s\n' "$*"
}
