---
type: project
name: ml-research-workflow
goal: "Build the ML-research workflow — a followed SKILL (the workflow contract) + supporting datatypes (framing, arrows-ledger) + metis discipline tools (oracle/arrow-test harness, plausibility checks, training sanity gates, provenance/anchors) — that MANUFACTURES hard-local grounding signals for exploratory ML research, where the domain gives only soft-global feedback (a bad number whose cause — bug / tuning / framing / real-negative — is unknowable from the number alone). Validate it by REBUILDING rogii rigorously under the workflow, building each component when the rebuild pulls it out (demand-driven, per the metis charter). The reset premise: rogii's many dead-ends may have been mis-marked (implementation/tuning vs a proven real-negative) for lack of rigor — the workflow exists to tell those apart and to SURFACE, not hide, the modeling gaps a real leaderboard beat requires (we know a beat is doable from the LB + forum)."
done_when: "The `ml-research` skill exists as a followed prose contract; rogii is rebuilt through it end-to-end with an arrows-ledger carrying, per hypothesis, {oracle-ceiling, honest-vs-baseline (dual-measured), mechanism, belief-weighted held-fixed WITH links, revisit-trigger, provenance, top-line contribution}; EVERY 'dead' verdict has passed the impostor-ladder (fit → oracle → dual-measure → held-fixed); and the components the rebuild demanded are built in metis (not speculatively). Success is NOT a specific LB number — it is (a) a rigorous, re-runnable map separating proven-dead from under-explored, (b) ≥1 previously-'dead' end reopened or confirmed-under-rigor, (c) the reusable skill + datatypes + tools, earned on the real case."
status: proposed
operator: xianxu
created: 2026-07-22
updated: 2026-07-22
deadline: 2026-09-15
sources:
  - brain/workshop/pensive/2026-07-21-01-pensive-metis-ml-workbench-modeling-loop.md
  - brain/workshop/pensive/2026-07-03-01-pensive-experiment-shape-workbench-design.md
  - workshop/projects/arena3-rogii-wellbore.md
  - brain/workshop/continuation/20260722T042851-rogii-buda-fix-covered-lever.md
explicitly_out:
  - "the model ZOO built speculatively — no loss-builder library / architecture menagerie / fusion-stacking harness until a validated arrow demands a refiner (gate the zoo behind a passed arrow-test)"
  - "metis ENFORCEMENT (hard-shell gates) of any workflow step before its PROSE shape has proven itself on the rogii rebuild — earn the abstraction on the real case, then compile it into a gate"
  - "ensemble / stacking work before ≥2 uncorrelated legs each beat baseline and clear the fusion bar (ρ < σ_strong/σ_weak)"
---

# ml-research-workflow — the modeling INNER loop, as a grounded workflow

The inner-loop sibling of **arena3-rogii-wellbore** (which nailed the *outer* loop: ingestion, cluster-CV,
channel-split — how to run reproducible configs without getting lost). This project builds the **inner loop**:
how to form, test, and adjudicate modeling hypotheses *rigorously*, so an agent stays grounded when the only
feedback is a soft number. It is the concrete realization of the 2026-07-21 pensive — earned on rogii, not
designed in a vacuum.

## The problem (why SDLC doesn't transfer)

In ariadne/SDLC the feedback is **hard and local**: it compiles or it doesn't; the test names the failing line.
The agent is grounded because *failure points at itself*. In ML research the feedback is **soft and global**:
you get a number, and a bad number has four indistinguishable causes — **bug · tuning · framing · real-negative**
— that all look identical. **That ambiguity is the whole difficulty.** Every rogii dead-end was an instance: a
silently-broken `geo_surface` feature *looked* like "the GRU doesn't transfer"; `K=60` *looked* like "the
interpolator is tapped out"; a phantom `5.99` *looked* like "covered is solved, the gap is isolated." Each cost
a whole thread.

So the design target is not "tools for building models." It is: **manufacture a hard-local feedback signal
where the domain only gives a soft-global one.** The workflow is an SDLC analog with **soft gates**, where the
grounding shell (the compiler) is replaced by a **measurement discipline**.

## Principle #1 — no number without a second, independent measurement

This is the substitute for the type-checker, and the top rule of the whole workflow. The "second measurement"
generalizes to the **nearest independent CONSTRAINT** the quantity must satisfy — six families; a number is
**implausible if it violates any**, and you can almost always afford *two*:

1. **Sandwich** — baseline-floor ≤ result ≤ oracle-ceiling (below floor → bug/absent; *above ceiling → leak*).
2. **Identity / conservation** — an exact law to tolerance (rogii: within-well `a` std ≈ 0 — the one-line
   `geo_surface`-bug detector; `a` std was 662 ft). *Cheapest, highest-value class.*
