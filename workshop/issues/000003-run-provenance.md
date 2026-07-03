---
id: 000003
status: open
deps: []
github_issue:
created: 2026-07-02
updated: 2026-07-02
estimate_hours:
---

# Run provenance: snapshot the resolved pipeline config (+ experiment git sha) so ## Runs is knob→score legible

**Stage: DESIGN — do not build yet.** Operator: "don't fix yet, we need to design
this well." The skeleton is already useful for discussing direction; this captures
the gap + intent. Part of the **metis v1** project (`brain/data/project/metis-v1.md`);
this issue is **L0** in the v1 layering — the resolved-**point** content-address
`(resolved-with, three repo SHAs, seed)` that is the cache key, repro key, and run
identity everything else keys on. Design:
`brain/workshop/pensive/2026-07-03-01-pensive-experiment-shape-workbench-design.md`.

## Problem

A run records **metrics but not the config that produced them**. `run.json` holds
`{id, experiment, seed, metrics, artifacts}` — no consolidated pipeline/steps
block. The runner does write each step's resolved `with.json` into
`runs/<id>/<step>/with.json`, so config *is* captured per-step in the run dir; but:

- The experiment `.md` frontmatter is **mutable and never snapshotted**. Edit a
  knob (e.g. `model: logreg`→`rf`) and re-run, and `## Runs` just appends a new
  metric line — you **cannot tell which frontmatter produced which score**.
- Reusing a run-id overwrites the per-step `with.json`, losing the prior inputs.

For a bench whose entire value is **knob → score**, this is the central miss.
Today's v0 workaround (a real convention, worth documenting): **treat a run-bearing
experiment file as ~immutable; fork a new file per variation** so each file owns its
own `## Runs` history. Provenance-snapshot is what would later make in-place editing
safe.

## Spec (intent, not yet a plan)

On each run, capture enough to answer "what config produced this score," durably:

- Snapshot the **resolved pipeline** (steps + resolved `with` + seed — the runner
  already parses it) into `run.json` (or a `config.json` in the run dir).
- Stamp the experiment file's **git SHA** (and dirty flag) for exact source
  provenance — git already versions the frontmatter; bind the run to the commit.
- Consider enriching `## Runs` (or a generated index) to show the salient
  config diff beside the metric, so the ledger reads as a knob→score table.
- Design in tandem with metis#2 (caching shares the "resolved-config hash") and
  the fork-per-experiment convention.

## Done when

- (design-stage) A design note settles what gets snapshotted (config shape, git
  binding), where it lives, how `## Runs` surfaces knob→score, and the
  relationship to the fork-per-experiment convention + caching keys.

## Plan

- [ ] Design note: provenance record shape + git binding + ## Runs legibility + relation to #2 and fork-per-experiment.
- [ ] (deferred, post-design) implementation milestones.

## Log

### 2026-07-02
- Filed design-stage from the kbench#1 discussion. Operator derived the fork-per-experiment / frontmatter-immutability convention independently and asked to design provenance well before building. Cluster with metis#2 (caching).
