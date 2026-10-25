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

---

## Re-review — 2026-10-25T04:39:10-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 70 — CI: build weave from a pinned ariadne commit |
| repo | metis |
| issue file | workshop/issues/000070-ci-weave-pin.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4cb2c455fd62430e4ab25fdddeb940236c30f74d..4c7ce7e5c53c45ed0ac5666a1fb81470a1f4b65f |
| command | sdlc close --issue 70 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-25T04:39:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff and workflow read; now checking run-merge-checks behavior in the substrate and running the new tests.

Dispositions mostly settled; checking whether the hub report (Done-when clause 5) is evidenced.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code in this window is correct and the CI run proves it. `.sdlc/weave-compiler-ref` holds the Spec SHA `63d6bfd…` plus a newline. That SHA is ariadne's only supported ref (`.sdlc/weave-compiler-refs` in the pinned substrate). The pin is not manifest-managed: neither metis's nor ariadne's `base.manifest` names it, and upstream `portable-ci.test.sh:167` asserts that. The two scope revisions are now recorded under `## Revisions`, and the new missing-tool guard has a real regression test. I ran `tests/test_merge_checks.py` and all 3 tests pass. I then put back the base versions of both experiment checks: both parametrized cases failed. The selftest returned 1 instead of 2, and `experiment-validate.sh` returned 0 instead of 2, so it had been passing silently. So the fix can be reached and the test pins it. Two things are left, both cheap. First, `## Done when` was not updated for the scope revisions. This is the 2nd finding in `scope-changed-after-close`. Second, the atlas has no entry for the new `ci-setup.sh` hook. There is also a Minor gap: the hub report has no recorded evidence.

Done-when: stale — the Revisions add three behaviours (https substrate clone, the `ci-setup.sh` PATH hook, and exit 2 when vocabulary is missing) and no `## Done when` clause covers any of them, so nothing in the acceptance contract checks them.

