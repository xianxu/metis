---
name: ml-research
description: "Use when running an exploratory ML research investigation — a Kaggle competition or any modeling problem whose only feedback is a soft metric (RMSE/AUC), not compile/test. The workflow that manufactures a hard, local grounding signal from soft-global feedback: tell a real dead-end from a bug/tuning/framing failure, keep a rigorous hypothesis (arrows) ledger, and extract reusable mechanisms into metis/metisser. Prose-first (the metisser binary is not built yet). Triggers: work the competition, why can't we beat the baseline, is this signal real, should we mark this end dead, starting or resuming a modeling investigation."
version: 0.2-prose
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

## The four impostors — the core ML debugging skill

Every disappointing number is one of four things, and **each has its own discriminating instrument.** Running
the wrong instrument (or none) is how investigations rot: an agent facing a bad number reaches for the cheapest
visible action — retune, swap architectures — which tests only impostor #3 while #1/#2/#4 hide underneath.

1. **A bug** — the code doesn't compute what you think it computes.
   *Instrument:* exact checks built INTO the probe that fail loudly — identities (a conserved quantity ≈ 0),
   degenerate cases (zero-strength model must equal the baseline EXACTLY, asserted), nulls (scrambled input
   must score at chance), known-answer calibration (run the detector on synthetic data with a known verdict
   before trusting it on real data). **A probe that has no check that COULD fire cannot rule a bug out.** These
   checks catch real failures constantly — silent-garbage returns, detectors that fail their own calibration,
   proposals killed by their own baseline floor. A result below the trivial floor or above the oracle ceiling
   is a bug (or a leak) until proven otherwise.
2. **A tuning issue** — the mechanism works but a knob is wrong.
   *Instrument:* the **held-fixed ledger + dose-response sweep**. Every verdict records what was frozen when it
   was declared, with a belief-strength per frozen knob; "tapped out" means "tapped out AT these values" until
   each load-bearing knob has a swept axis. The sweep's SHAPE is the finding: an interior optimum on every axis
   → the config is genuinely done, the cap is elsewhere; a slope at the boundary → headroom, follow it. A knob
   chosen by argument instead of measurement, sitting under a load-bearing verdict, is the classic false
   dead-end (a never-swept neighborhood size once hid a full leaderboard step).
