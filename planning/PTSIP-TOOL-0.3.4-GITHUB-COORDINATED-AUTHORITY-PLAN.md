# PTSIP Tool 0.3.4 — GitHub-Coordinated Authority Plan

> **Status:** APPROVED / COMPLETED  
> **Target Tool version:** `0.3.4`  
> **Original planning baseline:** `f49fd391ff8237360d9fe5ebc8438eca5bfb4076`  
> **Implementation merge:** PR `#24`, squash commit `555c528593f700a348d8da84545a62ce61291cae`  
> **Verification follow-up:** PR `#25`, squash commit `8cd0ddf16dc9b56f27f694138a37caae1c49bb4f`  
> **Final verification:** GitHub Actions run `31471025526`, Python `3.14.6`, `134 passed`  
> **Active Tool binding:** `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
> **Specification note:** proposed Specification `0.3.3-draft` remains an independent workstream and is not an automatic Tool `0.3.4` binding.  
> **Publication state:** implementation complete; no `tool-v0.3.4` tag, GitHub Release, or PyPI publication has been created.

## 1. Completion decision

Tool `0.3.4` completes the **Distributed Authority Consistency** workstream that was only prototyped in the Tool `0.3.3` source tree.

Tool `0.3.3` remains permanently source-only and permanently untagged/unpublished. Its early GitHub authority code is treated as the prototype baseline that Tool `0.3.4` completed.

The completed Tool `0.3.4` contract is:

```text
architecture-sensitive coordinated gate
        |
        v
resolve repository/component coordination scope
        |
        v
read GitHub authority state
        |
        v
compare with selected Project Profile
        |
        +-- no authority + complete local declaration
        |       -> NO_DECISION_REQUIRED
        |       -> do not fabricate authority history
        |
        +-- remote winner + local missing
        |       -> validate and safely project winner
        |
        +-- remote winner + local equivalent
        |       -> RESOLVED / CONSISTENT
        |       -> no profile rewrite
        |
        +-- remote winner + local conflicting
        |       -> AUTHORITY_PROFILE_CONFLICT
        |       -> no silent overwrite
        |
        +-- unresolved authority decision
        |       -> DECISION_REQUIRED
        |
        +-- authority unavailable
                -> COORDINATION_UNAVAILABLE
                -> no Local fallback winner
```

Continuous background polling is not part of the contract.

## 2. State ownership

Tool `0.3.4` preserves three different roles:

```text
PTSIP Specification
    -> normative architecture/conformance contract

Consumer Repository Project Profile
    -> project-owned, revision-bound architecture declaration

Decision Authority
    -> coordination state for unresolved/resolved architecture decisions
```

These roles are intentionally separate.

- repository-root `ptsip.yaml` remains the default project-owned declaration and is intended to be Git-tracked;
- Local `control-plane.sqlite3` remains Tool-owned operational state under `PTSIP_HOME` and is not Git-shared;
- GitHub coordination uses repository-scoped authority state outside the normal worktree;
- a resolved global decision does not imply that every clone has already applied its local Project Profile projection.

## 3. Backend selection

The completed backend selection contract is:

```text
explicit --control-plane <URL>
    -> hosted HTTP Control Plane

explicit --coordination local
    -> embedded Local DecisionStore

explicit --coordination github
    -> GitHub authority

otherwise GitHub origin present
    -> GitHub authority

otherwise
    -> Local DecisionStore
```

`--control-plane` and `--coordination` remain mutually exclusive.

If GitHub coordination is selected and the authority cannot be safely read, authenticated, validated, or mutated for the required operation, PTSIP fails closed. It does not silently create a separate Local DecisionStore winner.

## 4. GitHub authority storage and identity

The GitHub reference implementation continues to use:

```text
refs/heads/ptsip-policy

  authority.json
  decisions/<global-decision-id>.json
```

The authority branch is accepted only when its manifest identifies the expected PTSIP authority format, repository, and ref. An existing unrelated `ptsip-policy` branch is refused.

Global decision identity remains derived from:

```text
GitHub owner/repository identity
    + normalized discovered component include selectors
```

The resulting `gdec-*` identity does not depend on clone-local clarification IDs, branch-local missing fields, or Local SQLite identity.

## 5. Write-side first-winner serialization

The prototype CAS model is retained:

```text
read authority HEAD A
    -> create commit B with parent A
    -> update ptsip-policy A -> B with force=false
