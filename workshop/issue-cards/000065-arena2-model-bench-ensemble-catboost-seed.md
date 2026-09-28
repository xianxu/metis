---
id: '000065'
status: done
started: 2026-07-19T01:03:44-07:00
created: 2026-07-19
updated: 2026-07-19
estimate_hours: 1.36
actual_hours: N/A
---

# arena2 model bench: ensemble kind + catboost + seed passthrough

## Problem

Arena2's residual gap to the ~0.953 pack (our best public 0.94966) has, by the M3/M4
ledger, two independent tree families CONVERGED to honest OUTER ~0.9506 — the signature of
a data noise floor at the single-model level. Two operator-directed moves remain untried:

1. **Blend, measured honestly.** `metis blend` (metis#60 M2) is a post-hoc soft-vote over
   PROMOTED runs — leaderboard-only, **no in-sweep OOF**, so it cannot answer "does blending
   help the OUTER CV" without spending a submission slot. The honest way to measure a blend
   is to make it a config the nested-CV sweep scores like any model.
2. **A new mechanism + cheap variance reduction** (the M5 bench pensive): CatBoost — the one
   real mechanism argument (per-node ordered target statistics; the sharper test of the
   "cell signal matters *conditionally*" hypothesis the flat global encoding closed at M3;
   the most-different boosting bias → best blend partner) — and seed-bagging the incumbent,
   which needs only a `params.seed` override to unlock.

These are three additions to the pure model core (`metis/model.py`), each Python-only: the
Go layer derives the family structurally (`FamilyOf` reads the `$any`-map branch label), and
the atlas records "adding a model kind is Python-only (MODELS + make_model + complexity)".
