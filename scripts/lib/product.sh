#!/usr/bin/env bash

# Canonical product identity for human-facing terminal scripts.
# Do not source this file from machine-readable APIs or JSON-producing commands.

VPS_AGENT_PRODUCT_NAME="MCP VPS Agent Gateway"
VPS_AGENT_TAGLINE="Scoped, auditable MCP control for a Linux VPS"

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

vps_agent_banner() {
  printf '\n%s %s\n%s\n\n' "$VPS_AGENT_PRODUCT_NAME" "$(vps_agent_version)" "$VPS_AGENT_TAGLINE"
}

vps_agent_step() {
  printf '\n==> %s\n' "$*"
}

vps_agent_note() {
  printf '    %s\n' "$*"
}
