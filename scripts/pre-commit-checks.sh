#!/usr/bin/env bash
#
# Pre-commit checks for Authula.
#
# Runs the same checks as CI before a commit is allowed:
#   format -> vet -> lint -> build -> test
#
# Called by .githooks/pre-commit, which is installed for every developer via
# `make setup` (or `make hooks`). Can also be run directly:
#   ./scripts/pre-commit-checks.sh
#
# To skip the hook on purpose (e.g. a WIP commit on a private branch):
#   git commit --no-verify
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Only run Go checks when Go-related files are part of the commit.
staged_go_files=$(git diff --cached --name-only --diff-filter=ACMR -- '*.go' go.mod go.sum)
if [[ -z "$staged_go_files" ]]; then
  echo "pre-commit: no Go files staged, skipping checks."
  exit 0
fi

run_step() {
  local name=$1
  shift
  echo "pre-commit: running $name..."
  if ! "$@"; then
    echo ""
    echo "pre-commit: '$name' failed. Fix the issues above and try again."
    exit 1
  fi
}

# Formatting rewrites files in place. If it changed anything that is staged,
# stop so the developer can review and re-stage the formatted code.
run_step "make format" make format
if ! git diff --quiet -- $staged_go_files; then
  echo ""
  echo "pre-commit: formatting changed the following staged files:"
  git diff --name-only -- $staged_go_files | sed 's/^/  /'
  echo ""
  echo "Review the changes, run 'git add' on them, and commit again."
  exit 1
fi

run_step "make vet" make vet
run_step "make lint" make lint
run_step "make build" make build
run_step "make test" make test

echo "pre-commit: all checks passed."
