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

---

## Re-review — 2026-07-06T17:22:41-07:00 (SHIP)

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
| timestamp | 2026-07-06T17:22:41-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. I independently verified: build + vet clean, full `cmd/metis` and `pkg/record` test suites green (13.5s), the CUE closed disjunction has real teeth (rejects a bogus `capture_status`), and both round-1 Important findings are genuinely landed (the wiring test drives the real `runResolvedExperiment` seam; the conformance record now carries `CaptureStatus` and `cue vet` enforces it).

```verdict
verdict: SHIP
confidence: high
```

This is a re-review after round-1's FIX-THEN-SHIP. The diff completes and hardens metis#8's capture exactly per the Spec: a shared `captureRunCode` snapshots the code closure **and** the run-spec `.md` to git side-refs for both single runs (`refs/metis/runs/<id>`) and sweeps (`refs/metis/sweeps/<id>`), with a loud `CaptureStatus` (`captured|degraded|none`) in the record and on output. I re-verified both round-1 Important findings are now substantively fixed (not cosmetic): `TestRunExperiment_SingleRunCapturesViaWiring` exercises the `if !o.inSweep { captureSingleRun }` seam and fails if the call site is deleted; the CUE conformance record now sets `CaptureStatus` and I confirmed `cue vet` actually rejects an out-of-enum value. Build/vet/full test suites are green. No Critical or Important findings remain — only a handful of Minors (the same ones round-1 acknowledged, plus a couple of narrow ones), none blocking the close.

### 1. Strengths
- **Round-1 fixes genuinely landed, with teeth.** The wiring test is regression-proof against a dropped call site, and I independently confirmed the CUE disjunction rejects `capture_status: "bogus"` — so the conformance guard now vets the closed enum rather than `omitempty`-skipping it (conformance_test.go:63, experiment.cue:94).
- **Real ARCH-DRY consolidation.** `captureRunCode` is one shared IO seam for single + sweep; `closureFromRunDir` is reused by both `captureSingleRun` and `sweepClosure`; `sortedSets`/`resolveRoot` extracted (capture.go:92, 189, 212, 278). The old duplicated per-point union loop is gone.
- **ARCH-PURPOSE fulfilled — shadow-sweep clean.** Every consumer of the "capture" purpose derives from the shared function: single-run wiring (run.go:190), sweep suppression via `inSweep` (sweep.go:98-99), spec hook (`addSpecToClosure`, capture.go:232), loud status (`warnOnUncaptured` + record), CUE field, atlas. No hand-maintained restatement of the model left.
- **The two prior real bugs are root-caused, not patched, and lessoned.** `filepath.EvalSymlinks` before `Rel` (capture.go:243) and the existence-guard on the spec (capture.go:237) are correct, and both are distilled into `lessons.md`.
- **Honest degradation is durable.** `captured`/`degraded`/`none` distinguishes "no git" from "no closure", and `backfillCodeManifest` writes the gap into the record (capture.go:313), so a reproducibility failure survives as record state, not just a transient stderr line.

### 2. Critical findings
None.

### 3. Important findings
None. (Round-1's two Importants — untested wiring seam, blind CUE guard — are both fixed and verified above.)

### 4. Minor findings
- **`primaryRoot` is not symlink-resolved while the `commits` map keys are** (capture.go:128 vs 198/207). `commits` is keyed by `resolveRoot`'d repo paths; `primaryRoot = cacheProjectRoot(...)` is not. On macOS the `commits[primaryRoot]` lookup therefore *always misses* and relies on the sorted-first fallback (line 129-136). Single-repo runs (all current usage) stay correct via that fallback; a multi-repo+symlink dirty run would silently record a non-primary (still valid) repo's commit as the record's single `Commit`. Strongest candidate to fix now — one line: `commit = commits[resolveRoot(primaryRoot)]`.
- **Done-when "a dirty *sweep* captures the spec too" has no guard.** `TestCaptureSweepCode_BackfillsCodeManifest` never writes `sweep.md` to disk, so `addSpecToClosure` skips it and the test even asserts `len(D) == 1` (capture_e2e_test.go:89,117), actively excluding the spec. The mechanism is shared with the single-run path (which *is* spec-tested), so risk is low — but the specific item is unverified. Cheapest close: write `sweep.md` in that fixture and assert it lands in D.
- **The `inSweep` double-capture suppression is untested.** No test drives a sweep through `runSweep` in a git repo and checks that points don't each create `refs/metis/runs/<pointID>`. Deleting `pointOpts.inSweep = true` (sweep.go:99) wouldn't corrupt the final record (the sweep backfill overwrites), only do redundant per-point capture — so this is wasted-work risk, not correctness, hence Minor.
- **Comments/plan say "stderr" but capture notes go to `o.out` (= stdout).** `warnOnUncaptured` and run.go:192 write to `out`; the Spec's requirement is "in its output," which stdout satisfies — just reconcile the "stderr" wording in the comments/plan §3.
- **A mid-capture `git` failure is swallowed into a generic "degraded"** (capture.go:117-119): the underlying error text is dropped, and the warn message says only "no git work-tree, or a git failure." Best-effort is intentional per Spec, but folding the actual `err` into the warn would make a real git hiccup debuggable.
- **A spec outside git is silently dropped from the closure while status can still be `"captured"`** (capture.go:246-248): a spec that couldn't be pinned isn't reflected in the honesty status. Narrow (spec is normally co-located with the code repo); note only.

### 5. Test coverage notes
Heart tests are strong at the function level — real temp-git, real blob round-trip, loud-note assertion — and INTEGRATION via real git rather than mocks (ARCH-PURE-aligned). The single-run *seam* is now covered end-to-end (the round-1 gap). Remaining gaps are the two narrow items above (sweep-spec capture, sweep double-capture suppression); neither guards a correctness bug, only a Done-when restatement and a wasted-work regression.

### 6. Architectural notes for upcoming work
- ARCH-DRY: **pass** (one shared capture path, no copy-paste). ARCH-PURE: **pass** — the module is inherently git-glue and is tested with real git, not mocks; the status classification is simple enough inline (a pure `classifyStatus(total, captured, skipped)` would be a nicety, not a requirement). ARCH-PURPOSE: **pass** — every consumer derives from `captureRunCode`.
- Per-repo `Commit` remains single-valued (the follow-up flagged at capture.go:82-84). When that refinement lands, resolve the `primaryRoot` symlink inconsistency (Minor #1) at the same time so the per-repo commit-map keys and the primary lookup share one canonical (`resolveRoot`'d) form.

### 7. Plan revision recommendations
None required. The plan now matches the code: Task 4.1's claim that the conformance test "covers `capture_status`" is true as of this window (I verified `cue vet` enforces the disjunction). If the operator wants the record fully honest, optionally note in the plan Log that the *sweep*-spec and *inSweep-suppression* Done-when items are proven at the shared-function level rather than through a sweep-specific e2e (Minors #2/#3) — but that's a documentation nicety, not a blocking discrepancy.
