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

---

## Re-review — 2026-10-25T04:35:07-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 70 — CI: build weave from a pinned ariadne commit |
| repo | metis |
| issue file | workshop/issues/000070-ci-weave-pin.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0f272a94aaab3c14cbd5754b3e954bdfdba172e0..54810af9368b4cf28cc4168cff626a339c5ab46a |
| command | sdlc close --issue 70 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T04:35:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

This fix review should only have contained a fix for BR-1, but the window adds new CI behaviour that no prior finding asked for. There are three changes. `construct/deps` now clones over https instead of ssh. There is a new post-compile hook, `scripts/ci-setup.sh`, which appends `../ariadne/bin` and `bin` to `GITHUB_PATH`. And a new `scripts/merge-checks.d/_vocab.bash` guard makes both experiment checks exit 2 when the `vocabulary` tool is missing. The issue Log calls these "scope revision" and "second scope revision" (hub-approved). By rule 4 of the fix-review contract, a narrow review does not cover a scope change, so this window needs a full `sdlc close`. The code itself looks correct: the hook is wired in at `.github/workflows/merge-check.yml:166-167` and allowlisted at `.github/workflows/local-gates-guard.yml:60`, and the guard is sourced correctly. Separately, BR-1 is still not addressed. The Log records only the failing runs (40188584381 and 40188715202). The passing merge-check is claimed only in the subject of commit 54810af ("merge-check accepted"), and no run ID or result is written into the issue.

1. **Strengths**
   - `scripts/merge-checks.d/_vocab.bash:4-8`: a missing tool now exits 2 with its name, instead of looking like a schema rejection. That fixes the root cause of the misleading "valid-baseline.md should PASS but was rejected" failure.
   - `scripts/ci-setup.sh:9,11`: the script refuses to run outside GitHub Actions and refuses a missing layer directory, so it doesn't silently do nothing.
   - The hook names its removal trigger (ariadne#627) in both the script header and the Log.
   - Using one sourced helper for both experiment checks avoids copy-paste (ARCH-DRY).

2. **Critical:** none.

3. **Important**
   - Scope changed after close (rule 4): `construct/deps:1`, `scripts/ci-setup.sh` and `scripts/merge-checks.d/_vocab.bash`, plus their call sites in `experiment-schema-selftest.sh:12` and `experiment-validate.sh:22`, add behaviour that no prior finding requested. Run a full `sdlc close` so a full review covers it, and record the revision in `## Revisions`.

4. **Minor**
   - BR-1 is still open: write the passing PR #52 merge-check run ID and its result into the Log.
   - The `_vocab.bash` header says the file is skipped by run-merge-checks because it is "not executable". That depends on file mode alone. A `.bash` suffix filter would be sturdier.

5. **Test coverage**
   - No local test checks that the experiment checks exit 2 when `VOCAB` doesn't resolve. That's cheap to add, for example `VOCAB=/nonexistent ./scripts/merge-checks.d/experiment-schema-selftest.sh; [ $? -eq 2 ]`. The CI hook can only be proven by real CI.

6. **Architecture (interdiff only)**
   - **ARCH-DRY: pass.** The guard is shared by both experiment checks.
   - **ARCH-PURE: pass.** These are shell glue scripts with no business logic.
   - **ARCH-PURPOSE: pass.** The hook is a stop-gap with a named upstream removal trigger (ariadne#627), not a deferral of this issue's purpose.
   - **ARCH-EVOLVE: pass.** The `construct/deps` change keeps the same three-field format and only changes the URL value.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry for the two scope revisions (https deps; the ci-setup hook plus the vocabulary guard). Right now they appear only in `## Log`.
   - Record the run that proves the cold-checkout acceptance item, so the newly checked box has evidence behind it.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The Log records only the failing runs 40188584381 and 40188715202; the passing PR #52 merge-check appears only in the subject of commit 54810af, with no run ID or result in the issue.
findings:
  - id: new
    severity: Important
    kind: defect
    family: scope-changed-after-close
    title: |
      Scope changed after close: run a full sdlc close
    detail: |
      The interdiff adds behaviour no prior finding asked for. construct/deps now clones over https instead of ssh, the new scripts/ci-setup.sh post-compile hook adds layer bin dirs to GITHUB_PATH, and the new scripts/merge-checks.d/_vocab.bash guard makes both experiment checks exit 2 when vocabulary is missing. The Log calls these scope revisions 1 and 2. A narrow fix review does not cover them; run a full sdlc close and record them under the Revisions heading.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"33944692-9e9a-41b3-95b8-7fb6c7290239","repository":"github.com/xianxu/metis","issue_id":"000070","recipe":"fix-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/de2e5333afd39e97","reviewed_head":"54810af9368b4cf28cc4168cff626a339c5ab46a","slot":"metis:1","requested_at":"2026-10-25T11:34:22.989502Z"},"phase":"finished","pid":47356,"started_at":"2026-10-25T11:34:23.168431Z","finished_at":"2026-10-25T11:35:04.923465Z","session_id":"327fe450-4bb6-4b84-80ef-584e25ebd7b0","outcome":"returned"}] -->
