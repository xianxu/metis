---
type: pensive
date: 2026-07-14
topic: a feature-engineering algebra under cross-validation
mode: ideas
description: metis#35 escalated into a design theory — label as a domain-restricted keyed channel, feature constructors as fit/apply with declared scope signatures, aggregate algebraic class deciding fold-recompute cost; roughly relational algebra extended with leak-tracking and fold-indexed evaluation.
references: [../issues/000036-channel-split-y-as-runner-scoped-keyed-artifact-nested-cv-as-domain-restriction-metis-v3.md, ../issues/000037-feature-constructor-algebra-declared-scope-signatures-aggregate-classes-derived-placement.md, brain/workshop/continuation/20260714T073940-metis-v2-honest-beat-blocked-on-35.md, ../history/projects/metis-v2-experiment-algebra.md]
---

# Pensive: a feature-engineering algebra under cross-validation

Started as metis#35 (nested-CV sealed pass drops get-data; a `features` step reading `raw: get-data`
dangles). Root-caused as a design gap, not a bug: the seal substitutes a derived artifact
(`analysis_i`) and deletes its producers, which is only sound if that artifact is the *sole* road
from raw data to the pipeline — and `raw: get-data` is a second road. Pulling that thread reached
a general model. The engineering question ("how to fix the seal") became a research question:
what is the algebra of feature construction, and how does cross-validation interplay with it?

## The data model: two channels, one partial

A dataset is not one table. It is a row-key set K plus two channels: `X : K → features` (total —
defined on all rows, train and test) and `y : K ⇀ label` (partial — defined only on labeled rows).
Test rows are not "rows with a NaN label"; they are outside `dom(y)`. A CV fold is then not a
masking operation but a **domain restriction on y alone**: outer fold i hands the pipeline
`y|dom(y)\Oᵢ`; the inner sweep restricts further. Nesting is successive domain subtraction, and it
composes trivially: `(y|A)|B = y|A∩B`. X is never restricted (under transductive semantics — see
below). An assessment row is a test row whose label we happen to know and are choosing not to look
at — the existing `cross_fit_target_encode` already treats them identically (NaN y, non-fit mask),
which was the tell that the model is right.

The split is **access control, not separation of computation**: a target-encoding constructor
still receives both channels joined by key; what changes is that the *runner* decides y's domain,
so the constructor cannot reduce over labels it was never given. `fit_mask` stops being a
convention passed to steps and becomes derived: `fit_mask ≡ k ∈ dom(y)`.

## The canonical constructor form: fit ∘ apply with a scope signature

