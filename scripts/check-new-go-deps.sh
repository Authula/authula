#!/usr/bin/env bash
set -euo pipefail

# Get direct dependencies from go.mod
mapfile -t direct < <(go mod edit -json | jq -r '.Require[] | select(.Indirect != true) | .Path')

for mod in "${direct[@]}"; do
  # Query each module separately; ignore stderr noise
  go list -m -u -json "$mod" 2>/dev/null || true
done | jq -r 'select(.Update != null) | "\(.Path)\t\(.Version)\t\(.Update.Version)"'
