---
id: '000053'
status: done
started: 2026-07-17T23:05:12-07:00
created: 2026-07-16
updated: 2026-07-17
estimate_hours: 0.48
actual_hours: 0.6
---

# promote fingerprint-consistency guard — refuse when the working tree is not the cohort's code

## Problem

`metis select --fingerprint <fp> --best --promote` selects on ONE code cohort but executes the
promoted run against the CURRENT working tree — with no check that they are the same code.
Same-session promote (tree unchanged) is sound and is the common case. But promote-after-drift
(any edit to a closure file since the sweep) silently ships a submission from code that never
produced the honest estimate the operator selected on — the exact silent-blend class the #32
cohort guard stops at the LEDGER, left open at the PROMOTE seam. Today's only tell is the #39
`recording under code_fingerprint` line on the promote run differing from the pin — visible,
never enforced. (Operator question 2026-07-16: "do we need a way to restore to that state? is
that already happening when we run select --promote?" — answer: no; restore is metis#28,
detection is THIS issue.)