Every feature constructor decomposes as `fit: θ = A(X|R, y|S)` (a reduce over rows, producing a
parameter — a group-mean table, a median, a count map, an ECDF) and `apply: F(k) = g(X(k), θ)`
(a per-row map over ALL rows). All row-coupling lives in fit; apply is always label-free. The
constructor is characterized by its **scope signature** (R = which rows' features fit reads,
S = which rows' labels fit reads):

- map (Title from Name): R=∅, S=∅
- feature-reduce (fare_median, ticket_size): R≠∅, S=∅
- label-reduce (group survival rate): R≠∅, S≠∅

Two safety properties become one-liners: **no assessment leakage** is `S ∩ B = ∅` — enforced
structurally by handing the constructor `y|A` (out of scope, literally, not "the step was
careful"); **no self-leakage** is `k ∉ S(k)` — the scope is per-row for labeled rows (LOO:
`S(k) = A\{k}`; cross-fit: `S(k) = A\innerfold(k)`; unlabeled rows: `S(k) = A`). So: labels enter
features only through fit parameters, and the runner controls the label-scope of every fit.

## The aggregate's algebraic class decides cost (orthogonal to leakage)

Classify θ's reduce: **no reduce** (free) < **commutative monoid** (shardable, cacheable) <
**abelian group** (subtractable) < **holistic** (median, quantiles — no decomposition). The
abelian-group case is the scale answer: every fold's θ derives from one global pass by
subtraction — `θᵢ(g) = (total_g − Σ_{k∈Bᵢ} y(k)) / (cnt_g − |Bᵢ∩g|)`, LOO subtracts once more —
O(m + folds×groups) instead of O(folds×m), and it composes with nested CV because domain
restriction composes. (This matches the data-cube distributive/algebraic/holistic classification
and incremental view maintenance — from memory, to be verified in the research detour.) Holistic
aggregates under induction are irreducibly O(folds×m); the design prices that honestly and —
because features become constructor-level nodes, not one monolithic `features` step — a fold
recomputes ONLY the constructors whose scope changed, not the whole feature stage.

## Two independent axes, decided by different authorities

- **S (label scope) is a legitimacy question** — non-negotiable, fold-defined, runner-enforced.
  Kaufman/Rosset/Perlich's axis (their Fig 1(c): "only targets are illegit" — never operationalized).
- **R (feature scope) is an estimand question** — feature-reduces over all rows leak nothing but
  bias the CV estimate relative to the *inductive* estimand (Moscovich & Rosset 2022, JRSS-B:
  label-free preprocessing still biases CV; refutes ESL's claim that unsupervised screening is
  safe) while being exactly right for the *transductive* one (Kaggle: the deployed transform is
  literally the same θ). Transductive-vs-prospective must be a declared semantics in the shape,
  not an accident — under transduction X is fold-invariant and every S=∅ constructor hoists;
  under induction R must be restricted per fold too and the commuting analysis returns.

Placement falls out mechanically: R=∅,S=∅ → hoist always · R≠∅,S=∅ → hoist iff transductive ·
S≠∅ → never hoisted, y domain-restricted per fold, subtract-lowered if algebraic.

## The design-space gist: relational algebra + two extensions

The operator basis is relational algebra plus grouped aggregation — even kNN-style features
dissolve into it (a self-join on a distance predicate + aggregate over neighbors; leak-tracking
follows both table references). Our algebra differs from RA by exactly two extensions: (1) an
**annotation/type layer tracking label provenance** — y is one distinguished column whose scope
is runner-controlled, so "no second source of y" is a type rule (metis#35 becomes a type error,
not a missing repoint); (2) **fold-indexed evaluation semantics** — one query evaluated under a
family of domain restrictions of one relation, with the aggregate's algebraic class deciding
incremental vs. re-evaluated. Literature so far (verified in-session by a research agent): the
label-as-separately-keyed-artifact-with-fold-set-domain design appears to be an undrawn diagonal —
Feast's entity-dataframe does keyed labels with row-scope keyed by *time*; mlr3's
`.predict_dt(dt, levels)` drops `target` from the signature but row-scope stays with the external
Resampling; Yang et al. (ASE'22) do static map-safe/reduce-suspicious analysis but take rows as
the taint unit and explicitly scope out label leakage.

## Open questions

- Does the transductive estimand escape Moscovich & Rosset's bias theorem? My argument: under
  transduction there are no "new observations" — the scored rows and deployed rows are in the same
  position w.r.t. θ. Believed, not proven; the literature has nothing connecting Vapnik's
  transductive estimand to the CV-bias results (and Kapoor & Narayanan's L1.2 contradicts
  Kaufman's (9)→(2) reduction on exactly this point, unreconciled). **Empirically testable on the
  workbench**: run ticket_size hoisted vs fold-scoped, compare which honest estimate tracks the
  public leaderboard.
- The scorer needs held labels: split `train` into fit/predict + a terminal `score` step (only a
  scalar-producing step ever touches `y|B`), or declare metis's own steps trusted? Lean: split.
- Where does the transductive/prospective declaration live — shape header?
- The split step itself reads full y (stratification) — the runner's carve-out needs to be
  explicit in the type rule.
- Is the constructor-declared scope signature checkable/derivable, or purely trusted declaration?
  (Static analysis of step code is out of scope; a declared signature + runtime domain restriction
  makes false declarations *ineffective* for S, but R declarations under induction are trusted.)
- What of the current metis design survives: phases (data/pipeline) are too coarse — they were the
  binary approximation of the scope signature. `analysis_i` cloning, `METIS_READ_ROOT`, the sealed
  branch of `buildFoldExperiment`, and `fit_mask`-as-passed-parameter all become deletable in the
  masked/channel design. #35's two sibling bugs (analysis_i lacks test; adapt's fare_median fits
  above the split) dissolve or reclassify under it.

## References

- metis#35 (`workshop/issues/000035-*.md`) — the motivating bug; metis#36 (stage B) + metis#37
  (stage C) — the tickets this design feeds.
- `workshop/history/projects/metis-v2-experiment-algebra.md` — the project this feeds (done; archived into metis 2026-07-19).
- Kaufman, Rosset & Perlich, KDD'11 — legitimacy relation; Fig 1(c). Moscovich & Rosset, JRSS-B
  2022 — label-free preprocessing biases CV. Cawley & Talbot, JMLR 2010 — nested-CV selection
  bias. Yang et al., ASE'22 — static leakage detection, map/reduce edges. Kapoor & Narayanan,
  Patterns 2023 — leakage taxonomy. (All verified in-session; full citations in the research
  agent's report, session 2026-07-14.)
- Gray et al. Data Cube (aggregate classes) — **verified** (3-0, arxiv cs/0701155): distributive/
  algebraic/holistic match the ladder; class is operation-relative (MAX insert-distributive,
  delete-holistic) — anticipates the subtractable rung. Caveats: Gray never says monoid/group
  (our vocabulary is honest extrapolation); holistic-forces-recompute holds for exact aggregates
  only (mergeable sketches soften it).
- Chernozhukov et al. 2018 double-ML — **fetched, extraction-level**: coins cross-fitting;
  own-observation bias term provably vanishes; **no first-order efficiency cost** (S(k) is
  statistically free). Nuance: Neyman orthogonality is the other half of DML and is a property
  of the estimand, not the data-flow — scope signatures cannot capture it.

## Research detour findings (2026-07-14, deep-research run, 103 agents)

Verdict: components 2 (fit∘apply) and 3 (aggregate ladder) are established prior art; component 1
(typed y channel with fold-set domain) exists in NO surveyed algebra — the genuinely novel axis.
CV appears in every surveyed system only above or below the algebra (script control flow, example
workload), never in it; nobody frames folds as IVM deltas though all machinery exists.

- **fit∘apply confirmed** (all 3-0): factorized in-DB ML (Orion, LMFAO, IFAQ) computes training as
  group-by aggregate batches pushed through joins without materializing (LMFAO: 37KB sufficient
  statistics vs 23GB join); parameter-independent Σ=Σxxᵀ computed once, reused across iterations.
- **Subtractable rung confirmed via DBSP** (3-0, arXiv 2203.16684): Z-sets = abelian group;
  differentiation literally requires subtraction; linear ops Q=Q^Δ; MIN non-incrementalizable
  under deletes. Fold-as-deletion-delta is OUR extension — present in no source.
- **No label typing anywhere** (3-0 per system): LMFAO folds y into the feature vector with
  parameter fixed to −1; Orion/SystemDS/IFAQ/LARA treat it as an ordinary column/variable; zero
  mentions of leakage in any of them.
- **Nearest system: SystemDS** (CIDR'20 + LIMA SIGMOD'21): k-fold CV over linear models reuses
  per-fold X^TX/X^Ty partial products via lineage-hash caching, recombined by element-wise
  addition — implicit monoid exploitation as an opaque hardcoded rewrite; no typing, no y-channel,
  no leakage enforcement. **Design refinement this yields:** folds are a PARTITION, so a fold's
  complement-θ = sum of the other parts' partials — **monoid suffices for fold-level θ; subtraction
  is only needed for per-row S(k) (LOO/cross-fit) and unanticipated restrictions.** Pre-partition
  by fold + merge beats global-pass-minus-delta for the fold axis.
- **Prongs 3–5: targeted 3-lens adversarial verify pass ran 2026-07-14 (27 agents) — 8/9 claims
  survive, 1 refuted:**
  - **SeLINQ** (Schoepe/Hedin/Sabelfeld, ICFP 2014, DOI 10.1145/2628136.2628151) — **CONFIRMED
    3-0**: column-granular security labels via per-database type signatures; static type system
    with a noninterference soundness theorem (Thm 1) through LINQ-style queries; derived columns
    get computed levels; secret→public relaxation = type error (precision: the relaxed annotation
    is the query's declared RESULT-type field, not the DB column). Gap claim **CONFIRMED with a
    load-bearing refinement**: no in-query aggregation (grammar: `exists` only; aggregates
    deferred to future work *via declassification*) — but the HOST language does type numeric
    folds/sums through standard label propagation (a sum over secret ints is itself secret).
    **The insight this yields: standard IFC propagation is correct but useless for us — we WANT
    θ (a function of secret y) released to features. The fit boundary is a DECLASSIFICATION
    point, and the legitimacy conditions (S∩B=∅, k∉S(k)) are the declassification policy.
    Cross-fitting = a principled declassification rule for the y channel.** SeLINQ defers exactly
    this; that deferral is the precise hole our design fills.
  - **Cheney/Ahmed/Acar** (arXiv 0708.2173; MSCS 21(6):1301-1337, 2011) — **CONFIRMED 3-0**:
    NRC setting, sub-value (field/cell) annotations, dependency-correctness (noninterference-
    SHAPED via Abadi et al.'s dependency core calculus — the word itself doesn't appear);
    minimal/exact dependency provenance undecidable (via NRC query-constancy) → sound
    conservative approximation only. Legitimizes declared-signature + runtime domain restriction.
  - **Chernozhukov et al. 2018** (Econometrics J 21(1) C1-C68) — **CONFIRMED 3-0, verbatim**:
    coins "cross-fitting" (swap-role + average); own-observation bias term mean-zero with
    vanishing variance under sample splitting; averaging restores full efficiency (no first-order
    cost); Neyman orthogonality is a property of the score, not the partitioning.
  - **sklearn TargetEncoder** — **CONFIRMED 3-0, verbatim** (added 1.3, Micci-Barreca 2001;
    fit_transform cross-fits, default 5 folds, KFold/StratifiedKFold; fit().transform() ≠
    fit_transform() stated three times in the docs, leaky path callable; cv is an operator
    parameter; shrinkage blend, smooth='auto' empirical Bayes).
  - **featuretools** — **CONFIRMED 3-0** (exactly two primitive classes; label-unaware — dfs's
    `target_dataframe_name` names the entity, not a label; cutoff-time separate). Nuance: some
    transform primitives (CumSum) are row-coupled while staying one-output-per-row.
    **Hamilton** (now Apache Hamilton) — **CONFIRMED 3-0**: per-node caching (node_name +
    code_version + dependency versions) = column-level recompute; no fit-scope/label concept;
    splits expressible only as ordinary user nodes.
  - **skrub AggJoiner** — **REFUTED 2-1, my error**: the op vocabulary is {count, mode, min, max,
    sum, median, mean, std} — median and mode are HOLISTIC, so "all decomposable, none holistic"
    was wrong (rest confirmed: aux table frozen at __init__, never fold-restricted; stage-level;
    y ignored). Two upshots: real-world op vocabularies DO include holistic aggregates (the
    ladder's bottom rung is not exotic), and skrub has a separate **AggTarget** class for
    label-aware aggregation — another instance of the label path being a bolted-on sibling
    rather than a typed channel.
  - **Amsterdamer/Deutch/Tannen, "Provenance for aggregate queries"** — **CONFIRMED 3-0**: PODS
    2011 (pp. 153-164, DOI 10.1145/1989284.1989302), K-semimodules + tensor construction K⊗M
    (aggregate values as formal sums k₁⊗m₁+…). The semiring→aggregation bridge EXISTS; nobody
    has pointed it at label taint. getML claims remain extraction-level (unverified).
- **Refuted in verification** (0-3): "Orion's factorization is enabled by aggregate class" (ground
  the ladder in Gray/DBSP/Olteanu, not Orion); an over-broad LARA FO+aggregation expressiveness
  claim.
- **Still open after both passes:** transductive extension of Moscovich-Rosset (their abstract is
  explicitly inductive-only; flags further research); whether anyone anywhere did CV-folds-as-
  deltas (targeted search of differential-dataflow/Feldera + ML-systems communities warranted
  before claiming novelty); aggregation semimodules as the y-taint-through-aggregates bridge.
