---
id: 000005
status: open
created: 2026-07-02
updated: 2026-07-02
estimate_hours:
github_issue:
---

# metis/describe step: per-feature distribution + class balance for human inspection

## Problem

The pipeline goes raw → adapt → model with no place for a human to *look at the
data*. For a learning bench, seeing each feature's distribution and the target's
class balance is a core early step (EDA) — it's how you decide what to engineer and
whether the adaptation looks sane.
