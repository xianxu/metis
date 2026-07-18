---
id: 000034
status: codecomplete
deps: []
github_issue:
created: 2026-07-13
updated: 2026-07-17
estimate_hours: 0.38
started: 2026-07-17T17:12:38-07:00
actual_hours: 0.35
---

# repo-root-relative shape path as canonical key (cwd-independent run/select/submit)

## Problem

The user works one pipeline at a time and naturally sits **inside** that pipeline dir (e.g.
`competition/titanic/pipelines/`), invoking `metis run titanic-sweep.md`. But the shape path is used as
an identity/output anchor, so behavior can drift with cwd. We want the **repo-root-relative path** to be
the canonical key regardless of where `metis` is invoked from: `titanic-sweep.md` (from the pipeline dir)
and `competition/titanic/pipelines/titanic-sweep.md` (from the repo root) must resolve to the **same**
canonical key `competition/titanic/pipelines/titanic-sweep.md`. Then `metis run`, `metis select`, and
`kaggle submit` are all cwd-independent and consistently rooted in the pipeline dir. (Split out of metis#32's
brainstorm — orthogonal to the selection algebra.)

## Spec

- Resolve any passed shape path → repo-root-relative canonical form (walk up to the repo root; error clearly
  if outside a repo). Use that canonical string wherever the path is an identity/anchor term.
- Verify it does NOT perturb content-addressing (the point-address is the shape's *blob-hash* + config, not
  its path — confirm cwd-independence holds end-to-end for run dirs, the ledger sidecar location, and the
  record's slug rooting so `kaggle submit` resolves consistently).
- `metis run titanic-sweep.md` (from pipeline dir) and `metis run competition/titanic/pipelines/titanic-sweep.md`
  (from repo root) produce identical run identities + land outputs in the same place.

## Done when

- Invoking `metis run` / `metis select` on the same shape from the pipeline dir vs. the repo root yields
  identical run ids, ledger location, and output dirs (a cwd-independence test).
- `kaggle submit --run <id>` resolves consistently regardless of the cwd `metis run` was invoked from.


## Spec (at claim, 2026-07-17 — recon reframe: pin the invariant, fix the two real drifts)

A very-thorough path-as-identity audit (fresh recon agent; table in the session record) found the
premise MOSTLY ALREADY SATISFIED: identity is content-addressed everywhere (`shapeBlobHash`,
`PointAddress`, `shapeRunIdentity` — no path-string term), output anchors are
`filepath.Abs(Dir(expPath))`-derived, and the ledger sidecar sits NEXT TO the shape file — so
`metis run`/`select` from the pipeline dir vs repo root already yield identical ids, ledger, and
output dirs. The canonical-path-as-key mechanism the title asks for is NOT needed for
correctness (Simplicity First; ARCH-PURPOSE is served by the invariant, not the mechanism).
What actually drifts with cwd — exactly two spots:

1. **`kaggle submit --run <id>`** (kaggle/cmd/kaggle/main.go:66,76): `runs/<id>/…` joined
   against CWD, no anchor — breaks from anywhere but the pipeline dir. Fix: a git-style
   `-C <dir>` flag (default `.`); error names the missing `runs/<id>` under the anchor.
   (kaggle deliberately imports no metis package — self-contained fix.)
2. **`metis` steppath fallback** (cmd/metis/steppath.go:42-46): the no-construct-workspace
   fallback anchors `repo.Root` on `os.Getwd()` instead of the shape's own dir. One line
   (house style per the #11 close-review: anchor on the resolved path, not cwd) + test.

Plus the regression net the original done-when asked for: a **cwd-independence test** driving
`runExperiment` on the same shape from two cwds → identical run id + physical output dir
(pins the existing invariant against future drift).

### Done when (reframed — supersedes the original)

- The cwd-independence test passes (same run id + same physical runs dir from pipeline-dir and
  repo-root style invocations).
- steppath fallback is shape-anchored (test: bare repo, shape in subdir, cwd elsewhere → step
  path anchored at the shape's repo, not cwd's).
- `kaggle submit -C <pipeline-dir> --run <id>` resolves from any cwd (test via fake CLI);
  default `.` keeps the RUNBOOK/#50 paste-lines working unchanged.
- Issue Log records the audit verdict (premise false for run/select — invariant pre-existing).

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.04 impl=0.18
item: smaller-go-module   design=0.03 impl=0.12
design-buffer: 0.15
total: 0.38
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Rows: (1) kaggle `-C` flag + tests + metis steppath one-liner + test; (2) metis cwd-independence
regression test + Log/atlas line.

## Plan

Recon table = the design (no separate plan doc — two one-file fixes + one test; the audit did
the brainwork, plan-quality gate judges this issue file).

- [x] metis: cwd-independence regression test (runExperiment from two cwds → same id + dir)
- [x] metis: steppath fallback anchored on shape dir (test: cwd elsewhere)
- [x] kaggle: `-C` flag on submit --run + fake-CLI test from foreign cwd
- [x] Log evidence + atlas one-liner (path is location, never identity)

## Log

### 2026-07-13
- Filed from the metis#32 brainstorm (operator): "use the relative-to-repo-root path as the full key … so
  the user can `metis run titanic-sweep.md` from inside the pipeline dir and `metis select titanic-sweep.md
  --best` consistently, and kaggle submit stays rooted in the pipelines dir." Orthogonal to #32's selection
  algebra → split out.

### 2026-07-17 (built — evidence)
- 2026-07-17: closed — metis -race suite green incl. new cwd-independence net (same shape, two cwds -> same runs dir + identical point_address) + red-proofed steppath anchor test; kaggle suite green, -C foreign-cwd test + failure-mode test; kaggle commit 8addd9f pinned in Log. actual 0.35h = LABELED JUDGMENT (brain-dir session transcripts unattributable, same as #25); review verdict: SHIP
- **Audit verdict recorded:** premise FALSE for `metis run`/`select` — identity content-addressed
  (shapeBlobHash/PointAddress/shapeRunIdentity), anchors Abs(Dir(expPath))-derived, sidecar
  next-to-shape. No canonical-path key built (Simplicity First; the invariant IS the deliverable).
- **metis** (this branch): steppath bare-repo fallback now anchors on the shape's repo (was
  os.Getwd; red-proofed — reverting fails the new test) + `TestRun_CwdIndependentIdentityAndLocation`
  (two cwds, same shape → same physical runs/ dir + identical point_address). Full `-race` green.
- **kaggle** (commit `8addd9f`, pushed): `submit -C <pipeline-dir>` anchors `runs/<id>` from any
  cwd (the audit's one true drift); default `.` keeps #50's paste-lines unchanged; miss error
  names the anchor; usage updated; foreign-cwd test + failure-mode test. Full kaggle suite green.
