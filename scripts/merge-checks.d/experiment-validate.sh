#!/usr/bin/env bash
# experiment-validate — fail the merge if any CHANGED experiment file violates the
# CUE schema. The enforcement seam for the experiment datatype (metis#1 M1,
# ARCH-PURPOSE): the schema is enforced at the merge gate, not just documented.
#
# CONTRACT: invoked by scripts/run-merge-checks.sh as `experiment-validate.sh <base>
# <head>` (positional, a merge-base..head range). We honor those args (falling back
# to env/defaults only when unset), and compute the changed-file list into a VARIABLE
# so an unresolvable base ABORTS under `set -e` — not silently passes (the failure of
# a `git diff` inside a `< <(…)` process substitution is swallowed by set -e; a plain
# assignment is not).
#
# Scans changed *.md whose frontmatter declares `type: experiment` and runs
# `vocabulary validate-instance --type experiment` on each. Structural (shape)
# validation only — the runner enforces semantic checks (DAG/needs/uses) at read time.
# The schema itself is regression-tested by experiment-schema-selftest.sh.
#
# testdata/ is SKIPPED (intentionally-malformed unit-test fixtures). Set
# EXPERIMENT_VALIDATE_INCLUDE_TESTDATA=1 to scan them too.
set -euo pipefail

VOCAB="${VOCAB:-vocabulary}"           # on PATH during `make weave`; else ../ariadne/bin/vocabulary
base="${1:-${MERGE_CHECK_BASE:-origin/main}}"
headref="${2:-HEAD}"

# Assign first (NOT `< <(git diff …)`) so a bad base fails the script instead of
# being swallowed. run-merge-checks passes merge-base..head, so two-dot is exact.
files="$(git diff --name-only "$base" "$headref" -- '*.md')"

fail=0
while IFS= read -r f; do
  [ -n "$f" ] || continue
  [ -f "$f" ] || continue
  case "$f" in
    testdata/*) [ "${EXPERIMENT_VALIDATE_INCLUDE_TESTDATA:-0}" = 1 ] || continue ;;
  esac
  # Probe the FRONTMATTER BLOCK (lines between the first two `---` fences), so a
  # `type: experiment` below reordered id/seed fields is still detected.
  awk '/^---[[:space:]]*$/{n++; next} n==1' "$f" \
    | grep -qE '^type:[[:space:]]*experiment[[:space:]]*$' || continue
  if ! "$VOCAB" validate-instance --type experiment "$f"; then
    fail=1
  fi
done <<< "$files"

exit "$fail"
