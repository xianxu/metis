#!/usr/bin/env bash
# experiment-validate — fail the merge if any changed experiment file violates the
# CUE schema. The enforcement seam for the experiment datatype (metis#1 M1,
# ARCH-PURPOSE): the schema is enforced at the merge gate, not just documented.
#
# Scans changed *.md whose frontmatter declares `type: experiment` and runs
# `vocabulary validate-instance --type experiment` on each. Structural (shape)
# validation only — the runner enforces the semantic checks (DAG/needs/uses).
#
# testdata/ is SKIPPED: those are the validator's OWN unit-test fixtures, some
# intentionally malformed to prove rejection (see the M1 validator test). Set
# EXPERIMENT_VALIDATE_INCLUDE_TESTDATA=1 to scan them too — used by this repo's
# own test to prove the check actually rejects a bad experiment.
set -euo pipefail

VOCAB="${VOCAB:-vocabulary}"          # on PATH during `make weave`; else ../ariadne/bin/vocabulary
base="${MERGE_CHECK_BASE:-origin/main}"

fail=0
while IFS= read -r f; do
  [ -f "$f" ] || continue
  case "$f" in
    testdata/*) [ "${EXPERIMENT_VALIDATE_INCLUDE_TESTDATA:-0}" = 1 ] || continue ;;
  esac
  # cheap frontmatter probe: `type: experiment` in the leading fence
  head -8 "$f" | grep -qE '^type:[[:space:]]*experiment[[:space:]]*$' || continue
  if ! "$VOCAB" validate-instance --type experiment "$f"; then
    fail=1
  fi
done < <(git diff --name-only "$base"...HEAD -- '*.md')

exit "$fail"
