---
id: 000071
status: codecomplete
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-25
estimate_hours:
collaboration-mode: collaboration
card_mirror: '108b584f44c65b5fd66eca8464a58f9acaa257a8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T17:13:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
flow: {kind: full, provenance: inferred}
actual_hours: 0.19
---

# Step-level never-cache policy with safe downstream invalidation

## Problem

ariadne#577 is being re-planned as a Metis consumer (couch-bench). An agent session and a human grading step are deliberately nondeterministic; a cache hit would erase a repeat or reuse a person's old judgment. The current cache has only a global switch, and downstream keys use upstream input identities, so simply bypassing the session cache can still reuse a stale score.

## Spec

Add a step-level never-cache policy at the existing StepExecutor/cache boundary. It bypasses both lookup and storage for that execution. Preserve ordinary defaults. Ensure descendants cannot reuse results on input identity alone after an uncached nondeterministic producer: either propagate non-cacheability conservatively or bind reuse to the realized upstream artifacts. Document the chosen rule; do not infer determinism from absent reads.json. Update the vocabulary and its consumers through their existing owner.

## Done when

- A never-cache session executes on every repeat even with identical configuration and seed; no cache entry can substitute for it.
- A changed realized session candidate cannot reuse an old downstream hidden score. A sequence test exercises both unchanged and changed outputs through the real cache decorator.
- Ordinary deterministic steps retain existing hit/miss and transitive code-invalidation behavior.
- Human-grade steps can request the same policy without domain-specific logic in Metis; malformed/unknown policy values refuse before execution.

## Plan

**Rule (chosen): conservative propagation.** A step is *uncached* when its own `with: {cache: {policy: never}}` is set **or any transitive `needs` ancestor is uncached**. An uncached step skips lookup and store entirely (no index entry, no CAS put, no `kpres`/`transitiveD` write); it always runs `inner.Execute`. Everything outside that closure keeps today's input-addressed hit/miss and transitive-D invalidation, unchanged.