```

A stale writer cannot overwrite a newer authority HEAD. Competing contradictory answers for the same component scope converge to one accepted winner, and a later contradictory answer returns the already accepted authoritative state rather than replacing it.

## 6. Read-side authority freshness

This was the central Tool `0.3.4` correction.

A GitHub-coordinated `ptsip gate` no longer returns `NO_DECISION_REQUIRED` merely because the selected local Project Profile already contains a complete declaration.

The relevant component scope is checked against GitHub authority first. This closes the stale-complete-profile gap where one clone could otherwise continue using a local declaration without observing a repository-global winner.

Authority lookup has a read-only path. If `refs/heads/ptsip-policy` does not exist, a read-only freshness check does not bootstrap the branch or create a pending decision solely to record history.

## 7. Reconciliation matrix

The finalized behavior is:

| Local Project Profile | GitHub authority | Tool 0.3.4 result |
| --- | --- | --- |
| declaration absent | no decision | create/reuse pending only when active gate requires a decision |
| declaration absent | resolved winner | validated safe local projection |
| declaration present | no authority decision | `NO_DECISION_REQUIRED`; retain declaration; no fabricated authority record |
| declaration equivalent | resolved equivalent winner | `RESOLVED` with `CONSISTENT`; no rewrite |
| declaration conflicting | resolved different winner | `AUTHORITY_PROFILE_CONFLICT`; no silent overwrite |
| repository/profile changes during reconciliation | any relevant remote state | stale refusal / no unsafe write |

Equivalent comparison is semantic rather than dependent on YAML formatting.

## 8. Conflict semantics

Tool `0.3.4` exposes authority/profile disagreement instead of silently choosing one side.

Relevant machine-readable states include:

```text
NO_DECISION_REQUIRED
DECISION_REQUIRED
RESOLVED
ALREADY_RESOLVED
AUTHORITY_PROFILE_CONFLICT
STALE_EVIDENCE
COORDINATION_UNAVAILABLE
DECISION_ERROR
```

`AUTHORITY_PROFILE_CONFLICT` means a resolved distributed winner and the existing local Project Profile declaration disagree. The Tool does not automatically reclassify or overwrite the existing project declaration in that state.

## 9. Global decision vs local projection state

Tool `0.3.4` separates:

```text
GLOBAL DECISION STATE
    PENDING / RESOLVED

LOCAL PROJECTION RECEIPT
    LOCAL_APPLIED / FAILED / STALE
```

GitHub authority records determine the winner. Clone-local profile application is reported separately with:

```text
scope = LOCAL_PROJECTION
```

A local projection receipt does not alter the winning answer and does not claim that another clone has applied the same Project Profile update.

## 10. `ptsip adopt`

`ptsip adopt` remains the explicit project-owner declaration path introduced in Tool `0.3.3`.

In GitHub-coordinated mode:

- new shared architecture decisions coordinate through GitHub authority before local mutation;
- an existing equivalent winner may be reused;
- a different winner is not overwritten;
- when the component is already declared locally and no authority decision exists, adoption does not create a remote pending decision solely for bookkeeping;
- dry-run remains read-only.

Local-only adoption continues without fabricating a Local DecisionStore workflow record.

## 11. `ptsip gate`

GitHub-coordinated `ptsip gate` now performs action-time authority freshness for the relevant discovered component scopes even when local declarations are complete.

This is bounded synchronization, not continuous polling. Ordinary unrelated read-only PTSIP operations are not converted into background authority clients.

## 12. `ptsip resolve`

`ptsip resolve` retains explicit human/project-owner decision semantics and first-valid-resolution-wins.

For GitHub coordination it:

- reads the authoritative decision;
- refuses replacement of a different resolved winner;
- serializes pending resolution through CAS;
- applies the accepted answer through the validated local profile projection path;
- does not use clone-local branch/revision metadata to create a second global decision identity.

Explicit Local and hosted backends retain their documented behavior.

## 13. Verification completed

Final Tool `0.3.4` verification completed in GitHub Actions run:

```text
31471025526
```

Environment and results:

```text
Python 3.14.6
134 passed
PTSIP Tool 0.3.4 identity: PASS
bound Specification identity: PASS
ptsip conform --help: PASS
python -m build: PASS
python -m twine check dist/*: PASS
built-wheel reinstall: PASS
installed-wheel Tool/spec/CLI smoke: PASS
```

Built artifacts verified successfully:

```text
ptsip-0.3.4-py3-none-any.whl
ptsip-0.3.4.tar.gz
```

The first merged implementation run exposed only three stale tests that still expected Tool `0.3.3`; after those expectations were corrected in PR `#25`, the complete suite passed with `134 passed`.

## 14. Acceptance criteria result

All Tool `0.3.4` release acceptance criteria for implementation completion are satisfied:

1. repository-global first-winner writes remain CAS-safe — **PASS**;
2. coordinated reads account for relevant authority state even with an existing local declaration — **PASS**;
3. missing/equivalent/conflicting local declarations have deterministic behavior — **PASS**;
4. conflicting local/remote state is never silently overwritten — **PASS**;
5. GitHub coordination failure cannot silently create a Local winner — **PASS**;
6. global resolution and clone-local application are separate — **PASS**;
7. explicit Local and hosted backends remain supported — **PASS**;
8. `adopt`, `gate`, and `resolve` use the finalized coordination model — **PASS**;
9. full test/build/package/installed-wheel verification passes — **PASS**;
10. Tool binding remains explicitly `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45` — **PASS**.

## 15. Preserved non-goals

Tool `0.3.4` does not introduce:

- continuous authority polling;
- GitHub Projects as the authority database;
- shared SQLite through Git;
- a fourth PTSIP architecture classification;
- automatic/LLM architecture classification;
- automatic overwrite/reclassification of a conflicting Project Profile;
- a mandatory `.PTSIP/` or `.ptsip/` Consumer Repository directory;
- silent binding to the proposed Specification `0.3.3-draft` family;
- unrelated conformance-rule expansion.

## 16. Publication boundary

**Implementation completion does not itself publish Tool `0.3.4`.**

At this completed planning checkpoint:

```text
Tool source version: 0.3.4
Implementation: COMPLETED
Verification: PASSED
Tag: not created
GitHub Release: not created
PyPI 0.3.4: not published
```

A `tool-v0.3.4` tag, GitHub Release, and PyPI publication require a separate explicit release decision.
