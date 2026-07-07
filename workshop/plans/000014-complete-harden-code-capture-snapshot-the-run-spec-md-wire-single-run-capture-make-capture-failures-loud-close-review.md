# Boundary Review — metis#14 (whole-issue close)

| field | value |
|-------|-------|
| issue | 14 — Complete + harden code capture — snapshot the run-spec .md, wire single-run capture, make capture failures loud |
| repo | metis |
| issue file | workshop/issues/000014-complete-harden-code-capture-snapshot-the-run-spec-md-wire-single-run-capture-make-capture-failures-loud.md |
| boundary | whole-issue close |
| milestone | — |
| window | 30b90e103de284217b195be908364a47c92298a4..HEAD |
| command | sdlc close --issue 14 |
| reviewer | claude |
| timestamp | 2026-07-06T17:09:58-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All green (`go build`/`vet`/`test ./...` pass, 9/9 packages). I have enough to render the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diff completes and hardens metis#8's capture exactly as the Spec asks: a shared `captureRunCode` now snapshots code **and** the run-spec `.md` to git side-refs for both single runs (`refs/metis/runs/<id>`) and sweeps (`refs/metis/sweeps/<id>`), with a `CaptureStatus` (`captured|degraded|none`) surfaced loudly in the record and on stdout. Build/vet/full test suite are green, the DRY refactor (shared `captureRunCode`/`closureFromRunDir`/`sortedSets`) is clean, and the two symlink/existence traps the fork hit are correctly guarded and captured in `lessons.md`. Nothing here is a correctness bug or a missing feature — the purpose is delivered. What keeps it from a clean SHIP is two cheap test-coverage gaps, one of which directly contradicts a claimed Plan item; both are non-blocking at the gate but worth closing first.

### 1. Strengths
- **Genuine ARCH-DRY consolidation.** `captureRunCode` is one shared IO seam for single + sweep; `closureFromRunDir` is reused by `sweepClosure`; `sortedSets`/`resolveRoot` extracted (capture.go:92, 189, 212). The old duplicated per-point union loop is gone.
- **ARCH-PURPOSE fulfilled, not the cheap subset.** Shadow-sweep of consumers: single-run wiring (run.go:190), sweep suppression via `inSweep` (sweep.go:98-99), spec hook (`addSpecToClosure`, capture.go:232), loud status (record + `warnOnUncaptured`), CUE field, atlas — every consumer of the "capture" purpose derives from the shared function.
- **The two real bugs are correctly rooted, not patched.** `filepath.EvalSymlinks` before `Rel` (capture.go:243) and the existence-guard on the spec (capture.go:237) are the right fixes, and both are distilled into `lessons.md`.
- **Honest degradation.** `captured`/`degraded`/`none` distinguishes "no git" from "no closure", and the record carries the gap (backfillCodeManifest:313) so reproducibility failure is durable, not just a transient stderr line.

### 2. Critical findings
None.

### 3. Important findings
- **The single-run capture *wiring* is untested — the seam the whole issue hinges on.** `TestCaptureSingleRun_*` call `captureSingleRun()` **directly** (capture_e2e_test.go:120, 188); nothing exercises the `if !o.inSweep { captureSingleRun(...) }` call in `runResolvedExperiment` (run.go:190-194) or the sweep's `inSweep=true` suppression (sweep.go:99). Deleting the run.go call site leaves the entire suite green, yet the Done-when is "a dirty single **`metis run`** records a manifest…". Fix: one e2e that drives a dirty single experiment through `runExperiment` in a real git repo (like `runSweepCapture` does for sweeps) and asserts the record got `CaptureStatus=captured` + a `refs/metis/runs/<id>` ref — this also pins that the sweep path does *not* double-capture per point.
- **CUE conformance guard is blind to `capture_status` — and the Plan claims otherwise.** `TestRunRecordConformsToCUE` builds a `CodeManifest` without `CaptureStatus` (conformance_test.go:58-63), so `omitempty` drops it and the closed-disjunction `"captured"|"degraded"|"none"` is never vetted. A future Go typo (e.g. `"skipped"`) would pass this guard but fail `cue vet` on a real record.json — exactly the drift the guard exists to catch. Plan Task 4.1 asserts "the CUE `#RunRecord`/`#CodeManifest` conformance test covers `capture_status`"; it does not. Fix: add `CaptureStatus: "captured"` to the test record (one line).

### 4. Minor findings
- `primaryRoot` is not symlink-resolved while the closure map keys are (`resolveRoot`, capture.go:198/128), so `commits[primaryRoot]` misses under symlinked paths (macOS `/var`→`/private/var`) and silently falls back to the sorted-first repo's commit for the record's single `Commit`. Single-repo (all current use) is unaffected by the fallback; only a multi-repo+symlink run records a non-primary (still valid) commit. Cheap fix: `commit = commits[resolveRoot(primaryRoot)]`.
- Done-when "a dirty **sweep** captures the spec too" has no guard: `TestCaptureSweepCode_BackfillsCodeManifest` never creates `sweep.md` on disk (capture_e2e_test.go:37), so `addSpecToClosure` skips it. The mechanism is shared with the single-run path (which *is* tested), so low risk — but the specific item is unverified.
- `warnOnUncaptured` / capture notes write to `o.out` (= `os.Stdout` in main.go:60), yet the code comments and plan §3 say "stderr". Behavior matches the Spec ("in its output"); just reconcile the comment or route to stderr.
- A spec outside git is silently dropped from the closure while status can still be `"captured"` (capture.go:246-248) — a spec that couldn't be pinned isn't reflected in the honesty status. Narrow (spec normally co-located with code); note only.

### 5. Test coverage notes
Heart tests are strong on the *functions* (real temp-git, real blob round-trip, loud-note assertion) — INTEGRATION via real git, not mocks (ARCH-PURE-aligned). The gap is the *integration seam*: capture is verified as a unit but not as a consequence of `metis run`. Adding the one `runExperiment`-level e2e (Finding 3.1) plus the conformance line (3.2) would make the diff self-guarding against the two most likely regressions (a dropped call site, an out-of-enum status).

### 6. Architectural notes for upcoming work
- ARCH-DRY: **pass** (shared capture path, no copy-paste). ARCH-PURE: **pass** — the module is inherently git-glue and is tested with real git, not mocks; the status classification is simple enough inline (extracting a pure `classifyStatus(total,captured,skipped)` would be a nicety, not a requirement). ARCH-PURPOSE: **pass** — every consumer derives from `captureRunCode`; no hand-maintained restatement left.
- Per-repo `Commit` remains single-valued (comment at capture.go:83-84 flags it as the follow-up). When that refinement lands, resolve the `primaryRoot` symlink inconsistency (Minor above) at the same time so the per-repo commit map keys and the primary lookup share one canonical form.

### 7. Plan revision recommendations
- Add a `## Revisions` entry to `workshop/plans/000014-complete-harden-capture-plan.md`: Task 4.1's claim that the conformance test "covers `capture_status`" is not yet true — either land the one-line coverage (Finding 3.2) or correct the claim. Optionally note that the single-run and spec-in-sweep Done-when items are proven at the function level (`captureSingleRun`/`captureSweepCode`) rather than through `metis run`, pending the integration e2e (Finding 3.1).
