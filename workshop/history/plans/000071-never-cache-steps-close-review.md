# Boundary Review — metis#71 (whole-issue close)

| field | value |
|-------|-------|
| issue | 71 — Step-level never-cache policy with safe downstream invalidation |
| repo | metis |
| issue file | workshop/issues/000071-never-cache-steps.md |
| boundary | whole-issue close |
| milestone | — |
| window | 27995863bb76020f322317132dc2c1df7783e24b..83edcb9f490a3af0cce3078f6c2fe284a267a497 |
| command | sdlc close --issue 71 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-25T07:52:22-07:00 |
| verdict | SHIP |

## Review

Diff and issue read; now running the package tests (including the CUE drift guard if `cue` is present).

Tests pass. Now doing a mutation check: disabling the never-cache branch and the descendant propagation separately to confirm the sequence test catches each.

```verdict
verdict: SHIP
confidence: high
```

This change meets the Spec and all four Done-when items, and I found nothing that blocks shipping. There is one parser, `experiment.ParseCachePolicy`, and both `Validate` and the executor's `isImmutableLeaf` use it. `UncachedSteps` is a pure, memoized closure over `needs` that marks every descendant of a `policy: never` step. The caching decorator adds a single early branch: an uncached step skips lookup, store, the `kpres` entry and the `transitiveD` entry. I ran the tests against the pinned ariadne substrate. To do that I replaced `ariadne` with a temporary `-modfile` and pointed `GOTMPDIR`/`TMPDIR` inside the checkout, because the reviewer `git` refuses repos outside its directories. With that setup, the new tests and all existing cache-soundness tests pass, including the `cue vet` drift guard (`cue` is on PATH here). I also ran two mutation checks, each of which turns the new sequence test red:
- **Propagation turned off:** `score` gets the stale `"score-0"` for a changed candidate, and an extra index entry appears.
- **Executor branch turned off:** `session` and `score` both get cache hits, and the stale score is served.

The findings below are all Minor.

1. **Strengths**
   - `pkg/experiment/cachepolicy.go:19-46`: one typed parser for `with.cache`, shared by `Validate` and `isImmutableLeaf` (`cmd/metis/caching.go:383-385`). This also ends the old silent behaviour where a malformed `leaf` was treated as "not a leaf" (ARCH-DRY).
   - `pkg/experiment/cachepolicy_test.go:19-36`: one table of cases (`cachePolicyCases`) drives both the Go parser test and the `cue vet` drift guard. The only difference between the two validators, leaf + never together, is written down in the table (`cueOK`).
   - `cmd/metis/caching_soundness_test.go:285-337`: the sequence test goes through the real decorator. It covers unchanged and changed realized outputs, checks the stored index entries, and checks that the cacheable root above the session still hits. Both mutations above make it fail.
   - `UncachedSteps` fails closed: a malformed policy is treated as uncached. It also terminates on cycles, and the out-of-topo-order and cycle cases are tested.
   - The atlas (`atlas/index.md:241-249`, `atlas/experiment.md:33-34`) records the chosen rule, the rejected alternative and why, and states that determinism is never inferred from a missing `reads.json`. The Spec asks for all three.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - **Plan item not done as written.** The plan says to update "the hand restatement in `pkg/experiment/conformance_test.go`", but that file is unchanged. A new guard, `TestCachePolicyConformsToCUE`, covers the drift instead, which is arguably better. The plan text should say so.
   - **Stricter `Validate` can reject old files (ARCH-EVOLVE).** A key named `cache` used to be accepted with any value, because `with` is a free-form map. Now anything other than `{leaf: immutable}` or `{policy: never}` is refused. No experiment in the repo uses `cache:` (I grepped), and the refusal is intended per the Spec. The atlas mentions the refusal but does not call it a breaking change.
   - **Memo on cycles.** In a cycle, `UncachedSteps` saves a step's result while part of the cycle is still being visited, so the result may be incomplete. This can't matter in practice: `Validate` rejects cycles before execution, and the code comment says so.
   - **`newCachingExecutor` signature.** It now takes the whole `Experiment` rather than `steps`. That's fine; the plan wording just differs slightly.

5. **Test coverage notes**
   - Covered: the parser table (9 cases plus absent), `Validate` refusing a bad policy through the real YAML `Parse`, `UncachedSteps` on chain, diamond, out-of-order, fail-closed and cycle cases, the decorator sequence test, and the CUE drift guard.
   - Existing tests still pass: `UpstreamOutputChangeDoesNotRekeyDownstream`, `HitFeedsDownstreamClosure` and the leaf tests.
   - There is no end-to-end test that `Runner.Run` refuses a bad policy before any step runs. This is low risk: `pkg/experiment/run.go:62` calls `Validate` before `TopoSort` and before execution.

6. **Architecture notes**
   - **ARCH-DRY: pass.** One parser, one table of cases.
   - **ARCH-PURE: pass.** The parser, closure and `Validate` are pure; the executor adds one branch of glue.
   - **ARCH-PURPOSE: pass.** Propagation covers every consumer of `kpres` and `transitiveD`, because the uncached set includes all descendants. Human-grade steps use the same domain-free `policy: never`.
   - **ARCH-MOCK: pass.** There is no new external dependency, and the tests reuse the existing `traceFakeExec` and git fixture.
   - **ARCH-CONSTRAINTS: pass.** The closure is computed once per executor in O(steps+edges).
   - **ARCH-SECURE: pass.** The author's input is turned into a typed value in `Validate`, before execution.
   - **ARCH-ORDER: pass.** The uncached set is read-only after construction, and no new state carries across events.
   - **ARCH-FUNERAL: pass.** Uncached steps write nothing durable. This is verified: the index holds only root's entry.
   - **ARCH-EVOLVE: pass, with a note.** The format change is additive. An older metis would silently cache `policy: never` steps; the plan accepts this because couch-bench pins its metis version. The stricter `Validate` is noted above.

7. **Plan revision recommendations**
   - Add a "## Revisions" entry: "The `conformance_test.go` restatement item was replaced by `TestCachePolicyConformsToCUE` in `pkg/experiment/cachepolicy_test.go`, which runs the shared Go table through `cue vet -d #Step`. `newCachingExecutor` takes the `Experiment` rather than steps + seed."

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: plan-item-delivered-differently
    title: |
      Plan item to update the hand restatement in pkg/experiment/conformance_test.go was not done as written
    detail: |
      conformance_test.go is unchanged; the drift guard was delivered as TestCachePolicyConformsToCUE in pkg/experiment/cachepolicy_test.go instead. Record this in a plan Revisions entry so the plan matches the code.
  - id: new
    severity: Minor
    kind: defect
    family: validate-strictness-breaking-change
    title: |
      Validate now refuses a with.cache key that was previously accepted in the free with map
    detail: |
      Before this change any value under with.cache passed Validate. Now only leaf: immutable and policy: never are accepted. No in-repo experiment is affected (grep found none), but the atlas should call this a breaking change for consumer experiment files.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"299107cc-863a-41a4-bd1c-1b5170c62983","repository":"github.com/xianxu/metis","issue_id":"000071","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/a2accd2b89efe773","reviewed_head":"83edcb9f490a3af0cce3078f6c2fe284a267a497","slot":"metis:1","requested_at":"2026-10-25T14:50:14.121233Z"},"phase":"finished","pid":19755,"started_at":"2026-10-25T14:50:14.339064Z","finished_at":"2026-10-25T14:52:20.584406Z","session_id":"c210067d-1731-44e0-a7de-e01a7203eff7","outcome":"returned"}] -->
