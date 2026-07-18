# Boundary Review — 000034-repo-root-relative-shape-path-as-canonical-key-cwd-independent-run-select-submit#34 (whole-issue close)

| field | value |
|-------|-------|
| issue | 34 — repo-root-relative shape path as canonical key (cwd-independent run/select/submit) |
| repo | 000034-repo-root-relative-shape-path-as-canonical-key-cwd-independent-run-select-submit |
| issue file | workshop/issues/000034-repo-root-relative-shape-path-as-canonical-key-cwd-independent-run-select-submit.md |
| boundary | whole-issue close |
| milestone | — |
| window | df8844cb0dcc3211f084f82ea04f8196896c8092..HEAD |
| command | sdlc close --issue 34 |
| reviewer | claude |
| timestamp | 2026-07-17T17:29:38-07:00 |
| verdict | SHIP |

## Review

Verification is complete. Note on process: Bash was unusable this session (the harness couldn't create its `~/.claude/session-env` directory — an EPERM even with sandbox bypassed; `/sandbox` can adjust that), so I could not execute the test suite and instead verified every claim statically against the code. That's reflected in the confidence level.

```verdict
verdict: SHIP
confidence: medium
```

This boundary delivers exactly what the reframed Spec-at-claim commits to, and the reframe itself is well-evidenced: identity is content-addressed (`record.PointAddress` at `pkg/record/address.go:36`, minted in `buildRecord` at `cmd/metis/record.go:121` with no path term), the runs anchor is `Abs(Dir(expPath))` (`cmd/metis/run.go:228-237`), and the ledger sidecar derives from the shape path (`cmd/metis/ledger.go:105-108`). The one real in-repo cwd drift — the bare-repo steppath fallback anchoring `repo.Root` on `os.Getwd()` — is fixed to anchor on the shape's own dir (`cmd/metis/steppath.go:45-49`), and a grep confirms no production `os.Getwd` remains anywhere in `cmd/metis` (only test helpers and a historical plan doc). Both new tests are structurally sound: I traced the e2e test's full path through `runExperiment` → `runResolvedExperimentAdmitted` → `writeRecordJSON` and confirmed the fixture matches the existing e2e conventions (`blaspins_e2e_test.go` uses the identical script contract), that `t.Chdir` is available (go 1.26.3), and that `point_address` is minted non-empty even in the git-less temp dir (empty blob-hash is a legal `PointAddress` input). Nothing blocks SHIP; the findings below are minor.

**1. Strengths**

- The steppath fix is minimal and correct: one anchor swap (`cmd/metis/steppath.go:45-46`) reusing the existing `repo.Root`/`FindUp` walker rather than new path logic (ARCH-DRY pass), and the doc comment states the house rule and its provenance.
- `TestStepPath_BareRepoFallbackAnchorsOnShapeDir` (`cmd/metis/steppath_test.go:172`) is a genuine red-proof fixture — two disjoint `go.mod` repos with cwd deliberately in the wrong one; reverting the fix would resolve `B/steps` and fail. `EvalSymlinks` on both sides handles the macOS `/var → /private/var` trap correctly.
- `TestRun_CwdIndependentIdentityAndLocation` pins the invariant at the right level: same shape, two documented invocation styles, asserting both the *physical* output location (reading `record.json` via the pipeline dir's absolute path) and identity equality (`point_address`), with a non-empty guard so a degenerate `"" == ""` pass can't slip through.
- The atlas one-liner (`atlas/experiment.md:208-213`) names real, verified entities — `shapeBlobHash`, `PointAddress`, `shapeRunIdentity` all exist where claimed — and states the rule ("path is location, never identity") rather than restating implementation.

**2. Critical findings** — none.

**3. Important findings** — none in-window. One traceability caveat for the close verdict (not a code defect): Plan item 3, the `kaggle submit -C` flag, lives in a separate repo (no `kaggle/` exists here; the Log cites kaggle commit `8addd9f`, pushed, with foreign-cwd + failure-mode tests). That leg of the Done-when is unverifiable from this window; the close verdict should rest on the kaggle repo's own evidence, which the Log does record with a commit hash.

**4. Minor findings**

- `cmd/metis/steppath_test.go:172` — the new test omits `t.Setenv("METIS_STEP_PATH", "")`, which every sibling discovery test (lines 63, 100, 126) uses. A developer shell with that variable set makes the test flake (false failure, not false pass). One-line fix.
- `cmd/metis/steppath.go:31,45` — `filepath.Abs(expPath)` is now computed twice in successive blocks (ARCH-DRY nit); hoisting one call would read slightly cleaner. Pre-existing shape, made more visible by this diff.
- `cmd/metis/cwdindep_e2e_test.go` — the git-less fixture means `shapeBlobHash` degrades to `""` in both invocations, so the blob-hash *term* of the identity isn't exercised; a regression making `shapeBlobHash` cwd-dependent (it's `Abs`-anchored today, `cmd/metis/capture.go:310-317`) would escape this net. `git init`-ing the fixture would close that, at the cost of a git dependency the test currently avoids.
- Same test: run ids are passed explicitly (`r-from-pipe`/`r-from-root`), so the *defaulted* run-id path (`singleRunID` → `pointAddressOf`) is covered only transitively via `point_address` equality — acceptable, since the defaulted id *is* that address.

**5. Test coverage notes** — The shipped bug class (cwd leaking into an anchor) is now pinned at both levels: unit (steppath fallback) and e2e (run identity + location). I could not execute the suite (Bash unavailable in this review environment); the Log's "full `-race` green" claim is consistent with everything I could verify statically, hence medium rather than high confidence.

**6. Architectural notes** — ARCH-DRY: pass (reuses `repo.Root`; only the duplicate-`Abs` nit). ARCH-PURE: pass — the change stays in the thin IO shell; `stepPathFromLayers` remains the injected-`exists` pure unit; the new tests are appropriately INTEGRATION for an IO-boundary invariant. ARCH-PURPOSE: pass with the shadow-sweep run — consumers of the resolved shape path (runs-dir anchor, ledger sidecar, `shapeBlobHash`, steppath fallback) all derive from the shape's location, none from cwd; the `os.Getwd` sweep confirms no hand-maintained cwd anchor remains in this repo. The reframe (invariant-not-mechanism) is a legitimate fulfillment of the purpose, not an easy-subset dodge — the superseding Done-when is explicit and each bullet is delivered or (kaggle) evidenced cross-repo. The "path is location, never identity" rule is now stated in the atlas; future path-handling work should cite it.

**7. Plan revision recommendations** — none; the plan (including the Spec-at-claim reframe and ticked checklist) matches the code as shipped. No Core concepts table exists, consistent with the plan's declared "no separate plan doc" scope for a two-fix + one-test issue.
