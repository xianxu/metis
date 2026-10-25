# Boundary Review — metis#73 (whole-issue close)

| field | value |
|-------|-------|
| issue | 73 — Repeat-shaped run records and ledger rows without CV folds |
| repo | metis |
| issue file | workshop/issues/000073-repeated-run-ledger.md |
| boundary | whole-issue close |
| milestone | — |
| window | 23335ebc28bf331b8c54a98b1da16fcdd8500ec4..b353b0a521f95a07b7a0a8e5b5ba5c0967dad832 |
| command | sdlc close --issue 73 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T13:13:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Read the full diff for the library and CLI layers; now checking how the runner, sweep and `ledger show` paths interact with the new single-run publication.

Confirmed one gap (repeat labels aren't single-use for experiments outside git). Now running the test suite and reading the e2e tests.

All tests pass. Now I'm reading the e2e tests and the atlas changes, and checking whether a regression test fails without the fix.

The parallel-writer test goes red with the flock removed (file restored). Now checking README/atlas coverage of the new CLI surface.

Found a likely problem: `ledger publish` copies `rec.Repeat` from the record without validating it. Verifying with a throwaway test in the disposable checkout.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue is delivered. Two repeats of one config get separate claimed directories, records and ledger rows. Plain runs now publish a row that a rerun replaces. Republishing the same row is a no-op, and conflicting data is refused. A crash between the record and the row can be recovered with `metis ledger publish`. Repeat families aggregate as mean/se/n plus counts, and legacy ledgers round-trip byte-identically. I ran the whole suite against the pinned ariadne dependency (through a temporary `go.work` and an allowed TMPDIR): `./pkg/...`, `./internal/...` and `./cmd/metis` all pass. I also removed the `flock` in `publishRows`, and `TestPublishRows_ParallelWritersAndClaims` failed 3 out of 3 times. I then restored the file. Three cheap Important defects keep this from SHIP:
- A run record with an invalid `repeat` value gets published, and then the ledger can no longer be loaded. I confirmed this with a probe test.
- For an experiment outside git, a repeat label is not actually single-use.
- `ledger fingerprints` looks up records by point address, so when a code edit splits repeats at one point, it shows one fingerprint's commit and dirty state for the other.

**1. Strengths**
- `internal/fsx/atomic.go:19`: one temp-file-plus-rename helper now replaces the two earlier copies (CAS, step cache) and is used for run.json, record.json and the ledger. There is a test that fails the rename and checks the old file survives (`atomic_test.go:95`).
- `pkg/ledger/ledger.go:271` (`dedupKey`): the label is added to the key only when it is set, so legacy keys are byte-identical. A 200-trial property test pins this (`repeat_test.go:592`).
- `Publish` is kept separate from `Append` (`ledger.go:324`), so replacing a row happens only at the one call site that wants it. The kinds × duplicates table test pins this.
- `applyCodeCapture` (`capture.go:388`) moves the code-fingerprint backfill into memory before a single atomic record write, so a `record.json` that exists is final.
- `publishRows` (`cmd/metis/ledger.go:151`) is the one locked ledger writer, and the sweep ledger writes go through it too. The concurrency test does catch a missing lock.

**2. Critical findings**
None.

**3. Important findings**
- **`cmd/metis/ledger.go:197` `rowFromRecord` / `publishRunRow`: the writer accepts a label the reader refuses (ARCH-SECURE).** `rec.Repeat` is copied from the record without validation. `Encode` writes it, then `Decode` (`ledger.go:458`) refuses it, and every later publish or show on that ledger fails. A probe that published a row with `Repeat: "*"` was accepted, after which `loadLedger` failed with `invalid repeat label "*"`. A hand-edited or foreign record passed to `metis ledger publish` would cause this.
  - Fix: in `readFinalRecord` or `rowFromRecord`, run `ParseRepeatLabel` on a non-empty `rec.Repeat` and check that `filepath.Base(runDir) == repeatRunID(rec.PointAddress, rec.Repeat)`.
  - Better: validate in `Ledger.Append`/`Publish`, so no writer can produce what `Decode` refuses.
- **`cmd/metis/run.go:222` + `singleRunID` (`run.go:296`): repeat labels are not single-use outside git (ARCH-PURPOSE).**
  - If `shapeBlobHash` fails (no git), the run ID is `run-<timestamp>.r-L`. Reusing label L a second or more later creates a new directory and runs every step. Only then does publishing either refuse as "conflicting data", or silently do nothing if the metrics match. In both cases a second directory exists that the ledger does not represent.
  - That breaks decision 2: a used label must refuse before any effect.
  - Fix: refuse `--repeat` when there is no content address, or build the claimed directory name from the point address the record mints anyway (`assembleRecord`).
  - The implementer's Log already notes the related `rowRunID` miss. Neither is tested, because the e2e tests use a tracked experiment.
- **`cmd/metis/fingerprints.go:251` `loadLedgerRecords`: half-fixed (ARCH-PURPOSE, instance not class).** The read path now goes through `rowRunID(r)`, but the map is still keyed, and skipped, by `r.PointAddr`, and `cohortSummaries` looks it up by `PointAddr`. Repeats at one point share that address. After a code edit splits repeats `1` (fp1) and `2` (fp2), the fp2 cohort shows fp1's record: wrong commit, dirty and capture status in `metis ledger fingerprints` and in select's error output.
  - Fix: key by `rowRunID(r)` in both the loader and `cohortSummaries`.
  - Add a test with two fingerprints at one point.
- **`pkg/ledger/ledger.go:558` repeat family `.se` at n=1.** `meanSE` returns `se=0` for a single value, and the test pins `train.t.se = 0` with n=1. That is a made-up "zero uncertainty", which goes against the Spec's "uncertainty without inventing absent measurements".
  - Fix: omit `<m>.se` when n<2 in the repeat branch. Fold groups stay unchanged.

**4. Minor findings**
- `run.go:233`: if the claim succeeds but the experiment fails validation (`run.Started == ""`), the empty claimed directory is left behind. The label is used up, and a retry gets the misleading "already executed" message.
- `ledger.go:248` `rowPublished` re-implements key matching by hand instead of using the ledger's `dedupKey` (ARCH-DRY). Expose a `Ledger.Has(Row)`.
- `Ledger.Publish` replacing a plain row drops any `Extra` columns a newer metis wrote on it. `SameRow` compares `Extra`, so republishing a row that carries `Extra` reads as a conflict (ARCH-EVOLVE edge).
- Plain runs with `--run <id>`: `rowRunID` returns the point address, not `<id>`, so the fingerprint view misses the record.
- Temp files `.tmp-*` left by a killed process next to `exp.ledger.csv` or in run directories are never cleaned up; only the CAS eviction pass skips them (ARCH-FUNERAL).
- The plan's Tasks checkboxes are still `- [ ]`, while the issue shows `[x]`.

**5. Test coverage notes**
- Good: the legacy byte-identity property, unknown-column survival, the kinds × duplicates table, the n-counts property, fold groups compared byte for byte, interrupted publication and recovery, refusing a truncated or foreign record, plain-run replace, a failed row with no metrics, and parallel publishers (confirmed to fail without the fix).
- Missing:
  - an experiment outside git with `--repeat`
  - `ledger fingerprints` with two fingerprints at one point
  - publishing a record with an invalid `repeat`
  - an e2e `metis ledger show exp.md --sort train.score` that renders a family row
- Parallel writers are tested with goroutines only; the plan said goroutines and processes. That is enough because `flock` works per open file description.

**6. Architectural notes (ARCH-\*)**
- ARCH-DRY: pass. One atomic helper and one ledger writer; `rowPublished` is the one small duplicate (Minor).
- ARCH-PURE: pass. Label parsing, `rowFromRecord`, keys, conflict detection and aggregation are pure and tested without IO; mkdir, flock and rename are a thin shell.
- ARCH-PURPOSE: **flag**. The repeat path outside git, and the fingerprint loader where only the path was fixed and the key was not (Important findings above).
- ARCH-MOCK: pass. Only filesystem and git are used, and both are real in temp directories.
- ARCH-CONSTRAINTS: pass. The O(rows) rewrite per publish is declared, and the lock is held for one rewrite.
- ARCH-SECURE: **flag**. The record's `repeat` is trusted on publish (Important). Otherwise a truncated record degrades visibly.
- ARCH-ORDER: pass. The sequence is linear (claim, execute, run.json, record.json, row); the claim is the exclusive mkdir, and the e2e tests cover each crash point. The leftover empty claim is Minor.
- ARCH-FUNERAL: pass with a note. Run directories and ledger rows grow by design as the research record, and the plan says so. Orphaned temp files are Minor.
- ARCH-EVOLVE: pass. `Extra` is captured and written back, and the `repeat` column is added and ragged. The plain-row replace dropping `Extra` is Minor.

**7. Plan revision recommendations**
- In Core concepts, `withLedgerLock` / `publishRows` is now just `publishRows` (the lock is inline), and `repeatGroups` is a branch inside `AggregateView`, not a named function. Add a Revisions entry so the table matches the code.
- Add a Revisions line on experiments outside git: either `--repeat` refuses without a content address, or name the run directory from the minted point address.
- Mark Tasks 1–7 as `[x]` in the plan file.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: writer-reader-validation-asymmetry
    title: |
      ledger publish writes a record's unvalidated repeat label, which Decode then refuses, so the ledger can no longer be loaded
    detail: |
      rowFromRecord (cmd/metis/ledger.go) copies rec.Repeat without ParseRepeatLabel. A probe test publishing Repeat "*" was accepted, then loadLedger failed with "invalid repeat label". Validate the label in Ledger.Append/Publish (or readFinalRecord) and check the run-dir base equals repeatRunID(rec.PointAddress, rec.Repeat).
  - id: new
    severity: Important
    kind: defect
    family: repeat-claim-identity
    title: |
      For an experiment outside git, a repeat label is not single-use (timestamped dir, so the mkdir claim never collides)
    detail: |
      With no shapeBlobHash, singleRunID returns run-<ts>, so reusing --repeat L later creates a new dir and executes every step. Publish then refuses as conflicting, or silently does nothing when metrics match, leaving a directory the ledger does not represent. Refuse --repeat without a content address, or claim on the minted point address. Add a test outside git.
  - id: new
    severity: Important
    kind: defect
    family: row-identity-is-run-id
    title: |
      loadLedgerRecords and cohortSummaries still key records by PointAddr, so repeats at one point show one fingerprint's commit and dirty state for another
    detail: |
      fingerprints.go now reads runs/rowRunID(r) but skips and looks up by r.PointAddr. When a code edit splits repeats at one point, the fp2 cohort shows fp1's record in ledger fingerprints and in select's error output. Key both by rowRunID(r) and add a two-fingerprint repeat test.
  - id: new
    severity: Important
    kind: defect
    family: no-invented-measurements
    title: |
      A repeat family with n=1 reports se=0, a made-up zero uncertainty
    detail: |
      AggregateView's repeat branch emits <m>.se from meanSE, which returns 0 for n<2, and repeat_test pins train.t.se=0 at n=1. The Spec says uncertainty must not invent absent measurements. Omit .se when n<2 for repeat families; fold groups stay unchanged.
  - id: new
    severity: Minor
    kind: defect
    family: repeat-claim-identity
    title: |
      A run that fails validation leaves its claimed repeat dir empty, using up the label with an "already executed" message
    detail: |
      claimRunDir runs before runner validation. When run.Started is empty, remove the empty claimed dir, or word the refusal as "claimed, never started".
  - id: new
    severity: Minor
    kind: defect
    family: one-source-of-row-identity
    title: |
      rowPublished re-implements the ledger row key by hand instead of using dedupKey
    detail: |
      Expose a Ledger.Has(Row) built on dedupKey, so the key is defined in one place (ARCH-DRY).
  - id: new
    severity: Minor
    kind: defect
    family: unknown-fields-survive-rewrite
    title: |
      Replacing a plain row drops its Extra columns, and SameRow treats Extra-only differences as a conflict
    detail: |
      Ledger.Publish overwrites the row wholesale. Carry the old row's Extra forward on replace, and compare only known fields when checking an idempotent republish.
  - id: new
    severity: Minor
    kind: defect
    family: plan-table-matches-code
    title: |
      Core concepts names withLedgerLock and repeatGroups, but neither exists as a named entity; Tasks checkboxes are left unchecked
    detail: |
      The lock is inline in publishRows and the repeat grouping is a branch inside AggregateView. Add a Revisions entry and mark Tasks 1-7 done in the plan file.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"297d7ece-93d5-4a1e-a9ae-0d38c307ca17","repository":"github.com/xianxu/metis","issue_id":"000073","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/174e5fee3656e06c","reviewed_head":"b353b0a521f95a07b7a0a8e5b5ba5c0967dad832","slot":"metis:1","requested_at":"2026-10-25T20:04:11.816695Z"},"phase":"finished","pid":58338,"started_at":"2026-10-25T20:04:12.020166Z","finished_at":"2026-10-25T20:13:28.045414Z","session_id":"d3dd99ec-8c8d-4a5c-8335-f0b3c189144b","outcome":"returned"}] -->

