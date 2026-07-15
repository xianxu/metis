---
id: 000040
status: open
deps: [metis#41]
github_issue:
created: 2026-07-14
updated: 2026-07-14
estimate_hours:
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

## Spec

A metis skill (`.claude/skills/` per repo convention; shell-invokable knowledge either way):

- **Inputs:** a shape + ledger (+ pinned fingerprint). The skill instructs the agent to: load the
  cohort, render the per-family honest board (outer mean±SE), the top-k configs per family
  (pooled inner mean±SE), flag known estimator biases (co-occurrence/group features under-ranked
  at reduced n — cite the #35 fragmentation analysis), state LB resolution (subsample SE), then
  elicit/apply operator priors and converge on a pick.
- **Output:** a chosen config identified by ledger row (`point_addr`) → `metis select --point
  <addr> --promote` (metis#41 — the skill's actuation seam) + a rationale paragraph recorded in
  the issue/project Log (the auditable trail of a human-prior override).
- The skill encodes the PROCEDURE (what to compute, what to flag, what to ask); numbers always
  come from the ledger via commands — no LLM-remembered statistics.
- Explicitly out: `select --where` query language (deferred — operator: "we will drive select
  directly in this session"; the skill + point-select cover the near-term need).

## Done when

- Skill file exists + discoverable; a session invoking `/metis-select` on the titanic-sweep ledger
  reproduces (at minimum) the 2026-07-14 analysis: honest board, per-config drill-down, bias
  flags, and ends with a `--point` promote + logged rationale.
- Authored via superpowers-writing-skills (with verification pass).

## Plan

- [ ] After metis#41 (the actuation seam). Author with superpowers-writing-skills; the 2026-07-14
  session transcript is the reference procedure.

## Log

### 2026-07-14
- Filed from the metis#35 honest-beat session — the operator's selection-as-conversation call
  (option 1 over `select`-drives-an-LLM): judgment stays where it's inspectable; the binary owes
  it machine-legible measurement. Siblings from the same session: metis#38 (TUI), metis#39
  (fingerprint visibility), metis#41 (point-select), kaggle#6 (submit auto-description).
