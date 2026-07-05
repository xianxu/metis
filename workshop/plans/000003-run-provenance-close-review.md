# Boundary Review — metis#3 (whole-issue close)

| field | value |
|-------|-------|
| issue | 3 — Run provenance: snapshot the resolved pipeline config (+ experiment git sha) so ## Runs is knob→score legible |
| repo | metis |
| issue file | workshop/issues/000003-run-provenance.md |
| boundary | whole-issue close |
| milestone | — |
| window | d0dd7c96d4c48fcb480d5ebe7f12b82ce8c57727..HEAD |
| command | sdlc close --issue 3 |
| reviewer | claude |
| timestamp | 2026-07-05T14:30:52-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything needed. Build, vet, and the full affected test suite pass; the CUE drift guard runs (0.01s, not skipped) and `cue` is on PATH; atlas is updated; no README exists (no README surface). Let me write up the review.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** metis#3 whole-issue close is a clean, well-tested boundary. The window (`d0dd7c9..HEAD`) is exactly the five `#3` commits — no unrelated work bleeds in — and delivers every implementation Done-when: the pure `pkg/record` core (structs + `PointAddress` + `OutputHash`, drift-guarded against `#RunRecord`), the `pkg/experiment` per-step retention change, and the M2 `cmd/metis` assembly (`gitProbe` seam, `assembleRecord`/`buildRecord`, `record.json`, knob→score `## Runs`). I verified `go build`/`go vet`/tests all green, the CUE conformance guard actually runs against the real `record.json` (not skipped), graceful git degradation and cross-run point-address stability are e2e-tested, and the M1-review Minor (non-finite config) was genuinely root-caused into a returned error. No Critical or Important findings; everything below is Minor or a forward-looking architectural note. Nothing blocks SHIP.

### 1. Strengths

- **The point-address is pinned by exactly the right adversarial tests.** `record_test.go` hammers determinism 25× (`TestPointAddress_DeterministicAcrossCalls`) — the map-iteration-order footgun is the real risk and it's directly targeted — plus per-determinant sensitivity and the non-finite-config error path. `address.go:24` leans on `encoding/json`'s documented sorted-map-key guarantee at every nesting level, the correct minimal choice, with the RFC-8785 upgrade path noted.
- **The drift guard is real enforcement (ARCH-PURPOSE / ARCH-DRY).** `pkg/record/conformance_test.go` marshals a fully-populated `RunRecord` (including the `#2`-owned `d`/`deps`/`upstream` slots) and `cue vet`s it against the closed `#RunRecord`; a renamed/removed/extra Go field fails the build. The e2e (`record_e2e_test.go:113`) additionally vets the *real* emitted `record.json`, not just a fixture — so the CUE source is enforced on the actual output shape, not restated documentation.
- **`pkg/record` stayed a clean leaf over `pkg/cas`** (does not import `pkg/experiment`) — the `StepRun`→`StepRecord` mapping lives at the `cmd/metis` assembly site (`buildRecord`), so the record package carries zero orchestration/IO coupling. This is the M1-review improvement, and the plan's `## Revisions` correctly records the divergence from its original "imports experiment" claim.
- **Graceful git degradation is a genuine behavior test, not a mock reasserting itself** (`TestRunExperiment_DegradesWithoutGitProvenance`): the run completes, warns, and still writes a record with no repo-SHAs and a valid point-address. The failed-step path also exercises record assembly implicitly (the `## Runs` line is now rendered from the record, so `TestRunExperiment_FailedStepStillWritesLedger` proves the failed-run record was built).
- **ARCH-PURE holds throughout:** `PointAddress`/`OutputHash`/`buildRecord` are pure and unit-tested with no IO; all git/filesystem reads sit behind `gitProbe` + `hashArtifacts` in the thin `cmd/metis` shell and are injected. `formatMetrics` was extracted from the deleted `runSummary` and reused by `recordSummary` (real ARCH-DRY win — verified no dangling `runSummary` references remain).

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

