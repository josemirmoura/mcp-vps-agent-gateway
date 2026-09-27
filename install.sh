#!/usr/bin/env bash
set -euo pipefail

REPO="${VPS_AGENT_REPO:-josemirmoura/mcp-vps-agent-gateway}"
VERSION="${VPS_AGENT_VERSION:-latest}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

if [ "$(id -u)" -ne 0 ]; then
  echo "Run as root, for example: curl ... | sudo bash" >&2
  exit 1
fi

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

if [ -n "${VPS_AGENT_LOCAL_PACKAGE:-}" ]; then
  PKG="$VPS_AGENT_LOCAL_PACKAGE"
else
  if [ "$VERSION" = "latest" ]; then
    URL="https://github.com/$REPO/releases/latest/download/vps-agent-linux-$ARCH.tar.gz"
  else
    URL="https://github.com/$REPO/releases/download/$VERSION/vps-agent-linux-$ARCH.tar.gz"
  fi
  PKG="$TMP/package.tar.gz"
  echo "Downloading $URL"
  curl -fL --retry 3 --proto '=https' --tlsv1.2 "$URL" -o "$PKG"
fi

tar -xzf "$PKG" -C "$TMP"
chmod +x "$TMP"/vps-agent*
exec "$TMP/vps-agent-setup" install --source-dir "$TMP" "$@"
