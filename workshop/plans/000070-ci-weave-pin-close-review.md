# Boundary Review — metis#70 (whole-issue close)

| field | value |
|-------|-------|
| issue | 70 — CI: build weave from a pinned ariadne commit |
| repo | metis |
| issue file | workshop/issues/000070-ci-weave-pin.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4cb2c455fd62430e4ab25fdddeb940236c30f74d..c880dd42ca36d313e138407e3266940b529efde7 |
| command | sdlc close --issue 70 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T04:21:56-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This change adds one file and one atlas entry. The file is `.sdlc/weave-compiler-ref`, and it holds `63d6bfd86509d97649e10d7a5dd1e9a06db17d21` plus a newline. I checked it four ways:
- The validator copied from `.github/workflows/merge-check.yml:38-50` accepts it (`[0-9a-f]{40}\n`).
- The SHA exists as a commit in the pinned ariadne substrate.
- It matches the only entry in ariadne's `.sdlc/weave-compiler-refs`.
- Nothing under `construct/` manages it.

The atlas entry agrees with the workflow: the job fails when the pin is missing, malformed, or doesn't match what was built, and there is no fallback. It also agrees with the upstream §Compiler compatibility (#459) section, which says bumps go through operator-verified PRs. Its link anchor `#compiler-compatibility-459` points at a heading that exists. The remaining Done-when clauses (real cold-checkout CI acceptance, the operator applying the label, the hub report) depend on the PR. The plan honestly leaves that item unchecked, and they happen outside the repo. Nothing blocks SHIP.

1. **Strengths**
   - The scope is right: the seeded workflow already carries the template from upstream, so the only change is the authored pin. The plan says workflow edits are out of scope.
   - The pin file is byte-exact: 41 bytes, lowercase, ending in a newline, which is what the in-workflow validator requires.
   - `atlas/index.md:255-260` links to the upstream doc instead of restating its rollout and compatibility rules.

2. **Critical** — none.

3. **Important** — none.

4. **Minor**
   - The Done-when line "The consumer-authored pin survives compile" is proven only structurally, by the fact that no manifest names `.sdlc`. The log says `weave refresh` refused and `weave compile` stopped at `.claude/settings.json`, so no full compile ran with the pin present. That evidence is sufficient, but a full compile run should be recorded before close (or the PR's CI run noted as covering it).

5. **Test coverage**
   - The tests for malformed pins, fetch/build failure and PATH precedence live upstream, in `portable-ci.test.sh` and `weave-compatibility.test.sh`. That is appropriate because the workflow is seeded from upstream. This repo has no runtime code, so a local regression test isn't expected.

6. **Architecture**
   - **ARCH-DRY: pass.** The atlas entry is a pointer to upstream, not a copy.
   - **ARCH-PURE: not applicable** (no code changed).
   - **ARCH-PURPOSE: pass.** The pin is the missing piece. CI acceptance is tracked as the open PR step, not deferred to a follow-up.
   - **ARCH-EVOLVE: pass.** A new authored config file is additive and unmanaged; no existing format changes.

7. **Plan revisions** — none needed.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: done-when-evidence-partial
    title: |
      "Pin survives compile" is shown structurally, not by a full weave compile run
    detail: |
      The Log records that weave refresh refused and weave compile stopped at .claude/settings.json, so evidence rests on no manifest naming .sdlc (confirmed). Record a full compile run, or the PR CI run, as the evidence before closing. This is the only instance in the window.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"af406c07-04bc-4f84-a7ad-e15a006f0481","repository":"github.com/xianxu/metis","issue_id":"000070","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/5135b727569e6e8c","reviewed_head":"c880dd42ca36d313e138407e3266940b529efde7","slot":"metis:1","requested_at":"2026-10-25T11:21:06.557619Z"},"phase":"finished","pid":59050,"started_at":"2026-10-25T11:21:06.657592Z","finished_at":"2026-10-25T11:21:54.763846Z","session_id":"04f6914c-72f1-4681-9d8b-e16d755658ba","outcome":"returned"}] -->
