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
- (Revision 1) CI clones the substrate over https from `construct/deps`; a
  local checkout with ../ariadne present never clones.
- (Revision 2) CI finds the layer tools by name: `scripts/ci-setup.sh` puts
  ../ariadne/bin and bin on GITHUB_PATH until ariadne#627 lands, and is then
  deleted with its atlas line.
- (Revision 2) Both experiment merge-checks exit 2 naming a missing vocabulary
  tool instead of reporting a schema verdict; `tests/test_merge_checks.py`
  pins it.

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

## Revisions

- 2026-10-25T04:24 (hub-approved; the operator reviews it with the
  local-gates label) — reason: PR #52's real CI got past the pin build, then
  bootstrap could not clone the substrate over ssh ("Permission denied
  (publickey)", run 40188584381). Delta: `construct/deps` clones
  https://github.com/xianxu/ariadne.git; a local ../ariadne never clones.
- 2026-10-25T04:27 (hub-approved; same verification) — reason: the next CI
  run failed experiment-schema-selftest because compile builds vocabulary into
  ../ariadne/bin without putting it on PATH, and the check hid the
  command-not-found (run 40188715202). Delta: `scripts/ci-setup.sh` appends the
  layer tool dirs to GITHUB_PATH (temporary, removed by ariadne#627); both
  experiment checks source `scripts/merge-checks.d/_vocab.bash` and exit 2
  naming a missing tool; `tests/test_merge_checks.py` pins that.

## Log



- 2026-10-25: closed — pin 63d6bfd86509d97649e10d7a5dd1e9a06db17d21 validated and built with matching vcs.revision, vcs.modified=false; PR #52 merge-check green (runs 40189012386, 40189166668) through cold bootstrap and compile after two hub-approved revisions (https deps; ci-setup PATH hook plus experiment checks exit 2 on a missing vocabulary, pinned by tests/test_merge_checks.py: 3 pass, guard revert fails). Done-when now covers both revisions; per-clause evidence in the Log. local-gates-guard red awaiting the operator label.; review verdict: SHIP
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

2026-10-25 metis:1: real CI acceptance: PR #52 merge-check passed on
815ebebd6f442ac687aaf62fa787937f92db2add (run 40189012386) and on
c4d3e44551119057c83280310d7f5050b3316716 (run 40189166668, 1m19s), with the
cold bootstrap and compile under the pinned compiler; only local-gates-guard is
red, awaiting the operator label. The fixed-then-ship re-close returned REWORK
(BR-2: the post-close commits are scope revisions, so a full close is due);
recorded them under Revisions and added tests/test_merge_checks.py (3 pass;
reverting the selftest guard fails it).

2026-10-25 metis:1: evidence per Done-when clause. (1) Pinned build before
bootstrap: CI runs 40189012386 and 40189166668 build 63d6bfd8 with vcs.revision
matching and vcs.modified=false; the malformed-pin, fetch/build-failure and
PATH-precedence tests are upstream (portable-ci.test.sh, weave-compatibility.test.sh),
and the validator was rerun here on five malformed shapes. (2) Cold checkout: the
same runs compile cold under the pinned compiler with no authored-replacement
refusal; the tampered-output refusal is upstream's compatibility check. (3) Pin
survives compile and is unmanaged: no manifest names .sdlc; the CI compiles ran
with it present. (4) Operator label: pending, owned by the operator (the hub
put PR #52 in front of them); agents do not apply it. (5) Hub reports: Couch
receipts 15c3d811 (plan), 78b7ef4d (ssh blocker), a6af3ffd (ci-setup/vocabulary,
filings), 69dc2ddf (PR #52 green apart from the guard, compiler identity).
(Revision 1) https clone: run 40188715202 and later pass bootstrap. (Revision 2)
PATH hook: runs 40189012386, 40189166668 pass experiment-schema-selftest;
missing-tool refusal: tests/test_merge_checks.py 3 pass, reverting the guard fails.
