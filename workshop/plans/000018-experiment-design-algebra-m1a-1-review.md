# Boundary Review — metis#18 (milestone M1a-1)

| field | value |
|-------|-------|
| issue | 18 — experiment-design algebra M1a — three-phase shape + Sampler fold node (static samplers, per-fold pipeline, driver:single) |
| repo | metis |
| issue file | workshop/issues/000018-experiment-design-algebra.md |
| boundary | milestone M1a-1 |
| milestone | M1a-1 |
| window | f4d961d57ffc9e1f27edee2ad926e2a3e104d33d^..HEAD |
| command | sdlc milestone-close --issue 18 --milestone M1a-1 |
| reviewer | claude |
| timestamp | 2026-07-07T17:23:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M1a-1 schema boundary is correct and well-tested: the Go `Shape`/`Sweeper`/`Driver` structs, the strict `KnownFields(true)` parse, the combined-DAG `ValidateShape`, and the closed CUE `#ExperimentShape` rewrite all do what the plan's Tasks 1–4 promise, and the drift-guard + closedness CUE tests actually execute (cue on PATH — verified PASS, not skip). The deliberate `cmd/metis` RED is real, confined (`go build ./...` names only `cmd/metis`; `go build ./pkg/...` clean), and honestly documented in the plan + lessons.md — acceptable per the boundary-decomposition rule. Nothing here is a correctness bug, so nothing blocks the gate. What keeps it from a clean SHIP is the **Docs gate**: the schema change is breaking, but its two doc consumers (`construct/datatype/experiment-shape.md`, the authoring reference; `atlas/index.md`) still describe the deleted v1 vocabulary, and neither is guarded by any test — a hidden-consumer gap the plan didn't account for. Plus a cheap ARCH-DRY inconsistency (Go duplicates the header the CUE just single-sourced) and an unenforced phase-ordering invariant.

### 1. Strengths
- **CUE single-sourcing done right** (`construct/vocabulary/experiment.cue:37–86`): `_meta` + `_phase` factored and embedded into both `#Experiment` (via `_pipeline`) and `#ExperimentShape`; closedness preserved *and* asserted by `TestCUE_ClosednessPreservedBySingleSource` (shape_test.go:167) — a real regression guard, not a restatement.
- **Strict parse closes the yaml silent-drop footgun** (`shape.go:94–98`) with correct `io.EOF` handling that ignores only the empty-stream case (not real errors), and is tested for both top-level *and* nested (`sweeper`) unknown keys (shape_test.go:81–91).
- **Combined-DAG validation** (`shape.go:111–163`) correctly concatenates phases into one synthetic `Experiment` so cross-phase `needs` resolve — a subtlety the plan review flagged — and it's tested for the resolve path, duplicate-id-across-phases, and dangling need (shape_test.go:96–128).
- **`driver:cv` parsed-but-rejected** (`shape.go:77–80,149–151`) makes metis#23 a purely additive change with no schema churn — good forward-compat.
- The structural/semantic split is coherent and verified: I confirmed CUE allows empty-pipeline / driver-neither / driver-both, and the Go `ValidateShape` semantic checks are the sole enforcers — exactly as the vocabulary file's own doctrine states, and all are covered by the T3 mutator table.

### 2. Critical findings
None.

