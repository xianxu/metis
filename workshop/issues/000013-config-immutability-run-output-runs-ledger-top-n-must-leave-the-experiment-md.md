---
id: 000013
status: codecomplete
deps: []
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 0.9
started: 2026-07-06T15:07:51-07:00
actual_hours: 0.41
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
  **outside** the config via **on-demand `metis ledger show`** (already exists) over the `.ledger.csv`
  sidecar — **no new generated sidecar** (decision settled in the plan; keeps #13 a pure removal).
  Single-run *aggregated* history (the dropped `## Runs` bullets) defers to the metis#8
  "experiment = 1-config ledger" unification; per-single-run provenance stays in `record.json`.
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

Durable plan: `workshop/plans/000013-config-immutability-plan.md`. Single-pass atomic.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.15  impl=0.35
item: milestone-review       design=0.0   impl=0.2
item: atlas-docs             design=0.05  impl=0.1
design-buffer: 0.15
total: 0.9
```

Σdesign 0.20 × 1.15 = 0.23; Σimpl 0.65 × 1.00 = 0.65; total **0.9** (= `estimate_hours`). `smaller-go-module` = remove the two `.md` writes (appendRunLog + regenLedgerSummary's body regen) + reconcile tests; `milestone-review` = close boundary; `atlas-docs` = the config-immutability boundary.

## Plan

- [x] RED: a test asserting `krun` leaves the experiment `.md` unchanged (fails today — it's appended to).
- [x] Remove `appendRunLog`'s `.md` write + the in-body ledger-summary regen from the run/sweep path.
- [x] Human sweep view = on-demand `metis ledger show` (no new sidecar); single-run history deferred (metis#8 unification).
- [x] Update any `## Runs`/body-summary-asserting metis tests; the kbench e2e snapshot/restore drop is a kbench follow-up.
- [x] atlas: config-immutability boundary.

## Log

### 2026-07-06
- 2026-07-06: closed — Close-review round-1 (FIX-THEN-SHIP, 0 Critical / 2 Important) — all fixed. Important: (1) reconciled the base-layer datatype CONTRACT (construct/datatype/experiment.md Run-history convention + authoring rules + experiment-shape.md) to config-immutability — was still telling authors the runner appends ## Runs / the shape body carries a top-N; (2) finished the atlas (pkg/ledger bullet) + deleted the now-dead recordSummary/formatKnobs/formatMetrics + their test (plan explicitly said delete-if-unused; YAGNI, reintroduce with metis#8 unification). Minors: all stale ## Runs-append comments (run.go x3 + sweep.go + run_test/e2e_test/record_e2e_test), run-fail.md fixture prose (kept the ## Runs heading as a deliberate adversarial no-append case), lessons.md; renamed TestLedger_SweepWritesSidecarAndSummary→...NotBody. go build+vet+test ./... green.; review verdict: SHIP
- 2026-07-06: closed — Config immutability done: a metis run no longer mutates the experiment .md. Removed appendRunLog (single-run ## Runs write) + regenLedgerSummary body top-N regen (sweep); kept the useful objective-metric-missing warning (warnIfObjectiveMissing). Run output stays in runs/<id>/{run,record}.json + the .ledger.csv sidecar; human top-N view is on-demand `metis ledger show`. Inverted 5 tests that asserted the old .md mutation into config-immutability guards (byte-identical where the run-fail fixture carries its own ## Runs heading). go build+vet+test ./... green. Atlas reconciled (4 stale ## Runs-mutation refs across experiment.md + index.md). Single-run aggregated history deferred to the metis#8 1-config-ledger unification (per-run provenance stays in record.json). Prereq for #14 (capture the run-spec) now satisfied.; review verdict: FIX-THEN-SHIP
- Filed from the reproducible-dirty-run design pass (pensive). Prerequisite for #14 (capture the
  run-spec). The config `.md` becomes immutable input; run output stays in record.json + the ledger
  sidecar; the browse view moves to `metis ledger show` / a generated sidecar.