*Rejected: binding descendants to realized upstream artifacts* (fold `record.OutputHash` + metrics into the uncached step's `kpres`). It is sound only if the declared artifacts capture everything a descendant reads. A couch-bench session's candidate likely lives outside the run dir (a worktree/ref), metric-only outputs are not in `OutputHash`, and metis cannot verify completeness (absent reads.json proves nothing). For nondeterministic producers the outputs differ on almost every repeat anyway, so the reuse it would buy is near zero. Propagation is sound without that assumption; the cost is that a deterministic scorer under a session reruns each repeat, which repeats need regardless.

**Policy surface:** stays in the existing `with.cache` sub-map, next to `leaf: immutable` (one home for cache policy, ARCH-DRY). Allowed: `leaf: immutable`, `policy: never`. Refused at `experiment.Validate` (which `Runner.Run` and `ValidateShape` already call before any execution): `cache` not a map, an unknown key, an unknown value, or `leaf` + `policy: never` together. This also ends today's silent "malformed leaf means not a leaf". Metis stays domain-free: a human-grade step uses the same `policy: never`.

- [x] `pkg/experiment`: a typed `CachePolicy` parsed from `with.cache` (pure; one parser for Validate and the executor; `isImmutableLeaf` re-expressed over it); `Validate` refuses malformed values. Pure `UncachedSteps(steps) map[string]bool` (closure over `needs`, O(steps+edges)).
- [x] `cmd/metis/caching.go`: `newCachingExecutor` takes the steps (alongside seed), computes the set once; `Execute` branches first: uncached → print `step X (never-cache)`, run inner, return. Only glue in the IO shell (ARCH-PURE).
- [x] Vocabulary: `construct/vocabulary/experiment.cue` `#Step.with` gains `cache?: #CachePolicy` (closed: `leaf?: "immutable"`, `policy?: "never"`), and the rest of the map stays open; update the hand restatement in `pkg/experiment/conformance_test.go`.
- [x] Tests:
  - Validate table: each refusal above plus valid forms.
  - `UncachedSteps`: chain, diamond, unrelated sibling branch.
  - Sequence test through the real decorator (`runSound`/`traceFakeExec` template, `cache: true`, same config and seed). up=`policy: never`, down=deterministic: three runs with the nonce fixed (unchanged output), then a changed nonce. Both up and down execute on every run, and neither leaves an index entry. A cacheable root above `up` still hits.
  - Existing soundness tests unchanged: `UpstreamOutputChangeDoesNotRekeyDownstream` stays true for cacheable upstreams.
- [x] Atlas: the `pkg/cache` entry in `atlas/index.md` and `atlas/experiment.md` document the rule, the rejected alternative, and that determinism is never inferred from a missing reads.json.

**ARCH notes.**
- PURE: the parser, the closure and Validate are pure; the executor adds one branch.
- EVOLVE: the experiment format change is additive. The cache index format is unchanged. An older metis would ignore `policy: never` and cache the step; couch-bench pins its metis version, so there is no version bump.
- FUNERAL: creates nothing durable; uncached steps write no cache entries.
- ORDER: no new cross-event state; the set is computed once per point executor and is read-only.
- SECURE: the experiment file is author input, parsed into a typed value at Validate.
- CONSTRAINTS: O(steps+edges) once per point, no hot path.
- MOCK: N/A, no external dependency.

**Size:** about 60 added code lines; quick flow, no Mx.

## Revisions

- 2026-10-25 (close review advisories) — delivered differently from the Plan:
  the drift guard is `TestCachePolicyConformsToCUE` in
  `pkg/experiment/cachepolicy_test.go`, which runs the shared Go case table
  through `cue vet` against `#Step`, instead of extending the hand restatement in
  `conformance_test.go`; and `newCachingExecutor` takes the `Experiment` (seed
  and never-cache set derived inside) rather than steps plus a seed.

## Log



- 2026-10-25: closed — with.cache.policy: never parsed and validated (TestParseCachePolicy, TestValidate_RefusesMalformedCachePolicy, CUE drift guard TestCachePolicyConformsToCUE); UncachedSteps propagates to descendants fail-closed (TestUncachedSteps); cachingExecutor runs a never step and its descendants every repeat while the root still hits (TestCachingExecutor_NeverCacheRunsEveryRepeatWithDescendants; mutation without the guard fails). Recorded suite passed on 83edcb9f490a3af0cce3078f6c2fe284a267a497 (go ./... ok, pytest 140 passed).; review verdict: SHIP
- 2026-10-25: flow upgraded quick → full — 120 added lines in code files (limit 100)
2026-10-23: Filed unclaimed for the operator-approved ariadne#577 re-plan. Seams: cmd/metis/caching.go cachingExecutor.Execute/kpre; pkg/experiment.Step and construct/vocabulary/experiment.cue. No implementation or production experiment changed.
2026-10-23 (metis:1): The slot carried uncommitted local-gates leftovers from slot setup that blocked the fast-forward to main 0350013: merge-check.yml and .gitignore modified (an older seed), .gitattributes and local-gates-guard.yml untracked (identical to main). The hub (ops:0) approved discarding them. By the time I ran the backup, the tree was already clean at origin/main 0350013, so I discarded nothing. The scratchpad backup holds only main's two untracked files; the older modified seed was never captured, and main's versions replace it.
2026-10-23 (metis:1): The hub approved the plan; implemented it on the quick flow. `pkg/experiment/cachepolicy.go`: ParseCachePolicy and UncachedSteps. UncachedSteps is a memoized DFS. It fails closed on a malformed policy (Validate refuses that first anyway) and terminates on cycles. The executor checks first for uncached steps, and `isImmutableLeaf` now goes through the same parser (ARCH-DRY). The CUE `#CachePolicy` gets a drift guard (`TestCachePolicyConformsToCUE`) that runs the Go table's cases through `cue vet`; the leaf + never contradiction is Go-only. Mutation check: with the executor branch disabled, the new sequence test fails with "score-0" served for a changed candidate. Environment note: `TestCache_ToyPipelineHitsOnRerun` builds a uv venv and needs pypi network access, so it fails under the default sandbox.
2026-10-23 (metis:1): Concurrent claims in this slot: #71, #75, #76 and #72 (in planning) are open at once while #75 waits on the operator CI pin (metis#70). Active-time attribution across them is approximate until ariadne#580 lands.
