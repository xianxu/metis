---
id: 000072
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
---

# Bound step execution with timeout and live logs

## Problem

ariadne#577 needs a long Couch session as a Metis step. The legacy execStep path uses exec.Command and CombinedOutput: no step deadline, no live child log, and output accumulates in memory. A timeout cannot be mistaken for retirement of independently managed Couch actors.
