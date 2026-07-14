---
id: 000032
status: working
deps: [metis#23]
github_issue:
created: 2026-07-13
updated: 2026-07-13
estimate_hours:
started: 2026-07-13T15:52:26-07:00
---

# outer-CV model-family selection — close the loop (nested-CV selects, not just reports)

## Problem

**metis#23's nested CV (`driver: cv`) is a passive reporter — it produces an honest estimate but ships
NO winner, so the outer CV isn't used in any automated fashion.** That wastes exactly the signal we
need. Demonstrated empirically on the real Titanic honest-beat run (kbench#8):

- The sweeper's cross-family pick is inner-CV **argmax-mean** (metis#19's parsimony is intra-family only).
- Inner CV shipped `hist_gbm [title,family]` (mean **0.846**, cx 1500) → **public 0.749** — a ~0.10 overfit
  gap. The rf robust winner `[title…embarked, ticket_survival]` md=4 (mean 0.839, cx 14.3) would generalize
  far better (that family ~0.78 public historically).
- The outer CV *holds the signal that GBM overfits* (its honest estimate drops toward ~0.75 while rf's
  holds) — and we throw it away. The cross-family choice is made on the optimistic inner CV instead.

## Spec

*(Design converged via brainstorm 2026-07-13. The **principle** below survives from the original capture;
the **mechanism** evolved twice — from "a driver mode that selects+ships inline" → run/select separation →
the final **derived-run-mode** model here. See the [[## Log]] deltas.)*

**The principle (two-level selection, each signal used where it's valid):**
- **Within a family** → inner CV + parsimony (metis#19). Fine choice (dozens of hyperparams); complexity
  discriminates here, where its units are comparable.
- **Across families** → the **outer/nested honest estimate** (metis#23). Coarse choice (few classes).
  **Outer-CV accuracy is cross-family COMPARABLE** (held-out accuracy, same units) — which complexity is
  NOT (14 leaves vs 1500 leaves vs 5 coefs). So the outer CV is the correct cross-family signal.
  **Subsumes** the cross-family-argmax problem AND the (unsound) cross-family-complexity idea.

**Statistical spine — granularity.** Inner-CV selection overfits because it's a FINE choice (best of ~99)
on an OPTIMISTIC estimate; family selection is a COARSE choice (best of ~3) on an HONEST estimate → small
selection bias on a low-dim choice. Fine choice on the cheap estimate; coarse choice on the honest one.

### Mechanism — three commands, a derived run mode, one shared run engine

`run` **measures** and populates a ledger; `select` **chooses** (and, with `--promote`, ships); `kaggle
submit` **uploads**. The outer CV is a pure *measure* of ~3 per-family procedures, **never a config sweep**
(outer-scoring all configs + selecting on them would be "yet another sweep" on the held-out fold →
dishonest; it reintroduces the optimism #23 removed).

**1. `metis run <shape>` — MEASURE (never ships). The mode is DERIVED from the shape, not declared:**

| Shape | `metis run` does | records |
|---|---|---|
| **sweep** (has `$any` free vars) | **nested CV** (outer × inner; outer reuses `sweeper.resample.cv`) | inner + outer rows |
| **no free vars**, has a resample | **single-level CV** — the outer selection loop has one candidate, so nested-CV mathematically *degenerates* to a plain k-fold of that config (no special-casing) | inner rows |
| **bare experiment** (no sweeper) | run the steps as-is | — |

The nested run **records the whole thing to the ledger** (today it records NOTHING): per-`(outer-fold,
config)` **inner-CV** rows AND per-`(outer-fold, family)` **outer-CV** rows (each = that fold's family
inner-winner + its outer-held-out score), **level/fold-marked so the two never collide**. Family is
DERIVED from the recorded `train.model` free-param (`FamilyOf`) — no new family column. Per-family honest
estimate = `mean ± SE` over the outer folds of that family's outer scores.

**2. `metis select <shape> [--best | --best-per-model-class] [--promote]`** — new top-level command;
**retires `metis ledger select` + `metis promote`**.
- Reduces the ledger: **family** from the outer rows, **config-within-family** from the inner rows.
- **Family rule:** among families whose honest mean is within **1 SE of the best family's mean**, pick the
  **lowest-SE** one (documented tiebreak; not over-designed — revisit only if real runs show families
  tying). NOT a static family-complexity order — class complexity isn't statically orderable (a
  regularized gbm can use fewer effective params than a deep rf), so a static order just smuggles
  cross-family complexity comparison back in.
- **Config rule (within family):** inner CV + `sweeper.objective.select` (the metis#19 rule, e.g. pct-loss).
- **`--best`** → single ship rec. **`--best-per-model-class`** → one pick per family (the metis#22 seam).
- **Dry (no `--promote`):** print the selected config(s) + per-family honest `mean±SE` + the honesty caveat.
- **`--promote`:** for each selected config, **reconstruct it from the ledger** (never a pre-materialized
  winner) and run it in **ship mode — `data + pipeline + ship`, fit on ALL data, no CV → `submission.csv`**,
  into `runs/best-{family}-{hash}/`. **Prints the run ids** (the handles for `kaggle submit`). Config always
  reconstructed from the ledger → no committed winner `.md`, no `--point` selector (retires that friction).

**3. `kaggle submit --run best-{family}-{hash}`** → uploads that run's `submission.csv` (+ slug from
`record.json`). Never runs the pipeline.

**One shared run engine, two assemblies (ARCH-DRY).** Both `metis run`'s measure and `--promote`'s ship
funnel through the SAME per-experiment runner (`runResolvedExperiment` — cache, step executor,
record/run.json), differing only in how the fixed-config experiment is *assembled*: measure = `data +
pipeline + cv-split` (fold-scored); ship = `data + pipeline + ship`, all-data fit. No second engine → the
cache/provenance/determinism guarantees are identical on both paths.

**Shape simplification.** The `driver:` block is **deleted** (the run mode is derived; the outer loop
reuses `sweeper.resample.cv`). `data` / `pipeline` / `ship` / `sweeper` all stay. `ship` is **latent
during measurement** and activated only by `select --promote` (which reconstructs the winner *from this
shape* and needs its `ship` steps) — so it lives in the sweep shape but `metis run` never runs it.

**Workflow:**
```
metis run titanic-sweep.md                              # nested CV → ledger (inner + outer rows)
metis select titanic-sweep.md --best                    # dry: print the pick + honest mean±SE + caveat
metis select titanic-sweep.md --best-per-model-class --promote   # reconstruct + ship each on ALL data →
                                                        #   runs/best-{family}-{hash}/submission.csv (prints ids)
kaggle submit --run best-rf-{hash}
```
metis#31's parallelism makes the nested `run` affordable — the reason it was sequenced first.

**Honesty caveat (reported, not hidden).** The *selected family's* honest estimate is mildly optimistic — a
1-SE pick over ~3 families, ~0.01 upward bias (NOT a per-config claim — the shipped config has no outer
estimate of its own; only the family/procedure does). A large honesty gain over the inner-CV path's ~0.10
gap, not a perfectly-unbiased number (that needs another outer loop; not worth it at ~3 families). `select`
surfaces the caveat alongside the number.

**Join soundness.** The family (outer rows) ↔ config (inner rows) reduction MUST be pinned to one
`code_fingerprint` — an unscoped reduce over a mixed-code ledger silently blends versions (the
`workshop/lessons.md` footgun). `select` errors sharply if the ledger lacks the rows it needs (e.g. a
non-sweep ledger with no outer rows) rather than nil-deref.

**Scope.** IN #32: the nested-run ledger recording (inner+outer, level-marked); the derived run mode +
`driver:`-drop; `metis select` (the select+promote merge, reconstruct-and-ship); the family rule; the
caveat. OUT (separate issues): the **repo-root-relative path key** (metis#34). COMPOSES with: metis#19
(intra-family, unchanged), metis#23 (the estimator this extends + records), metis#22 (consumes
`--best-per-model-class`), metis#33 (GBM's own overfit — orthogonal).

## Done when

- **`metis run` on a sweep records the full nested CV to the ledger** — per-`(outer-fold, config)` inner-CV
  rows + per-`(outer-fold, family)` outer-CV rows, level/fold-marked so inner and outer rows for the same
  config+fold don't collide (today the nested path records nothing). A no-free-var shape degenerates to a
  single-level CV automatically (hermetic test).
- **`metis select <shape>`** reduces those → per-family honest `mean±SE`; **`--best`** applies the
  lowest-SE-within-1-SE family rule + inner-CV config choice; **`--best-per-model-class`** reports per family;
  **`--promote`** reconstructs each config from the ledger and ships it on ALL data (`runs/best-{family}-{hash}/`),
  printing the run ids; **retires `metis ledger select` + `metis promote`**; the family↔config join is
  fingerprint-scoped and errors sharply when required rows are absent.
- **The `driver:` block is removed** from the shape schema; the run mode is derived; `ship` is retained and
  activated by `--promote`.
- On a fixture cv+inner ledger, `select --best` picks the **rf** family over the **gbm** overfitter
  (hermetic gate). *(The real-Kaggle numbers — GBM 0.749 vs rf 0.79186 — are recorded evidence, not a
  runnable test.)*
- The max-over-families honesty caveat is surfaced in the `select` report (as the *family's* estimate).

## Plan

- [x] **Brainstorm** (2026-07-13, operator) — converged the mechanism (see Log): derived run mode (nested
  for sweeps, single-level CV degenerate), one nested run recording inner+outer, `metis select`
  reconstruct-and-ship on all data, `driver:` dropped, one shared run engine. Spec above.
- [ ] Durable plan via `superpowers-writing-plans` → `workshop/plans/000032-*-plan.md`, then `change-code`.

## Log

### 2026-07-13
- Filed from the kbench#8 honest-beat run (operator). THE headline metis-v2 capability — the honest
  estimate STEERING, not just reporting (closes the loop metis#23 left open; deps it). Empirically:
  inner-CV cross-family argmax shipped a GBM overfitter (0.846 inner → 0.749 public) over the rf-ticket
  generalizer. **Subsumes** the earlier "cross-family selection robustness" + "GBM cross-family complexity"
  ideas — the outer CV is the comparable cross-family signal, so no complexity hack is needed there.
  Sibling: metis#33 (GBM's own overfit + intra-family complexity), metis#31 (parallelism to afford the
  nested run), metis#30 (progress). Brainstorm-first — it extends the outer-node contract.
- **Brainstorm 2026-07-13 (operator) — mechanism refined; principle unchanged.** The original capture put
  selection INSIDE a driver mode (the outer node's `Tell/Done` selects+ships inline). The operator reframed
  to **run/select separation**: `run` populates a ledger, choosing is a read-time op over it. Deltas from
  the original spec:
  - **`driver: cv` records** per-`(outer-fold, family)` honest rows (it recorded nothing before); the outer
    node does NOT select — it stays a pure honest *measure* (operator: "the outer is only meant to provide
    honest measure, not yet another sweep"). Outer-scoring all configs + selecting on them = dishonest.
  - **`metis select` merges select+promote** (retires `ledger select` + `promote`); `--promote` materializes
    all-data runs named `best-{family}-{hash}` and prints the ids for `kaggle submit --run`.
  - **Family tie-break = lowest-SE-within-1-SE**, NOT "simplest class" — the operator killed a static
    family-complexity order as unsound (effective params vary; it re-smuggles cross-family complexity). SE
    also penalizes inner-selection instability for free.
  - **Two-run workflow** (single → config ledger + ship; cv → family estimates), ship on all data.
  - **Split out:** the repo-root-relative path key → its own issue (cwd-ergonomics, orthogonal). *(Filed as metis#34.)*
- **Brainstorm 2026-07-13 (operator) — second evolution: DERIVED run mode; supersedes the two-run workflow.**
  The spec review (fresh-eyes) surfaced C1: `driver:` is a shape field with an "exactly one of single|cv"
  validator + no `--driver` flag, so "run under two drivers" means editing the .md → changes its blob-hash →
  shifts all point-addresses + breaks `best-{hash}` naming. Resolving it collapsed the whole model:
  - **ONE `metis run` does the nested CV** and records BOTH inner (per config) + outer (per family) rows —
    no separate `driver: single` run. The two-run workflow is retired.
  - **Run mode is DERIVED, not declared:** sweep (free vars) → nested CV; no free vars → single-level CV (the
    outer selection loop has one candidate, so nested-CV mathematically *degenerates* — no special case);
    bare experiment → run as-is. **The `driver:` block is deleted**; the outer loop reuses `sweeper.resample.cv`.
    This dissolves C1 (no driver toggle → blob constant by construction) — the named-drivers machinery was
    solving a problem the derived mode designs away.
  - **`select --promote` reconstructs the config from the ledger** and runs it in ship mode (all data, no CV,
    → submission) — the winner is NEVER pre-materialized. `metis run` measures + `select --promote` ships
    funnel through the SAME run engine (`runResolvedExperiment`), two experiment assemblies (ARCH-DRY).
  - **`ship:` stays** (latent during measure; activated only by `--promote`, which reconstructs the winner
    from the shape and needs its predict/submission steps). Only `driver:` is obsolete.
  - **Verbs:** `metis run` (kept, not `metis sweep`) / `metis select` / `kaggle submit` — three commands.
  - Spec `## Spec` rewritten to this model; the C2 ledger extension (inner+outer rows, level-marked) is the
    one substantial new backend piece — everything else reassembles existing parts.
