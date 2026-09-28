---
id: '000014'
status: done
started: 2026-07-06T16:33:59-07:00
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 1.79
actual_hours: N/A
---

# Complete + harden code capture — snapshot the run-spec .md, wire single-run capture, make capture failures loud

## Problem

metis#8's side-ref capture is supposed to make a dirty run reproducible (snapshot the exact
code+config bytes to `refs/metis/*`, record the `(path, blob-SHA)` manifest + commit in
`record.json`). Today it under-delivers on three fronts:
1. **The run-spec `.md` is never captured.** The capture closure = the Python read-set
   (`sweepClosure` ← each point's `reads.json`); the experiment `.md` is parsed by the *Go* runner,
   read by no Python step, so it never enters the closure. Only its resolved *values* reach the
   point-address. So "this `titanic-sweep.md` produced the result" isn't actually pinned to a blob.
2. **Capture is sweep-only.** `captureSweepCode` runs from `runSweep`; a plain `metis run`
   (`runResolvedExperiment`) captures nothing — a single dirty experiment run is unreproducible.
3. **Failure is silent/best-effort.** No git / no closure / a git hiccup → capture is a no-op that
   only warns. So you can believe a dirty run was durably captured when it wasn't.

(These three are one issue — all "complete + harden the capture the record promises". Cross-repo
*code* capture is separately metis#11; this issue assumes it and adds the spec-hook + single-run
wiring + loudness.)
