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

*(Refined via brainstorm 2026-07-13 — the principle below survives from the original capture; the
mechanism changed from "a driver mode that selects+ships inline" to run/select separation. See the
[[## Log]] delta.)*

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

**Mechanism — `run` records the honest measure; `select` steers (run/select separation).** NOT a driver
mode that selects+ships inline. `run` populates ledgers; choosing is a **read-time** operation over the
recorded ledgers (consistent with today's inner-CV `ledger show`/`select`):

1. **`driver: cv` (the metis#23 nested CV, extended to record).** Today it outer-scores only the single
   overall inner-winner and records nothing durable. #32 changes it to outer-score **each family's
   inner-winner** and **append per-`(outer-fold, family)` rows to the shape ledger** — level-marked
   (distinct from `driver: single` inner rows), each carrying: family, that fold's inner-winner
   free-params, its inner-CV mean, its **outer-held-out score**. **The outer CV stays a *measure* of ~3
   per-family procedures, never a config sweep** — outer-scoring all configs and selecting among them
   would be "yet another sweep" on the held-out fold → dishonest (reintroduces the optimism #23 removed).
   Per-family honest estimate = `mean ± SE` over the outer folds of that family's outer scores.
2. **`driver: single` (today's).** The all-data inner-CV config ledger + the ship source.

3. **`metis select <shape.md> [--best | --best-per-model-class] [--promote]`** — new top-level command;
   **retires `metis ledger select` + `metis promote`**.
   - Reduces the `cv` rows → per-family honest estimate (mean±SE).
   - **Family rule (the cross-family select):** among families whose honest mean is within **1 SE of the
     best family's mean**, pick the **lowest-SE** one (most stable outer estimate). NOT a static
     family-complexity order — class complexity isn't statically orderable (a regularized gbm can use
     fewer effective params than a deep rf), so a static order just smuggles cross-family complexity
     comparison back in. The SE also folds in **inner-selection instability** (a family whose inner pick
     jumps fold-to-fold has a wider outer spread → loses the tie) — a free, honest fragility penalty.
   - **Config within the family:** from the inner CV (the `driver: single` ledger, metis#19 rule).
   - **`--best`** → the single ship recommendation (family rule → its inner config). **`--best-per-model-class`**
     → each family's honest estimate + its ship config (the metis#22 ensembling seam).
   - **`--promote`** → materialize each selected config as a **run on ALL data** (the `driver: single`
     refit — no outer holdout), producing `runs/best-{family}-{shorthash}/…/submission.csv` + `record.json`;
     run id = `best-{shorthash}` / `best-{family}-{shorthash}` (hash = the config's content-address).
     **Prints the generated run ids** — the handles for `kaggle submit --run <id>`. Without `--promote`:
     reports what it would pick (dry view). (Also retires the hand-written-winner + `--point`-selector
     friction hit on the rf-ticket ship.)
4. **`kaggle submit --run best-{family}-{shorthash}`** → submits (submission.csv + slug from record.json).

**Workflow (two runs, then select):**
```
metis run sweep.md                  # driver: single → inner-CV config ledger + ship source
metis run sweep.md   (driver: cv)   # → per-family honest outer estimates in the ledger
metis select sweep.md --best --promote    # family by lowest-SE-within-1-SE on the honest estimate,
                                           # config by inner CV, materialized as best-{family}-{hash}
kaggle submit --run best-{family}-{hash}
```
metis#31's parallelism makes the `driver: cv` run affordable — the reason it was sequenced first.

**Honesty caveat (reported, not hidden).** The *selected* family's estimate is mildly optimistic — a
1-SE pick over ~3 families, ~0.01 upward bias. A large honesty gain over the inner-CV path's ~0.10 gap,
not a perfectly-unbiased number (that needs another outer loop; not worth it at ~3 families). `select`
surfaces the caveat alongside the number.

**Scope.** IN #32: the `driver: cv` per-family ledger recording; `metis select` (the select+promote
merge); the family rule; the caveat in the report. OUT (separate issues): the **repo-root-relative path
key** (cwd-ergonomics — own issue). COMPOSES with: metis#19 (intra-family, unchanged), metis#23 (the
estimator this extends), metis#22 (consumes `--best-per-model-class`), metis#33 (GBM's own overfit — orthogonal).

## Done when

- **`driver: cv` records** per-`(outer-fold, family)` honest rows to the shape ledger (family inner-winner
  free-params + inner-CV mean + outer-held-out score), level-marked distinct from `driver: single` rows.
- **`metis select <shape>`** reduces those → per-family honest estimate; **`--best`** applies the
  lowest-SE-within-1-SE family rule + inner-CV config choice → the ship rec; **`--best-per-model-class`**
  reports per family; **`--promote`** materializes all-data runs (`best-{family}-{hash}`) and prints the run
  ids; **retires `metis ledger select` + `metis promote`**.
- On the Titanic honest-beat run, `select --best` ships the **rf** family (not the GBM overfitter), and the
  shipped config's outer estimate tracks its public score far better than the inner-CV path did (which
  shipped GBM 0.846-inner → 0.749-public; the rf generalizer scored public 0.79186).
- The max-over-families honesty caveat is surfaced in the `select` report, not hidden.

## Plan

- [x] **Brainstorm** (2026-07-13, operator) — resolved the mechanism: run/select separation (not a
  select+ship driver mode), `metis select` merging select+promote, lowest-SE-within-1-SE family rule,
  two-run workflow, all-data ship. Spec written above.
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
  - **Split out:** the repo-root-relative path key → its own issue (cwd-ergonomics, orthogonal).
