#!/usr/bin/env bash
# Report production statement coverage from a Go coverage profile.
#
# Excludes test-support packages, entrypoints, and generated code so the
# percentage reflects production code only. See docs/test-coverage.md.
set -euo pipefail

profile=${1:-coverage.out}
target=${COVERAGE_TARGET:-90}

if [[ ! -f "$profile" ]]; then
  echo "coverage profile not found: $profile" >&2
  exit 1
fi

filtered=$(mktemp)
trap 'rm -f "$filtered"' EXIT

# Keep the mode line and drop test-support, generated and entrypoint packages.
awk 'NR == 1 ||
     ($1 !~ /\/tests\// &&
      $1 !~ /\/internal\/tests\// &&
      $1 !~ /\/openapiexport\// &&
      $1 !~ /\/cmd\// &&
      $1 !~ /openapi_docs\.go:/)' "$profile" > "$filtered"

read -r covered total < <(
  awk 'NR > 1 { t += $2; if ($3 > 0) c += $2 } END { printf "%d %d\n", c + 0, t + 0 }' "$filtered"
)

if [[ "$total" -eq 0 ]]; then
  echo "no production statements found in $profile" >&2
  exit 1
fi

awk -v c="$covered" -v t="$total" -v target="$target" '
  BEGIN {
    needed = int(t * target / 100)
    if (needed * 100 < t * target) needed++
    gap = needed - c
    if (gap < 0) gap = 0
    printf "production coverage: %.1f%% (%d/%d statements)\n", c * 100 / t, c, t
    printf "statements needed for %d%%: %d (+%d)\n", target, needed, gap
  }'

echo
echo "Package coverage (ascending):"
echo

awk -v target="$target" '
  NR > 1 {
    split($1, a, ":")
    f = a[1]
    p = f
    sub(/\/[^\/]+$/, "", p)
    sub(/^github\.com\/Authula\/authula\/?/, "", p)
    if (p == "") p = "(root)"
    total[p] += $2
    if ($3 > 0) covered[p] += $2
  }
  END {
    for (p in total) {
      needed = int(total[p] * target / 100)
      if (needed * 100 < total[p] * target) needed++
      gap = needed - covered[p]
      if (gap < 0) gap = 0
      printf "%6.1f%%  %6d/%-6d  +%-5d  %s\n", covered[p] * 100 / total[p], covered[p], total[p], gap, p
    }
  }' "$filtered" | sort -n
