---
id: '000006'
status: done
started: 2026-07-05T16:41:40-07:00
created: 2026-07-03
updated: 2026-07-05
estimate_hours: 2
actual_hours: 1.12
---

# experiment-shape datatype: lift the experiment config schema into a config-space (Space[T])

## Problem

v0 has `experiment` — a single reproducible config *instance*. To explore many
configurations (the point of an ML workbench), we need the *type/space* above the
instance: a `experiment-shape` that declares a set of configs and expands into
concrete points. The key insight (design pensive): experiment-shape is the
experiment config schema **lifted** — each leaf value-type `T` becomes a *space over
T* — and `experiment` is the special case where every leaf is a singleton.
