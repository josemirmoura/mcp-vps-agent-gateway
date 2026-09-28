#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh
printf '%s %s\n' "$VPS_AGENT_PRODUCT_NAME" "$(vps_agent_version)"