---

## Re-review — 2026-10-25T13:49:02-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 73 — Repeat-shaped run records and ledger rows without CV folds |
| repo | metis |
| issue file | workshop/issues/000073-repeated-run-ledger.md |
| boundary | whole-issue close |
| milestone | — |
| window | 23335ebc28bf331b8c54a98b1da16fcdd8500ec4..5cfad8dbf163f703dbac0ecbeff6bc022551d496 |
| command | sdlc close --issue 73 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T13:49:02-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Let me look at the round-1 fix commit first, then the issue/plan.

The pinned tests are running; the git-dependent e2e tests are blocked by the sandbox's git-init fence, so I'm finding where the temp dir comes from.

Full suite is green at head. Now reverting each claimed fix to confirm its regression test goes red.

All seven code fixes go red when reverted. Now reading the issue spec and the head code to sweep each finding's class.

Still reviewing: fix reverts all verified; now sweeping the ledger writer call sites for remaining instances of each finding class.

Nearly done: checking the atomic-write helper and the leftover lock file, then writing the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All eight round-1 findings are fixed, and the seven behavior fixes are backed by tests that go red when the fix is reverted. I checked each one by reverting it in this disposable checkout and running its regression test in the foreground; all seven failed and were then restored. The working tree is clean. With a `go.work` pointing at the pinned ariadne checkout, and `TMPDIR`/`GOTMPDIR` set inside the reviewer's directories (the e2e tests run `git init`), the full suite passes at head: `go test ./...` is green, including the `cmd/metis` package (56.8s). Nothing blocks shipping. Two Minor findings remain. First, the atlas still says every metric in a repeat family gets `.se`, which contradicts the BR-4 fix; it also doesn't mention the new BR-2 rule that `--repeat` needs a tracked experiment. Second, the BR-2 guard checks a stand-in condition rather than the identity that actually names the run dir. That is the 3rd finding in `repeat-claim-identity`, so I state the rule below.

