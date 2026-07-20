---
type: project
name: metis-v2-experiment-algebra
goal: "Evolve the metis workbench from a flat grid-sweep into a real experiment-design algebra that finds HONEST (non-overfit) performance — resampling (CV) and selection as first-class, declarative layers — proven on Titanic."
done_when: "The workbench expresses experiment design as driver · sweeper · pipeline over a three-phase shape (data │ pipeline │ ship): a black-box sweeper (training data → winner) owning inner-CV + a configurable select rule (1-SE / mean−std, not raw cv-max), an outer driver for NESTED-CV (honest procedure estimate), and LEAKAGE-SAFE features (per-fold structurally + internal cross-fit for target features). Honesty test (not just a bigger number): the honest nested-CV estimate must ACTUATE SELECTION — the outer CV *selects* the model family (metis#32), not merely reports it — so a Titanic run's SHIPPED-config public score tracks its honest estimate within noise. An honest *estimator* (metis#23) is necessary but NOT sufficient: the 2026-07-13 honest-beat run showed inner-CV cross-family argmax still ships the overfitter (GBM inner 0.846 → public 0.749) while the passed-over honest generalizer (rf md=4 + ticket_survival, inner 0.8395) scored public 0.79186. Done = a driver:cv run selects the generalizing family on its honest estimate and the shipped public tracks it, past v1's 0.77990 (rf+ticket already clears it at 0.79186 under inner-CV selection; #32 makes that selection principled, and GBM/ensembling/ticket remain the levers to push the honest number up)."
status: done
operator: xianxu
mvp_scope: [metis#18, metis#23, metis#19, metis#20, metis#21, metis#22, kbench#8]
explicitly_out: [metis#22, kbench#11, metis#33]
created: 2026-07-07
updated: 2026-07-17
sources: [metis/workshop/pensive/2026-07-07-experiment-design-algebra.md]
---

# metis-v2 — the experiment-design algebra

The next workbench chapter, on top of metis-v1 (the sweep→ledger→promote MVP, done). v1's Titanic
winner scored **~0.81 cv → 0.77990 public** — a textbook overfit gap. The lesson: piling on
features/models while selecting by raw cv-max just overfits harder. v2 makes the workbench **find
honest, non-overfit performance**, and the operator's insight is that this is an **algebra** problem,
not a pile of loops.

**Design source-of-truth:** the pensive in `sources` — read it first (**converged** after a long design
conversation + a 3-front prior-art survey). The converged model: experiment design is **driver · sweeper
· pipeline** over a **three-phase shape** (`data │ pipeline │ ship`):
- **sweeper** — a black box `training data → winner`; owns its inner-CV, objective, and select rule
  (mlr3 `AutoTuner`). Reduces by **selection**. The select rule (1-SE/robust, not cv-max) is the knob.
- **driver** — the outer, honest evaluator (optional). `single` ships; `cv`/`nested` gives the honest
  procedure estimate. Reduces by **aggregation**. This is nested-CV.
- **pipeline** — the swept (algorithm × hyperparameter) atom.

Two orthogonal knobs fall out (the operator's distinction, cleanly located): **estimation** (the
driver: flat vs nested) and **selection** (the sweeper's rule). The `data│pipeline` phase boundary is the
ONE structural cut → cross-fold leakage-safety with **no per-step markers** (`over:`/`fit_scope` both
dropped as error-prone; target-feature safety is the step's own internal cross-fit). Prior art is mature
and validated the model — **mlr3 is the structural twin** (`resample(AutoTuner(resample(learner)))`);
tidymodels (three-phase, `select_by_one_std_err`); sklearn (Pipeline per-fold, `TargetEncoder` cross-fit).

The two parts the operator asked for:
1. **Improve the ML algebra** — the sweeper substrate + nested-CV + robust selection (M1a, M1b, M2).
2. **The Titanic-score improvements** — leakage-safe features, GBM, ensembling, ticket feature (M3–M5).

## Milestones

### M1a — the sweeper substrate  *(metis#18 — DONE, codecomplete 2026-07-08)*
The core. Three-phase shape (`data│pipeline│ship`); a **black-box sweeper** (`training data → winner`)
owning its inner-CV + objective + select rule (mlr3 `AutoTuner`); **read-time reduction** of raw fold
rows → per-config `(mean, SE)`; **fold-as-artifact** scatter + **fan-in reducer** (content-addressed,
order-independent CV score); `driver: single` ship path. The `data│pipeline` cut gives cross-fold safety
with no markers. Everything else depends on this. **DONE (codecomplete 2026-07-08, pending merge):** all 5
review boundaries shipped — M1a-1 schema · M1a-2 pure Sampler core · M1a-3a IO rewire · M1a-3b input-addressed
cache + transitive-D soundness · M1a-5 driver:single ship + honest e2e + atlas. The offline mechanism is proven
(Go fake-exec e2e + a real-step smoke sweep shipping a valid submission); the honest-numbers acceptance (the
42-config Titanic run reproducing/beating v1's 0.77990 without selection-overfit inflation) is the
**operator-gated** next step (Kaggle creds, RUNBOOK). #23/#19/#20 now build on this substrate.

### M1b — nested-CV  *(metis#23 — deps M1a)*
The **outer driver** wrapping the sweeper (`driver:cv` = `resample(AutoTuner(...))`): result-dependent
select-then-assess-on-sealed-outer-fold → the honest procedure estimate (produces **no** winner; ~5×
compute). The estimation knob, separate from selection.

### M2 — selection objectives  *(metis#19)*
The sweeper's **select rule** (separate from estimation): `one-std-err` (simplest config within 1 SE of
the best — the operator's "less-overfitting near-winner", named) / `mean − λ·std`, over the `(mean, SE)`
from M1a's read-time reduction + a parsimony ordering from the tagged `$any` tree. **Uncontested across
all surveyed frameworks — our differentiator.** M1b then honestly estimates this better policy.

### M3 — leakage-safe target features  *(metis#20 — deps M1a)*
M1a already makes `features` per-fold structurally (it's in the `pipeline` phase). M3 is the
**target-feature's own within-fold cross-fit/shrinkage** (sklearn `TargetEncoder` / tidymodels
`step_lencode_mixed`) so a passenger's own label doesn't leak into their own feature. No engine marker
(dropped); the step owns it. Hard-blocks a whole bug class (kbench#8's ticket survival).

### M4 — models + ensembling  *(metis#21 GBM, metis#22 ensembling)*
- **metis#21:** add **gradient boosting** (HistGradientBoosting) as a model branch — the strongest
  tabular model, and a clean exercise of the metis#17 tagged-`$any` model set. Independent of M1;
  can start anytime.
- **metis#22:** **ensembling/stacking** as a new step-type (blend logreg + rf + gbm) — a new
  workbench primitive; top Titanic solutions ensemble.

### M5 — Titanic guinea-pig validation  *(kbench#8)*
The **ticket-group survival** feature (mean survival of a passenger's *other* ticket-mates — one of the
strongest Titanic signals; needs M3's internal cross-fit) + the **honest beat**: a run whose nested-CV
estimate tracks public within noise, ideally past 0.77990. Titanic is the guinea pig; the deliverable is
the workbench capabilities, not the number.

### Cross-cutting *(surfaced by the design's caching survey)*
- **metis#24** — cache addressing decision: input-addressed vs output-hash-chained interior (operator
  leans input-addressed for static plannability). Touches existing cache architecture.
- **metis#25** — get-data root-hash gap: dataset keyed on path string, not content → silent stale hit.
  Orthogonal soundness bug; makes the content-addressed interior end-to-end trustworthy.

### Honest-beat findings *(kbench#8 real 891-row run, 2026-07-13 — the ESTIMATOR is honest, the SELECTOR isn't)*
The first real `driver: single` ship (`titanic-winner-v3.2`) scored **public 0.749** vs inner-CV **0.846**
— a ~0.10 gap, *below* baseline. Root cause is NOT the ticket feature (it works — see below) and NOT random:
the sweep shipped a **GBM overfitter** (`[title,family]`, 1500 leaves → memorizes 891 rows) because
**cross-family selection is inner-CV argmax-mean** (metis#19 parsimony is intra-family only), and GBM's
inner CV (0.846) edged rf's (0.839) for a 0.007 mirage. So metis-v2 delivered the honest **estimator**
(#23) but the **selector** still overfits at the cross-family seam. Two follow-ons:
- **metis#32** — **outer-CV family selection** (THE headline fix): close #23's loop — the nested CV
  *selects* the family on its honest, cross-family-comparable estimate (+ 1-SE-across-families), not just
  reports. Within-family stays #19 inner-CV parsimony. Subsumes cross-family-argmax + cross-family-complexity.
  Brainstorm-first (extends the driver/sweeper outer-node contract). *deps metis#23.*
- **metis#33** — **GBM overfit**: bug-vs-regularize (grid floor / early stopping / reg defaults for 891
  rows) + the **effective-complexity measure** (cx is leaf-cap-pinned = `max_iter×max_leaf_nodes`,
  shrinkage-blind → intra-family parsimony is a *no-op* for GBM; the metis#21-foreseen weakness, now biting).
- **Ticket feature validated:** matched-pair inner-CV Δ of **`ticket_survival` ≈ +0.007–0.009** across
  rf/logreg/gbm (leakage-safe cross-fit → should hold out-of-sample); `ticket_size` weak (+0.002, hurts
  logreg). The rf md=4 + `ticket_survival` config (cx 14.3) is the robust ship.
- **Number recovered — rf-ticket SHIPPED 2026-07-13 → public 0.79186** (`kbench titanic-winner-rf.md`,
  run `titanic-winner-rf-live`). Beats v1's 0.77990 AND the GBM overfitter's 0.749 (**+0.043**). Three
  empirical confirmations in one submit: (1) **rf > GBM at the cross-family seam** — the generalizer
  argmax-mean passed over beats the family it shipped, so an honest (outer-CV) family selector *should*
  pick rf → **the direct empirical case for metis#32**; (2) **ticket's public value** — 0.79186 vs the
  historical pre-ticket rf 0.782 ≈ **+0.01 on the board** (tracks the +0.007–0.009 inner Δ → kbench#8
  paid off out-of-sample); (3) **the inner→public gap halved** (0.048 for rf vs GBM's 0.097) — rf's
  inner-CV is far less optimistic, though still not honest (that's #32). *`promote --point` can't select a
  list-valued free-param (parses `[a b c]` as a string) — this ship was hand-authored; fix in the metis#22
  `promote --family` work.*

### Runner infra follow-ons *(surfaced by the kbench#8 sweep-scale discussion — 495/2,475 per-fold runs)*
Both hang off the **existing ask/tell `Run` loop** (grid + adaptive share one runner — no per-sampler
runner); they're the two injected seams that make it scale. Independent of M4b.
- **metis#30** — runner progress reporting: `SizeHint` on `Sampler` (n = exact | budget | unknown) +
  a `progress` callback per `Tell` → live `k/n` + running outer-cv. The cheap, near-term one (you feel
  the blindness at 2,475 folds).
- **metis#31** — parallel batch executor: an injected concurrent `exec` on `Run` (parallelism = the
  `Ask` batch width, sampler-declared; grid = embarrassingly parallel). Determinism-preserving via M1a's
  content-addressed, order-independent reduce (the determinism test is the spine). ONE global cap `n`
  (default `NumCPU`) enforced at the leaf subprocess spawn — not per-level (nesting would multiply it).
  Bigger perf win, more care. **= the operator's "implement parallelism so grid sweeps faster."**

## tasks

- [x] **M1a** metis#18 — sweeper substrate (three-phase shape, black-box sweeper, read-time reduction, fold-as-artifact). **DONE — codecomplete 2026-07-08** (5 review boundaries M1a-1..M1a-5, all shipped; pending `sdlc merge`). The Sampler fold node algebra (driver⊃sweeper⊃resample), input-addressed cache + transitive-D soundness, driver:single ship, honest per-config (mean,SE) all landed; real 42-config Kaggle honest-numbers run operator-gated (RUNBOOK).*
- [x] **M1b** metis#23 — nested-CV (outer driver; honest procedure estimate). **DONE — codecomplete 2026-07-12, 2 boundaries M1/M2 both FIX-THEN-SHIP + fixed** (est 3.1h / actual 2.75h, 0.89×). `driver:cv` = a pure `CVDriver` outer resample over the unchanged `Run` loop (zero engine change); per outer fold: **sealed** sweep on a physically-subset `analysis_i/` → winner, then refit-and-score on the held outer-assessment (full-data fold at OUTER k; `cv_folds` determinism reproduces the partition) → `Aggregate` → **mean±SE**, the honest estimate. Ships **NO** winner. **Sealing = L1 structural** (outer-split subset dirs) **+ L2 trace-confinement** (`METIS_READ_ROOT` asserted at `metis/io.py:exp_path`, covers parquet; **proven through the real chain** — a real uv `cv-split` via `execStep` reading out-of-root is caught). `GuardComplexity` runs on the nested path too (no silent mis-select). **M1 sealing spine is shared** — #20/kbench#8 inherit it. **MERGED (PR #18) 2026-07-12.** Follow-up metis#29 (real-data driver:cv confinement e2e, blocked on a toy data-step) filed. *Honest-estimate-tracks-public acceptance operator-gated (Kaggle).*
- [x] **M2** metis#19 — sweeper select rule over (mean, SE, **measured complexity**). **DONE — 2026-07-09** (2 milestone boundaries M1/M2, both shipped; est 3.7h / actual 6.15h). Tagged-union `objective.select` (argmax-mean|one-std-err|pct-loss|mean-std, mirrors `driver`); pure two-level `SelectConfigs` (group-by-family → band → ε-binned min-complexity → mean tie-break; cross-family argmax-mean) reused by both the in-memory ship path and offline `metis ledger select`; complexity **measured on the fitted model** (rf mean leaves/tree, logreg coef count) not guessed from hyperparams. **VERIFIED acceptance** (real Titanic, warm cache): `pct-loss` selects rf **md=4 / all-6-features** (cx 14.6 → public 0.782) over argmax-mean's **md=8** overfitter (cx 66.3 → public 0.770) — the differentiator, empirically confirmed, NOT the sparse nfeat=1 corner. *the differentiator.*
- [x] **M3** metis#20 — leakage-safe target features (internal cross-fit). **DONE — codecomplete 2026-07-12, review FIX-THEN-SHIP/INFO** (est 1.05h / actual 1.62h, 0.65×). Two repos: reusable primitive `metis/encode.py::cross_fit_target_encode` (internal K-fold cross-fit reusing `cv_folds` + m-estimate shrinkage; `strategy ∈ {kfold, loo}` for cross-competition reuse) + kbench group-protocol extension (`apply_features` +seed + `TARGET_GROUPS` branch, 6 stateless groups byte-identical; `target_encode_group` adapter; `pclass_survival` demonstrator, NOT wired into the sweep). **Leak proof at the feature level:** naive-incl-self `corr=0.73` with own label vs cross-fit `corr=−0.04` on no-signal small-group data; fold leakage-cut proven through the real `apply_features` chain. Titanic thread green (42 configs, demonstrator absent). K-fold chosen over LOO (LOO's `1/(n−1)` within-group invertibility leaks worst for small groups + GBM). *M1's per-fold `fit_mask` gives cross-fold safety for free; #20 adds the within-fold cross-fit. **Unblocks kbench#8 (M5).** **MERGED** — metis PR #19 + kbench PR #7 (both to main, both suites green); #20 archived to metis `workshop/history/`.*
- [x] **M4a** metis#21 — GBM (HistGradientBoosting) model branch. **DONE — codecomplete 2026-07-11, review SHIP** (est 0.6h / actual 0.62h, 1.0× — clean calibration). Python-only (3 touch points: `MODELS`+`make_model`+`complexity`); Go layer derives the family structurally (`FamilyOf`), zero edits. Complexity = **total realized leaves summed across boosted trees** (sum, not rf's mean — boosting is additive; grounded in an ESL/Friedman/Bühlmann–Hothorn/XGBoost literature pass). The learning_rate-shrinkage caveat contained structurally: baseline shape fixes ν, sweeps max_iter×max_leaf_nodes. `metis run -dry-run` → 33 configs incl. 12 hist_gbm. *pending `sdlc merge`; real-data ledger run operator-gated (Kaggle).*
- [ ] ~~**M4b** metis#22 — ensembling/stacking step-type.~~ **PUNTED 2026-07-16 (operator, rescope):** the 2026-07-14 reframe made its payoff the **gated rule+residual** (Deotte's 0.84688 architecture) — the operator then ruled out hard-coded-rule submissions on principle, parking the whole 0.80+ rule tier; plain stacking was already downgraded (scores below the no-ML WCG rule). Status open→punt, Revisions entry in the issue. Reopen trigger: a future competition where model combination is a live lever on its own merits.
- [x] **M5** kbench#8 — ticket-group survival feature + honest Titanic validation. **OFFLINE HALF DONE — codecomplete 2026-07-13, review SHIP** (est 0.67h / actual 0.59h, 1.1×). Two independent toggles: `ticket_survival` = leakage-safe cross-fit (a one-line `partial(target_encode_group, key="Ticket")` — inherits M3's #20 primitive) + `ticket_size` = both-frames count over train+test (fold-independent). Registry renamed `TARGET_GROUPS`→`GROUP_FEATURES` (a non-target member joined). **Leak proof empirical:** naive-whole-train corr 0.75 with own label vs cross-fit 0.08. Sweep wired: +3 ticket rungs (size/survival/both — attribution) + `hist_gbm` (M4a #21) → **99 configs** (dry-run confirmed); all stale `42` docs reconciled. *The honest **beat** (nested-CV estimate tracks public, ideally past 0.77990) is **operator-gated** — needs the real 891-row Kaggle data (toy testdata is all-singletons); RUNBOOK §6 sets it up + flags the fit_mask-both-levels check (`ticket_survival` = first target feature swept under nested CV). kbench branch `000008-ticket-group-feature` — pending merge.*
- [x] **X** metis#24 — cache addressing decision. **DONE (docs-only close, direct push) 2026-07-17** (actual 0.15 labeled-judgment; close review SHIP). Closed as **decision-complete**: input-addressed was decided 2026-07-07 and shipped/hardened in #18 M1a-3b (Kpre-on-upstream-Kpres + transitive-D); this close added the explicit TRADE-OFF record to atlas (early-cutoff given up — cheap in ML where outputs aren't byte-reproducible; buys static plannability + nondeterminism robustness) and DEFERRED the pre-run cache-hit-map printout by decision (demand-driven, next competition — issue Revisions). With #25's pins the content-addressed interior is end-to-end trustworthy. *cross-cutting.*
- [x] **X** metis#25 — get-data root-hash gap (soundness). **DONE + MERGED (PR #36) 2026-07-17** (est 0.47 / actual 0.9 — labeled judgment, transcripts unattributable from the brain-dir session; close review SHIP). Spec-at-claim reframe: the local-path premise predated M1a — the live gap was REMOTE ingest identity, closed with **config-declared content pins** (Nix fixed-output model): `kaggle/download` gains `with.sha256`, verifies post-unzip (mismatch/missing/extra = loud fail; contract files excluded mirroring collectArtifacts), unpinned ingest prints a loud paste-ready block. Pins ride the existing `with → Kpre` channel — zero metis cache change. `titanic-sweep.md` pinned from live-run artifacts; e2e-dual-use shapes deliberately unpinned (one static block can't serve two data truths). kaggle 0960f34+a9aadcf · kbench 742238c. *cross-cutting; orthogonal.*
- [x] **X** metis#30 — runner progress reporting. **DONE + MERGED (PR #27) 2026-07-15** (est 1.63 / actual 1.51, 1.1× — clean calibration point). `SizeHint(total, kind)` on the Sampler interface (exact|budget|unknown; all 4 production samplers exact) + `Run` fires `ProgressEvent[P,O]` at POINT COMPLETION (spec revised: per-Tell would land at batch end under #31's batch exec — Revisions entry) + the cmd/metis sink: ONE throttled line `metis: progress outer j/k · configs a/b · folds x/y · est mean±SE` (totals seeded at wiring from SizeHint; 1s throttle on the injected clock; always-emit on outer completions; plain lines, no escape codes). **#38 seam designed in: outer-fold identity via per-pass closure binding (`forPass(i)`), never a payload field.** Close review SHIP; real smoke evidence: live `folds 1/36→21/36`, est evolving to `0.8103 ± 0.0062`. *runner infra.*
- [x] **X** metis#38 — parallel-run TUI. **DONE + MERGED (PR #28) 2026-07-15** (est 2.19 / actual 1.50, 1.5×). On a TTY a sweep pins a live board to the bottom (step logs scroll above): aggregate line · one row per outer fold (`✓ held-out` / `▸ configs a/b · folds x/y · best` / queued, ≤12 + overflow) · `leaves b/c · R folds/min · ETA` — the moving-average rate DECAYS on stalls (the k10-probe BLAS-thrash signature, visible in seconds). **Design deviations (Revisions'd): hand-rolled ANSI pin-bottom, no TUI lib** (output-only board; 2-dep module) **+ pinned-bottom over full-screen** (hiding step logs would lose the hung-vs-working signal). Zero pkg/sampler change — rides #30's forPass(i) closure-bound hooks. Plan review caught a real Critical (writer identity is temporal — the fork pool captured the pre-board writer; fixed by parse-first reorder, bypass routes pinned by test). Close review FIX-THEN-SHIP → fixed in close commit. Real pty evidence (script -q, cold cache): 246 repaints, leaves 0/8→8/8 live, in-flight incumbents, clean final frame; redirected run byte-clean. *operator UX; filed from the #35 honest-beat's minutes-of-silence.* **Post-merge bugfix metis#46 (PR #29, same day, est 0.61 / actual ~0.2):** the per-write erase/repaint strobed at ~500Hz under warm-cache bursts — real terminals/mux layers (operator's ghostty-in-cmux) tore mid-sequence ("unorganized lines"). Compositor now double-buffered with a 250ms flush budget: one atomic erase→dump→repaint per window (live warm pty: 7 erases/run vs ~150+), quiet writes still inline, tick force-flushes. Diagnosis methodology: live pty repro + pyte terminal-emulator replay proved sequence CORRECTNESS, isolating the failure to real-terminal timing under sequence VOLUME. **UX-iteration cluster (2026-07-16, operator live-testing):** #47 flash fix (DEC 2026 synchronized output — atomic apply on ghostty/iTerm/kitty) DONE+MERGED · #50 run-end summary (elapsed, rows→ledger, cohort, paste-ready select commands) DONE+MERGED · filed: #48 default leaf BLAS pins (the 3h-ETA root cause bare), #49 board readability (labels/cold-phase indicator/leaves smoothing/ETA damping), #51 ledger-show point_addr column, #45 partial inner CV. First operator full-grid run: 7,200 folds in ~20 min pinned (ETA panic was the #49 cold-phase artifact); cohort ee3d36bf honest pick rf md4/n200+WCG 0.8384±0.0072 over 10 folds — best honest estimate yet. **Operator live-submitted it (best-rf-6dde4f89) → public 0.78947 = identical to kbench#9's WCG rf submission (330/418): the k10 full-grid honest selection CONVERGED on the same config → confirmation, not new information. Operator raised the ceiling question + ruled out hard-coded-rule submissions — pure-ML plateau ~0.78-0.79 per the research ground truth; remaining learned-feature levers (surname-keyed groups etc., archived kbench#9 rungs) worth ~1-3 flips; the 0.84 gap is mostly explicit rule expression (out by preference). Project done_when is MET (honest estimate actuates selection, shipped public tracks it — twice).**
- [x] **X** metis#39 — fingerprint visibility. **DONE + MERGED (PR #26) 2026-07-15** (est 1.55 / actual 0.65, 2.4× over — the plan carried complete code, so impl was largely transcription). Shipped: `metis run` prints `recording under code_fingerprint <hash> (commit <sha>, clean|dirty)` from both capture sites · **`metis ledger fingerprints <shape>`** (per-cohort rows-by-level, first…last record times, commit+dirty+capture, `(legacy)` group, newest-last; tolerant of cleaned run dirs → `?`) · **git-style `--fingerprint` prefixes** in select + ledger show (one `pinFingerprint` resolver — ends the --fingerprint/--point semantics split) · multi-cohort guard + zero-match errors render the cohort table inline + name the command (the "no scored configs" lie is dead, pinned by test). Close review FIX-THEN-SHIP (sweep first-record-missing test gap; degraded-latest dropped the dirty marker) → fixed in the close commit. **Live smoke: `select --fingerprint 566995b9` (the operator's exact failing repro) now resolves.** *operator UX; filed from the honest-beat's bare-hash guard wall.*
- [x] **X** metis#31 — parallel batch executor. **DONE — codecomplete 2026-07-13, plan fresh-eyes-reviewed (1C+5I+3m folded in) + both change-code judges INFO** (est 2.8h / actual N/A — interleaved-session mention-fallback across #30-33 + fork-executed impl; not laundered). Injected `exec(batch,runPoint)[]O` on `Run` (`SeqExec`/`ParExec`/`ExecFor`, order-preserving) + ONE global leaf semaphore at `execStep` (cap `n`=NumCPU, `--parallel`/`METIS_MAX_PARALLEL`) — orchestration goroutines unbounded+budget-free, only real subprocess spawns draw → peak ≤ n across `driver⊃sweeper⊃resample`, deadlock-free. Byte-identical `Done` (index-addressed fan-out + order-independent reduce). Fixed 2 concurrency bugs the review caught: atomic cache-index write (was torn `os.WriteFile`) + the C1 git-probe false-abort (`s != "" && s != codeID`). Hermetic wall-clock **4.5×** (serial 432ms→parallel-8 95ms); full `-race` suite green (7 load-bearing tests, atomicity/C1/determinism RED-first). *runner infra; perf — the operator's "sweep faster".* **MERGED (PR #20) 2026-07-13.**
- [x] **X** metis#32 — outer-CV family selection. **DONE + MERGED (PR #21) 2026-07-13/14 — THE headline metis-v2 capability.** The honest outer-CV estimate now **ACTUATES** family selection (closes the loop #23 left open). Long operator-driven brainstorm → spec **twice** fresh-eyes-reviewed → durable plan → both change-code judges INFO → 2 milestones (M1 measure/record, M2 choose/ship), each boundary-reviewed FIX-THEN-SHIP→fixed, final close SHIP; all 9 pkgs green `-race` throughout. Acceptance gate `TestSelect_PicksGeneralizerNotInnerOverfitter` proves the flip: `select --best` ships the generalizer over the inner-CV overfitter. Shipped: `metis run` (measure; mode DERIVED by config-count, `--fast`; records inner+outer `Level`-keyed rows; **no auto-ship**; `driver:` deleted) · `metis select [--best|--best-per-model-class] [--promote]` (family by lowest-SE-within-1-SE honest estimate + config by inner CV; reconstruct-and-ship all-data `best-{family}-{hash}`; fingerprint join-soundness guard) · retired `metis ledger select` + `metis promote`. kbench RUNBOOK migrated to the 3-command model (side-quest). *(est 4.5h / actual N/A — interleaved-session mention-fallback + fork-executed; not laundered.)* **DESIGN NOTE:** the whole design (run/select separation, derived mode, reconstruct-never-materialize, `--fast`, outer-as-pure-measure) came from the operator's steering across the brainstorm.
- [x] **X** metis#34 — cwd-independent run/select/submit. **DONE + MERGED (PR #37) 2026-07-17** (est 0.38 / actual 0.35 labeled-judgment; close review SHIP). **Audit reframe: the premise was false for `metis run`/`select`** — identity is content-addressed and anchors are exp-path-derived, so no canonical-path key was built (Simplicity First); the invariant is now PINNED by a cwd-independence regression test (two cwds → same runs dir + identical point_address). The two real drifts fixed: the bare-repo steppath fallback anchors on the shape's repo (was cwd; red-proofed) and `kaggle submit -C <pipeline-dir>` anchors `runs/` from any cwd (kaggle 8addd9f). *split from #32 brainstorm.*
- [x] **X** metis#42 — `--sample m` m-of-k sparse fold sampling + the k10 attenuation probe. **DONE + MERGED (PR #24) 2026-07-14** (est 0.95 / actual 0.27). Probe CONFIRMED the #36 attenuation hypothesis (pre-committed rule): `+ticket_survival` inner increment k5→k10 rf 0.0020→0.0078 (~4×), gbm 0.0059→0.0098; label-free `ticket_size` control flat. *k stays the estimand knob; m the precision/cost knob.*
- [x] **X** kbench#9 — WCG feature (woman-child-group survival over ticket-exact groups; `WCG_AllLived`/`WCG_AllDied`, WC labels only, self-excluded, no shrinkage). **BUILT + SHIPPED 2026-07-14** — public **0.78947** (330/418; one passenger off the 0.79186 all-time best; honest outer 0.8283±0.0140). Rule fully expressed at predict time (8/8 Masters, 7/7 females). Sweep slimmed in place (72 configs, k:10, logreg dropped). **CLOSED + MERGED 2026-07-15 (kbench PR #9)** — done-when's "beat 0.79186" miss (1 flip) documented in an operator-approved Revisions entry; close review FIX-THEN-SHIP (atlas registry line + mixed-evidence predicate test, falsification-verified) → fixed in the close commit; est 0.95 / actual 1.89 (diagnostics + 2 live submissions un-estimated). Candidate rungs live in its (archived) Log. *Side discovery: public LB grades ALL 418 rows (integer proof) — 1 flip = 0.239%, SE ±2.0%.*
- [x] **T1** metis#44 — leaf executor fork-server. **DONE + MERGED (PR #25) 2026-07-15** (est 1.08 / actual 2.35 — review findings + benchmark discipline). Warm per-root server (third-party-only preload for D precision), fork-per-step preserving every per-step semantic; loud legacy fallbacks; `--forkserver=false` hatch. Close review FIX-THEN-SHIP caught a REAL fork-vs-stdout-lock child deadlock (C1, fixed: fork under protocol lock + fresh child streams) + the dispatched-and-lost double-execution hazard (I1, now errors the step; server process-grouped since `uv run` doesn't exec-replace). **Measured: real kbench smoke legacy 21.8s→9.8s wall, 104→30s user-CPU (3.5× leaf-bound); warm marginal ~43ms vs ~290ms/leaf; per-leaf import tax on real leaves ~1.85s.** *throughput tranche 1 — first half landed.*
- [x] **T1** kbench#10 — e2e workspace isolation. **DONE + MERGED (PR #10) 2026-07-15.** Disposable tmp workspace miniature (construct/steps/mds/testdata copied; .venv+peers symlinked, `UV_NO_SYNC` — a tmp-project `uv run` otherwise SYNCS the real venv, observed+lesson'd); suite-wide autouse guard in root conftest.py fingerprints the live competition/ tree + venv editable pointers, falsification-verified. Close review FIX-THEN-SHIP (guard scope + breadth) → fixed. **Throughput tranche T1 COMPLETE** — next per operator order: T2 (#38 TUI+runs/sec, #39 fingerprint UX). *throughput tranche 1 — second half landed.*
- [x] **T2** metis#38 — (already listed above, DONE + MERGED 2026-07-15) — the moving-average runs/sec + ETA line landed as specified. **T2 "better UX" tranche COMPLETE (#39, #30, #38 all merged 2026-07-15). Next per operator order: the levers — #22 gated rule+residual, priced first by kbench#11 (one operator submission); #43 depth-first scheduling adjacent; #36 channel split (stage B).** *throughput tranche 2 — complete.*
- [x] **T2** metis#39 — (already listed above, now DONE + MERGED 2026-07-15) — both 2026-07-14 additions landed: `--fingerprint` prefix matching + truthful zero-match error listing cohorts. **T2 remaining: #30→#38.** *throughput tranche 2 — first item landed.*
- [x] **X** metis#43 — depth-first leaf scheduling. **DONE + MERGED (PR #34) 2026-07-17** (est 4.74h / actual 4.70h, 1.0×). One sweep-scoped `2n` whole-run admission controller now sits above the independent `n` leaf budget, reaching complete cold runs before admitting the whole grid; its set-once first-error authority also suppresses queued side effects, sampler placeholders, persistence/reporting, and stale TUI redraws. Deterministic cold-order, nested-cap, artifact-equivalence, controlled-tick failure, full `-race`, and pinned-peer real-process smoke passed (7 trains in 45s; first train before fifth run). *Scheduler half of the paired #43/#49 work is complete; #49 is next.*
- [x] **W** metis#48 — default leaf BLAS pins. **DONE + MERGED (PR #33) 2026-07-16** (est 0.96 / actual 0.71, 1.4×). The orchestrator now injects single-thread OMP/OPENBLAS/VECLIB/MKL defaults at both leaf spawn seams unless the operator exported a value; `--parallel` is the one budget. Full `-race` suite green; bare cold-cache real-data smoke completed 720/720 folds in 1m23s with exactly one note. Close review FIX-THEN-SHIP required the peer RUNBOOK change to be pinned to kbench commit `bf57c5c`; fixed in the close anchor. *first 2026-07-16 rescope quick win.*
- [x] **W** metis#49 — board readability. **DONE + MERGED (PR #35) 2026-07-17** (activity-backed board telemetry; built by the parallel session — see its Log entry when it lands; row reconciled here 2026-07-17 since the merge is on metis main). *adopted at the 2026-07-16 rescope.*
- [x] **W** metis#51 — ledger show point column. **DONE + MERGED (PR #38) 2026-07-17** (est 0.17 / actual 0.35 labeled-judgment; close review REWORK → fixed → SHIP). The review caught the real bug: `--sort` aggregates via AggregateView, which stamped a SYNTHETIC group key into PointAddr — the column rendered garbage exactly on the sorted leaderboard. Fixed at the source: aggregate rows now carry a REAL member addr (any member resolves — resolvePointRows expands to the config), so ledger-show AND select's #52 handles derive from one authority; end-to-end test drives the literal Done-when flow. *adopted at the 2026-07-16 rescope.*
- [x] **W** metis#53 — promote fingerprint-consistency guard. **DONE + MERGED (PR #39) 2026-07-17** (est 0.48 / actual 0.6 labeled-judgment; close review FIX-THEN-SHIP → fixed → SHIP). `select --promote` (both --best* and --point) now REFUSES when the working tree is not the selected cohort's code: per-path blob compare over the cohort's captured D closure (same gitBlobHashes as capture), diff-shaped refusal + capture-commit restore hint, `--no-fingerprint-check` loud override, legacy provenance warns-and-proceeds. Review fold: per-path retry so one deleted file can't poison the batch (delete e2e pins it); the fake hasher now mirrors production's batch-failure semantics. Restore half stays metis#28. *adopted at the 2026-07-16 rescope.*
- [x] **W** metis#45 — partial inner CV. **DONE + MERGED (PR #40) 2026-07-17** (est 0.86 / actual 1.2 labeled-judgment; close review FIX-THEN-SHIP ×2 → folded). **Lever (a) shipped:** `sweeper.resample.cv.inner_k` — k stays the ESTIMAND knob (outer + inner default; every existing shape identity-stable, pinned by a marshal-identity regression test after the plan review caught the would-be churn), inner_k overrides the INNER per-config CV only (10×72×5 = 3,600 leaf-folds vs 7,200). partitionRef minted from the resolved fold count; flat runs ignore the knob loudly (their CV IS the estimate); the leakage tooth asserts held-out scoring stays at outer k via decoded records; progress totals toothed (12/12). **Lever (b) — the racing/successive-halving sampler — filed as metis#54** (demand-driven, next competition). *adopted at the 2026-07-16 rescope.*
- [ ] ~~**X** kbench#11 — WCG rule-only submission diagnostic.~~ **PUNTED 2026-07-16 (operator, rescope):** existed to price the #22 rule+residual build; punted alongside it (a rule-only submission is the purest instance of what the no-rules call excludes). Revisions entry in the issue.
- [ ] **X** metis#33 — GBM overfit (bug-vs-regularize + effective-complexity measure; intra-family parsimony no-op). *surfaced by the honest-beat run.* **OUT of v2's close gate (2026-07-16 rescope): stays open in the metis tracker, likely picked up in the next competition.*
- [x] **X** metis#35 — **nested-CV one-road fix (stage A) — BUILT + the real honest-beat RAN 2026-07-14** (close pending merge). The brainstorm escalated "missing repoint" to the design gap: the seal substitutes a derived artifact + deletes its producers, sound only if it's the SOLE road — `raw: get-data` was a second road. Research detour (2 deep-research passes + 27-agent adversarial verify → the feature-algebra pensive, `metis/workshop/pensive/2026-07-14-01-*`) → 3-stage plan: **A = this** (adapt carries source cols under new `source` schema role; features drops `raw:`; outer-split carries `test` → analysis_i shape-identical; transductive declared) · **B = metis#36** (channel split: y as runner-scoped keyed artifact) · **C = metis#37** (constructor algebra). Nested smoke e2e un-xfailed → first-ever green nested run through the real pipeline. Siblings filed from the run: #38 (parallel-run TUI), #39 (fingerprint visibility). *(the #32 cohort guard fired correctly mid-run — caught last session's flat rows from blending into the nested reduce.)*

**`done_when` status — machinery DELIVERED; the tracking clause needs an operator decision.** The
2026-07-14 honest-beat (5 outer × 99 configs × 5 inner on 891 rows): per-family honest estimates
hist_gbm 0.8361±0.0103 · **rf 0.8328±0.0045 (selected — lowest-SE-within-1-SE declined the GBM's
higher-variance mean; the 0.749 trap avoided by RULE)** · logreg 0.7879±0.0074 → promote
`rf [title,family,age] md=4 n=500` → **public 0.77751**. **Leakage tell PASSES** (15 fold×family
winners, outer−inner ≈ −0.002 mean, two-sided; the ticket_survival winner +0.0068 ≪ 1 SE) — outer
leakage ruled out on the real pipeline for the first time. **But public sits ~12 estimator-SE below
the honest estimate**: a train↔test distribution gap + LB subsampling (public ≈209 rows → its own
SE ±0.029; 0.77751 / v1's 0.77990 / rf-ticket's 0.79186 are mutually within ~1.4 LB-SE — the LB
cannot resolve differences at this scale). Nested CV delivered its actual promise (SAME-distribution
honesty, outer≈inner); the done_when's "public tracks the estimate within noise" clause conflated
that with cross-distribution prediction no CV can deliver, and "past 0.77990" is below LB
resolution. **Decision pending (operator): revise the done_when to separate internal honesty
(delivered) from the LB beat (push via #22 ensembling / #36's ticket estimand experiment), or hold
it open as-is.** The selector's no-ticket pick vs rf-ticket's 0.79186 (Δ0.014, inside LB noise) is
exactly metis#36's transductive experiment. *[RESOLVED by events 2026-07-16: no revision needed —
the clause as written was met twice (cohorts b7aee3de, ee3d36bf); see the rescope Log entry.]*

**Same-day follow-up (metis#41, operator-prior production experiment):** the operator insisted on
the ticket features; `select --point` (built+merged same day — publish any ledger row by
point_addr) promoted rf md=8 n=200 [all-6+ticket_size+ticket_survival] (honest inner 0.8297±0.0043,
BELOW the rule pick's 0.8305) → **public 0.78229 — ABOVE the rule pick's 0.77751 and v1's 0.77990.**
Public board: rf-ticket 0.79186 > point-rf(tickets) 0.78229 > v1 0.77990 > best-rf(no tickets)
0.77751 — ticket configs hold the top two public spots while nested CV ranks them lower; pairwise
inside LB noise (±0.029) but three same-direction samples = directional evidence for the estimator
under-ranking co-occurrence features (coverage 38.6%→~30% under the seal, m=10 shrinkage) — the
sharpened metis#36 hypothesis.

**Next action** *(superseded 2026-07-16 — see the rescope Log entry; current close gate lives
there)*: ~~operator's `done_when` call above → merge + close metis#35 → #22, #36, #30/#38, #39,
#34~~ — #35/#30/#38/#39 all landed; #22 punted; #36 is v3 territory.

**2026-07-14 (evening) — direction brainstorm + metis#42 probe (DONE+MERGED same session, PR #24,
est 0.95/actual 0.27):** web research (digest: `kbench/workshop/pensive/2026-07-14-01-*`) reframed
the LB levers — the 0.78→0.83 gap is **WCG group-survival propagation** (Deotte: rule-only 0.81818,
rule+XGB residual 0.84688; stacking without group rules caps at ~0.808), NOT model blending — so
**#22's Titanic payoff is downgraded** (stays valid as a platform primitive; the highest-value
"ensemble" here is a gated rule+ML-residual, worth expressing in #22's contract). New: **kbench#9**
(WCG feature, surname∧ticket groups — operator-directed next build). **metis#42** (`--sample m`
m-of-k sparse fold sampling + k10 probe) CONFIRMED the #36 attenuation hypothesis with a
pre-committed rule: `+ticket_survival` inner increment k5→k10 rf 0.0020→0.0078 (~4×), gbm
0.0059→0.0098; label-free `+ticket_size` control flat (the shift is specific to the label-dependent
channel); sealed selection flips toward ticket configs at k10 (rf 2/3 vs 1/5 outer folds). #36's
estimand knob is now evidence-backed, not hypothetical. Ops lesson institutionalized (metis
lessons + RUNBOOK): pin BLAS threads + cap `--parallel` for real sweeps; #38 gains the
moving-average runs/sec requirement.

**2026-07-15 — operator prioritization + kbench#9 shipped.** kbench#9 (WCG feature) shipped:
public 0.78947 (330/418, one passenger off the 0.79186 best); rule fully expressed at predict
time (8/8 Masters, 7/7 females); LB discovered to grade ALL 418 rows (integer proof) — noise
math corrected everywhere. Operator's point-experiment (gbm it50, top pooled-inner 0.8424) →
public 0.77033: third same-direction vindication of the outer-estimate 1-SE selector. **Operator
priority order: throughput first — metis#44 (fork-server leaf executor, CLAIMED) + kbench#10
(e2e workspace isolation), then metis#38 (TUI + runs/sec) + metis#39 (fingerprint UX) — "iterate
faster" before pushing the board again.** Performance levers queued behind: metis#22 reframed as
gated rule+residual (the evidenced headroom), kbench#11 (rule-only submission diagnostic, filed),
kbench#9 close decision pending (candidate rungs logged in its Log).
**Then the metis-v2 honesty payoff is one operator-gated Kaggle run:** the `driver: cv` honest-beat on the
real 891-row data (titanic-sweep.md now has ticket features + GBM + nested-CV) — does the nested-CV
estimate track public, ideally past v1's 0.77990? That's the `done_when`, and it needs only creds + a run
(RUNBOOK §6), no more code. The per-family leaderboard #19
ships is exactly what #22 blends and #23 estimates one-per-family. #21
gave the ledger a third model family, so #22 (blend logreg+rf+gbm) and #23 (nested-CV per family)
now have all three to work with.

## Log

### 2026-07-17 (PROJECT CLOSED — gate 9/9; metis#45 merged)
**metis-v2 is DONE.** done_when was MET 2026-07-16 (honest estimate actuates selection; shipped
public tracks it — twice); the 2026-07-16 rescope defined a 9-issue close gate, and the last
item (#45, PR #40) merged today. Final tally: the algebra core (#18/#23/#19/#20/#21/#32/#35 +
kbench#8/#9) · throughput (#31/#44/kbench#10) · UX (#30/#38/#39/#41/#42 + the #46-#53 cluster)
· platform (#24/#25/#34) · the inner_k cost knob (#45). Punted by the no-hard-rules call:
#22 + kbench#11. Open in the metis tracker beyond v2: #33 (GBM overfit — next competition),
#54 (racing sampler — demand-driven), #36/#37/#40 (v3 design track), older backlog.
**Two sessions closed the gate in ~26 hours**, every issue full-SDLC with a boundary review
that caught a real defect on nearly every close (#48 traceability, #51 aggregate-key
rendering, #53 batch-poisoning, #45 identity churn + estimand semantics).
**NEXT ARENA (decided 2026-07-18): Playground Series S6E7** — successor project
`metis/workshop/projects/arena2-playground-s6e7.md` (per the new charter, in the work repo);
bring-up issue kbench#12. The standing frame holds: zero new workbench features until the
competition demands them (#54 and #33 are the queued demand candidates).
*(Housekeeping: per the 2026-07-17 charter change, projects now live in the
center-of-gravity repo (`metis/workshop/projects/`), never in brain — this file is
legacy-located; migrate via `sdlc migrate` if/when a v3 project opens, rather than churning
a closed artifact.)*

### 2026-07-17 (close-gate items 7-8/9 — metis#51 + #53 merged; lane transferred)
- Operator transferred the remaining lane to session B ("continue to metis#51, #53, #45").
- **#51 (PR #38)** and **#53 (PR #39)** shipped — details in their task rows. Both close
  reviews caught real defects pre-merge (the aggregate-key rendering bug; the batch-poisoning
  refusal lie) — the boundary-review gate keeps paying for itself.
- **ONE gate item remains: #45** (partial inner CV — decided (a)-first: the `inner_k` split
  ships now, the racing sampler files as a demand-driven follow-up issue). Then metis-v2
  closes.

### 2026-07-17 (close-gate item 6/9 — metis#24 closed; SESSION-B LANE COMPLETE)
- **metis#24 closed decision-complete** (docs-only, direct push, SHIP): trade-off recorded in
  atlas; hit-map printout deferred by decision (demand-driven, next competition). **Session B's
  platform tranche is done (#25 → #34 → #24). Gate remaining: #51, #53, #45 — the other
  session's lane. When those close, metis-v2 closes** (flip `status:` here, closing Log entry,
  next-arena decision).

### 2026-07-17 (close-gate item 5/9 — metis#34 merged; session-B lane)
- **metis#34 shipped via PR #37** (SHIP; est 0.38 / actual 0.35 labeled-judgment). The audit
  found run/select already cwd-invariant (content-addressed identity + exp-path anchors) — the
  deliverable became the regression net pinning that invariant plus the two genuine drift fixes
  (steppath bare-repo fallback; `kaggle submit -C`). Session-B lane remaining: **#24 only**
  (decision-record + the pre-run cache-hit map question). Other lane: #51/#53/#45.

### 2026-07-17 (close-gate item 4/9 — metis#25 merged; session-B lane)
- **metis#25 shipped via PR #36** (close review SHIP; est 0.47 / actual 0.9 labeled-judgment).
  External-ingest identity = declared content pins (fixed-output model) riding `with → Kpre`;
  the operator-facing consequence is in the RUNBOOK: editing a pin ⇒ full cold run + a new
  ledger cohort, by design. Gate remaining: #34 (claimed next, session B) · #24 (session B) ·
  #51/#53/#45 (other session's lane).

### 2026-07-17 (lane split — two live sessions)
Operator split the remaining gate across the two active sessions: **session B (this entry's
author) takes the platform tranche #24/#25/#34** (metis#25 claimed → working; #34/#24 queued
behind it, worked sequentially in an isolated worktree); the other session's lane is the UX
filings **#51/#53/#45**. Neither session claims into the other's lane.

### 2026-07-16 (RESCOPE — operator verdict; the close gate defined)

**done_when is MET** — twice over: the honest nested-CV estimate actuates family selection
(metis#32's rule declined the higher-variance GBM both times), and the shipped public tracks the
honest estimate (cohort b7aee3de → 0.78947-family results; cohort ee3d36bf, the first full k=10
grid, honest pick rf md4/n200+WCG 0.8384±0.0072 → live-submitted `best-rf-6dde4f89` → public
**0.78947**, identical to kbench#9's WCG submission — the k10 selection CONVERGED on the same
config). Pure-ML Titanic plateau ≈ 0.78–0.79 reached per the research ground truth; everything
above ~0.80 runs through explicit rule expression, which the operator **ruled out on principle**
(the workbench's learned intelligence is the point, not the leaderboard number).

**Operator rescope verdict** (against the full open-item inventory):
- **PUNT metis#22 (M4b)** — the gated rule+residual serves the excluded rule tier; plain
  stacking already downgraded. Status open→punt + Revisions entry (reopen trigger recorded).
- **PUNT kbench#11** — the diagnostic that priced #22; no consumer without it. Same treatment.
- **metis#33 stays OPEN but OUT of v2's close gate** — GBM overfit / effective-complexity work,
  likely picked up in the next competition.
- **CLOSE GATE — work these, then close metis-v2** (9 issues): the four project-listed
  cross-cutting items **#24** (cache addressing) · **#25** (get-data root-hash gap) · **#34**
  (repo-root-relative path key) · **#43** (depth-first scheduling), PLUS the five 2026-07-16
  live-testing filings adopted into v2 (they came from v2's own operator UX testing): **#48**
  (default BLAS pins — first, fully specced) · **#49** (board readability; pairs with #43) ·
  **#51** (point_addr column) · **#53** (promote guard) · **#45** (partial inner CV).
- **Explicitly NOT adopted** (stay in their trackers, no v2 obligation): the v3 design track
  (#36 channel split, #37 constructor algebra, #40 select-skill), old metis backlog (#4, #5,
  #10, #26, #28, #29), kaggle #4/#6/#7, kbench#2.

Suggested build order (from the wrap-up continuation): #48 → #43+#49 together (#43 changes what
#49's board should show) → #51/#53/#45 → #24/#25/#34 in any order. Engineering happens in the
metis repo per charter; this file tracks the portfolio. When the 9 close, flip `status:` to done
here and decide the next arena (candidate framing: second Kaggle competition through the
workbench, zero new workbench features until the competition demands them — the v1→v2 pattern).

### 2026-07-07
- Created from a design conversation (operator) that started at "how do we improve Titanic further" and
  converged on: the bottleneck is the cv→public overfit gap, and the fix is a real experiment-design
  algebra (resampling + selection as first-class axes) — not more knobs. Pensive written (`sources`);
  milestones + issues below.

### 2026-07-07 (design converged)
- Long design conversation + a 3-front prior-art survey (ML frameworks · config/sweep/adaptive · caching)
  converged M1. Model: **driver · sweeper · pipeline** over a three-phase shape (`data│pipeline│ship`);
  the sweeper is a black box owning inner-CV + select; the driver is the outer honest evaluator
  (nested-CV). Dropped `over:`/`fit_scope` (cross-fold safety is structural; target-safety is the step's
  own cross-fit). Cache leans input-addressed (metis#24); get-data root-hash gap (metis#25) filed. mlr3 is
  the structural twin; 1-SE selection is the uncontested differentiator. Split M1 → M1a (#18, substrate)
  + M1b (#23, nested-CV). Full design + reshaped titanic-sweep.md in the pensive.

### 2026-07-09 (M2 / metis#19 DONE — the differentiator, verified)
- **M2 (select rule) shipped** across 2 milestone boundaries (M1 select machinery + M2 measured
  complexity), full SDLC (brainstorm → spec ×2 reviews → RF-complexity literature pass → durable plan
  ×1 review → change-code → M1 build+review FIX-THEN-SHIP → M2 build → close). **A mid-design pivot
  worth recording:** the first spec declared complexity per-knob in CUE (`{form,basis}`, `2^depth`); a
  fresh-eyes review traced it over the real ledger and found it shipped an unvalidated sparse corner
  (md=4/nfeat=1), not the 0.782 config. Operator reframed → **complexity is MEASURED on the fitted
  model** (rf realized leaves, logreg coef count), which the RF-complexity literature independently
  endorsed (`2^depth` overstates; n_estimators-neutral; cross-family param-count unsound). That
  collapsed three layers of machinery into one measured scalar and fixed the corner for the right
  reason. **VERIFIED**: `pct-loss` recovers rf md=4/6-feature (cx 14.6, public 0.782) over argmax-mean's
  md=8 (cx 66.3, public 0.770) — the honesty differentiator, empirically confirmed over the real
  ledger. est 3.7h / actual 6.15h. The per-family leaderboard is the seam #22 (ensembling) + #23
  (nested-CV) build on.

### 2026-07-11 (M4a / metis#21 GBM DONE — third model family)
- **GBM branch shipped**, full SDLC (claim → start-plan → recon subagent → **boosting-complexity
  literature pass** → spec+estimate → change-code [plan+estimate judges INFO] → TDD → close [review
  **SHIP**]). Python-only (3 touch points); the Go layer's structural `FamilyOf` picked up `hist_gbm`
  with zero edits. **Complexity decision (the one real call):** total realized leaves *summed* across
  boosted trees — the deliberate inverse of rf's *mean*-per-tree, because boosting is **additive**
  (F=Σ trees) so capacity sums and more rounds overfit. Grounded in a literature pass (ESL §10.2/10.12,
  Friedman 2001, Bühlmann–Hothorn df(m)=trace(𝐁ₘ)↑m, XGBoost's Ω=γT) — mirroring the #19 RF-complexity
  literature pass. The literature also surfaced the **learning_rate-shrinkage caveat** (leaf-count
  decouples from effective DoF across ν), contained *structurally* by fixing ν in the baseline shape
  (fixed-ν stratum) rather than an unvalidated ν-weighted measure (measure-before-rebuild; ν-general
  model stays sweepable). est 0.6h / actual 0.62h (1.0× — the estimate-derivation gate caught a 1.5h
  gut-guess and the honest 0.6 landed on the nose). #22 (blend logreg+rf+gbm) is now unblocked.

### 2026-07-12 (M1b / metis#23 nested-CV DONE — the honesty axis)
- **The keystone honest-reporting capability shipped** (operator prioritized it: "more real reporting…
  one of the key goals"). Full SDLC: claim → 2 recons (nested-CV architecture + read-trace) → design →
  durable plan → fresh-eyes plan review (1 Critical + 4 Important, all fixed) → change-code → M1 + M2
  (fork-implemented, main-owned boundaries), both FIX-THEN-SHIP + fixed. est 3.1h / actual 2.75h (0.89×).
- **The design the operator shaped:** structural separation (L1, `outer-split` subset dirs) **+
  trace-enforced read-confinement** (L2, `METIS_READ_ROOT` at the `exp_path` chokepoint) — leakage is
  unrepresentable AND verified, not just hoped. A key finding surfaced by review: the existing read-trace
  is a *code* closure (data-blind), so confinement lives at the `metis.io` data chokepoint (covers
  parquet); and analysis dirs must be referenced **exp-relative** or the read bypasses the chokepoint.
- **The honest split:** only *selection* is sealed; *scoring* the chosen winner is a fold-expressed
  held-out eval (no leakage post-selection; `cv_folds` determinism reproduces the partition — no extra
  wiring). Bound assumption (honest while features stateless) stated, not left to lie; #20's fold-safe
  features are inherited automatically. **M1's sealing spine is the shared foundation #20 + kbench#8 ride on.**
- **The confinement is REAL, not reasoned:** a real-subprocess test drives execStep → uv `cv-split` →
  `exp_path` and catches an out-of-root read (within-root succeeds). `GuardComplexity` runs per outer
  fold too (a parsimony-select + non-reporting-model shape is rejected, not silently mis-selected).

### 2026-07-13 (M5 kbench#8 MERGED + rf-ticket SHIPPED → public 0.79186 + done_when revised)
- **kbench#8 merged to main** (M5, ticket-group feature). Re-close was needed first: the post-close
  `metis promote` artifact (`titanic-winner-v3.2.md`) tripped the publish gate's reviewed-HEAD-unchanged
  invariant; re-review of the 1-file delta returned SHIP. PR #8 merged server-side, issue archived.
- **Cheap-win #2 — recover the number.** Shipped the honest generalizer the inner-CV cross-family
  argmax passed over: `rf md=4, n_est=500` + `ticket_survival` (`kbench titanic-winner-rf.md`) →
  **public 0.79186**. Beats v1 (0.77990) and the GBM overfitter (0.749, +0.043). Confirmed empirically:
  **(1)** rf > GBM at the cross-family seam (the direct case for #32 — an honest family selector picks
  the generalizer); **(2)** ticket's public value ≈ +0.01 over pre-ticket rf (kbench#8 held OOS);
  **(3)** inner→public gap halved (0.048 vs 0.097). Hand-authored the winner (`promote --point` can't
  select a list-valued free-param — folded into the #22 `promote --family` work).
- **done_when revised** (operator-directed): the honesty test now requires the honest estimate to
  **actuate selection** (#32), not merely report (#23). An honest *estimator* is necessary but not
  sufficient — the honest-beat proved inner-CV argmax still ships the overfitter. Evidence bound into the
  clause (0.749 / 0.79186), not left aspirational.
- **Sequence chosen by operator:** cheap wins (this) → **metis#31 (parallel batch executor)** →
  metis#32 (outer-CV family selection). #31 first so the 2,475-fold `driver:cv` nested runs #32 lives on
  are bearable to iterate.

### 2026-07-16 (close-gate item 1/9 — metis#48 merged)
- **metis#48 shipped via PR #33** (est 0.96h / actual 0.71h, 1.4×). The bare-run BLAS
  oversubscription footgun is now prevented at both executor seams, while explicit operator env wins.
  Fresh `go test ./... -race` passed and the prior real-data smoke remains the behavioral proof:
  720/720 folds in 1m23s with one default-pin note. Boundary review FIX-THEN-SHIP found no code defect;
  its one traceability finding was fixed by recording the downstream kbench RUNBOOK commit. Next close
  gate item remains the paired metis#43 + metis#49 scheduler/readability work.

### 2026-07-17 (close-gate item 2/9 — metis#43 merged)
- **metis#43 shipped via PR #34** (est 4.74h / actual 4.70h, 1.0×; close review SHIP). Whole-run
  admission removes the cold phase-wave starvation while preserving deterministic artifacts and an
  independent leaf cap; cancellation is experiment-wide through sampler, persistence, and TUI
  consumers. The disposable smoke pinned Ariadne and completed seven real trains in 45s, with the
  first train before the fifth admission. Next is **metis#49**, which builds truthful board activity,
  clearer counter vocabulary, smoothed occupancy, and confidence-gated ETA on the stabilized schedule.