3. **A framing issue** — you are predicting the wrong THING, or with the wrong output shape.
   *Instrument:* three, in order. (a) **Oracle the link**: place inputs at ground truth — if the signal exists
   there but the product fails, the failure is framing/tuning downstream, NOT signal absence; isolate which
   link broke. (b) **Match the model class to the measured structure of the target**: measure what the labels
   actually ARE (piecewise? smooth? segmental? the output of an external process you can reverse-engineer?)
   and check your output parameterization against it — a Gaussian-walk state tracking labels that are measured
   piecewise-constant-dip segments is a framing mismatch no amount of tuning fixes. (c) **The
   findings-to-framing loop** (Principle #1c) — the discipline that keeps (b) from being skipped.
4. **A real dead end** — the signal genuinely isn't there / isn't exploitable.
   *Instrument:* the **impostor ladder** — this verdict is only earnable AFTER the other three are excluded:
   fit-check (could it overfit in-sample? → rules out broken pipeline) → oracle (does the signal exist at
   ground truth? → rules out wrong-thing-measured) → dual-measure (independent recompute agrees? → rules out
   bug) → held-fixed (every load-bearing knob recorded, swept or belief-justified? → rules out tuning). Only
   through all four is it `dead`; anything less is `deprioritized` or still `open`.

**Quick diagnosis table** — symptom → first suspect:
- violates an identity / below floor / above ceiling → **bug** (or leak). Stop and find it; nothing else is
  interpretable until it's gone.
- oracle passes, product fails → **framing or tuning** in a downstream link — never write "signal absent."
- verdict rests on a knob with no swept axis and no external justification → **tuning undecided**; sweep it.
- model class contradicts a measured structural finding about the target → **framing**; revise the framing
  FIRST, only then resume tuning.
- all of the above excluded, ladder complete → a **real negative**. Record mechanism + revisit-trigger; move on.

## Principle #1 — no number without a second, independent measurement

**This applies to every number in every stage.** The "second measurement" is the nearest **independent
constraint** the quantity must satisfy. A number is **implausible if it violates any** — and you can almost
always afford *two*. The six families:

1. **Sandwich** — `baseline-floor ≤ result ≤ oracle-ceiling`. Below floor → bug/absent. **Above ceiling →
   you're leaking.**
2. **Identity / conservation** — an exact law to tolerance. *Cheapest, highest-value class* (a per-group
   constant whose std should be ≈ 0 is a one-line bug detector for an entire pipeline).
3. **Invariance** — must not move under an irrelevant perturbation (scramble forbidden labels → prediction
   unchanged; same seed → same number).
4. **Cross-method agreement** — the *same* quantity computed two structurally different ways (module vs probe;
   a refactored evaluator must reproduce the scalar path bit-for-bit before its sweep is trusted).
5. **Degenerate / known-answer** — a special case with a known value (untrained residual model == baseline;
   synthetic recovery; a detector must separate a synthetic positive from a synthetic negative before running
   on real data).
6. **Dose–response** — right *direction and shape* as a knob turns (beat grows with data; a bias-variance knob
   *plateaus*, doesn't spike). A dose-response over an EXTERNAL metric (successive scored submissions along one
   knob) is the strongest confirmation of a mechanism available.

**Recipe** for any learned `f()`: answer six questions — *trivial baseline? · perfect-info ceiling? · what must
it NOT depend on? · what exact identity relates its I/O? · what case do I know the answer to? · which knob moves
it predictably?* You won't get all six cheaply; get **two**.

**The DOF ladder — run before ANY flexible aligner/sequential model.** Sweep the *match freedom* itself (rigid
1-param → +1 DOF → …) on an oracle-anchored probe and watch signal-margin vs false-optimum rate. If the first
added DOF already narrows the margin (alias optima grow faster than the error mode it absorbs), a full elastic
model extrapolates worse — kill the family before building it.

## Principle #1b — check a load-bearing premise before you build on it

A *number* needs a second measurement (#1); a *premise* needs a check. Before building on a load-bearing
observation — "the model uses X", "signal Y is present", "the interpolator is tapped out" — verify it with the
**cheapest possible probe** (a `grep`, a one-liner, a schema peek). A false premise wastes a whole thread, and
it hides *inside* plausible work. The check costs seconds; the thread it saves costs hours.

**Corollary — direct observables before proxies.** When hunting a predictor/gate/signal, enumerate what is
DIRECTLY measurable in the data before reaching for proxies. Declaring "no test-legal signal exists" after
trying four proxies, while a direct measurement (the model's own residual on labeled rows available at test)
sat unexamined, is the canonical audit failure. Ask first: *what does the test-time data let me measure
outright?*

## Principle #1c — the findings-to-framing loop

**Every finding must either revise the framing or be cited by an arrow. An orphan finding is a leak.** Findings
are measured to CHANGE something; the costliest audited failure mode is a structural finding (the labels are
segmental; the residual is observable in-group) recorded as a trivia row while the models keep their old shape.
Concretely:
- After every finding lands, ask explicitly: *does this change WHAT we predict, the OUTPUT SHAPE, a model
  class, or a prior?* If yes → revise `framing.md` **now** (append a `## Revisions` entry), and open/update the
  arrow that exploits it. If no → the finding row must name the arrow(s) it informs.
- `framing.md` is *revised at pivots* — a form-changing finding with an untouched framing file is a process
  bug, not a judgment call.
- At session close, scan for: findings cited by nothing · arrows `open` and never run for multiple sessions ·
  a `dead` verdict whose held-fixed list contains an unswept load-bearing knob. Surface them in the frontier.
  (These three scans are prime `metisser lint` candidates.)

---

## The loop

```
0 charter → 1 framing+baseline → 2 arrows(hypothesize+ORACLE) → 3 assemble → 4 train → 5 honest-CV → 6 adjudicate+record
                                        ↑___________________________________________________________________|
```

### 0 · Charter — understand the domain (`framing.md` §charter)
What IS this problem? What data exists **train vs test** — and critically **what is train-only** (labels,
auxiliary columns, reference logs)? The metric, and the eval regime (extrapolation? grouped? held-out region?).
Keep it concise. Train-only info is where oracles live (§2). **Define the domain vocabulary here** — a short
glossary of the named parts of the data/task — so every later doc, arrow, and probe refers to them precisely;
imprecise naming is a slow leak that compounds. Show operator those vocabulary to avoid miscommunication. Add
new terms to the vocabulary as research progresses.

### 1 · Framing + baseline (`framing.md` §framing)
Model **input** (encodings mostly known) and — the interesting part — **output shape**: autoregressive vs
joint-over-positions, target parameterization (direct value? residual to a proposal? a latent surface the value
derives from?). If the labels are the output of some external process (a human interpretation, an instrument
pipeline), *measuring the structure of that process* is framing work — reverse-engineering the labeler is a
legitimate model family. Then **establish the baseline** — the trivial-but-strong thing to beat — and **measure
it under honest CV FIRST**. Every later result is reported *paired vs this baseline*.

### 2 · Arrows — hypothesize, then ORACLE before you build
An **arrow** is an idea to extract signal: a falsifiable A→B link. **Before building anything:**
- **Oracle the load-bearing link.** Place the inputs at ground truth and measure: *is the signal there AND does
  it localize/exploit?* Signal-present-but-non-localizing kills a family before any model is built.
- **Chain oracle** for A→B→C: test *does true B determine C* first.
- Oracle result gates the build: **no signal / not exploitable at the oracle → do not spend model budget.**
- Record the arrow as `open` in `arrows.md` with its oracle-ceiling.

### 3 · Assemble — baseline-first, weak-models-first
Build the **simplest** predictor on **validated arrows**. Beat the baseline. Weak models before strong. **Gate
the zoo** (below) — no architecture/tuning knobs before an arrow is oracle-validated.

### 4 · Train — sanity gates that fail loudly
- **Monitor the loss** — converging? (Long runs: incremental progress reports, always.)
- **Overfit experiment** — can it fit in-sample? (Degenerate check: rules out broken-pipeline/can't-express.)
- **Leak assertion** — scramble the forbidden labels; prediction must not move.

### 5 · Honest CV — the verdict
Paired vs baseline, under the **honest CV structure**, as-measured, no laundering. **This is the number that
counts.** metis already owns much of this — extend it, don't rebuild.

**Design the CV to mimic the test regime — the checklist** (simple k-fold is only the trivial case):
- **Holdout UNIT = the test unit.** Hold out whatever the hidden test holds out — a whole group / cluster /
  region, never a row finer than it (adjacent rows are near-identical in most sequential data).
- **Reproduce the distribution SHIFT — but verify the shift hypothesis against external anchors.** A
  harsher-than-test rung misleads as badly as a leakier one. When you hold external scores (past submissions),
  CALIBRATE the rung: the split whose baseline reproduces the external anchor is the one that mimics the test;
  a split that pins every model at the floor while the external test scores fine has the wrong holdout unit.
- **Mask what the test masks.** Whatever is absent/`NaN` at test, blank it in the CV folds too.
- **Assert no leak channel crosses the fold** — target-derived features, group-shared statistics, neighbour
  rows. Use the invariance check: scramble a forbidden signal → the fold score must not move.
- **Report the LADDER, not one number** — the leaky estimate AND the honest one, so the removed optimism is
  visible. Trust the rung that matches the test regime. **Measure the per-model-family CV→external gap**: if
  one family's gap is 2× another's, its CV gain is partly a fold artifact — diagnose the population shift
  (which subpopulation does the test emphasize?) rather than blanket-distrusting the CV.
- **Population-shift robustness for deployment choices**: pick config for the *test* population, not the CV
  argmax. Report per-regime (per-difficulty-quintile) tables alongside totals; a knob chosen at the CV argmax
  when the test population skews toward a regime where that knob hurts is a measured failure mode. Prefer the
  regime-robust choice; better, replace selection with *calibration* — fit confidence parameters to truth-path
  likelihood, not to score argmax. An over-confident component betrays itself when smoothing/aggregating makes
  results worse; calibration then derives the honest weight from first principles instead of re-tuning.

### 6 · Adjudicate + record — the impostor ladder before "dead"
Run the four-impostor discrimination (top of this skill). Only a full ladder earns `dead`; else `deprioritized`
(signal present, unexploited — a priority, not a verdict) or still `open`. Then **write the arrow row**, run
the findings-to-framing loop (#1c), and loop: the ledger's gaps are your next arrows.

---

## The arrows ledger (`arrows.md`, one per investigation)

**Structured markdown is the single source of truth** (LLM-native, git-native). Relational queries run over a
**throwaway** CSV/duckdb view regenerated from it — **never a second authoritative store** (single-mechanism).
Layout = **index table** (queryable scalars) + **detail sections** (`### <id>`) for what a cell can't hold. At
≤~100 arrows/file this greps and pandas-reads instantly; no database.

**Taxonomy — three kinds of entry:**
- **arrow** (`a-*`; index + detail) — a *hypothesis / lever* to test: a **building block**, at any grain —
  coarse (nearly a whole model) or fine (one knob). Always a claim; **never itself the product**.
- **finding** (`f-*`; §findings) — a measured structural *fact* about the data (informs the framing; not a
  lever to exploit). Subject to the findings-to-framing loop (#1c): every finding names what it changed or
  which arrow cites it.
- **anchor** (§anchors) — an achieved *top-line number* (baseline / best / external-LB): the **product**. A
  *model* is an assembly of win-arrows; its measured number is the anchor, which references the arrows it's
  built from. Anchor `kind`: `baseline` · `best` · `LB` (external submission) · `ceiling`.

**Status enum**: `open` (untested) · `active` (in progress) · `deprioritized` (signal present, not yet
exploited — a priority, NOT a verdict) · `dead` (proven-absent, cleared the impostor ladder) · `win` ·
`baseline`. Index columns: `id · hypothesis · status · oracle · honest(vs-baseline) · belief · prov ·
topline-Δ`.

**Provenance on every number**: `ACHIEVED` (externally scored — a leaderboard, a deployed metric) · `MEASURED`
(our honest CV) · `ORACLE` (ceiling) · `CLAIMED` (external, unreproduced — **never** build strategy on this).
**Strategy may build only on ACHIEVED/MEASURED.**

**Source rule (revalidation debt)**: every number carries its provenance tag **AND a source** — a probe/trace
**PATH** (git supplies the version: the verification commit makes `git blame` on the cell resolve to the exact
code+trace; inline shas are churn), or an external submission id. A number with **no source** gets a trailing
**`†`** = *unsourced, revalidate*; **`grep '†' arrows.md`** is the standing debt list. Reproducing it (with a
trace) drops the `†`. `†` is only for "no source anywhere" — `CLAIMED` means "sourced, not by us."

**Prior-run policy**: verdicts from an earlier, un-audited investigation of the same problem are **not
imported** — old probes are bug-suspect by default, and under this loop a real dead-end re-derives cheaply
(a whole paradigm can be re-adjudicated in ~5 probes). Only externally-scored numbers (`ACHIEVED`) carry over.
If an old claim seems worth having, RE-MEASURE it; never import the verdict.

**Each detail section carries**: **mechanism** (durable prose — *why* the outcome, not just the number; a
number-only dead arrow gets re-explored) · **held-fixed** (`(value, belief-strength, justification-LINK)` per
frozen knob; no unlinked justification) · **revisit-trigger** (the belief-update that reopens it — makes a
later sweep a *scheduled* re-test, not luck) · **provenance + source** per number · **topline-Δ** (how it moved
the headline).

**Query the index with awk on the field, not raw grep** (status words recur in prose):
```bash
awk -F'|' '/^\| a-/ && $4~/open/                  {print $2}'          arrows.md  # untested hypotheses
awk -F'|' '/^\| a-/ && $4~/win|dead|deprioritized/{print $2"→"$4}'     arrows.md  # tested arrows + verdict
awk -F'|' '/^\| [af]-/{gsub(/ /,"",$2);gsub(/ /,"",$4);print $2","$4}' arrows.md  # id,status → csv
grep -n '†' arrows.md                                                             # the revalidation debt
```

The experiment's `arrows.md` keeps only a THIN header (frontmatter, a pointer to this skill for process, plus
any experiment-specific policies) — the process prose lives here, single-sourced.

## Reproducible probes (the inner-loop reproducibility model)

Exploratory runs are **cheap and one-shot**, so reproduce them by **RE-EXECUTION, not caching** — CAS /
content-addressing is for the expensive, *repeated* outer loop (metis sweeps). A probe is reproducible when:

1. **Deterministic** — pin every seed; read only pinned inputs; no wall-clock / unseeded randomness.
2. **Self-contained invocation** — the exact run command (all args, copy-paste) in the probe's docstring.
3. **The verification commit** — the `.py`, its same-basename `.log` trace, AND the `arrows.md` row it settles
   land in **one atomic commit**. That commit *is* the pin; the `source` cell names a PATH, never a sha.

**Bar: reproducible CONCLUSIONS, not bit-identical bytes.** Same-conclusion-on-re-run is the target.

**Reify before you claim.** Scratchpad = looking; committed probe = claiming — the ledger is the boundary
between exploration and record. **Pin the inputs once**: a workspace-level data-provenance file (source + a
content checksum); if inputs change, every `MEASURED` number reverts to `†`.

**No separate unit tests for probes** — the in-probe instruments (known-answer calibration, nulls, identities,
degenerate asserts) ARE the tests, run on every execution against real data and archived in the trace. A
conventional pinned-fixture test is warranted only when shared machinery is promoted into a module that future
probes import (a silent regression there corrupts *future* measurements — a different risk class). Build the
floors/ceilings/nulls BEFORE the first run and read them FIRST: a design flaw announces itself as an absurd
floor before you waste interpretation on anything downstream.

## Session continuity (research spans many sessions)

**No continuation file — the workspace + git ARE the handoff**, via one convention:

- **Entry points, in order:** experiment `README` (the map + protocol pointer) → `framing.md` (the problem) →
  `arrows.md` (state + `## The open frontier` = live thread + **NEXT ACTIONS**) → **`lessons.md`** (the
  experiment's own mistakes + worked examples — see below) → `git log` (what happened).
- **Next-steps live in the arrows frontier** — live, editable, single-source. NOT in commit messages (a commit
  is immutable; a "next step" written into it goes stale the moment it's done).
- **Before a session boundary, reify the live thread into the ledger** — nothing durable-worthy lives only in
  chat.
- **Version control: single-threaded on `main` (v0).** Research runs single-threaded — verification commits
  land directly on `main`; no feature branches by default. A research thread is already serialized through
  the ledger, so branch-per-issue buys nothing and hides the arrow history from the frontier. Revisit only
  when genuinely parallel model lines appear — and note that merging parallel RESEARCH lines is not a git
  merge (weights, configs, and ledger verdicts need their own reconciliation design); until that design
  exists, don't fork.

## Per-experiment `lessons.md`

**Each experiment folder carries a `lessons.md`** (sibling of `arrows.md`): the experiment-grain
self-improvement file, mirroring the constitution's `workshop/lessons.md`. It holds:
- **process lessons learned IN this experiment** — what went wrong + the rule that prevents the repeat, each
  with the concrete numbers that made it vivid;
- **the worked examples** — this skill states rules generically; the vivid case studies with real numbers live
  in the experiments that produced them (the canonical populated example: `competition/rogii-v2/lessons.md`);
- **domain gotchas** true of this dataset/task but not of ML research generally.

Read it at session start (it's in the entry-point list). When an audit or post-mortem finds a new failure
mode: the *generic* rule graduates into this skill; the *specific* story stays in `lessons.md`.

## Gate the zoo (why this SERVES modeling, not replaces it)

A net is **a refiner of a proposal, not a diviner of absent signal.** An agent facing a bad number reaches for
the cheapest visible action — another architecture, more tuning — which varies *capacity* while a *framing* bug
hides underneath. **So gate the model zoo behind a passed arrow-test (§2 oracle).** Then a washout is
*interpretable* ("the arrow was real but this refiner couldn't exploit it → a genuine modeling gap") instead of
ambiguous. Baseline-first; weak before strong; **ensemble/stacking LAST** — only once ≥2 legs beat baseline AND
clear the fusion bar `ρ < σ_strong/σ_weak`.

**The fixed-form-first ladder for sequential/structured models**: a probabilistic filter (HMM/Kalman-style,
~2 free parameters, components fixed by physics/measured priors) BEFORE any learned recurrence. It works at
small-data scale, can't hallucinate forbidden dynamics, and its failures are interpretable per-component.
Replace a component with a learned one ONLY when its mis-specification is MEASURED (emission residuals
heavy-tailed; calibration off; smoothing hurts) — the HMM→RNN spectrum is walked one measured rung at a time,
never jumped.

## Status — what's prose vs built

Everything here is **prose you follow by hand.** Nothing is enforced by a binary yet — deliberately (fast
iteration on the process itself). As steps prove themselves, they earn `metisser` subcommands — the metis
analog of ariadne's `sdlc`. **The earned-subcommand backlog is a living issue: metis#68**
(`workshop/issues/000068-metisser-earned-subcommand-backlog.md`) — lint scans, ledger verbs, probe discipline,
evaluation machinery, session ritual, each entry citing the failure that earned it. Append there as
investigations surface new candidates. **Do not build `metisser` before the prose shape settles.** Companion artifacts: `framing.md` (§charter +
§framing), `arrows.md` (the ledger), `lessons.md` (per-experiment). Project:
`metis/workshop/projects/ml-research-workflow.md`.
