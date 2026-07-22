---
name: ml-research
description: "Use when running an exploratory ML research investigation — a Kaggle competition or any modeling problem whose only feedback is a soft metric (RMSE/AUC), not compile/test. The workflow that manufactures a hard, local grounding signal from soft-global feedback: tell a real dead-end from a bug/tuning/framing failure, keep a rigorous hypothesis (arrows) ledger, and extract reusable mechanisms into metis/metisser. Prose-first (the metisser binary is not built yet). Triggers: work the competition, why can't we beat the baseline, is this signal real, should we mark this end dead, rebuild rogii, starting or resuming a modeling investigation."
version: 0.1-prose
status: forming
---

# ML Research Workflow

You are doing research where **a bad number has four indistinguishable causes — bug · tuning · framing ·
real-negative — that all look identical.** In ariadne/SDLC the compiler grounds you: failure points at itself.
Here nothing does. **This workflow is the substitute for the compiler:** it manufactures hard-local signals out
of soft-global ones so your conclusions are trustworthy and your dead-ends are real.

**Dual purpose — hold both, always.** (1) Push the leaderboard/metric. (2) **Extract reusable mechanisms into
metis/`metisser`** so the next investigation is more structured. (2) is not secondary — a win you can't
generalize is half a win. When a step here feels like a repeated manual ritual, that's a `metisser` subcommand
waiting to be earned; note it, keep going in prose.

---

## Principle #1 — no number without a second, independent measurement

**This is the top rule. It applies to every number in every stage below.** The "second measurement" is the
nearest **independent constraint** the quantity must satisfy. A number is **implausible if it violates any** —
and you can almost always afford *two*. The six families:

1. **Sandwich** — `baseline-floor ≤ result ≤ oracle-ceiling`. Below floor → bug/absent. **Above ceiling → you're
   leaking.**
2. **Identity / conservation** — an exact law to tolerance. *Cheapest, highest-value class.* (rogii: within-well
   `a` std must be ≈0 — that one check IS the `geo_surface`-bug detector; it was 662 ft.)
3. **Invariance** — must not move under an irrelevant perturbation (scramble the labels you're forbidden to see →
   prediction unchanged; same seed → same number).
4. **Cross-method agreement** — the *same* quantity computed two structurally different ways (module vs probe to
   0.001; kriging vs local-plane).
5. **Degenerate / known-answer** — a special case with a known value (untrained residual model == baseline;
   synthetic recovery).
