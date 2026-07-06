---
status: active
type: pensive
created: 2026-07-06
---

# Reproducible dirty-run capture — the workbench reproducibility spine

Settled with the operator 2026-07-06 (walking-through the dirty-iteration scenario). This note
is the source-of-truth for the follow-on metis issues (#11 + #13/#14/#15 below); the issues cite it.

## The decision: allow-dirty (not require-clean)

Two coherent worlds for run identity (see the metis-v1 close discussion):
- **require-clean** — `krun` refuses on a dirty tree; the git commit SHA *is* the identity; the
  side-ref/freeze/dirty machinery evaporates. Simpler, but gates fast iteration.
- **allow-dirty** — run with uncommitted edits; snapshot the dirty code+config to git off-branch so
  the run is still reproducible. metis#8's direction.

**Operator chose allow-dirty** — fast dirty iteration (edit → run → edit, no commits) is a
first-class workbench workflow (the "42 shots"). So the job is to make the dirty run *actually*
reproducibly captured — today it is NOT (the capture is sweep-only, best-effort, and blind to both
the consumer repo's code and the run-spec itself).

## The target scenario (operator's words, validated)

1. `features.py` + `titanic-sweep.md` are dirty/new; `krun titanic-sweep.md`.
2. Both files stay dirty in the worktree, but metis **side-commits** their exact bytes to git
   (`refs/metis/*`) and gets back a `(path, blob-SHA)` manifest + a commit SHA.
3. The run's `runs/<id>/record.json` records that **those exact bytes** (code + run-spec + seed)
   produced the result — recoverable even though nothing was committed to a branch.
4. Iterate freely, no commits, each run reproducibly captured.
5. When a result is good, commit on the branch + send for review.
6. Promote the best sweep point → a standalone committed experiment + submission.

## Invariants (why the pieces are where they are)

- **git = the durable code CAS.** Code/config are irreplaceable; git blobs are GC-protected and
  survive a CAS wipe. (This is why dirty capture goes to git side-refs, NOT the metis CAS —
  the CAS is deliberately `rm -rf`-safe / wipeable, holding only *recomputable* output blobs.)
- **The metis CAS = wipeable output blobs only.** Nothing irreplaceable lives there.
- **The config `.md` is immutable input.** Its content-hash must depend only on the author's
  intent, never on run output — so a committed config is a stable identity.
- **Two capture hooks, by kind:**
  - *code a step executes* (`features.py`, `metis/*.py`) → the **read-set trace** (Python audit
    hook → `reads.json`) already wants to see it. Blind spot = cross-repo (metis#11).
  - *the run-spec itself* (`titanic-sweep.md`) → **no step reads it** (the Go runner parses it), so
    the trace will never pick it up. Needs an explicit "hash + side-commit the `.md` I was handed".

## The five-item punch-list (→ issues)

Dependency order matters (4 gates 2):

1. **[metis#11]** Trace multi-root — first-party code from *every* repo on `METIS_STEP_PATH` (or root
   the read-set at the traced module's own repo) enters `reads.json`. Today `features.py` (a kbench
   file) is dropped because `trace.py` roots at the metis repo. Without this, "capture the code" is a
   lie for consumer-repo steps.
2. **[#13] Config immutability — run output leaves the `.md`.** Stop `appendRunLog` (`run.go:184`) +
   the sweep top-N regen from mutating the experiment file. Run output → a sidecar / `runs/` only.
   **Prereq for capturing the spec** (can't snapshot a file the run rewrites). The `## Runs` /
   `<!-- metis:ledger:begin -->` blocks come OUT of the config.
3. **[#14] Capture the run-spec + single-run capture + loud failures.**
   - Hash + side-commit the experiment `.md` bytes into the record's manifest (the second hook).
   - Wire capture into the **single-run** path (`runResolvedExperiment`), not just `runSweep`.
   - Make capture failure **loud** — a run that couldn't durably capture its code must say so
     (today best-effort/silent → you can believe a dirty run is reproducible when it isn't).
   (These three are one issue — they're all "complete + harden the capture the record promises".)

(#13 and #14 could be milestones of one issue; keeping them separate since #13 is an independent
behavior change consumers see — output moves — and #14 is the capture mechanism.)

## Open sub-questions to settle at plan time

- **Where does run output go once it leaves the `.md`?** Options: `runs/<id>/` only (record.json
  already there) + the `.ledger.csv` sidecar for sweeps; a per-shape `.runs.md`/summary sidecar for
  human browsing. Leaning: `record.json` + the ledger sidecar are the record; a *generated* summary
  (regenerated, gitignored, or a sidecar) for eyeballing — never the committed config.
- **Ref namespace for single/experiment runs** — `refs/metis/sweeps/*` is sweep-scoped; single runs
  want `refs/metis/runs/*` (or unify under `refs/metis/captures/*`).
- **Does the point-address need the spec blob-hash?** Today it's `(resolvedWith, repoSHAs, seed)`.
  The resolved values already pin semantics per-point; the spec blob is for *recovering the sweep
  definition* + the human address. Likely record it in the manifest, not fold it into the address
  (avoid churn) — decide at plan time.