1. **Strengths**
   - `pkg/ledger/ledger.go:93-99`: `dedupKey` adds the repeat label only when it is set, so legacy keys stay byte-identical. `Ledger.Has` (`:159`) now makes it the only row-identity definition; the hand-written copy in `rowPublished` is gone (`cmd/metis/ledger.go:257`).
   - `cmd/metis/ledger.go:151-172`: `publishRows` is the only ledger writer, and both the sweep path and the single-run path go through it. A flock plus an atomic rename means parallel repeats can't lose rows, and a crash mid-rewrite leaves the old ledger intact.
   - `cmd/metis/ledger.go:220-238`: `readFinalRecord` now ties a repeat record to its own directory (`repeatRunID(point_address, repeat)`). A hand-edited label can no longer reach the ledger.
   - `pkg/ledger/ledger.go:428-492`: repeat aggregation is a separate branch, and fold groups are pinned unchanged by a byte-comparison test (`TestAggregateView_FoldGroupsUnchangedByRepeats`).
   - Each round-1 fix has a focused test that reaches the fixed code, and every one fails without its fix.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **Atlas contradicts the BR-4 fix.** `atlas/experiment.md` ("Every metric gets mean, `.se` and `.n`") and `atlas/index.md` ("every metric mean/se/n") predate it, but the code now omits `.se` below n=2 (`pkg/ledger/ledger.go:486`). The atlas also never says `--repeat` refuses an untracked experiment (BR-2).
   - **The BR-2 guard checks a stand-in condition** (3rd finding in `repeat-claim-identity`). `cmd/metis/run.go:232` checks that `shapeBlobHash` succeeds, but the dir name comes from `singleRunID`. That function can still fall back to `run-<ts>` when `pointAddressOf` fails (`run.go:318`, for example a non-finite `with:` value), and then the mkdir claim never collides. Impact is small today, because `buildRecord` fails the same way and no row is published.
   - The plain-row replace keeps the old row's `Extra` (BR-7). Columns written by a newer version that describe the replaced execution (host, duration) then sit on the new execution's row. That's acceptable for now; just keep it in mind once a real newer column exists.

