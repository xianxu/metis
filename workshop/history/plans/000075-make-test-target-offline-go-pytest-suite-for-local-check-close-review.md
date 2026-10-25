# Boundary Review — metis#75 (whole-issue close)

| field | value |
|-------|-------|
| issue | 75 — make test target: offline Go + pytest suite for local-check |
| repo | metis |
| issue file | workshop/issues/000075-make-test-target-offline-go-pytest-suite-for-local-check.md |
| boundary | whole-issue close |
| milestone | — |
| window | e8cafbf867e30a740218674b2808b1696404111c..180ac8416f3c23900ce2b7a4cb37e1e476d27796 |
| command | sdlc close --issue 75 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-23T19:12:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Confirmed sdlc runs `make test` by default when there's no `.sdlc/local-checks.json`. Next I'm checking how sdlc exports `SDLC_DEP_SUBSTRATE`, then running the target.

Test temp dirs still point at `/Volumes/TestScratch/tmp/metis-slot1`, so something is overriding TMPDIR. Checking for the override.

Only a pre-existing perf test failed (timing-sensitive, under load). Make stopped before pytest, so I'm running the Python step on its own.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The change does what the issue asks. sdlc's default suite check runs `make test` when no `.sdlc/local-checks.json` exists (`cmd/sdlc/localchecks.go:208-238` in the pinned ariadne). So adding the `test` target is enough to unblock `sdlc local-check run --name suite`, with no local-gates config, which matches the Spec. I ran `make test` in this checkout, which has no sibling `../ariadne`, with `SDLC_DEP_SUBSTRATE` set. The `-modfile` redirect worked: it copied `go.mod` and `go.sum`, rewrote the `replace`, and compiled and ran all packages against the substrate. The Python step (`UV_OFFLINE=1 uv run --offline pytest -q tests`) passed 137 tests offline.

The full run did not go green here, for two reasons outside this diff:
- My first two runs failed because the review's git confinement rejected test dirs under the global `GOTMPDIR`.
- With that overridden, the existing timing test `TestForkServerPerf_LooseBound` (`cmd/metis/forkexec_test.go:258`) failed under load.

What stands between this and SHIP is one cheap fix: the temp modfile has a fixed name in a shared temp dir.

1. **Strengths**
   - `Makefile:34-39`: the `-modfile` fallback only turns on when `../ariadne/go.mod` is missing **and** `SDLC_DEP_SUBSTRATE` is set. Slots and clones keep the real `go.mod`, and the tracked `go.mod` is never touched (`git status` stayed clean after the runs).
   - `$(basename $(TEST_MODFILE)).sum` gives the right companion `go.sum` path (`metis-test.sum`), as Go's `-modfile` expects. I confirmed the files appeared and the build used them.
   - The Log explains why a temp `go.work` was rejected (the symlinked cwd breaks `use` matching). That reasoning is worth keeping.
   - `UV_OFFLINE=1` is set on the `go test` line as well, so the Python subprocesses the Go tests spawn inherit it, as the Spec requires.
   - The atlas entry (`atlas/index.md:9-13`) documents the new surface, including the modfile fallback.

2. **Critical findings:** none.

3. **Important findings**
   - **The temp modfile has a fixed name in a shared temp dir** (`Makefile:35`, `TEST_MODFILE := $(or $(TMPDIR),/tmp)/metis-test.mod`). Two `make test` runs in local-check checkouts on the same host share that path. That happens when the suite queue runs heads for different callers, each with its own `SDLC_DEP_SUBSTRATE`. One run's `cp` and `go mod edit` can overwrite the other's file mid-run, so a test may build against the wrong ariadne or fail to parse a half-written file. Nothing removes the file afterwards either (ARCH-FUNERAL). The bound is one file, but it is unowned.
     - Fix sketch: key the path to the checkout, e.g. `$(shell git rev-parse --absolute-git-dir)/metis-test.mod`. That is private to each checkout, never committed, and deleted with the checkout. Alternatively use `mktemp -d` inside the recipe and `rm` it at the end.

4. **Minor findings**
   - `-modfile=$(TEST_MODFILE)` on the `go test` line is unquoted, while the `cp` and `go mod edit` lines quote it. A `TMPDIR` containing a space would break only this line.
   - With no `../ariadne` and no `SDLC_DEP_SUBSTRATE`, the target falls through to a plain `go test`, which fails with Go's replace-path error. That is acceptable, but a one-line `$(error …)` or comment would make the cause obvious.
   - `TestForkServerPerf_LooseBound` is now part of the required `sdlc-local/suite` gate. It failed for me under load (forkserver cold 3.10s vs legacy 2.93s). It is not in this diff, but it may block merges intermittently, so consider tracking it under `known_flakes` or a follow-up issue.

