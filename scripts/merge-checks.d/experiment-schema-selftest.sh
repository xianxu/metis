#!/usr/bin/env bash
# experiment-schema-selftest — regression-test the experiment CUE schema on EVERY merge:
# assert the committed fixtures still behave (valid-baseline accepted, invalid-bad-status
# rejected). metis#1 M1 (review finding I1).
#
# Deliberately DIFF-INDEPENDENT (ignores the <base> <head> args run-merge-checks passes):
# it always checks the fixtures, so a schema change that widens/breaks #Experiment (e.g.
# adding a status to the enum) is caught even though experiment-validate.sh skips
# testdata/. This is the automated backstop behind M1's "enforcement" claim.
set -euo pipefail

VOCAB="${VOCAB:-vocabulary}"
cd "$(git rev-parse --show-toplevel)"
valid=testdata/experiment/valid-baseline.md
invalid=testdata/experiment/invalid-bad-status.md

rc=0
if ! "$VOCAB" validate-instance --type experiment "$valid" >/dev/null 2>&1; then
  echo "experiment-schema-selftest FAIL: $valid should PASS but was rejected"; rc=1
fi
if "$VOCAB" validate-instance --type experiment "$invalid" >/dev/null 2>&1; then
  echo "experiment-schema-selftest FAIL: $invalid should be REJECTED but passed"; rc=1
fi
[ "$rc" = 0 ] && echo "experiment-schema-selftest: ok (valid accepted, invalid rejected)"
exit "$rc"