- **Point-address repo-name = local checkout basename** (`cmd/metis/record.go:41`, `filepath.Base(top)`). The point-address hashes `repo_shas` keyed by the directory basename of `git rev-parse --show-toplevel`. Identical commit+config+seed cloned into `metis/` vs `metis-2/` mint *different* point-addresses. Within v1 single-checkout/Kaggle use this never bites (the e2e proves within-environment stability), but this is the very identity key #8's global ledger derives from — see architectural note. Non-blocking; flag so #8 pins the repo name to a stable source (remote URL / configured identity), not the checkout dir name.
- **`repoRoot(t)` test helper now duplicated 3×** — `pkg/record/conformance_test.go:16`, `pkg/experiment/helpers_test.go:15`, `cmd/metis/helpers_test.go:13`, each a thin wrapper over `internal/repo.Root`. The M1 review flagged 2; M2 added the third path's usage. Test-only ARCH-DRY; hoist to an exported `repo.TestRoot(t)`.
- **Non-finite config half-succeeds** (`cmd/metis/run.go:88-97`). A `.inf`/`.nan` knob lets steps execute and `run.json` (status `ok`) get written, then `PointAddress` errors and `runExperiment` returns an error with no `record.json`/`## Runs`. Correctly no longer panics (the M2 fix), but the cleaner root cause is to reject non-finite config at parse/validate time, before any step runs — so the command doesn't leave an `ok` run.json behind a failed invocation. Edge input only.
- **`hashArtifacts` failure aborts a succeeded run** (`cmd/metis/record.go:72-76`). If a completed step reports an artifact path that doesn't exist under `runDir`, the whole run errors *after* `run.json` was written `ok`. Defensive and unlikely (artifacts come from the executor), same partial-success shape as above; note only.
- **`OutputHash` sorts by `Path` only** (`pkg/record/address.go:46`) — duplicate-path inputs have unspecified relative order under `sort.Slice` (not stable). Unreachable with well-formed unique-path artifact sets; a `(path, hash)` secondary key would make it total (mirrors the sibling CAS eviction tie-break). Carried from M1.
- **`formatKnobs` renders values with `%v`** (`cmd/metis/record.go:178`) — a float knob `c: 1.0` prints as `c=1`. Cosmetic, `## Runs`-only.

### 5. Test coverage notes

Coverage targets the exact bug classes this diff could ship: map-order non-determinism (25× loop), walk-order dependence (`OutputHash` order-independence), CUE↔Go drift (real-output `cue vet`), lost/misordered per-step data (`TestRunner_Run_ReturnsPerStepResults`), and the two behavioral contracts the issue promised — cross-run point-address stability and no-git graceful degradation — both e2e. Tests pin properties, not the implementation. Remaining gaps are the Minor edge cases above (duplicate-path ordering, artifact-missing abort), none reachable with well-formed inputs.

**Docs gate:** atlas updated in-range for the new surface — `atlas/index.md:14` adds the `pkg/record` entry and `atlas/experiment.md` documents the record package, `Runner.Run`'s new `[]StepRun` return, and the #3/#2/#7/#8 scope line. No `README.md` exists in the repo, so there is no README surface to update. Docs gate satisfied.

### 6. Architectural notes for upcoming work

- **ARCH-DRY / ARCH-PURE / ARCH-PURPOSE all pass.** DRY: `formatMetrics` reused, drift-guard single-sources the schema, only test-helper duplication remains (Minor). PURE: pure core unit-tested with zero IO, injected `gitProbe`. PURPOSE: the shadow-sweep over the `#RunRecord` "single source" finds only enforced consumers (Go structs via `cue vet`, real `record.json` via `cue vet`) — no hand-maintained restatement that isn't derived; all three implementation Done-when items are delivered, not deferred as the "real" point.
- **Point-address stability is the #8 seam.** The Minor above (basename-derived repo name) is where #8's global ledger key inherits a cross-environment instability. When #8 builds the global key on the point-address, pin repo identity to something environment-invariant. The `dirty` flag being *outside* the point-address is by-design (design: clean-vs-dirty is legibility; the read-set trace `D` is #2's precise encoding) — record it here so #8 doesn't mistake the coarse address for a byte-exact identity.
- **`CodeManifest{D, Deps}` + `Upstream` are correctly left empty** for #2 to populate; the drift guard already covers those optional slots, so a one-sided Go/CUE edit when #2 fills `D` will fail the build. Good forward pressure.

### 7. Plan revision recommendations

None. The plan's `## Revisions` (2026-07-05) already records the two divergences that mattered — `pkg/record` being a clean leaf over `cas` (not importing `experiment`), and the M2 NaN/Inf error-not-panic fix — and both match the shipped code. The plan and code are consistent.