### 3. Important findings
- **Docs gate — authoring reference is fully stale (`construct/datatype/experiment-shape.md`).** The whole doc (lines 20–24, 38–50, 66–67) still documents v1: `steps` + `sweep` block + `range_steps` + `_pipeline` single-source. An author following it produces a shape that **both** `ParseShape` (KnownFields) and `cue vet` now reject. This is a hidden consumer of the schema change, unmentioned anywhere in the plan (only Task 21/M1a-5 touches atlas). *Fix:* rewrite the frontmatter table + `sweep` section to `data│pipeline│ship` + `sweeper.resample.cv` + `driver`, or, if deferring, add a prominent "⚠ v1 — being rewritten for metis#18 v2" banner this boundary.
- **Docs gate — `atlas/index.md:69–71` describes deleted/renamed entities as current:** `experiment.Shape`/`Sweep` parse "the `sweep:` block"; `#ExperimentShape … single-sourced via the shared `_pipeline`". Post-M1a-1 there is no `Sweep` type, no `sweep:` block, and `_pipeline` no longer single-sources both (`_meta`/`_phase` do). Plan defers full atlas to M1a-5, but this text is now *actively wrong*, not merely incomplete. *Fix:* correct these three lines (or mark stale) now; the full three-phase write-up can still land at M1a-5.
- **ARCH-DRY — Go header duplicated across `Experiment` and `Shape`.** `Shape` (`shape.go:22–31`) restates the five header fields (`Type/ID/Competition/Seed/Status`) that `Experiment` (experiment.go:29–35) already declares, and `ValidateShape` re-maps them field-by-field in `combined := Experiment{...}` (`shape.go:112–119`). The CUE side just single-sourced this exact header via `_meta`; the Go side doesn't mirror it → adding one header field is a 4-place edit. *Fix:* extract a shared `header` struct embedded `yaml:",inline"` into both (the v1 code already relied on inline embedding, so KnownFields stays happy); `combined` becomes `Experiment{header: sh.header, Steps: …}` with `combined.Type = "experiment"`.
- **ARCH-PURPOSE / defense-in-depth — phase ordering is unenforced.** `ValidateShape` checks acyclicity + needs-resolution over the combined DAG but never enforces that edges run monotonically (a `data` step must not `needs` a `pipeline`/`ship` step; `pipeline` must not `needs` `ship`). The `data│pipeline` cut is *the* structural leakage-safety guarantee of the whole design; a backward cross-phase edge validates clean today and would silently break the run-once/per-fold invariant when M1a-4 wires execution. *Fix:* add an edge-direction check keyed by phase membership in `ValidateShape` (sharp diagnostic at authoring), or explicitly record in the plan that ordering is enforced by execution wiring at M1a-4, not the validator.

### 4. Minor findings
- Go `ValidateShape` is looser than CUE on `objective`: empty `direction` passes (`shape.go:133` guards with `d != ""`) and `metric` is never checked, while CUE requires both — a semantic validator being looser than the structural one is a mild smell (both gates run, so harmless in practice).
- The `len(sh.Pipeline)==0` guard (`shape.go:124`) is not isolated by a test: the T3 "empty pipeline" mutator nils `Pipeline` on the full fixture, so `predict needs [train]` goes dangling and `Validate` fails *first* — removing the guard wouldn't fail that test.
- All of `ValidateShape`'s semantic checks are currently reachable only from tests (no runnable command calls it until `run.go` is rewired in M1a-4). Expected at this boundary; just be sure M1a-4 keeps `ValidateShape` on every run path.

### 5. Test coverage notes
- The kind of bug this diff could ship (silent yaml drop diverging from CUE; per-phase validation wrongly rejecting cross-phase needs; closedness lost on a CUE re-edit) is covered by T1/T2/T3 + the two CUE tests, which I verified execute rather than skip.
- Gaps: no test isolates the empty-pipeline guard (Minor 2); no test asserts phase ordering (because it's unenforced — Important 4). Add one shape with an empty `pipeline` + no cross-phase need to pin the guard, and one with a backward cross-phase edge once ordering is enforced.

### 6. Architectural notes for upcoming work
- M1a-4 consumes the new `Shape` phase surface heavily — land the header-DRY fix (Important 3) now, before consumers multiply.
- I verified the CUE phase fields are optional-with-empty-default (a shape omitting `data` passes `cue vet`); the non-empty-pipeline and driver-exactly-one invariants live **only** in Go `ValidateShape`. M1a-4 must therefore guarantee `ValidateShape` is invoked on every execution path, since the merge-check (`cue vet`) will not catch a driver-neither/both or empty-pipeline shape.

### 7. Plan revision recommendations
- Add to the plan a `## Revisions` entry (or a Task 5-adjacent step in Chunk 1): **"Update `construct/datatype/experiment-shape.md` and the stale `atlas/index.md:69–71` lines in the M1a-1 boundary."** The datatype doc is a hidden consumer of the schema change and is currently unmentioned in the plan — the exact "a drift-guard/reference is a hidden consumer of a schema change" lesson the diff itself just recorded in `workshop/lessons.md`, applied to a doc instead of a fixture.
- Add a `## Revisions` / `## Notes` line recording the **phase-ordering decision**: either "enforce monotonic phase edges in `ValidateShape`" (preferred — cheap, sharp diagnostic) or an explicit statement that phase ordering is enforced by M1a-4 execution wiring, not the validator, so the invariant isn't silently assumed.
