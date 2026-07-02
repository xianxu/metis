# Boundary Review — metis#1 (milestone M1)

| field | value |
|-------|-------|
| issue | 1 — metis ML-workbench core: experiment datatype + step-runner + Dataset/Split/cv-split + step-types (Titanic skeleton) |
| repo | metis |
| issue file | workshop/issues/000001-metis-ml-workbench-core-experiment-datatype-step-runner-dataset-split-cv-split-step-types-titanic-skeleton.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | ceb7f7e8657fbce100b55c169ad18ba81dcffd17^..HEAD |
| command | sdlc milestone-close --issue 1 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-01T14:30:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers exactly the declarative core it scopes: a genuinely single-sourced CUE schema (`#Experiment`/`#Step`/`#Status`/`#Run`) whose validator I confirmed works end-to-end (valid fixture → exit 0; invalid → exit 1 with a sharp `status: "running" is not valid (want: active|archived|draft)` diagnostic), a datatype prototype that compiles into the generated `xx-datatype` skill, honest SHAPE-vs-SEMANTIC framing, and a properly cross-linked atlas. What blocks a clean SHIP is the one piece that is the milestone's ARCH-PURPOSE headline — the enforcement merge-check. It does **not** honor the `run-merge-checks.sh` `<base> <head>` contract and **silently passes when its git-diff base fails to resolve** — both reproduced below. The schema/prototype/atlas are solid; the fix is ~5 lines in one file, so this is fix-then-ship, not rework.

### 1. Strengths (confirmed-good ground)

- **The schema is single-sourced and actually enforced by shape.** `construct/vocabulary/experiment.cue` is a closed definition; I verified the inherited `vocabulary validate-instance --type experiment` reads the `.cue` directly (blanked the generated `experiment.json` → still rejected), so there's no stale-artifact coupling. Valid passes, invalid rejects. This is the real ARCH-DRY win.
- **The two *live* consumers derive.** `experiment` appears in the generated `construct/generated/datatype/SKILL.md` trigger list (prototype → skill), and the validator derives from the `.cue`. ARCH-DRY holds for M1's shipped consumers; the Go consumer is legitimately M2.
- **The SHAPE/SEMANTIC split is honest and ARCH-PURE-clean.** The `.cue` header and plan both state plainly that `needs`-resolution / DAG-acyclicity / `uses`-format are not expressible in `cue vet` and are deferred to M2's pure Go validator. No business logic is buried in IO because there is none yet — the seam is correctly reasoned, not hand-waved.
- **Atlas gate satisfied.** `atlas/experiment.md` + `atlas/index.md` added; index links `experiment.md` and the `workflow` symlink (resolves to `../../ariadne/atlas/workflow`).
- **The negative fixture is well-isolated** (`invalid-bad-status.md` differs from valid only by the out-of-enum `status`), so a rejection unambiguously means the enum is enforced.

### 2. Critical findings

**C1 — `experiment-validate.sh` ignores the merge-runner's `<base> <head>` contract and silently passes on git-diff failure.** (`scripts/merge-checks.d/experiment-validate.sh:17,30`; ARCH-DRY + silent-swallow)

`scripts/run-merge-checks.sh:49` invokes every check as `"$c" "$BASE" "$HEAD"`, and CI (`.github/workflows/merge-check.yml`) computes `base="$(git merge-base …)"` and passes it positionally. This hook discards `$1`/`$2` and instead recomputes its own range with `base="${MERGE_CHECK_BASE:-origin/main}"` … `git diff … "$base"...HEAD`. Two demonstrated consequences:

- **Ignores the passed range** — invoked with an *empty* positional range (`… HEAD HEAD`), a contract-honoring check scans nothing; this hook still diffs `origin/main...HEAD` and flags the fixture:
  ```
  $ EXPERIMENT_VALIDATE_INCLUDE_TESTDATA=1 experiment-validate.sh HEAD HEAD   → exit 1
  ```
- **Silent pass when the base doesn't resolve** (fresh CI checkout without `origin/main`, or a repo whose default branch isn't `main`) — the `git diff` failure inside the `< <(…)` process substitution is swallowed by `set -e`, the loop reads nothing, `fail` stays 0:
  ```
  $ MERGE_CHECK_BASE=origin/definitely-not-a-ref EXPERIMENT_VALIDATE_INCLUDE_TESTDATA=1 experiment-validate.sh HEAD HEAD
    fatal: bad revision 'origin/definitely-not-a-ref...HEAD'   → exit 0   # invalid experiment unchecked
  ```

