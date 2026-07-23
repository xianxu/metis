---
id: 000068
status: open
deps: []
github_issue:
created: 2026-07-23
updated: 2026-07-23
estimate_hours:
---

# metisser: the earned-subcommand backlog (living list from ml-research investigations)

## Problem

The `ml-research` skill (`construct/local/ml-research/SKILL.md`) is deliberately prose-first: every step is
followed by hand until it proves itself, and only *earned* steps become `metisser` subcommands (the metis
analog of ariadne's `sdlc`). The rogii-v2 rebuild (kbench `competition/rogii-v2/`) is the first full
investigation under the skill, and it has now produced a concrete earned list — each entry backed by a real
failure or a repeated manual ritual, recorded in that experiment's `lessons.md` / skill-compliance audit.

This issue is the **living backlog**: the single place the candidate list accretes across investigations.
Append with provenance (which experiment/failure earned the entry); do not build ahead of the prose (the
skill's own rule — "do not build `metisser` before the prose shape settles"). When an entry is implemented,
tick it here and note the verb it became.

## Spec

Candidates, grouped; ordering within groups ≈ earned-strength (each cites what earned it).

### 1. Lint (`metisser lint`) — the standing-debt scans, all earned by the 2026-07-23 rogii-v2 audit

- [ ] **`†`-scan** — every number in `arrows.md` carries a provenance tag AND a source (path / submission id),
      else a trailing `†`; list the debt. (Earned: the `CLAIMED`-as-`MEASURED` 5.99 phantom; the standing
      `grep '†'` ritual.)
- [ ] **orphan findings** — a `f-*` row cited by no arrow and marked as having changed nothing in
      `framing.md`. (Earned: `f-label-form` — labels measured segmental, model class never updated; the
      audit's costliest miss.)
- [ ] **stale framing** — a form-changing finding newer than the last `framing.md` revision entry. (Earned:
      same case; "framing revised at pivots" had no enforcement.)
- [ ] **open-never-run arrows** — `open` status with no source cell, older than N sessions. (Earned:
      `a-nearest-typewell` sat open through the entire GR program including a verdict it directly tested.)
- [ ] **dead-verdict-with-unswept-held-fixed** — a `dead`/load-bearing verdict whose held-fixed list contains
      a knob with neither a swept axis nor a justification link. (Earned: old-run "tapped out at K=60" hiding
      LB 13.25→12.45; v2's LSEG=50 and HL_MAX_BLOCKS=12 repeats.)
- [ ] **strategy-on-CLAIMED** — anchors/frontier actions citing `CLAIMED` numbers as load-bearing. (Earned:
      the 5.99 phantom shaping a whole thread.)
- [ ] **index/detail consistency** — every `a-*` index row has a `### a-*` detail section and vice versa;
      status values from the enum.

### 2. Ledger mechanics (`metisser arrows`) — the projection/query surface

- [ ] **query verbs** — `open` / `verdicts` / `csv` projections (today: awk one-liners documented in the
      skill; earned by every session re-typing them; also removes the copy-paste hop created when the awk
      block moved from arrows.md into the skill).
- [ ] **`arrow kill <id>`** — the `dead`-verdict gate: refuses without the four impostor-ladder fields
      (fit-check / oracle / dual-measure / held-fixed) + mechanism + revisit-trigger. The sdlc-close analog.
      (Earned: the ladder is what kept v2's five dead verdicts trustworthy; nothing enforces it.)
- [ ] **`arrow open <id>`** — template with oracle-ceiling field, so the oracle-before-build gate has a slot
      from birth.

### 3. Probe discipline (`metisser probe`) — the verification-commit + plausibility machinery

- [ ] **`probe close`** — stage `.py` + same-basename `.log` trace + the `arrows.md` diff and commit
      atomically; refuse if any leg is missing. (Earned: the invariant held all session only by hand; one
      near-miss — a ledger edit almost committed before its trace landed.)
- [ ] **plausibility asserts** — probes declare floors/ceilings/nulls (header convention or tiny API);
      runner reads the trace and fails loudly on violation: below-floor, above-ceiling (leak), null-beats-real.
      (Earned: dz-proposal killed by its own 111.6 floor; flipped-null catching autocorrelation artifacts;
      known-answer calibration killing the curvature-sparsity detector.)
- [ ] **data pin** — workspace-level input checksum file; `pin` / `verify`; on mismatch, flip every
      `MEASURED` to `†`. (In the skill since v0.1; still manual.)

### 4. Evaluation (`metisser cv` / `metisser lb`) — honest-CV + external-anchor machinery

- [ ] **rung-calibration report** — given ladder rungs (row/group/spatial-k…) and external anchors (scored
      submissions), report which rung reproduces the anchors, per model family. (Earned: spatial-block CV at
      every k pinned models at ~15.8 vs LB 12.45 — the "honest = harsher" instinct measurably wrong;
      `f-test-interleaved`.)
- [ ] **per-family CV→external gap table** — the optimism ladder per leg (persistence ≈0 / geometry +1.0 /
      tracker +1.8 was the diagnosis that cracked the LB inversion).
- [ ] **submission ledger** — record each external submission (id, message, config, score) and group
      dose-responses along a knob (the λ = 0/0.5/1 → 12.452/12.496/12.782 monotone chain was ACHIEVED-level
      mechanism confirmation; today it lives in prose).
- [ ] **per-regime report** — quintile-by-difficulty tables alongside totals for any config choice (earned:
      the λ argmax vs regime-robust choice; CV-argmax selection as the inversion mechanism).

### 5. Session ritual (`metisser status`)

- [ ] **session-start brief** — one command printing: frontier NEXT ACTIONS, open arrows, `†` debt, lint
      results, lessons.md pointer (the README entry-point ritual, automated).
- [ ] **session-close check** — frontier updated since last commit? lint clean? uncommitted probe/trace
      pairs? (The "reify before stopping" rule, enforced.)

## Done when

- This issue exists as the single accreting backlog (this file), referenced from the ml-research skill's
  "Status — prose vs built" section, and
- entries get ticked as they are implemented (each implementation is its own issue/milestone; this backlog is
  not itself an implementation plan).

## Plan

- [ ] Keep appending candidates with provenance as investigations run (rogii-v2 ongoing).
- [ ] When ≥1 group feels settled in prose, open a scoped implementation issue for that group and link it here.

## Log

### 2026-07-23
- Created from the rogii-v2 skill-compliance audit + session retrospective (kbench
  `competition/rogii-v2/lessons.md` holds the worked cases with numbers; skill v0.2 = metis `e06c6d1`).
  Groups 1–5 seeded; every entry cites the failure/ritual that earned it.
