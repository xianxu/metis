---
id: 000013
status: working
deps: []
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours:
started: 2026-07-06T15:07:51-07:00
---

# Config immutability — run output (## Runs / ledger top-N) must leave the experiment .md

Design source: `workshop/pensive/2026-07-06-reproducible-dirty-run-capture.md` (item 4; the
**prerequisite** for #14 — you can't cleanly snapshot a run-spec the run itself rewrites).

## Problem

`metis run` **mutates the experiment `.md` with run output** on every run:
- single run → `appendRunLog(o.expPath, rec)` (`run.go:184`) appends a `- <knobs → score>` line to a
  `## Runs` section (creating it if absent), rewriting the file (`run.go:220`).
- sweep → the ledger top-N summary is regenerated into the body between
  `<!-- metis:ledger:begin -->` / `end` markers (metis#8 `regenLedgerSummary`).

So the config file's content changes even when the *input* didn't → its content-hash churns, and a
committed config is **not** a stable identity. Concretely: this is what forced the repeated
`## Runs`-stripping of `titanic-sweep.md` all through the metis-v1 build, and it makes the
reproducibility model (git rev / blob of the committed config = its identity) unsound. The config
`.md` must be **immutable input**; run output belongs in the record/ledger, not the spec.

## Spec

- **A run leaves the experiment `.md` byte-for-byte unchanged.** Remove the `appendRunLog` write and
  the in-`.md` ledger-summary regeneration from the run path.
- Run output already has durable homes — keep them: `runs/<id>/record.json` (per-run provenance) +
  the `.ledger.csv` sidecar (sweeps). Nothing is lost by not touching the `.md`.
- **The human "recent runs / top-N" browse view** (which the body summary provided) is preserved
  **outside** the config — decide at plan time between: (a) on-demand only via `metis ledger show`
  (already exists); (b) a **generated summary sidecar** (e.g. `<shape>.runs.md`, gitignored or
  clearly-marked generated). Leaning (a)+(b): the ledger sidecar is the record, a generated summary
  file for eyeballing, the committed `.md` stays pure input.
- The `<!-- metis:ledger:begin/end -->` markers + `## Runs` heading are no longer written into the
  experiment body.

## Done when

- `krun <experiment.md>` and `krun <shape.md>` (single + sweep) leave the experiment file
  **byte-identical** — a test runs a thread and asserts the `.md` is unmodified (no snapshot/restore
  needed anymore — the e2e's `clean_workspace` snapshot/restore of the experiment file can be dropped).
- Run output is fully in `record.json` + the ledger sidecar; the human summary view is available via
  `metis ledger show` and/or a generated (non-config) sidecar.
- Existing tests that asserted the `## Runs` append / body-summary regen are updated to the new home.
- atlas: the run-output-vs-config-input boundary.

## Plan

- [ ] RED: a test asserting `krun` leaves the experiment `.md` unchanged (fails today — it's appended to).
- [ ] Remove `appendRunLog`'s `.md` write + the in-body ledger-summary regen from the run/sweep path.
- [ ] Re-home the human summary (decision: `metis ledger show` + a generated sidecar).
- [ ] Update the e2e (drop the experiment-file snapshot/restore) + any `## Runs`-asserting tests.
- [ ] atlas: config-immutability boundary.

## Log

### 2026-07-06
- Filed from the reproducible-dirty-run design pass (pensive). Prerequisite for #14 (capture the
  run-spec). The config `.md` becomes immutable input; run output stays in record.json + the ledger
  sidecar; the browse view moves to `metis ledger show` / a generated sidecar.