This is exactly the "silent error swallowing" + "behavior drift from a documented contract" the gate guards against, in the seam whose entire justification (per the plan and the Log's "proven") is *enforcement*. The Log's proof used a hand-set `MERGE_CHECK_BASE=$(git rev-list --max-parents=0 HEAD)` — i.e. it only ever passed because the operator supplied the arg the real gate does not. Blast radius on metis *today* is low (no non-testdata experiment instances exist yet; instances live in kbench), but this hook is the template kbench inherits for its real experiments, so it should be correct before crossing.

Fix sketch — consume the runner's args and fail loudly:
```bash
base="${1:-${MERGE_CHECK_BASE:-origin/main}}"
head="${2:-HEAD}"
files="$(git diff --name-only "$base" "$head" -- '*.md')"   # runner already passed merge-base..head → two-dot is exact
while IFS= read -r f; do … done <<< "$files"
```
(Assigning to a variable first makes the git failure abort under `set -e` instead of being swallowed by process substitution.)

### 3. Important findings

**I1 — No automated test exercises the fixtures; the negative case never runs in CI.** The merge-check *skips* `testdata/` by default (`:22-24`), so the only committed thing that could validate `invalid-bad-status.md` never does in normal operation. Nothing else in the tree references the fixtures (grep across `*.sh`/`*.go`/`Makefile*`/`*.bats` → only the issue/plan/atlas/hook/fixtures themselves). Consequence: a schema regression (e.g. someone widens `#Status`) ships unnoticed — the exact bug the fixtures were meant to catch. The hook comment even cites "the M1 validator test," which does not exist as an automated artifact. Recommend a tiny committed self-test (a `merge-checks.d` self-check or shell/bats target) asserting `valid → 0` **and** `invalid → 1`; the `EXPERIMENT_VALIDATE_INCLUDE_TESTDATA=1` flag already exists to drive it.

### 4. Minor findings

- `head -8` frontmatter probe (`:26`) requires `type:` within the first 8 lines — brittle if a real experiment reorders frontmatter (id/competition/seed above type). Prefer parsing the fenced frontmatter block, or widen/document the assumption.
- The datatype prototype hand-restates the enum (`draft | active | archived`) in both the shape table and the `description:` — a drift risk vs the `.cue`. It matches the inherited ariadne `issue`/`project` prototype convention (authoring guidance, not an enforced consumer), so acceptable, but worth a watch-line.
- Plan snippet drift: plan Task 3 shows `head -5` + `grep -q '^type: experiment$'`; the shipped hook uses `head -8` + a lenient regex (better). Harmless, but the plan should reflect what shipped.

### 5. Test coverage notes

The validator behavior is real (I ran both fixtures directly). The gap is *automation*: the positive path is only exercised incidentally when a real experiment file is in a PR diff, and the negative path is exercised **never** (testdata skipped). For a milestone whose deliverable is "enforcement," add I1's self-test so the enforcement is itself regression-tested. C1's two failure modes also have no test — the fix should come with a case that passes `HEAD HEAD` and asserts scope, and one that asserts a bad base fails loudly.

### 6. Architectural notes for upcoming work (M2)

- M2's Go validator must implement the deferred semantics (`needs`→id resolution, DAG acyclicity via topo-sort, `uses = <layer>/<steptype>`), and the plan already promises the merge-check will invoke it — fold that in when C1 is fixed so the hook has one correct range-and-invoke path.
- Once C1 is fixed, this hook becomes kbench's inherited enforcement template. Getting the runner-contract adherence right now avoids propagating the silent-pass to the repo that actually holds experiment instances.
- Consider making the negative-fixture self-test the standard pattern for future metis-owned datatypes (kaggle/kbench will each add schemas + fixtures).

### 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000001-experiment-datatype-plan.md`:

- **Task 3 (merge-check) — contract fix.** The planned hook (Task 3 Step 1 code block: `base="${MERGE_CHECK_BASE:-origin/main}"` + hardcoded `HEAD`, ignoring positional args) is the source of C1. Revise the plan to require the check to **consume the `run-merge-checks.sh` `<base> <head>` positional args** and to compute the changed-file list into a variable (so a bad base aborts under `set -e` rather than silently passing). Note this applies to the shipped `head -8`/lenient-regex form, not the `head -5` snippet in the plan.
- **Task 3 — add the negative self-test** (I1) as an explicit checklist step: a committed assertion that `invalid-bad-status.md` is rejected and `valid-baseline.md` passes, so the milestone's "proven" claim is backed by an automated artifact rather than a one-off manual run.