5. **Test coverage notes**
   - This is build glue, and the real test is running it. The Log records the implementor's offline run in a fresh clone. I independently confirmed two things: the modfile path works with no `../ariadne`, and pytest passes offline (137 tests).
   - I could not get a clean full Go run in the review environment. One failure was confinement (worked around via `GOTMPDIR`); the other is the timing test above. All other packages printed `ok`.
   - I did not reproduce the "no `.venv`" condition, because a `.venv` already existed after the Go tests ran.

6. **Architectural notes**
   - ARCH-DRY: pass. The new target reuses `uv sync` and `go test` without duplicating anything.
   - ARCH-PURE: N/A. The diff is only build glue.
   - ARCH-PURPOSE: pass. All three Done-when items are addressed, and the substrate fallback was needed for the third.
   - ARCH-MOCK: N/A. The diff adds no new external dependency seam.
   - ARCH-CONSTRAINTS: pass. The offline contract is enforced by `UV_OFFLINE` and `--offline`, so a missing cache fails loudly instead of waiting on the network.
   - ARCH-SECURE: minor. A world-writable `/tmp` fallback holding a fixed-name modfile that the build trusts is the same issue as the Important finding; the fix covers both.
   - ARCH-ORDER: flagged. The target holds no state between events except the shared modfile path, so concurrent runs interleave on it (see the Important finding).
   - ARCH-FUNERAL: flagged. The temp modfile and sum file are never removed.
   - ARCH-EVOLVE: pass. No persisted format changes; the tracked `go.mod` is untouched.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry: "Plan item 1 grew a `-modfile` fallback (e950317): when `../ariadne` is absent and `SDLC_DEP_SUBSTRATE` is set, `make test` uses a temp `go.mod` copy whose `replace` points at the substrate. This was required for Done-when #3 in local-check checkouts." Today the Log explains this but the Spec and Plan don't.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: shared-temp-artifact-unowned
    title: |
      make test writes its temp go.mod to one fixed path in a shared temp dir, so concurrent runs can collide and nothing deletes it
    detail: |
      Makefile:35 sets TEST_MODFILE to $(TMPDIR or /tmp)/metis-test.mod. Two local-check runs on the same host, possibly with different SDLC_DEP_SUBSTRATE values, can overwrite each other's file mid-run and build against the wrong ariadne, and the file is never removed (ARCH-ORDER, ARCH-FUNERAL). Fix: put it somewhere private to the checkout, e.g. $(shell git rev-parse --absolute-git-dir)/metis-test.mod, or use mktemp -d plus rm inside the recipe.
  - id: new
    severity: Minor
    kind: defect
    family: inconsistent-shell-quoting
    title: |
      The -modfile path on the go test line is unquoted, unlike the cp and go mod edit lines
    detail: |
      Makefile:46 passes -modfile=$(TEST_MODFILE) without quotes, so a TMPDIR containing a space breaks only this step.
  - id: new
    severity: Minor
    kind: defect
    family: plan-scope-drift
    title: |
      The Spec and Plan don't mention the SDLC_DEP_SUBSTRATE -modfile fallback added in e950317
    detail: |
      The Log explains it, but the plan should get a Revisions entry so the Spec and Plan describe what make test actually does in local-check checkouts.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ec272bcc-efce-4e42-9d85-14471ca6d80d","repository":"github.com/xianxu/metis","issue_id":"000075","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/5208294fb1210307","reviewed_head":"180ac8416f3c23900ce2b7a4cb37e1e476d27796","slot":"metis:1","requested_at":"2026-10-24T02:03:05.167497Z"},"phase":"finished","pid":45069,"started_at":"2026-10-24T02:03:05.262731Z","finished_at":"2026-10-24T02:12:11.40416Z","session_id":"fb2f68a4-cbe0-413f-869a-dc0c0e7bae4d","outcome":"returned"}] -->

---

## Re-review — 2026-10-23T19:32:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 75 — make test target: offline Go + pytest suite for local-check |
| repo | metis |
| issue file | workshop/issues/000075-make-test-target-offline-go-pytest-suite-for-local-check.md |
| boundary | whole-issue close |
| milestone | — |
| window | e8cafbf867e30a740218674b2808b1696404111c..aad6f5c5d310ca2dd05dc246bd7a26359d039ee9 |
| command | sdlc close --issue 75 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-23T19:32:35-07:00 |
| verdict | SHIP |

## Review

I'm checking whether the target works: next I'll run `make test` against the pinned ariadne substrate.

```verdict
verdict: SHIP
confidence: high
```

All three findings from round 1 are fixed, and I found nothing new that blocks the close. The Makefile now runs a private, per-run temp-dir recipe (`mktemp -d` plus `trap 'rm -rf "$d"' EXIT`). It points the module's `replace` at `SDLC_DEP_SUBSTRATE` through `-modfile` only when `../ariadne/go.mod` is absent, so slots and clones still use the real go.mod. I ran it against the pinned substrate with no sibling `../ariadne`:
- **Go half:** it printed `make test: ariadne => …`, compiled and ran every package, and left no `metis-test.*` directory behind.
- **Failure handling:** when `go test` failed, make stopped before pytest and exited non-zero, which matches Done-when #1.
- **Python half:** run offline with `UV_OFFLINE=1 uv run --offline pytest -q tests`, 137 tests passed.