6. **Dose–response** — right *direction and shape* as a knob turns (beat grows with data; a bias-variance knob
   *plateaus*, doesn't spike).

**Recipe** for any learned `f()`: answer six questions — *trivial baseline? · perfect-info ceiling? · what must
it NOT depend on? · what exact identity relates its I/O? · what case do I know the answer to? · which knob moves
it predictably?* You won't get all six cheaply; get **two**.

---

## The loop

```
0 charter → 1 framing+baseline → 2 arrows(hypothesize+ORACLE) → 3 assemble → 4 train → 5 honest-CV → 6 adjudicate+record
                                        ↑___________________________________________________________________|
```

### 0 · Charter — understand the domain (`framing.md` §charter)
What IS this problem? (geosteering; health-risk factors.) What data exists **train vs test** — and critically
**what is train-only** (rogii: markers, typewell, heel labels)? The metric, and the eval regime (extrapolation?
grouped? held-out region?). Keep it concise — it orients everything. Train-only info is where oracles live (§2).

### 1 · Framing + baseline (`framing.md` §framing)
Model **input** (encodings mostly known) and — the interesting part — **output shape**: autoregressive vs
joint-over-positions, target parameterization (rogii: predict `TVT` directly, or the surface `S=TVT+Z`, or a
persistence *residual*?). Then **establish the baseline** — the trivial-but-strong thing to beat (rogii:
persistence) — and **measure it under honest CV FIRST**. Every later result is reported *paired vs this baseline*.

### 2 · Arrows — hypothesize, then ORACLE before you build
An **arrow** is an idea to extract signal: a falsifiable A→B link. **Before building anything:**
- **Oracle the load-bearing link.** Place the inputs at ground truth and measure: *is the signal there AND does
  it localize/exploit?* (rogii GR: oracle corr +0.5 at true TVT → signal real; but equally good at wrong TVTs →
  **doesn't localize** → not exploitable. Knowing this BEFORE building five aligners would have saved the thread.)
- **Chain oracle** for A→B→C: test *does true B determine C* first. (The formation model died here in hindsight:
  a perfect layer label carries ~0 info about within-layer drift — the load-bearing link was never oracled.)
- Oracle result gates the build: **no signal / not exploitable at the oracle → do not spend model budget.**
- Record the arrow as `open` in `arrows.md` (§ ledger) with its oracle-ceiling.

### 3 · Assemble — baseline-first, weak-models-first
Build the **simplest** predictor on **validated arrows**. Beat the baseline. Weak models before strong. **Gate
the zoo** (see below) — do not reach for architecture/tuning knobs before an arrow is oracle-validated.

### 4 · Train — sanity gates that fail loudly
- **Monitor the loss** — is it converging/progressing? (Long runs: incremental progress reports, always.)
- **Overfit experiment** — can it fit in-sample? (Degenerate check: rules out *broken pipeline / can't-express*.
  rogii: the "ties" were untrained models sitting at their persistence init — the overfit probe proved the model
  *could* fit, retracting a false negative.)
- **Leak assertion** — scramble the forbidden labels; prediction must not move.

### 5 · Honest CV — the verdict
Paired vs baseline, under the **honest CV structure**, as-measured, no laundering. **This is the number that
counts.** metis already owns much of this — extend it, don't rebuild.

**Design the CV to mimic the test regime — the checklist** (what "honest" means, generalized; simple k-fold is
only the trivial case):
- **Holdout UNIT = the test unit.** Hold out whatever the hidden test holds out — a whole group / cluster /
  region, never a row finer than it. A split finer than the test unit leaks (rogii: the *whole well*, not the
  row — adjacent ~1-ft rows are near-identical).
- **Reproduce the distribution SHIFT.** If the test is unseen wells / an unseen region / a future period, the
  fold split must impose the same shift (rogii: *spatial-block* for unseen region — even whole-well CV is
  optimistic because neighbour wells drill the same rock).
- **Mask what the test masks.** Whatever is absent/`NaN` at test, blank it in the CV folds too (rogii: the toe
  `TVT_input`).
- **Assert no leak channel crosses the fold** — target-derived features, group-shared statistics, neighbour
  rows. Use the invariance check (Principle #1): scramble a forbidden signal → the fold score must not move.
- **Report the LADDER, not one number** — the leaky estimate AND the honest one, so the optimism you removed is
  visible (rogii: row ≪ well ≪ spatial-block). Trust the rung that matches the test regime.

### 6 · Adjudicate + record — the impostor ladder before "dead"
A hypothesis cannot be filed **`dead`** until it survives, in order:
1. **Fit check** — could it overfit in-sample? (rules out broken pipeline / can't-express)
2. **Oracle** — does the signal exist at ground truth? (rules out measuring-the-wrong-thing)
3. **Dual measurement** — does an independent recompute agree? (rules out bug)
4. **Held-fixed** — what was frozen when you declared it? (makes the negative falsifiable — "tapped out **at
   K=60**" is not "tapped out")

Only through all four is it a **real negative**. Else it's `deprioritized` (signal present, unexploited — a
priority, not a verdict) or still `open`. Then **write the arrow row** and loop: the ledger's gaps are your next
arrows.

---

## The arrows ledger (`arrows.md`, one per investigation)

**Structured markdown is the single source of truth** (LLM-native, git-native). Relational queries run over a
**throwaway** CSV/duckdb view regenerated from it — **never a second authoritative store** (single-mechanism).
Layout = **index table** (queryable scalars — the CSV-projection surface) + **detail sections** (`### <id>`) for
what a cell can't hold. At ≤~100 arrows/file this greps and pandas-reads instantly; no database.

Index columns: `id · hypothesis · status · oracle · honest(vs-baseline) · belief · provenance · topline-Δ`.
Status ∈ `{open, active, deprioritized, dead, win}`. Each detail section carries:
- **mechanism** — *why* the outcome (durable prose: "GR autocorrelated → broad optimum → can't localize", NOT
  "washed −0.02". A number-only dead arrow gets re-explored.)
- **held-fixed** — `(value, belief-strength, justification-LINK)` per frozen knob. `K=60` = "arbitrary guess, no
  link → HIGH revisit priority"; `Adam default LR` = "literature-backed → [link] → LOW". **No unlinked
  justification** (anti-hallucination; internal = cross-ref to a catalog section, external = URL).
- **revisit-trigger** — the belief-update that reopens it (a swept knob, a forum post, a paper). "Interpolator
  dead → revisit if K/POWER/AR swept" makes the later K-sweep a *scheduled* re-test, not luck.
- **provenance** — every number tagged `ACHIEVED` (LB-confirmed) / `MEASURED` (our CV) / `ORACLE` (ceiling) /
  `CLAIMED` (external, unreproduced). **Strategy may build only on ACHIEVED/MEASURED.** (A `CLAIMED` 5.99 treated
  as `MEASURED` shaped a whole thread falsely.)
- **source rule** — every number carries its tag **AND a source** (trace path / commit sha / LB-submission id;
  carried numbers use `old-run`). A number quoted with **no source** gets a trailing **`†`** = *unsourced,
  revalidate*; **`grep '†'`** is the standing revalidation debt. Reproducing it (with a trace) drops the `†` and
  adds the link. `†` is only for "no source anywhere" — `CLAIMED`/`old-run` mean "sourced, not by us." (Future
  `metisser arrows lint`: assert no number without a source or a `†`.)
- **topline-Δ** — how this arrow moved the headline. Headline results become **prefix-searchable anchors**
  (`~12` → grep `12\.`).

---

## Reproducible probes (the inner-loop reproducibility model)

Exploratory runs are **cheap and one-shot**, so reproduce them by **RE-EXECUTION, not caching** — CAS /
content-addressing is for the expensive, *repeated* outer loop (metis sweeps); caching a 5-second one-shot buys
nothing and fights the freedom to jot. A probe is reproducible when three invariants hold:

1. **Deterministic** — pin every seed; read only pinned inputs; no wall-clock / unseeded randomness. Same code +
   same data → same trace.
2. **Self-contained invocation** — the exact run command (all args, copy-paste) lives in the probe's docstring.
3. **Trace committed WITH the code, atomically** — the `.py`, its same-basename `.log` trace, AND the `arrows.md`
   row it settles land in **one commit**. That commit *is* the pin: `git blame` on any `arrows.md` source-cell
   resolves to the exact commit holding the exact code + trace that produced the number. So the `source` cell
   names a **PATH** (`probes/<name>.py`), never a sha — git supplies the version; inline shas are churn.

**Bar: reproducible CONCLUSIONS, not bit-identical bytes.** A re-run giving 9.962 vs 9.960 still confirms the
finding — chase same-conclusion-on-re-run, not bit-determinism (that's CAS's job, overkill here).

**Reify before you claim.** Jot scratchpad python freely to *look*; a number becomes a ledger fact only when its
probe is committed (else it carries `†`). Scratchpad = looking; committed probe = claiming — the ledger is the
boundary between exploration and record.

**Pin the inputs once** (the one CAS idea worth importing): a workspace-level data-provenance file (source + a
content checksum), so re-runs verify the same inputs — not per-run. If the inputs change, every `MEASURED` number
reverts to `†`.

## Gate the zoo (why this SERVES modeling, not replaces it)

A net is **a refiner of a proposal, not a diviner of absent signal.** An agent facing a bad number reaches for
the cheapest visible action — another architecture, more tuning — which varies *capacity* while a *framing* bug
hides underneath; infinite plausible-looking actions all failing for the same unseen reason. **So gate the model
zoo behind a passed arrow-test (§2 oracle).** Then a washout is *interpretable* ("the arrow was real but this
refiner couldn't exploit it → a genuine modeling gap to push") instead of ambiguous ("arrow? framing? model?").
This is how you discover whether an end was marked dead *wrongly*. Baseline-first; weak before strong;
**ensemble/stacking LAST** — only once ≥2 legs beat baseline AND clear the fusion bar `ρ < σ_strong/σ_weak`.

---

## Worked example — rogii (the ledger this skill is reverse-engineered from)

`brain/.../continuation/20260722T042851-rogii-buda-fix-covered-lever.md` + kbench#24 `## Log` are a fully-
populated arrows-ledger. What each rule caught, in the real case:
- **Oracle-before-negative** — GR "washed" 6 ways, but the oracle (+0.5 at true TVT) proved the signal was real,
  just non-localizing → `deprioritized`, not `dead`.
- **Identity check** — `geo_surface` returned garbage silently; within-well `a` std 662 ft (should be ≈0) was
  the one-line detector that had been missing for a whole thread.
- **Held-fixed / revisit** — "interpolator tapped out" was false; it meant "at K=60". Sweeping K (a never-varied
  knob) moved LB 13.25 → **12.45** (`ACHIEVED`).
- **Provenance** — the `CLAIMED` 5.99 phantom framed "covered is solved" falsely; real covered was 10.7
  (`MEASURED`).
- **Gate the zoo** — five model families (GBM/Viterbi/CNN/cross-attn/GRU) washed because each was fed a
  weak/absent proposal; the win came from getting the *framing* right (cross-well geometry) then tuning one knob.

---

## Status — what's prose vs built

Everything here is **prose you follow by hand.** Nothing is enforced by a binary yet — deliberately (fast
iteration on the process itself; early drift is fine). As a step proves itself on the rogii-v2 rebuild, it earns
a `metisser` subcommand (the arrows projector/query, plausibility asserts, lifecycle verbs) — the metis analog of
ariadne's `sdlc`. **Do not build `metisser` before the prose shape settles.** Companion artifacts: `framing.md`
(§charter + §framing), `arrows.md` (the ledger). Project: `metis/workshop/projects/ml-research-workflow.md`.
