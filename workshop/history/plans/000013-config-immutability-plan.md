---
issue: 000013
title: Config immutability — run output leaves the experiment .md
status: active
created: 2026-07-06
---

# Config immutability Implementation Plan

> **For agentic workers:** AGENTS.md §3. TDD (superpowers-test-driven-development).

**Goal:** A `metis run` (single or sweep) leaves the experiment `.md` **byte-for-byte unchanged** — run output lives only in `runs/<id>/record.json` + the `.ledger.csv` sidecar, never in the config. This makes a committed config a stable content-hash (the reproducibility model) and is the prerequisite for #14 (you can't snapshot a spec the run rewrites).

**Architecture:** Remove the two writes that mutate the experiment file: `appendRunLog` (single-run `## Runs` bullet, `run.go:184`) and the in-body ledger-summary regen (`regenLedgerSummary`, called from `writeSweepLedger`). Keep the machine record (`record.json`) and the sweep ledger CSV (`writeSweepLedger` still writes `<stem>.ledger.csv`); the human top-N view is `metis ledger show` (on demand). Single-run aggregated history (the `## Runs` bullets) is **dropped** for now — `record.json` per run is the record; the elegant "experiment = 1-config ledger" unification (metis#8) is a separate follow-up, noted in the Log.

**Tech Stack:** Go, the metis run/ledger path.

## Core concepts

### Pure/IO entities touched

| Name | Lives in | Status |
|------|----------|--------|
| `appendRunLog` | `cmd/metis/run.go` | deleted |
| `runResolvedExperiment` (drops the appendRunLog call) | `cmd/metis/run.go` | modified |
| `regenLedgerSummary` | `cmd/metis/ledger.go` | deleted (or reduced to no-op-on-.md) |
| `writeSweepLedger` (keeps the CSV; drops the body regen) | `cmd/metis/ledger.go` | modified |

- **The experiment `.md` is immutable through a run.** After the change, no code path writes to `o.expPath` during a run. `record.json` (per run) + `<stem>.ledger.csv` (sweep) hold everything.
- **DRY note:** removing two ad-hoc writers of the same file *toward one invariant* (config = input). The human summary single-sources to `metis ledger show` over the CSV, rather than a cached copy in the body that drifts.

## Tasks (TDD)

### Task 1: prove + stop the single-run mutation

- [ ] **1.1 RED** — a test: run a plain experiment (the toy pipeline) via the run path, assert the experiment `.md` is **byte-identical** before/after. Fails today (`appendRunLog` appends `## Runs`).
- [ ] **1.2 GREEN** — delete `appendRunLog` + its call at `run.go:184` (and `recordSummary` if now unused). Run → PASS. Commit.

### Task 2: stop the sweep body-summary mutation (keep the CSV)

- [ ] **2.1 RED** — a test: run a multi-point shape (sweep), assert the shape `.md` is byte-identical before/after AND the `<stem>.ledger.csv` sidecar still has the rows. Fails today (`regenLedgerSummary` rewrites the body between markers).
- [ ] **2.2 GREEN** — in `writeSweepLedger`, stop calling `regenLedgerSummary`'s `.md` write (delete `regenLedgerSummary`, or reduce it to nothing that touches the file); keep the CSV write + the objective-not-found warning. Run → PASS. Commit.

### Task 3: reconcile existing tests + docs

- [ ] **3.1** — update/remove metis tests that asserted the `## Runs` append or the body top-N block (grep `## Runs`, `ledger:begin`, `appendRunLog`, `regenLedgerSummary` in `tests/`/`*_test.go`). The sweep sidecar-write test stays (asserts the CSV, not the body).
- [ ] **3.2** — `metis ledger show` is the human sweep view (verify it still renders after the body regen is gone).
- [ ] **3.3** — atlas: the **config-input vs run-output boundary** — the `.md` is immutable input; run output is `record.json` + `.ledger.csv`; the browse view is `metis ledger show`.
- [ ] **3.4** — `go build/vet/test ./...` green. Commit.

### Task 4: close

- [ ] **4.1** `sdlc close --issue 13`. Note in the Log: single-run aggregated history (the dropped `## Runs`) is deferred to the metis#8 "experiment = 1-config ledger" unification; the kbench e2e's experiment-file snapshot/restore (`clean_workspace`) becomes unnecessary and can be dropped in a kbench follow-up.

## Done when (issue) — mapped

- [ ] `krun`/`metis run` (single + sweep) leaves the `.md` byte-identical — Tasks 1,2
- [ ] run output in `record.json` + `.ledger.csv`; sweep view via `metis ledger show` — Task 2,3
- [ ] `## Runs`/body-summary-asserting tests updated — Task 3
- [ ] atlas: config-immutability boundary — Task 3

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

Reconciliation: Σdesign 0.20 × 1.15 = 0.23; Σimpl 0.65 × 1.00 = 0.65; total ≈ 0.9. `smaller-go-module` = remove the two `.md` writes + reconcile the tests; `milestone-review` = the close boundary; `atlas-docs` = the config-immutability boundary note. Single-pass atomic.
