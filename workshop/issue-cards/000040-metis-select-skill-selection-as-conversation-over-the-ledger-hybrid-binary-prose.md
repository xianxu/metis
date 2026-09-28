---
id: 000040
status: open
created: 2026-07-14
updated: 2026-07-14
estimate_hours:
github_issue:
---

# metis-select skill — selection-as-conversation over the ledger (hybrid binary+prose)

## Problem

The metis#35 honest-beat (2026-07-14) showed selection is a JUDGMENT, not an argmax: the operator +
agent weighed the honest estimates (rf 0.8328±0.0045 picked by 1-SE rule), a quantified estimator
bias (nested measurement under-ranks co-occurrence features — ticket coverage 38.6%→~30% under the
seal, m=10 shrinkage), LB subsample noise (±0.029 on ~209 rows), fold-level evidence (one ticket
config won outer fold 3 outright), and an operator prior ("insist on ticket_survival; worst case a
slightly larger model") — none of which `metis select --best` can or should encode. The session's
ad-hoc python-over-csv queries were the prototype of the missing surface.

**The hybrid-system observation (operator, verbatim intent):** we started with a binary, needed
more intelligence, and the answer is to re-implement some of the selection logic in PROSE via the
skill system — the deterministic shell measures (run, ledger, estimates, guards); the skill
encodes the judgment procedure an LLM walks WITH the operator. `metis select` stays the mechanical
default; `/metis-select` is the conversational override path.