5. **Test coverage notes**
   - Revert checks:

     | Finding | Fix reverted | Test that went red |
     |---|---|---|
     | BR-1 | `readFinalRecord` check | `TestLedgerPublish_RefusesInvalidOrMisplacedRepeatLabel` |
     | BR-1 | `Publish` label validation | `TestPublish_ValidatesLabelAndKeepsExtra` |
     | BR-2 | untracked-experiment guard | `TestRepeat_RefusesUntrackedExperiment` |
     | BR-3 | `fingerprints.go` keying | `TestRepeat_FingerprintCohortsUseOwnRecords` |
     | BR-4 | n≥2 guard on `.se` | `TestAggregateView_RepeatFamily` |
     | BR-5 | claim release | `TestRepeat_ValidationFailureReleasesClaim` |
     | BR-7 | `Extra` carry-forward | `TestPublish_ValidatesLabelAndKeepsExtra` |

   - BR-6 (`Has`) is pinned by the `Has` assertions in `TestPublish_ValidatesLabelAndKeepsExtra`.
   - Not covered: a run dir named by the timestamp fallback when point-address minting fails (the Minor above).

6. **Architecture notes (ARCH-\*)**

   | Marker | Result | Why |
   |---|---|---|
   | ARCH-DRY | pass | One atomic-write helper (`internal/fsx`), one ledger writer (`publishRows`), one row key (`dedupKey` behind `Has`), one record-to-row mapping (`rowFromRecord`). |
   | ARCH-PURE | pass | Key, conflict check, label parsing and aggregation are pure and tested without IO. Mkdir, flock and rename are thin glue. |
   | ARCH-PURPOSE | pass | Repeats, single-run rows, idempotent and conflicting publish, recovery and family stats are all delivered, with no couch-bench JSONL ledger. Sweep of the writer-validation class: there are only two ledger writers (sweep `Append`, which has no repeats, and `Publish`, which validates), so that class is closed. Sweep of the row-to-dir class: `loadLedgerRecords`, `cohortSummaries` and `claimRunDir` all go through `rowRunID` or `repeatRunID`. `select` is shape-only. |
   | ARCH-MOCK | pass | Filesystem only. The e2e tests use real git in temp repos plus the existing `traceFakeExec` seam. |
   | ARCH-CONSTRAINTS | pass | Batch workload. Each publish loads and rewrites the whole ledger (O(rows)), as declared in the plan. |
   | ARCH-SECURE | pass | The slug label blocks path traversal. A repeat record read from disk is checked for a valid label and its own dir. Publish refuses a dir outside the experiment's `runs/`. |
   | ARCH-ORDER | pass | The publication order is linear (claim → execute → record → row), and each crash point has a recovery action. Rows are serialized by flock and the claim by mkdir. The minor gap is the stand-in guard above. |
   | ARCH-FUNERAL | pass | Growth of run dirs and ledger rows is declared in the plan. The `.lock` file is one per ledger. |
   | ARCH-EVOLVE | pass | `Extra` survives decode, Append, Publish-replace and Encode. The `repeat` column is ragged, and legacy ledgers round-trip byte-identically. |