3. **Invariance** — must not move under an irrelevant perturbation (scramble-the-toe leak assertion; seed).
4. **Cross-method agreement** — same quantity two structurally different ways (module == standalone probe to
   0.001; kriging vs local-plane).
5. **Degenerate / known-answer** — a special case with a known value (untrained residual model == baseline —
   the check that caught "the ties were untrained models"; synthetic planar recovery).
6. **Dose–response** — right direction AND shape as a knob turns (beat grows with well-count; K bias-variance
   plateaus, not spikes).

**The recipe** — to define hard signals for any learned `f()`, answer six questions: *trivial baseline? ·
perfect-information ceiling? · what must it NOT depend on? · what exact identity relates its I/O? · what case do
I know the answer to? · which knob moves it predictably?* The skill makes *declaring* these per-arrow cheap and
*checking* them (eventually) automatic.

## The impostor ladder — before any "dead" verdict

A hypothesis cannot be filed **dead** until it survives, in order, a cheap ladder that rules out each impostor —
the research analog of the SDLC close-gate:

1. **Fit check** — can it overfit in-sample? → rules out *broken pipeline / can't-express*.
2. **Oracle** — does the signal exist at ground truth? → rules out *measuring-the-wrong-thing* (rogii GR: +0.5
   at true TVT but doesn't localize → **not dead**, deprioritized).
3. **Dual measurement** — does an independent recompute agree? → rules out *bug*.
4. **Held-fixed record** — what was frozen when you declared it? → makes the negative *falsifiable* ("tapped
   out **at K=60**").

Only a hypothesis through all four is a **real negative**. Everything else is `open`/`active`/`deprioritized`.

## The arrows-ledger schema (the living record)

`dead` is a *proven-absent* verdict, not a default. Status ∈ `{open, active, deprioritized, dead, win}`. Each
arrow row carries:

**Storage — structured markdown, single source, projected to tabular (NOT a database).** At ≤1000 arrows
lifetime (~50–100 per competition) a persistent DB is all cost, no payoff — and fails the repo-centric
build-vs-buy test (opaque to the agent, binary git-diffs). Markdown stays the ONE source of truth (LLM-native,
git-native); any relational query runs over a **derived, throwaway** CSV/duckdb view regenerated from the
markdown — never a second authoritative store (single-mechanism). Layout mirrors issues (frontmatter scalars +
prose body) and MEMORY.md (index + files): one `arrows.md` per investigation = an **index table** (the queryable
scalars — `id · hypothesis · status · oracle · honest · baseline · belief · provenance · topline-Δ` — this table
IS the CSV-projection surface) + **detail sections** (`### <id>`) for what a cell can't hold (mechanism prose,
held-fixed WITH links, revisit-trigger, trace links). Projector = grep/awk/pandas now → a `metisser arrows query`
subcommand once the columns stabilize. Anchor search (`~12` → `12\.`) is grep on the file or a filter on the frame.

Each arrow row carries:

- **hypothesis** — the A→B link, stated so it's falsifiable.
- **oracle-ceiling** — best achievable with perfect info (the headroom).
- **honest-result** — vs the baseline, **paired, dual-measured** (per Principle #1).
- **dual-check** — which independent constraint(s) verified the number.
- **mechanism** — *why* the outcome (durable: "GR autocorrelated → broad optimum → can't localize", not
  "washed, −0.02"). A number-only dead arrow gets re-explored.
- **held-fixed** — `(value, belief-strength, justification-LINK)` per frozen hyperparameter. `K=60` = "arbitrary
  guess, no link → HIGH revisit priority"; `Adam default LR` = "literature-backed → link → LOW". Revisit-priority
  is a computed function of belief-strength. **No unlinked justification** (anti-hallucination; internal links =
  cross-ref to a catalog section, external = URL).
- **revisit-trigger** — the condition (usually a belief update from new evidence — a swept knob, a forum post, a
  paper) that reopens it. "Interpolator dead → revisit if K/POWER/AR swept" — then K=250 is a *scheduled* re-test.
- **provenance** — every number tagged `ACHIEVED` (LB-confirmed) / `MEASURED` (our CV) / `ORACLE` (ceiling) /
  `CLAIMED` (external, unreproduced). **Strategy may build only on ACHIEVED/MEASURED.** (The 5.99 phantom was a
  CLAIMED number treated as MEASURED for a whole thread.)
- **top-line contribution** — how this arrow moved the headline number. Top-line results become **searchable
  anchors** (the day/config we hit 12.45); markdown-prefix-searchable (`~12` → regex `12\.`).

## Why this SERVES modeling, not replaces it

The reset is **not** "modeling is a trap, just sweep." rogii got to LB 12.45 via a humble K-sweep while five
model families washed — but that is exactly the symptom the workflow must cure. The **model zoo is a trap only
when entered before the arrow is validated**: an agent facing a bad number reaches for the cheapest visible
action (another architecture, more tuning), which varies *capacity* while a *framing* bug hides underneath —
infinite plausible-looking actions all failing for the same unseen reason. The fix is to **gate the zoo behind a
passed arrow-test**: oracle the load-bearing link first (*is there a proposal worth refining?* — a net refines a
proposal, it doesn't divine absent signal). Then a model's washout is *interpretable* ("the arrow was real but
this refiner couldn't exploit it → a genuine modeling gap to push") instead of the ambiguous "arrow? framing?
model?" that plagued us. **This is how we find whether we marked ends dead wrongly** — the entire point of the
reset. Baseline-first; weak models before strong; ensemble/stacking last (only once ≥2 legs clear the fusion bar).

## Fleet (the build-out — mostly metis, driven by the kbench rogii rebuild)

Demand-driven, per the metis charter ("build the component when the case pulls it out of you"). The rogii
rebuild is the DRIVER; each metis component is created + claimed **when the rebuild demands it**, not upfront.
Issue numbers assigned at `sdlc issue new` time.

**Sequence:** `ml-research-skill` (prose) FIRST → then `rogii-rebuild` starts and PULLS the rest out on demand.

| # | issue (slug) | repo | what it builds |
|---|---|---|---|
| A1 | `ml-research-skill` | metis | the workflow contract as a SKILL.md — the ladder, the 6 hard-signals + recipe, the arrows schema, the principles. **Prose first; the umbrella.** |
| A2 | `rogii-v2-under-workflow` | kbench | a FRESH **rogii-v2** workspace (do NOT edit the old rogii — freeze it as reference; kbench#24's `## Log` is the ready-made worked example to seed the arrows format). Rebuild from the charter down, following the skill; the vehicle that pulls every component out and re-adjudicates the old dead-ends under rigor. |
| A3 | `framing-datatype` | metis | the static problem-model doc (charter + I/O framing merged — one low-churn doc, minimum-mechanism). |
| A4 | `arrows-datatype` | metis | the hypothesis-ledger datatype (the enriched schema above). The heart. |
| A5 | `plausibility-taxonomy` | metis | the 6 hard-signal families as a reference; later, metis assert-helpers (identity/invariance/sandwich). |
| A6 | `oracle-arrow-test-harness` | metis | a standard oracle + paired-null scaffold so every arrow probe emits a comparable, provenance-tagged row. (Open Q: first-class metis primitive vs probe pattern.) |
| A7 | `training-sanity-gates` | metis | overfit-probe, loss-curve capture, leak-invariance (scramble-test), **incremental progress reporting on every long run**. |
| A8 | `honest-cv-extensions` | metis | CV structures beyond the current cluster/spatial-block, as the rebuild pulls them (metis is already strong here — extend, don't rebuild). |
| A9 | `measurement-provenance-anchors` | metis | number-provenance tags + prefix-searchable top-line anchors. |

## Non-goals / guardrails

- **Earn the abstraction on rogii — stay in prose, name the binary later.** Every datatype/tool starts as PROSE
  used in the rebuild; it becomes a hard-shell gate in a binary (**`metisser`** — the metis-research lifecycle/
  workflow verbs, the arrows projector/query, the plausibility asserts; the metis analog of ariadne's `sdlc`)
  only after its prose shape has proven itself. Do NOT build `metisser` prematurely — prose keeps the *process
  itself* fast to iterate; early drift is fine and expected. (The pensive's own "don't design in a vacuum.")
- **No speculative model zoo / loss library / stacking harness** — gated behind a validated arrow (see §"serves
  modeling").
- **This project owns the INNER loop only.** The outer loop (ingestion, sweep, ledger, cluster-CV) is
  arena3-rogii-wellbore + metis#36/#37 — reference, don't duplicate.

## Log

### 2026-07-22 — project opened
Opened out of the rogii-buda continuation's closing discussion (operator brain-dump + the grounding-gap
exchange). rogii LB moved 13.248 → **12.452** via a K-sweep this session while every model family washed —
which crystallized the thesis: the scarce resource is **epistemic grounding under soft feedback**, not
model-construction sophistication. The rogii investigation (kbench#24 `## Log`) is a ready-made, fully-populated
worked example for the arrows-ledger. Next: draft the `ml-research-skill` (A1) as prose, then start the rogii
rebuild (A2) and pull components out on demand.

Execution decisions (operator, 2026-07-22): (1) **rogii-v2 = a fresh kbench workspace**, old rogii frozen as
reference; (2) the skill lives in **metis**; (3) the eventual binary is **`metisser`** (lifecycle/workflow
subcommands) — NOT built yet, prose-first for fast process iteration, drift acceptable early. (4) Catalog =
**structured markdown as single source, projected to a throwaway CSV/duckdb view for relational query — no
persistent DB** (≤1000 arrows; agent-legible + git-native win; single-mechanism guardrail).
