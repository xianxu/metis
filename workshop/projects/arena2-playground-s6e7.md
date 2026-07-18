---
type: project
name: arena2-playground-s6e7
goal: "Second arena: run Kaggle Playground Series S6E7 (Predicting Student Health Risk) through the metis workbench end-to-end — prove the workbench GENERALIZES beyond titanic with zero new features until the competition demands them (the v1→v2 pattern's next turn)."
done_when: "A live S6E7 submission produced entirely by the honest flow (metis run → select --best --promote → kaggle submit -C), on a content-pinned dataset, with the nested-CV honest estimate recorded — and a Log entry stating which workbench features the competition actually DEMANDED (possibly none). Leaderboard position is evidence, not the goal (the tight ~0.953 pack makes flips noise; the deliverable is the generalization proof + the demand list)."
status: active
operator: xianxu
explicitly_out: [new workbench features built speculatively]
created: 2026-07-18
updated: 2026-07-18
sources: [brain/data/project/metis-v2-experiment-algebra.md]
---

# arena2 — Playground Series S6E7 through the workbench

Successor to metis-v2 (closed 2026-07-17, done_when met; see `sources`). The arena: Kaggle
**Playground Series S6E7 — "Predicting Student Health Risk"** (closes 2026-07-31; 62MB
train.csv ≈ 100×  titanic's rows; leaderboard packed at ~0.953, metric to confirm from the
overview — likely AUC). Chosen 2026-07-18 (operator) over rogii-wellbore (grouped sequence
data — a step-type lift, candidate third arena) and autonomous-agent-prediction-beta (the
meta-competition mirror — parked, different game).

**The standing rule:** zero new workbench features until S6E7 demands them. Queued demand
candidates, in likelihood order: a new objective metric declaration (AUC — small step-type
addition), metis#33 (GBM regularization defaults — live at real row counts), metis#54
(racing sampler — if the grid cost bites at 100× data), possibly per-competition step reuse
friction (kbench layering). Everything else is out unless the competition says otherwise.

## Milestones

- [ ] **M0 — access (operator):** accept S6E7 rules on kaggle.com (the download 403s until
  then). One click; unblocks everything.
- [ ] **M1 — bring-up (kbench#12):** `competition/playground-s6e7/` workspace mirroring
  titanic's layout: get-data (kaggle/download + `with.sha256` pins from the first download's
  paste-ready block — the #25 flow), an `adapt` step for the S6E7 schema, a starter shape
  (small grid, `inner_k` from day one), smoke on `--fast`.
- [ ] **M2 — first honest submission:** full grid → `select --best --promote` → `kaggle
  submit -C` → record public score vs honest estimate in the Log (the tracking datum).
- [ ] **M3 — iterate:** feature blocks + families as the leaderboard/estimate gap directs;
  file demanded workbench features as they surface (the demand list IS a deliverable).

## Log

### 2026-07-18
- Project opened (operator: "ok, let's do this as 2nd test of metis"). Arena research + the
  candidate ranking recorded in the session; S6E7 chosen. Charter note: this file lives in
  metis/workshop/projects/ (the 2026-07-17 charter — projects in the center-of-gravity repo,
  never brain).
