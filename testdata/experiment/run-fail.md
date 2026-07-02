---
type: experiment
id: run-fail
seed: 7
status: active
steps:
  - id: boom
    uses: test/echo
    with: {fail: true}
---
# run-fail — M2 failed-run ledger fixture

One `test/echo` step told via `with.fail` to exit non-zero. Structurally and
semantically valid (it runs), so it reaches execution and fails there —
exercising the branch where `metis run` still writes `runs/<id>/run.json` with
status `failed` and appends a `## Runs` line, then returns an error.

## Runs
