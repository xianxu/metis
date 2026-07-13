---
id: 000032
status: open
deps: [metis#23]
github_issue:
created: 2026-07-13
updated: 2026-07-13
estimate_hours:
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

**Two-level selection, each signal used where it's valid:**
- **Within a family** → inner CV + parsimony (metis#19). Fine-grained (dozens of hyperparams); complexity
  discriminates here, where its units are comparable.
- **Across families** → the **outer/nested honest estimate** (metis#23). Coarse (few model classes).
  Crucially, **outer-CV accuracy is cross-family COMPARABLE** (held-out accuracy, same units for every
  family) — which is exactly what complexity is NOT (14 leaves vs 1500 leaves vs 5 coefs). So the outer CV
  is the correct cross-family generalization signal; complexity was only ever a within-family proxy. **This
  subsumes the cross-family-argmax problem AND the need for a cross-family-comparable complexity measure.**

**Statistical spine — granularity.** Inner-CV selection overfits because it's a FINE choice (best of ~99)
on an OPTIMISTIC estimate. Family selection is a COARSE choice (best of ~3) on an HONEST estimate → small
selection bias on a low-dimensional choice. Select the coarse choice on the honest estimate; the fine
choice on the cheap one (textbook hierarchical model selection). Add a **1-SE-across-families rule** (5
outer folds are noisy, SE ~0.015): among families within 1 SE of the best outer estimate, take the
simplest / lowest-variance one — don't naive-argmax the families either.

**Architecture — it fits the existing `sampler` fold algebra.** The outer node's *reduction* changes from
*aggregate* (today's `driver: cv` → mean±SE, ships nothing) to *select-family-by-honest-estimate* (→ ship
the winning family's inner winner). In `sampler` terms it's a **sweeper at the outer level over families**
— `sweeper(sweeper(resample))` — rather than `driver(sweeper(resample))`: the outer node's `Tell/Done`
selects instead of aggregates. Same ask/tell `Run`. **Nearly free:** nested CV already runs every config on
every outer fold — you just score each family's inner-winner on the outer fold (≈3 scorings vs 1) and reduce
per-family.

**Honesty caveat (state it, don't over-engineer it).** Once the outer CV *selects* the family, the winning
family's reported estimate is mildly optimistic (a max over ~3 families, ~0.01 upward bias). The deliverable
is the ROBUST CHOICE (ship rf, not GBM) + an estimate that's ~0.01 optimistic — a large honesty gain over
the inner-CV path's 0.10 gap, not a perfectly-unbiased number (that needs another outer loop; not worth it
at 3 families). Report the caveat alongside the number.

## Done when

- A `driver` mode selects the model family by the outer/nested honest estimate + a 1-SE-across-families
  rule, and ships that family's (inner-selected) winner — with the per-family outer estimates reported.
- On the Titanic honest-beat run it ships the **rf** family (not the GBM overfitter); the shipped config's
  outer estimate tracks its public score far better than the inner-CV path did.
- The honesty caveat (max-over-families optimism) is surfaced in the report, not hidden.

## Plan

- [ ] **Brainstorm first** (changes the driver/sweeper outer-node contract): the outer selecting-node
  (`sweeper(sweeper(resample))` reduction), the 1-SE-across-families rule, what the shipped artifact +
  report look like, and how it composes with metis#19 (intra-family) + metis#23 (the estimator). Then spec.

## Log

### 2026-07-13
- Filed from the kbench#8 honest-beat run (operator). THE headline metis-v2 capability — the honest
  estimate STEERING, not just reporting (closes the loop metis#23 left open; deps it). Empirically:
  inner-CV cross-family argmax shipped a GBM overfitter (0.846 inner → 0.749 public) over the rf-ticket
  generalizer. **Subsumes** the earlier "cross-family selection robustness" + "GBM cross-family complexity"
  ideas — the outer CV is the comparable cross-family signal, so no complexity hack is needed there.
  Sibling: metis#33 (GBM's own overfit + intra-family complexity), metis#31 (parallelism to afford the
  nested run), metis#30 (progress). Brainstorm-first — it extends the outer-node contract.