7. **Plan revision recommendations**
   - None for the table: the 2026-10-25 Revisions entry already supersedes the stale Core concepts names (`repeatGroups`, `withLedgerLock`, `cmd/metis/atomic.go`).
   - If the stand-in-guard Minor is fixed, add a single line saying the repeat claim is named from the minted point address, with no fallback.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Ledger.Publish validates the label and readFinalRecord checks label plus dir; reverting either turns its test red (TestPublish_ValidatesLabelAndKeepsExtra, TestLedgerPublish_RefusesInvalidOrMisplacedRepeatLabel).
  - id: BR-2
    disposition: addressed
    note: |
      run.go:232 refuses --repeat on an untracked experiment; reverting it turns TestRepeat_RefusesUntrackedExperiment red. The point-address fallback that remains is raised as a separate Minor naming the rule.
  - id: BR-3
    disposition: addressed
    note: |
      loadLedgerRecords and cohortSummaries key by rowRunID; restoring the old fingerprints.go turns TestRepeat_FingerprintCohortsUseOwnRecords red.
  - id: BR-4
    disposition: addressed
    note: |
      ledger.go:486 omits se below n=2; reverting it turns TestAggregateView_RepeatFamily red (train.t.se:0 reappears).
  - id: BR-5
    disposition: addressed
    note: |
      run.go:248 removes the empty claimed dir; reverting it turns TestRepeat_ValidationFailureReleasesClaim red.
  - id: BR-6
    disposition: addressed
    note: |
      Ledger.Has is built on dedupKey and rowPublished calls it; pinned by the Has assertions in TestPublish_ValidatesLabelAndKeepsExtra.
  - id: BR-7
    disposition: addressed
    note: |
      Publish carries the old Extra forward on replace and SameRow ignores Extra; reverting the carry turns TestPublish_ValidatesLabelAndKeepsExtra red.
  - id: BR-8
    disposition: addressed
    note: |
      The plan's 2026-10-25 Revisions entry records the inline flock and the AggregateView branch; Tasks 1-7 are ticked in the plan and the issue.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: repeat-claim-identity
    title: |
      The --repeat guard checks shapeBlobHash, but the claim dir comes from singleRunID, which can still fall back to a timestamp
    detail: |
      This is the 3rd finding in family repeat-claim-identity. Rule: a repeat's claim dir is named only from the minted point address (pointAddressOf), and --repeat refuses when minting fails; it never uses a fallback. run.go:232 checks shapeBlobHash instead, while singleRunID (run.go:318) still returns run-<ts> when pointAddressOf errors (e.g. a non-finite with: value), so the mkdir claim never collides. Fix: in the repeat branch call pointAddressOf directly and refuse on error, instead of using singleRunID plus a separate precondition. Prevalence: 3 instances (BR-2 untracked fallback, BR-5 claim spent at validation, and this one).
  - id: new
    severity: Minor
    kind: defect
    family: atlas-matches-behavior
    title: |
      The atlas still says every metric in a repeat family gets .se, and omits the rule that --repeat needs a tracked experiment
    detail: |
      atlas/experiment.md ("Every metric gets mean, .se and .n") and atlas/index.md ("every metric mean/se/n") contradict the BR-4 fix at pkg/ledger/ledger.go:486, which omits .se below n=2. The BR-2 refusal for an untracked experiment is not documented either. Update both atlas passages in the same round as the behavior fixes.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"676459ec-cc77-4407-b291-67b81aacbc02","repository":"github.com/xianxu/metis","issue_id":"000073","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/87a447b75cdbcf00","reviewed_head":"5cfad8dbf163f703dbac0ecbeff6bc022551d496","slot":"metis:1","requested_at":"2026-10-25T20:42:27.697285Z"},"phase":"finished","pid":68759,"started_at":"2026-10-25T20:42:27.857229Z","finished_at":"2026-10-25T20:49:00.280171Z","session_id":"b719b4e9-8cc4-4bc5-a301-a55b1fbb7abc","outcome":"returned"}] -->

