# Partial Inner CV (inner_k split) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Steps use checkbox (`- [ ]`) syntax.

**Goal:** Let a shape declare the inner resample's fold count separately from the outer driver's (`sweeper.resample.cv.inner_k`), so the decision grid's inner sweep — where nearly all compute goes — has a cost knob (metis#45 lever (a); lever (b), the racing sampler, is filed separately as the demand-driven follow-up).

**Architecture:** `k` KEEPS its existing meaning — the outer/estimand fold count AND the inner default — so every existing shape runs byte-identically; `inner_k` (optional, ≥2) overrides the inner only. One pure accessor (`CVResample.InnerFolds()`) is the single source both consumers derive from (ARCH-DRY); the semantic rule (metis#42's principle): **outer k = the ESTIMAND knob (train fraction each outer fold simulates); inner k = selection-precision/cost.** `--sample m`/`--fast` stay outer-only.

**Tech Stack:** Go + CUE vocabulary (drift-guard test exists).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `CVResample.InnerK` + `InnerFolds()` | `pkg/experiment/shape.go` | modified |

- **`InnerFolds() int`** — `inner_k` if set, else `k`. The ONE derivation; no consumer reads `.InnerK` directly. Validation: `inner_k` absent or ≥2 (and a note when `inner_k > k` is legal — more inner precision than outer folds is odd but sound).
  - **DRY rationale:** two consumers (nested inner sweep, flat single-level CV) + the progress totals all derive from one accessor.
  - **Future extensions:** lever (b)'s racing sampler replaces the *sampler* over the same inner budget; the knob stays.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| CUE schema `cv: {k, inner_k?, stratify?}` | `construct/vocabulary/experiment.cue` | modified | shape frontmatter |
| `runShapeSweep`/`runNestedCV`/`runOuterFold` k-threading | `cmd/metis/sweep.go` | modified | the sweep engine |

**The k-split map (from recon — verify by grepping `splitK` + `scoreOnOuterFold` at impl; count claims are to be re-grepped, not trusted):**
- INNER (→ `innerK := sh.Sweeper.Resample.CV.InnerFolds()`):
  - nested: `runOuterFold`'s `pass.splitK` (sweep.go:511 — `FixedKFolds{K: splitK}` + `cvSplitStep`)
  - flat single-level path: `pass.splitK` (sweep.go:324)
  - `seededTotals(…, k)` last param (sweep.go:267) + the two preamble prints ("%d inner folds", sweep.go:404 + the runShapeSweep banner)
- OUTER (stays `k`):
  - `materializeOuterAnalysis(k)` + `outerPart` ref string (sweep.go:409-416)
  - `CVDriver{K: runFolds}` (runFolds ≤ k, the #42 sample knob)
  - `scoreOnOuterFold(…, k, …)` (sweep.go:549 — held-out partition MUST stay outer-k for cv_folds determinism to reproduce analysis_i's assessment rows — the #23 invariant; getting this wrong is silent leakage, so the e2e asserts the outer rows' fold partition unchanged by inner_k)

## Chunk 1: all tasks

### Task 1: schema + accessor (TDD)
- [ ] CUE: `resample: {cv: {k: int, inner_k?: int, stratify?: bool}}` + semantic comment (k = estimand + inner default).
- [ ] shape.go: `InnerK int \`yaml:"inner_k"\`` + `InnerFolds()`; validation (`inner_k` 0 or ≥2, sharp error naming the field). Failing tests first: parse shape with inner_k → InnerFolds()==inner_k; absent → k; inner_k:1 → validation error. Conformance/drift-guard suite still green.
- [ ] Commit.

### Task 2: sweep threading (TDD via the nested fake-exec e2e)
- [ ] Failing e2e first (nestedcv_e2e_test.go patterns, fake exec): shape `k:2, inner_k:3` →
  (i) banner prints "2 outer fold(s) × (N configs × 3 inner folds)";
  (ii) ledger INNER rows: per (config, outer fold) exactly folds {0,1,2} (3 inner folds);
  (iii) OUTER rows: exactly outer folds {0,1} and the held-out scoring runs at OUTER k=2 (assert the cv-split step's `with.k`==2 in the scoring run's recorded config — the leakage-guard tooth);
  (iv) outer-split dirs: analysis_0..1 only.
- [ ] Thread: `innerK` in runShapeSweep (flat `pass.splitK`, `seededTotals`, banner) → `runNestedCV(…, innerK, …)` (preamble print, `runOuterFold` param → `pass.splitK: innerK`); `scoreOnOuterFold` untouched on k. Re-grep `splitK` for missed sites.
- [ ] Existing suite green (same-k shapes byte-identical — the k==inner_k degenerate case IS the whole current suite).
- [ ] Commit.

### Task 3: docs + the (b) follow-up issue
- [ ] RUNBOOK: the knob + cost arithmetic (10 outer × 72 × inner_k — inner_k:5 halves the 7,200-fold grid; the estimand k stays 10).
- [ ] atlas experiment.md: one paragraph at the sweeper/resample section (k vs inner_k semantics).
- [ ] `sdlc issue new` — racing/successive-halving inner sampler (lever (b)): carry over the Spec(b) design notes verbatim + the SizeBudget/board readiness note; #45 Revisions records the (a)-first decision and points at it.
- [ ] Issue Log evidence; close (single boundary).

**Verification gate:** full `-race` suite; the new e2e red-proofed (revert the splitK threading → (ii) fails); RUNBOOK cost numbers arithmetic-checked.

## Revisions

### 2026-07-17 — plan review folded (3 Important, 3 Minor)

- **IDENTITY COMPAT (Important 1):** `InnerK` gets `yaml:"inner_k,omitempty" json:"inner_k,omitempty"` —
  without the json tag, `record.CanonicalHash(Sweeper)` (shapeRunIdentity, sweep.go:863-874)
  would inject `"InnerK":0` and churn EVERY existing shape's run identity/manifest dir. A
  regression test pins that an inner_k-absent Sweeper marshals byte-identically to today.
- **partitionRef IS INNER (Important 2):** sweep.go:797-800 mints `cv-k%d-strat%t-seed%d` from
  `cv.K` — the partition identity flowing into every inner `_fold.partition` (point addresses,
  Kpre) and told-set key. Derives from `InnerFolds()` (backward-safe: absent inner_k → same
  string). Completeness net rekeyed: **grep the FIELD (`Resample.CV`) + whole-struct marshals**,
  not the local `splitK` — full reader set: sweep.go:216-217, :798, shape.go:151, the
  shapeRunIdentity marshal.
- **FLAT PATH DECISION (Important 3): flat stays at `k` — `inner_k` is a NESTED-ONLY knob.**
  Rationale: on a 1-config flat run the sweeper CV IS the reported estimate, so inner_k there
  would silently change the estimand's train fraction — contradicting the architecture line
  (outer k = estimand, #42's principle). The flat path ignores inner_k with ONE loud note
  ("inner_k ignored — a flat run has no inner level; k is the estimand"); pinned by a flat×
  inner_k test; documented in RUNBOOK/atlas + a code comment at sweep.go:324.
- Minor 4: dry-run banners (sweep.go:241-246) print the inner count → innerK on the nested
  dry-run line (flat dry-run stays k, per the decision).
- Minor 5: the CUE drift guard's fixture omits inner_k — add an inner_k-bearing cue-vet case
  so a typo'd schema key can't pass green.
- Minor 6: assertion (iii) compares `with.k` as float64 (JSON round-trip).