1. **Strengths**
   - `scripts/merge-checks.d/_vocab.bash` puts the tool check in one helper that both checks source (ARCH-DRY). It is non-executable on purpose, so `run-merge-checks.sh:37` (`[ -x "$c" ]`) skips it, and `test_vocab_helper_is_not_run_as_a_check` pins that.
   - `tests/test_merge_checks.py` asserts the behaviour (exit 2, the stderr naming the tool, no "rejected" text), not how it is implemented. Taking the fix out turns it red.
   - `scripts/ci-setup.sh` refuses to run outside Actions (`${GITHUB_PATH:?}`), fails closed when a tool dir is missing, and states its removal trigger (ariadne#627). It uses the seam the upstream workflow already has (`merge-check.yml:166`), and `local-gates-guard.yml:60` protects it.
   - The atlas pin entry (`atlas/index.md:255-260`) matches the workflow's no-fallback behaviour and the bump rule.

2. **Critical findings** — none.

3. **Important findings**
   - **`## Done when` is stale after two scope revisions** (`workshop/issues/000070-ci-weave-pin.md:43-54`). This is the 2nd finding in family `scope-changed-after-close`. Earlier rounds fixed instances, so don't fix this one alone. The rule: a `## Revisions` entry that adds behaviour must add, in the same commit, the `## Done when` clause that accepts it. Every instance in this window:
     - (a) The https clone in `construct/deps`, where a local `../ariadne` never clones.
     - (b) `scripts/ci-setup.sh` putting the layer tool dirs on PATH, until ariadne#627.
     - (c) Both experiment checks exiting 2 and naming the missing tool, pinned by `tests/test_merge_checks.py`.

     Fix: add one clause each for (a), (b) and (c).
   - **The atlas has no entry for the `ci-setup.sh` hook** (`atlas/index.md`, CI compiler pin entry). This is new CI surface, and its temporary status and removal trigger appear only in the script comment and the Log. Fix: one sentence next to the pin entry. It should say `scripts/ci-setup.sh` adds `../ariadne/bin` and `bin` to `GITHUB_PATH` until ariadne#627, and that the experiment checks refuse with exit 2 when vocabulary is missing. This is the only instance. There is no README in the repo.

4. **Minor findings**
   - **Hub report not evidenced.** Plan item 4 is ticked, including "Report … to the hub", and Done-when clause 5 requires it, but no Log entry records the report. This is the 2nd finding in family `done-when-evidence-partial`. The rule: every Done-when clause needs a Log line naming its evidence (a run ID, a message or a SHA) or saying explicitly that it is pending and who owns it. Instances: clause 5 (hub report, no evidence); clause 4 (operator label, which the Log correctly marks as pending); clause 3 (now covered, see BR-1).
   - `experiment-validate.sh:22` lost the old hint `else ../ariadne/bin/vocabulary`. The helper's message ("put the layer tool dirs on PATH or set VOCAB") mostly replaces it.

5. **Test coverage notes**
   - The only executable behaviour change, the vocabulary guard, is covered for both checks.
   - `ci-setup.sh` has no local test. Its only evidence is CI run 40189012386 / 40189166668, which is acceptable for a temporary Actions-only hook.
   - No CI job runs pytest (`merge-check.yml` runs only `merge-checks.d`). That is true of all existing `tests/` too, so it is not new.

6. **Architectural notes**
   - ARCH-DRY: pass. The shared `_vocab.bash` is used by both checks, and the PATH logic in `ci-setup.sh` is a declared stopgap for ariadne#627.
   - ARCH-PURE: pass. This window is thin shell glue with no business logic.
   - ARCH-PURPOSE: pass. The pin is delivered, CI is accepted on the pinned compiler, and the other repos' pins are properly out of scope.
   - ARCH-EVOLVE: pass. The new pin file is a single authored line, and the `construct/deps` format is unchanged (only the URL value changed).
   - When ariadne#627 lands, delete `scripts/ci-setup.sh` and its atlas line together.

7. **Plan revision recommendations**
   - Extend `## Done when` with clauses (a)–(c) above, and reference the two existing Revisions entries.
   - Add a Log line recording the hub report (PR #52, compiler `63d6bfd…`, runs 40189012386 / 40189166668), or mark it pending.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Log now records passing cold-checkout CI runs 40189012386/40189166668 under the pinned compiler; structurally verified no manifest (metis or ariadne base.manifest) names .sdlc and upstream portable-ci.test.sh:167 asserts the pin is never generated.
  - id: BR-2
    disposition: addressed
    note: |
      This is the full close; both scope revisions are recorded under the Revisions heading, and the missing-tool guard is pinned by tests/test_merge_checks.py (verified: reverting both checks fails 2 of 3 tests).
findings:
  - id: new
    severity: Important
    kind: defect
    family: scope-changed-after-close
    title: |
      Done-when not updated for the two hub-approved scope revisions
    detail: |
      2nd in family. Rule: a Revisions entry adding behaviour must add the matching Done-when clause in the same commit. Instances in window: (a) construct/deps https clone with local ../ariadne untouched; (b) scripts/ci-setup.sh putting ../ariadne/bin and bin on GITHUB_PATH until ariadne#627; (c) both experiment checks exit 2 naming a missing vocabulary tool (tests/test_merge_checks.py). None appear in the Done-when list at lines 43-54.
  - id: new
    severity: Important
    kind: defect
    family: atlas-missing-new-surface
    title: |
      atlas update appears missing for the scripts/ci-setup.sh CI hook and vocabulary guard
    detail: |
      atlas/index.md covers the compiler pin only. Add a sentence: ci-setup.sh appends the layer tool dirs to GITHUB_PATH (temporary, removed by ariadne#627), and the experiment merge-checks exit 2 when vocabulary is missing. Only instance in window; the repo has no README.
  - id: new
    severity: Minor
    kind: defect
    family: done-when-evidence-partial
    title: |
      Hub report (Done-when clause 5) is ticked in the Plan but has no Log evidence
    detail: |
      2nd in family. Rule: every Done-when clause needs a Log line naming its evidence (run ID, message, SHA) or saying explicitly that it is pending and who owns it. Instances: clause 5 hub report (no evidence); clause 4 operator label (correctly logged as pending); clause 3 now covered per BR-1.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"bc42968f-393f-4a0c-8485-c4cf6853c694","repository":"github.com/xianxu/metis","issue_id":"000070","recipe":"small-diff-review","round":3,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/0c71a2a145c281d4","reviewed_head":"4c7ce7e5c53c45ed0ac5666a1fb81470a1f4b65f","slot":"metis:1","requested_at":"2026-10-25T11:36:47.060401Z"},"phase":"finished","pid":28778,"started_at":"2026-10-25T11:36:47.261083Z","finished_at":"2026-10-25T11:39:06.250149Z","session_id":"34376b90-cfdf-492d-8c22-c00929d3ccb7","outcome":"returned"}] -->
