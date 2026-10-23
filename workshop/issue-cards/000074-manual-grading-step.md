---
id: 000074
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
---

# Manual grading step with attributable result and separate human wait

## Problem

The operator added human grading to couch-bench (ariadne#577): after hidden scoring, a person must sometimes inspect and try the result, especially for UI tasks. Metis needs a truthful pending manual-step outcome, with separately measured human waiting; a cached grade or inferred acceptance is invalid evidence.
