---
id: '000001'
status: done
started: 2026-07-01T13:51:21-07:00
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 6
actual_hours: 3.83
---

# metis ML-workbench core: experiment datatype + step-runner + Dataset/Split/cv-split + step-types (Titanic skeleton)

## Problem

metis is an empty scaffold. The `kaggle-ml-base-layer` project (brain) needs the platform-independent ML core: a way to **define, run, and record a reproducible pipeline** (the `experiment` datatype + a Go step-runner), plus the tabular data primitives (Dataset / Schema / Split / cv-split) and the metis step-types (`cv-split`, `train`, `predict`) that the Titanic thread walks through. "Platform-independent" test: *would this be identical on a non-Kaggle platform?* — if yes it lives here.
