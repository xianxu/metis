---
id: '000060'
status: done
started: 2026-07-18T22:35:41-07:00
created: 2026-07-18
updated: 2026-07-19
estimate_hours: 1.6
actual_hours: 1.98
---

# predict probabilities + decision step: threshold tuning and family blending

## Problem

`metis/predict` emits hard labels, which forecloses the two highest-EV M3 moves for arena2
(and any future imbalanced-metric competition): per-class threshold tuning (the balanced-
accuracy-optimal decision rule — argmax is only optimal for accuracy; `class_weight` is the
crude training-time tilt, and tree-ensemble probabilities are miscalibrated anyway, so the
empirical tune beats the divide-by-prior formula) and family blending (`select
--best-per-model-class --promote` already materializes one run per family — metis#22 — but
nothing can combine them). Both need the SAME missing primitive: probability outputs plus a
decision step. Demand #3 from arena2 (operator-approved direction, 2026-07-18 session).

