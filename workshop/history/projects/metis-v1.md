---
type: project
name: metis-v1
goal: "Evolve metis from a v0 walking-skeleton runner into a v1 ML workbench that lets an engineer explore many configurations reproducibly, without getting overwhelmed."
done_when: "An `experiment-shape` sweeps a config-space (grid) into many pinned, content-addressed, reproducible runs recorded in a navigable free-param-keyed ledger, with promotion of the best point to a standalone `experiment` — proven by the Titanic acceptance demo (kbench#4): a features×model×hyperparams sweep whose promoted winner is live-submitted to beat the v0 baseline of 0.76794 (or honestly records a non-improvement)."
status: done
operator: xianxu
mvp_scope: [metis#9, metis#3, metis#2, metis#6, metis#7, metis#8, kbench#3, kbench#4]
explicitly_out: [metis#4, metis#5, metis#10]
created: 2026-07-03
updated: 2026-07-07
closed: 2026-07-07
sources: [workshop/pensive/2026-07-03-01-pensive-experiment-shape-workbench-design.md]
---

# metis-v1

The workbench layer on top of the v0 base layer (`kaggle-ml-base-layer`, done). v1 is
about **trying many configurations without drowning**: a lifted config-space
(`experiment-shape`) over the single-config `experiment`, swept into pinned runs and
recorded in a navigable, human-keyed ledger. The concrete "why" — and the acceptance
test — is the **Titanic improvement sweep** (kbench#4): express feature/model/hyperparam
improvements, sweep them, pick the best, and beat the v0 baseline of 0.76794.
**Headline omissions (deliberately NOT in
MVP):** (1) combining independent modeling *streams* into a final model — the "multiple
people experiment on different portions, merge later" problem — is out entirely; (2)
intelligent sampling beyond **grid** — the sampler seam (`propose_next`/`should_stop`)
is built so Optuna/Ax/Hyperband slot in later, but only grid ships; (3) the cloud CAS
(S3) — local filesystem only, which covers all learning-scale work. Adjacent
workbench-quality issues (step manifests metis#4, describe/EDA metis#5, mid-sweep
code-freeze hardening metis#10) are `explicitly_out` — useful, not v1-core.

The full design is the source-of-truth pensive (see `sources`). Execution order below is
top-down; the first `[ ]` is next.

**Design phase complete (2026-07-05):** all MVP issues carry settled `## Design` notes — the six
metis issues (#9/#3/#2/#6/#7/#8: storage/cache/identity + experiment-shape/sweeper/ledger) and the
two Titanic demo issues (kbench#3 feature blocks, kbench#4 acceptance sweep). Ready to implement in
sequence. Continuation anchor: `brain/workshop/continuation/20260705T121804-metis-v1-build-start.md`.

## tasks

- [x] CAS blob store — content-addressed put/get, size-bounded eviction (storage floor) [metis#9]
- [x] L0 — resolved-point identity + unified per-step record (repro/provenance atom) [metis#3]
- [x] content-addressed step caching — validating trace over the record (pulled up: cache-early for dev speed) [metis#2]
- [x] experiment-shape datatype: lift the config schema into a space (Space[T]) [metis#6]
- [x] sweep runner + grid sampler (propose_next / should_stop seam) [metis#7]
- [x] shape run-ledger: CSV sidecar keyed by free-param tuple + promotion [metis#8]
- [x] Titanic feature-engineering building blocks (toggleable groups) [kbench#3]
- [x] Titanic improvement sweep — pick best, beat 0.76794 (v1 acceptance demo) [kbench#4]

## details

### CAS blob store [metis#9]
The storage floor: a generic content-addressed blob store (`put`/`get`/`has` by content
hash, integrity-verify, size-bounded eviction), mechanism-only. In v1 it's a **pure wipeable
cache** (`rm -rf cas/` always safe — wiped entries recompute), behind a swappable interface
(S3 later). Sole MVP consumer is #2 (+ pointer materialization); #3's record only *references*
CAS addresses (hashing ≠ storing), so #3 doesn't depend on it. Split out of #2 during the
caching design so mechanism (store) stays separate from policy (cache).

### L0 — point-address + unified per-step record [metis#3]
Design-first per the operator ("design this well"). The whole v1 keys off a resolved-point
identity — but **two keys, two jobs** (settled alongside #2's design, 2026-07-03): #3's point
content-address `(resolved-with, three repo SHAs, seed)` is the *repro/provenance* atom that
#8's global ledger key derives from; #2's *cache* key is a separate **per-step content-trace**
(deliberately not the repo SHAs — whole-repo SHAs over-invalidate, which is the problem #2
solves). They share only `resolved-with + seed`. Now **unified**: #3 owns the *per-step
record* whose key-material #2 hashes (cache key) and whose point-address #8 derives — one raw
record, hashed late into three views (metis#3 `## Design`).
**DONE 2026-07-05** (est 1.9h / actual 0.94h): `pkg/record` (pure leaf over `cas`) — `RunRecord`/
`StepRecord` (→ `runs/<id>/record.json`, CUE-drift-guarded), `PointAddress` (canonical config+repo-SHAs+seed
→ `cas.HashOf`), `OutputHash` (multi-file reducer); `Runner.Run` returns `[]StepRun`; `cmd/metis` assembles
the record (git provenance via an injected `gitProbe`, graceful no-git degradation), writes `record.json`,
renders the knob→score `## Runs` line. M1 (pure core) + M2 (integration) both review-SHIP; verified in the
real CLI. Trace/cache-key deferred to #2; side-ref capture to #7/#8.

### content-addressed caching [metis#2]
Design-first per the operator ("don't hurry; uniform + consistent"). Makes sweeps cheap
(shared upstream steps aren't recomputed per point). **Design settled 2026-07-03** (metis#2
`## Design`): a single *validating trace* — ex-ante `K_pre` + recorded read-set `D`, validate
by re-hashing `D` (code-version invalidation for free); two input classes (keyed immutable
artifacts vs. traced source files) with network/env/clock quarantined at ingest leaves under
a pin/etag policy. Sits on the CAS (#9) and indexes #3's unified record. **Sequencing changed
2026-07-03:** pulled up together with #9 (cache-early — faster dev-loop while building the
sweep, no Kaggle re-downloads across sweep points) rather than sequenced after the sweep as a
pure cost-optimization.
**DONE 2026-07-05** (est 3.0h / actual 1.57h; 3 milestones): `pkg/cache` (pure `Kpre`/`Validate`/`Entry`)
+ a Python `sys.addaudithook` read-sensor (`metis/trace.py` → `reads.json` first-party code closure) + Go
`gitBlobHashes`/`buildD` + the `cachingExecutor` (K_pre lookup → HIT materialize from a `pkg/cas` FSStore +
skip / MISS run+store). `uv.lock` folded into D (dep-upgrade invalidation); leaf policy for pinned fetches;
`metis run --cache`. **Cheap sweeps proven** (identical re-run HITs all; one-knob change re-runs only
downstream; real-uv pipeline reproduces cv_score from cache). Record `Code.D` provenance deferred to #8.

### experiment-shape datatype [metis#6]
Lift the v0 experiment config-space into `experiment-shape`; `expand(shape) → [point]`, with
`#Experiment` the all-singleton collapse (single-source). **Design settled 2026-07-04**
(metis#6 `## Design`): the lift is **value-level** on v0's untyped `with` bag (reserved
`$`-keys), not CUE-typed leaves — algebra of `$any` (set) / `$oneof` (labeled conditional sum)
/ product / `$linear-range`·`$log-range` (domain+metric, not a distribution); a `sweep:` block
(sampler/objective/range_steps) in-frontmatter; `expand()` bundles `$oneof` and emits
v0-shaped `with`. The L2 substrate #7 (sweeper) and #8 (ledger) build on.
**DONE 2026-07-05** (est 2.0h / actual 0.55h; both milestones review-SHIP-crossable): `pkg/shape`
(pure `Expand` — the 36-point titanic keystone proves `$oneof` ADDs; ragged free-param paths; grid
ranges); `experiment.Shape`/`ParseShape`/`ValidateShape`; CUE `#ExperimentShape` single-sourced with
`#Experiment` via a shared `_pipeline`; `metis run` on a shape (singleton→runs like v0; multi-point→#7
pointer). Verified in the real CLI. #7 (sweeper) + #8 (ledger) build on this.

### sweep runner + grid sampler [metis#7]
Depends on #6 (+#3 for run identity). **Design settled 2026-07-05** (metis#7 `## Design`): one
driver — `metis run` handles experiment (1 point) and shape (N) uniformly; a **stateful ask/tell**
sampler (seeded), grid now, adaptive later with no loop change; run id = the point's
content-address; the **shape-run** gets a content identity grouping/stamping its N point-runs;
mid-sweep code mutation → **detect-and-abort** for v1 (hermetic+perf upgrade → metis#10);
per-point failure → recorded `failed` row, sweep continues. #8 owns pick-best/promote.
**DONE 2026-07-05** (est 1.9h / actual 0.63h; M1 sampler SHIP + M2 driver): `pkg/sweep` (pure ask/tell
Grid + MaxPoints/TargetReached stops); `metis run` on a multi-point shape SWEEPS (`runSweep` loops
Ask → the shared `runResolvedExperiment` cached runner keyed by `record.PointAddress` → Tell;
per-point-failure-continues; shape-run manifest = the #8 handoff); `--max-points`/`--dry-run`;
detect-and-abort on HEAD-sha drift. **Cheap sweeps verified in the real CLI** (shared upstream HITs
the cache across points). Real-CLI caught a freeze bug (whole-repo dirty flag tripped by the sweep's
own outputs) — fixed to HEAD-sha-only. #8 (ledger/pick-best/promote) builds on the manifest.

### shape run-ledger + promotion [metis#8]
Depends on #6, #7, #3. The L1 tracking that actually solves "don't get overwhelmed": a
queryable CSV sidecar, human free-param key + global content-address per row, top-N body
summary, and promotion (row → standalone experiment). The durable spine = the sequence of
promotions. **Design settled 2026-07-05** (metis#8 `## Design`): **append-only** ledger, row
identity = point-address, three keys (free-param tuple [ragged/sparse] + sweep-SHA + point-address),
namespaced metrics, objective-driven pick-best; **promotion is a command** → all-singleton
experiment committed at its code SHA (self-contained); the ledger is the lifted `## Runs`
(experiment = 1-config). Carries the **git-owns-code durability refinement** (updates #2/#3/#9):
a sweep commits its traced code closure to a side ref; metis keeps a `(path, git-blob-hash, commit)`
pointer-manifest; the CAS holds only wipeable large-output bytes.
**DONE + MERGED 2026-07-05** (est 3.00h / actual 2.73h, ratio 1.1×; PR #7): full settled design
delivered across M1 (pure `pkg/ledger` — ragged append-only CSV + dedup-by-point-address + objective
pick-best) / M2 (integration — `rowsFromManifest`, `metis ledger show`, `metis promote` round-trip) /
M3 (side-ref dirty-code capture — `refs/metis/sweeps/*`, backfills `CodeManifest.D`/`Commit`).
Boundary review: REWORK → 6 close-review rounds → **SHIP** (0 Critical/Important); each fix
TDD-regression-proofed, recurring lesson = drive the REAL CLI (promote-never-commits, arg-order,
freeze dirty-flag all slipped past direct-call e2es). Residual dirty-run row-fidelity (HEAD-based
`SweepSHA` vs `CodeManifest.Commit`) tracked as **metis#10**.

**► metis-v1 SUBSTRATE COMPLETE (2026-07-05)** — all six infra issues merged (#9→#3→#2→#6→#7→#8).
The workbench spine is live: shape → sweep → content-addressed cached runs → navigable free-param
ledger → promote. Remaining MVP scope is the Titanic acceptance demo (kbench#3 → kbench#4) that
*uses* the substrate to close `done_when`.

### Titanic feature-engineering building blocks [kbench#3]
The "building blocks on the side" — the classic Titanic features (title, family-size,
age impute+bin, fare-per-person, deck/has-cabin, embarked) as pure, independently
**toggleable** transforms behind a `features` knob. Independent of the v1 infra
(just richer pipeline steps), so buildable early to derisk the demo; `features: []`
reproduces the v0 5-feature baseline exactly. **Design settled 2026-07-05** (kbench#3 `## Design`):
new `titanic/features` step, knob = a runtime-sorted set; deliberately **ad-hoc titanic code** —
the conformance testbed that the substrate absorbs arbitrary model code (no generic abstraction).
**DONE + MERGED 2026-07-05** (est 3.14h / actual N/A — fork-agent impl, active-time-v3 can't measure it; PR #2): 6 pure toggleable groups (each fits imputation on **train only** — no leakage) + shared `_derive_*` helpers + the `CANONICAL_ORDER` selector (order-independent → value-identical Dataset) + `features: []` pass-through anchor + the `titanic/features` step + a features-thread hermetic e2e. Review: plan-quality round-1 FAILURE (blocking trace-sensor F1 → **metis#11** filed) → change-code pass; close-review FIX-THEN-SHIP (1 Important: stray probe artifact) → **SHIP**. 34 tests incl. both e2e threads. Conformance-testbed **proven**: the substrate absorbed the ad-hoc titanic code with no generic abstraction. Two substrate follow-ups surfaced: **metis#11** (cross-repo trace root — features not cache-validated) + a committed-dir-cache robustness note.

### Titanic improvement sweep — the acceptance demo [kbench#4]
**This closes `done_when`.** A `titanic` experiment-shape sweeping `features × model ×
hyperparams` → grid sweep → ledger → pick best by cv_score → promote the winner →
operator live-submit to **beat 0.76794**. The concrete "why" of v1: express improvements
to the Titanic submission, sweep, pick the best, without drowning. Last in the chain —
needs kbench#3 + metis#6/#7/#8 (and benefits from #2). The live-submit clause mirrors
v0's operator live-run (needs kaggle creds); an honest non-improvement is still a valid
workbench demo. **Design settled 2026-07-05** (kbench#4 `## Design`): assembly/acceptance; shape =
#6's worked titanic-sweep; **offline sweep + submit the winner once**; rank by cv_score (honest
about the cv→public gap).
**MECHANISM DONE + MERGED 2026-07-06** (est 1.61h / actual N/A — interleaved with metis#12; PR #3):
`titanic-sweep.md` (42 pts) proven end-to-end on the hermetic fixture — sweep → ledger ranks by
`train.cv_score` → `promote --best` round-trips (result reproduces). A 4-point full-thread smoke
(`titanic-sweep-smoke.md`) + e2e guards the composition (differentiated cv_scores, each point emits
a submission). Operator runbook `pipelines/RUNBOOK-sweep.md` for the creds-gated real run.
Review: REWORK-free; close FIX-THEN-SHIP → **SHIP** (3 rounds; Docs/test-fidelity fixes). **The
acceptance demo did its job** — it surfaced the one real substrate gap (**metis#12**: `metis/train`
didn't consume the `$oneof` model bundle nor apply hyperparams — every point failed), which was
filed, fixed (TDD), merged, and re-validated before proceeding.
**► THE `0.76794` BEAT-CLAUSE IS THE OPERATOR'S PENDING ACTION** (creds-gated: real-data
`kaggle/download` + `kaggle/submit`, both need `~/.kaggle/kaggle.json`). Everything up to it is
built + proven; run `kbench/competition/titanic/pipelines/RUNBOOK-sweep.md` to close it.

## Status — DONE_WHEN MET (2026-07-07)

**✅ `done_when` SATISFIED — the acceptance demo beat the baseline.** Operator live-submitted the
promoted sweep winner to Kaggle: **`public_score = 0.77990` > the v0 baseline `0.76794`**
(submission `metis-v1 sweep winner`, 2026-07-07). A modest but real improvement, produced by the
workbench's systematic `features × model × hyperparams` sweep + pick-best + promote — exactly the
"explore many configs without drowning, and beat the number" the project set out to prove.

**The metis-v1 workbench is BUILT, PROVEN, and its acceptance test PASSED.** All 8 MVP issues merged
(metis#9/#3/#2/#6/#7/#8 + kbench#3/#4); the full spine works end-to-end: shape → grid sweep →
content-addressed cached runs → navigable free-param ledger → promote winner → reproducible
experiment → **live submission that beat v0**.

Substrate follow-ups completed after the demo (the reproducible-dirty-run effort): **metis#12**
(sweep-train hyperparam gap, the kbench#4 blocker), **metis#13** (config immutability), **metis#11**
(cross-repo trace multi-root), **metis#14** (capture run-spec + single-run + loud), **metis#15**
(sensor target-capture + data-exclusion), **kbench#5** (wrappers → `metis.trace`) — all DONE+MERGED,
so a dirty `features.py` edit is now genuinely captured + cache-invalidated. Layering
refinements (post-v1, not blocking) — **both DONE+MERGED 2026-07-07** (full SDLC, SHIP): **metis#16**
(`metis run` discovers its step-path by walking the workspace's `construct/deps` chain via the
already-public `ariadne/pkg/layergraph` — weave's own topology source; no `METIS_STEP_PATH`/`krun`
needed; leaf-first = nearest-wins) + **kaggle#5** (thin `kaggle submit --run <id>` reusing an extracted
`internal/submit` shared with the step; slug from `record.json` or `-c`; prints `public_score`) +
**kbench#6** (`bin/krun` collapsed → `metis run` — e2e execs metis directly, docs swept, wrapper
deleted, e2e green; side-quest: RUNBOOK now uses `kaggle submit --run`). **All three merged — the
layer model is fully realized: metis owns run+step-discovery, kaggle owns steps+submit CLI, kbench is
a pure workspace with no wrapper.**

**Project status → this MVP is complete.** (Bigger scores are now a modeling exercise the workbench
makes cheap — widen the sweep; the `done_when` proof is banked.)

[metis#9]: ../../metis/workshop/issues/000009-cas-blob-store.md
[metis#3]: ../../metis/workshop/issues/000003-run-provenance.md
[metis#6]: ../../metis/workshop/issues/000006-experiment-shape-datatype.md
[metis#7]: ../../metis/workshop/issues/000007-sweep-runner-grid-sampler.md
[metis#8]: ../../metis/workshop/issues/000008-shape-run-ledger.md
[metis#2]: ../../metis/workshop/issues/000002-step-caching.md
[kbench#3]: ../../kbench/workshop/issues/000003-titanic-feature-blocks.md
[kbench#4]: ../../kbench/workshop/issues/000004-titanic-improvement-sweep.md
