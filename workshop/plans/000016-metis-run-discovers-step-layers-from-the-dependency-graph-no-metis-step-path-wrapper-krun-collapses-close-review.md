# Boundary Review — metis#16 (whole-issue close)

| field | value |
|-------|-------|
| issue | 16 — metis run discovers step layers from the dependency graph — no METIS_STEP_PATH wrapper (krun collapses) |
| repo | metis |
| issue file | workshop/issues/000016-metis-run-discovers-step-layers-from-the-dependency-graph-no-metis-step-path-wrapper-krun-collapses.md |
| boundary | whole-issue close |
| milestone | — |
| window | 08298a090a4018ad43b1c97a49d11701eea4be87..HEAD |
| command | sdlc close --issue 16 |
| reviewer | claude |
| timestamp | 2026-07-06T23:05:44-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All claims verified. Writing the review.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** This is a clean, well-scoped implementation that delivers the issue's purpose: `metis run` now discovers its step-path by walking the workspace's `construct/deps` chain via `ariadne/pkg/layergraph` — the same topology source `weave` reads — with no `METIS_STEP_PATH` and no `krun`. I verified the code against the plan's Core-concepts table (every entity exists at the stated path with the stated kind/status), confirmed `layergraph.Walk`'s base-first / EvalSymlinks-canonicalized contract in the ariadne source, confirmed metis's own `construct/` setup makes the new branch safe (metis → [ariadne, metis] → drops ariadne → `[metis/steps]`, same as before), and ran the changed-package tests (green) plus `go build ./... && go vet ./...` (clean). The core mechanism is proven by a genuine hermetic invocation-path test (`TestStepPath_DiscoversLayersFromDepChain`) driving the *real* `stepPath` + *real* `execStep.resolve` over a fabricated 3-layer dep chain. Nothing blocks SHIP; the two items below are advisory follow-ups.

**1. Strengths**
- `stepPathFromLayers` (`cmd/metis/steppath.go:56`) is a genuinely pure unit — the `exists func(string) bool` injection lets `TestStepPathFromLayers_LeafFirstDropsNoSteps` pin the base-first→leaf-first reversal and the no-`steps/` drop with zero IO (ARCH-PURE done right).
- ARCH-DRY is fully honored on both axes: `repo.FindUp` generalizes the up-walk and `Root` delegates (`internal/repo/repo.go:33`), and `layergraph.Walk` is *consumed*, not re-parsed — so metis derives its layer topology from the one source `weave` uses rather than inventing a second `construct/deps` parser.
- The invocation-path tests (`steppath_test.go:47`, `:98`) exercise the real resolver end-to-end, not a mock reasserting the implementation — a workspace step shadowing a base-layer step (`TestStepPath_NearestLayerWins`) pins the actual nearest-wins policy that first-match `resolve` delivers.
- The broken-graph handling (`steppath.go:31`) surfaces `layergraph`'s actionable #155 error to stderr instead of silently degrading to a misleading `[steps]` — the right call for a returns-`[]string`-with-no-error seam, and the atlas reflects it accurately.

**2. Critical findings** — none.

**3. Important findings**
- **Test coverage: the new error/degrade branch is unexercised** (`cmd/metis/steppath.go:31-39`). No test covers (a) `layergraph.Walk` returning an error (the #155 present-but-broken-peer case) → stderr note + fall-through, nor (b) anchor-found-but-Walk-succeeds-with-no-`steps/`-layer → fall-through. The mitigating factor is that both paths are *loud* by design (a shipped bug here would be visible, not silent), and (b) lands on the pre-existing `repo.Root(cwd)/steps` fallback — so this is Important-but-cheap, not blocking. Fix sketch: a fixture where a `substrate` target is present on disk but missing `construct/base.manifest`, asserting `stepPath` degrades to the fallback (and, if you want, capture stderr to assert the note fires).

**4. Minor findings**
- Test prefix-boundary robustness: `strings.HasPrefix(exe, wr)` (`steppath_test.go:82`, `:114`) would false-positive if one layer root were a string-prefix of another; harmless for the current fixtures (`metis`/`kaggle`/`kbench`/`ariadne` — none prefixes another) but `filepath.Rel`-based containment or a separator-terminated prefix is sturdier.
- Done-when bullet 1 names `titanic-features.md`; the `## Log` proof used `titanic-baseline.md`. Both resolve all three layers, so the proof is equivalent — cosmetic mismatch only.

**5. Test coverage notes**
- Pure ordering/filter: covered. Real dep-chain discovery + resolution across all three layers: covered. Nearest-wins clash: covered. Env override verbatim: covered. `repo.FindUp` hit + no-marker error: covered; existing `TestRoot_*` still green through the delegate (verified). Gap is only the discovery **error/empty** branch (item 3). The real-kbench e2e (Done-when bullet 1) is a `## Log` *claim* I cannot verify from the diff — but the hermetic automated test proves the same mechanism against a real `construct/deps` + `steps/` tree, so the mechanism is genuinely pinned regardless.

**6. Architectural notes for upcoming work**
- ARCH-DRY: **pass.** ARCH-PURE: **pass** (pure policy fn injected into the thin `stepPath` seam; `layergraph.Walk`/`os.Stat` are the only IO, at the boundary). ARCH-PURPOSE: **pass** — the single-source shadow-sweep confirms metis *derives* its step-path from `layergraph` (the enforced topology source), leaving no hand-maintained restatement of the dep graph; the `krun` collapse is correctly scoped as a separable kbench follow-up per the issue's own Spec, not a deferred core purpose. For that follow-up: the atlas already flags that leaf-first **inverts** krun's base-first precedence — the collapse must not assume byte-identical resolution (harmless today only because `metis`/`kaggle`/`titanic` namespaces are disjoint).

**7. Plan revision recommendations**
- `workshop/plans/000016-metis-step-layer-discovery-plan.md`, Task 3 "Step 3: Implement" snippet: the plan's code + comment say a `layergraph.Walk` error is *"deliberately swallowed … degrade to the bare-repo fallback"* with a bare `if order, err := …; err == nil`. The shipped code instead **prints the error to stderr** before degrading (`steppath.go:31-36`), matching the atlas ("surfaces layergraph's actionable error rather than degrading silently"). Add a `## Revisions` entry noting the error is surfaced (loud), not swallowed, so the plan stops claiming behavior the code improved on.
