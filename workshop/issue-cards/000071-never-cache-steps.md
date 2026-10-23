---
id: 000071
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
---

# Step-level never-cache policy with safe downstream invalidation

## Problem

ariadne#577 is being re-planned as a Metis consumer (couch-bench). An agent session and a human grading step are deliberately nondeterministic; a cache hit would erase a repeat or reuse a person's old judgment. The current cache has only a global switch, and downstream keys use upstream input identities, so simply bypassing the session cache can still reuse a stale score.