Some `cmd/metis` tests failed, but only because this reviewer environment refuses `git init` outside its allowed directories ("outside the reviewer's directories (sdlc #339)"). That is not a defect in this diff. The only new finding is Minor: the Log has no evidence for Done-when #3.

1. **Strengths**
   - `Makefile:36-44`: the temp go.mod/go.sum copy lives in a private `mktemp -d` directory that the EXIT trap removes. That covers both the success and the `set -e` failure paths. I confirmed by running it that nothing is left behind.
   - `Makefile:44`: `${modflag:+"$modflag"}` passes no argument when there is no override and one quoted argument when there is. This fixes the quoting problem and also avoids passing an empty argument.
   - The fallback only activates when it's needed (`! -f ../ariadne/go.mod && -n SDLC_DEP_SUBSTRATE`). The tracked go.mod is never changed.
   - The Log records why a temp go.work was rejected: a symlinked cwd breaks its `use` matching. That's useful for whoever touches this next.
   - The atlas entry (`atlas/index.md:9-13`) describes the new surface, including the substrate fallback.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - Done-when #3 is ticked as part of Plan item 2, but the Log never records a successful `sdlc-local/suite` run on a pushed head after the fix. The only local-check result in the Log is the failed run on 8e8b3c96. The verification it does record was done in a plain clone. Add a Log line citing the recorded head.
   - Not raised as a finding: `sh` implementations differ on whether the EXIT trap runs after SIGINT, so an interrupted run might leave the temp dir behind. This is negligible.

5. **Test coverage notes:** The Makefile recipe has no automated test. I treated it as an executable contract and checked it by running it, as described above, and I accepted the fixes on that evidence. I didn't try to reproduce the old concurrent-run collision, because the private per-run directory removes the shared path by construction. The pytest half passed offline (137 tests). The Go half compiled and ran against the substrate, and the failures came only from the reviewer environment.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Nothing is duplicated.
   - **ARCH-PURE: pass.** This is build glue with no business logic.
   - **ARCH-PURPOSE: pass.** The issue's goal was to unblock `local-check --name suite`, and the substrate fallback serves that directly.
   - **ARCH-MOCK: N/A.** The change adds no new external dependency seam. The offline uv cache is the deliberate envelope.
   - **ARCH-CONSTRAINTS: pass.** The offline contract is enforced with `UV_OFFLINE`, so a cold cache fails loudly instead of waiting on the network.
   - **ARCH-SECURE: pass.** `SDLC_DEP_SUBSTRATE` comes from sdlc and is always quoted. No credentials are involved.
   - **ARCH-ORDER: pass.** The recipe holds no state between runs, because each run gets a private directory, so concurrent runs no longer share an artifact.
   - **ARCH-FUNERAL: pass.** The temp dir is removed on exit. Nothing durable is created besides the uv cache, which uv owns.
   - **ARCH-EVOLVE: N/A.** No persisted format changes. go.mod is copied, never rewritten.

7. **Plan revision recommendations:** none needed. The existing Revisions entry already describes the `-modfile` fallback and the per-run private directory.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Makefile:37 uses mktemp -d plus an EXIT trap rm. A run against the pinned substrate left no metis-test.* dir and resolved ariadne through the substrate.
  - id: BR-2
    disposition: addressed
    note: |
      Makefile:44 passes the flag as ${modflag:+"$modflag"}, and modflag is assigned quoted, so a TMPDIR with spaces is now safe.
  - id: BR-3
    disposition: addressed
    note: |
      The issue's Revisions entry (2026-10-23) adds the SDLC_DEP_SUBSTRATE -modfile delta to the Spec and Plan item 1, including the per-run private dir.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: done-when-claim-unevidenced
    title: |
      Done-when 3 (sdlc-local/suite recorded on the pushed head) is ticked, but the Log has no successful local-check record after the fix
    detail: |
      The only local-check run in the Log is the failed one on 8e8b3c96. The later verification was done in a plain clone. Add a Log line citing the head where sdlc-local/suite was recorded.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"680c10c7-7577-4bd6-8be9-d4629ae78e3c","repository":"github.com/xianxu/metis","issue_id":"000075","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/ca422ea244c2d5ed","reviewed_head":"aad6f5c5d310ca2dd05dc246bd7a26359d039ee9","slot":"metis:1","requested_at":"2026-10-24T02:29:37.905438Z"},"phase":"finished","pid":36522,"started_at":"2026-10-24T02:29:38.015025Z","finished_at":"2026-10-24T02:32:32.564623Z","session_id":"8a3bdfdd-0458-47aa-a2bc-07a545c08d98","outcome":"returned"}] -->
