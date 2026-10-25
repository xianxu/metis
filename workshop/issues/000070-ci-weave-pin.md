---
id: 000070
status: codecomplete
deps: [ariadne#459]
github_issue:
created: 2026-10-19
updated: 2026-10-25
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9c01b4f0ae35d11e4162f6a81f622c07f7db870c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T04:16:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
flow: {kind: quick, provenance: inferred, spec: "a651e122", done: "a4d38aa0"}
actual_hours: 0.06
---

# CI: build weave from a pinned ariadne commit

## Problem

ariadne#459 reproduced the cold generated-vocabulary failure in published weave
0.1.0, before ariadne#422's shipped-output sidecar ownership. This repository's
active GitHub main merge-check installs the published formula without an exact
source pin. Verified 2026-10-19 against [the exact workflow blob](https://api.github.com/repos/xianxu/metis/git/blobs/29440f124ceaff4c682d4a82fde63a339ae6a1d1).

## Spec

Repository-owned migration after ariadne#459's upstream compatibility guard
lands. Build weave from approved full Ariadne commit
63d6bfd86509d97649e10d7a5dd1e9a06db17d21 before bootstrap, using an authored
.sdlc/weave-compiler-ref outside weave's managed manifest. Follow the reviewed
upstream merge-check template. Install Go from that pinned source's go.mod;
verify fetched HEAD and binary vcs.revision/modified metadata, and prepend its
private output directory to PATH. No floating HEAD/formula or old PATH fallback
on pin acquisition/build failure. Preserve bootstrap, setup and check ordering.

## Done when

- The committed workflow builds the exact selected supported full SHA and uses
  that executable before bootstrap; tests reject malformed pins and fetch/build
  failure and prevent an older PATH weave from winning.
- A cold CI checkout with tracked old generated vocabulary and matching sidecars
  regenerates changed exports and sidecars without authored-replacement refusal;
  an authored/tampered output still refuses without losing its bytes.
- The consumer-authored pin survives compile and is not manifest-managed.
- Operator verifies the local-gates PR and applies local-gates-verified. Agents
  never apply that label. Future pin bumps require the same verification.
- The owner reports the PR, compiler identity and real CI acceptance to the hub.

## Plan

Quick flow. The reviewed upstream template is already on metis main:
`.github/workflows/merge-check.yml` is weave-seeded from ariadne#459 and already
reads, fetches, verifies and builds `.sdlc/weave-compiler-ref` before
bootstrap, with no fallback. Its malformed-pin, fetch/build-failure and
PATH-precedence tests live upstream (`scripts/test/portable-ci.test.sh`,
`scripts/test/weave-compatibility.test.sh`). The only missing piece is the
consumer-authored pin, whose absence is what fails CI today (PR #51).

- [x] Author `.sdlc/weave-compiler-ref` = `63d6bfd86509d97649e10d7a5dd1e9a06db17d21`
      plus a newline; it equals ariadne's own selected and only supported ref
      (`ariadne/.sdlc/weave-compiler-refs`).
- [x] Local evidence: run the workflow's own validator on the file; fetch that
      SHA into a temp checkout, build `./cmd/weave` the workflow's way
      (CGO_ENABLED=0, GOTOOLCHAIN=local, -trimpath) and confirm
      `vcs.revision` equals the pin with `vcs.modified=false`; feed a malformed
      pin (short, uppercase, missing newline) to the validator and see it refuse.
- [x] Not manifest-managed: `construct/base.manifest` has no `.sdlc` entry; run
      `weave refresh` and confirm the pin survives byte-identical.
- [x] Open the PR; real CI is the cold-checkout acceptance (the regenerated
      vocabulary and sidecars, with no authored-replacement refusal). The
      `local-gates-guard` check is expected to fail until the operator verifies
      and applies `local-gates-verified`. Report the PR, compiler identity and CI
      result to the hub. Never apply the label.
- [x] Atlas: note the pin and its bump procedure (operator-verified,
      a supported ariadne ref) in the atlas CI entry.

Out of scope: workflow edits (the seeded template is upstream's), and the
other repositories' pins.

## Log


- 2026-10-25: closed — pin authored at the Spec SHA; workflow validator accepts it and refuses 5 malformed shapes; pinned weave builds with vcs.revision=63d6bfd86509d97649e10d7a5dd1e9a06db17d21 and vcs.modified=false; no manifest names .sdlc; pin byte-identical across weave compile. no-plan-check: the one open item is opening the PR and its real CI run, which follow close (local-gates-guard red until the operator labels).; review verdict: SHIP
Filed under explicit hub authorization from this repository's primary :0
checkout, unclaimed, depending on ariadne#459. Source class A; inventory:
[verified active GitHub consumers](https://github.com/xianxu/ariadne/blob/000459-weave-release-downstream-ci-still-runs-weave-0-1-0-pre-422-vocabulary-changes-break-every-downstream-merge-check/workshop/plans/000459-downstream-compiler-inventory.md).
The upstream owner files the hand-off; this repository's assigned slot and the
operator own implementation and acceptance.

2026-10-25 metis:1: claimed and started. Several claims are open at once in
this slot (#71, #72, #73, #74, #76; ariadne#580 fixes active-time attribution).
The seeded workflow already carries the template, so the plan is the pin file
plus verification.

2026-10-25 metis:1: pin authored (`63d6bfd86509d97649e10d7a5dd1e9a06db17d21`).
Evidence: the workflow's own validator (copied verbatim) accepts it and refuses
short, uppercase, no-newline, extra-line and missing pins (exit 1, its error).
Fetched that SHA from github.com into a temp checkout (HEAD verified), built
`./cmd/weave` with CGO_ENABLED=0 and -trimpath (local go1.27.3; CI selects
go 1.26.3 from the pin's go.mod): `vcs.revision` equals the pin and
`vcs.modified=false`. Unmanaged: neither metis's nor ariadne's manifests name any
`.sdlc` path, and ariadne's `portable-ci.test.sh` asserts the pin is never
generated. `weave refresh` refuses a branch ahead of main, and `weave compile`
stopped in the sandbox at `.claude/settings.json` (not retried outside, per
the hub); the pin stayed byte-identical (sha1 7a28a3fa) across that run.

2026-10-25 metis:1: scope revision (hub-approved; the operator reviews it with
the label). PR #52's merge-check got past the pin (it built the pinned compiler)
and then failed in bootstrap: "git clone -- git@github.com:xianxu/ariadne.git
... git@github.com: Permission denied (publickey)" (run 40188584381). The runner
has no ssh key; construct/deps had carried the ssh URL since 0350013, and every
main merge-check since failed the same way. Switched construct/deps to
https://github.com/xianxu/ariadne.git (ariadne is public; pair uses the same
URL). A local checkout with ../ariadne present never clones, so local slots are
unaffected.

2026-10-25 metis:1: second scope revision (hub-approved, same operator
verification). With https deps, CI passed the pin build and the cold compile,
then experiment-schema-selftest failed ("valid-baseline.md should PASS but was
rejected", run 40188715202): compile builds vocabulary into ../ariadne/bin and
only prints the dir, so CI had no vocabulary on PATH, and the selftest hid the
command-not-found. Added scripts/ci-setup.sh (the seeded post-compile hook),
which appends ../ariadne/bin and bin to GITHUB_PATH; both experiment checks
now source scripts/merge-checks.d/_vocab.bash and exit 2 naming the tool when
it is missing. Removal trigger for the hook: ariadne#627 (the seeded workflow
puts weave's tool dirs on PATH itself). Also found, filed by the hub:
ariadne#628 (40-duplicate-issue-id.sh exits 141, SIGPIPE under pipefail, when
construct/dev-aliases.sh exists; local only).