---

## Re-review — 2026-10-25T13:52:11-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 73 — Repeat-shaped run records and ledger rows without CV folds |
| repo | metis |
| issue file | workshop/issues/000073-repeated-run-ledger.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6dd9bb55c76dd25a67e9f59aa406e206896e4672..214f1d00904e90d35acfb6e4dc6ef0fcb5b31521 |
| command | sdlc close --issue 73 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T13:52:11-07:00 |
| verdict | SHIP |

## Review

Running the repeat tests against the pinned dependency; fixing the workspace Go version.

The git-backed e2e tests can't run here: the sandbox refuses `git init` in the test temp dir. I'm checking the fix statically instead.

```verdict
verdict: SHIP
confidence: medium
```

Both open findings from the earlier rounds are fixed, and this window adds no new defects. For BR-9, the `--repeat` branch now gets the point address directly from `pointAddressOf` and refuses if that fails (`cmd/metis/run.go:233-241`). Before, it used `singleRunID`'s address and a separate precondition, so a failed address could still name the claim dir with a timestamp. The rule for the family is now built into the code: a repeat's claim dir comes only from the address it just computed. That closes all three fallback cases: an untracked experiment, an unaddressable config, and any future reason the address can't be computed. For BR-10, both atlas passages now say `.se` appears only at n ≥ 2 and that `--repeat` needs a tracked or addressable experiment. I could not run the git-backed e2e tests: the sandbox refuses `git init` in the test temp dir ("outside the reviewer's directories (sdlc #339)"), and setting `TMPDIR` didn't change where the tests ran. So I checked the regression test by tracing the code instead (details below). That is why confidence is medium. Nothing blocks SHIP.

1. **Strengths**
   - `cmd/metis/run.go:236-241`: the refusal now comes from the same function that names the dir, so there is no separate precondition that can drift from it.
   - `cmd/metis/repeat_e2e_test.go:478-486` checks two things: the specific refusal message, and that no dir was claimed. The second check is what proves the claim is never spent.
   - The atlas edits (`atlas/experiment.md:157-159,180-182`, `atlas/index.md:44-45`) match the code: the claim-release behaviour, the se-at-n≥2 rule (`pkg/ledger/ledger.go:486` per BR-4), and the untracked refusal.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** in the repeat path, `singleRunID` (`run.go:222`) still computes the point address once, and the result is then thrown away. It's harmless; it isn't worth raising as a finding.

5. **Test coverage notes**
   - `TestRepeat_RefusesUnaddressableConfig` reaches the fix: `.nan` is valid YAML, and nothing in `pkg/experiment` or `cmd/metis` rejects a non-finite value before `pointAddressOf`. `record.CanonicalHash` then errors on it (`pkg/record/address.go:19-24`).
   - The message the test expects ("needs a point address") exists only in the new code. Under the old code the run claims `run-<ts>.r-1` and fails later, so the test would go red.
   - I tried to revert the fix and run the test, but that run hit the same `git init` refusal, so the red result is inferred from the code, not observed. `pkg/record` tests pass, and `go vet ./cmd/metis/` builds cleanly against the pinned ariadne.

6. **Architecture (interdiff only)**
   - **ARCH-DRY: pass.** One function names the dir; the hand-written precondition is gone.
   - **ARCH-PURE: pass.** No new IO mixed into logic.
   - **ARCH-PURPOSE: pass.** BR-9 was fixed at the level of the rule, not just the site it named, and BR-10 was fixed in both atlas passages in the same round.
   - **ARCH-EVOLVE: pass.** No persisted format changes in this window.

   There is no conflict resolution to judge. The issue file changes only status, log and card metadata; `## Spec`, `## Revisions` and `## Done when` are untouched, so scope did not change.

7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      run.go:233-241 gets the address via pointAddressOf and refuses on error; the claim dir uses only that address (rule fixed, not the instance); TestRepeat_RefusesUnaddressableConfig reaches it via .nan (verified by tracing; the sandbox's git-init refusal stopped the e2e run).
  - id: BR-10
    disposition: addressed
    note: |
      atlas/experiment.md:157-159,180-182 and atlas/index.md:44-45 now state se only at n>=2 (matches ledger.go:486) and that --repeat needs a tracked/addressable experiment.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ffc21355-9c95-407d-98d0-6e9ecaabe208","repository":"github.com/xianxu/metis","issue_id":"000073","recipe":"fix-review","round":3,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/20be9ef6bc3cbbcd","reviewed_head":"214f1d00904e90d35acfb6e4dc6ef0fcb5b31521","slot":"metis:1","requested_at":"2026-10-25T20:51:00.096497Z"},"phase":"finished","pid":80457,"started_at":"2026-10-25T20:51:00.19467Z","finished_at":"2026-10-25T20:52:09.615269Z","session_id":"20ad1185-b0f7-44a7-b0dc-de8206c18dab","outcome":"returned"}] -->
