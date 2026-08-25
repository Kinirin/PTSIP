# WU-07 — Preview, Confirmation, Safe Sequential Apply, and Canonical Promotion

> **Status:** COMPLETE / FOCUSED TEST VERIFIED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-06 — target proposals and Final Point plan (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-07 exact entry baseline:** `c69f93eb19c6b7e227faaf6c9d38b563658f3b6a`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **Repository self-adoption boundary:** ADR-0017 — Tool Version and Project Profile Contract Independence  
> **Focused verification boundary:** PTSIP repository Project Profiles remain read-only; mutation/promotion verification uses controlled fixture repositories  
> **Focused verification exact SHA:** `68b29308e8531f93267acc2aec585f6e751021f1`  
> **Verification workflow run:** `32825710974` (`tooling-test`, workflow-dispatch; overall run failed only on the repository self-profile regression described in Section 14)  
> **Successor:** WU-08 — repository self-analysis, regression, package and release readiness

## 0. Purpose

Own the architecture-mutating execution capability of the Tool `0.3.7` migration pipeline without forcing the PTSIP repository itself to perform a Project Profile version migration.

WU-07 implements the guarded path that, **when an actual Project Profile draft migration exists**, can apply accepted source-specific target deltas into a selected Final PTSIP Point File, prove source completion, remove completed temporary sources, process canonical `ptsip.yaml` last, and perform guarded canonical promotion.

ADR-0017 establishes that Tool `0.3.7` does not itself select a repository-local `0.3.7-draft` Project Profile target. Therefore:

```text
PTSIP repository root profiles
    -> read-only verification input

controlled fixture repositories
    -> mutation / deletion / interruption / recovery / promotion verification
```

WU-07 MUST NOT create `ptsip_0.3.7.yaml`, rewrite repository `ptsip.yaml`, delete a repository source profile, or perform repository-local canonical promotion solely to prove the executor works.

## 1. Focused verification boundary

WU-07 focused verification is fixture-based.

The real PTSIP repository may be inspected for:

- current canonical profile identity;
- discovery behavior;
- source compatibility;
- evidence and analysis behavior;
- non-mutating WU-06 planning/preview behavior;
- confirmation that no repository-local target profile is selected.

The real PTSIP repository MUST NOT be used as the mutation target for WU-07 focused verification.

All tests that exercise write behavior MUST operate on isolated temporary/fixture repositories and MAY intentionally model draft migrations such as:

```text
0.3.6-draft canonical -> 0.3.7-draft Final Point
```

or:

```text
0.3.4-draft canonical
+ 0.3.6-draft temporary source
+ 0.3.7-draft temporary source
+ 0.4.0-draft Final Point
```

These are test transition states only. They do not authorize or imply the same Project Profile transition for the PTSIP repository.

Focused verification evidence SHOULD explicitly prove that the repository's actual:

```text
ptsip.yaml
profiles/example.ptsip.yaml
profiles/hybrid-python-package.ptsip.yaml
profiles/template-python-package.ptsip.yaml
```

were not rewritten as a side effect of WU-07 tests.

## 2. Mutation pipeline capability

```text
WU-06 deterministic plan
    -> project-owner preview
    -> explicit confirmation where architecture decisions are required
    -> exact source/final-point snapshot validation
    -> apply accepted delta to Final Point
    -> deterministic post-apply validation
    -> prove current source Required Work Elements complete
    -> delete current temporary source when eligible
    -> advance to next source
    -> canonical ptsip.yaml last
    -> global completion validation
    -> remove old canonical source
    -> rename Final Point -> ptsip.yaml
```

Mutation must stop at the first stale, conflicting, unauthorized, or incomplete boundary.

This pipeline is a reusable Tool capability. It is not a requirement that every Tool release create a new Final Point or migrate the repository's own Project Profile.

## 3. Source order enforcement

WU-07 must use the WU-01/WU-06 ordered source list and must not invent another sequence.

For Sequential Work:

```text
temporary sources: highest to lowest, excluding Final Point
canonical ptsip.yaml: always last
```

A removed temporary source may never become an intermediate migration destination or hop.

## 4. Per-source completion gate

Before deleting a temporary source, WU-07 must prove:

```text
source identity unchanged since accepted analysis
AND Final Point identity is the expected current state
AND source required_unresolved == 0 after apply
AND target validation for represented source obligations passes
AND no unresolved accepted-target conflict remains
```

Removal Migration Elements and Asynchronous Work Targets do not affect this completion gate.

If any Required Work Element is unresolved, source deletion is forbidden.

## 5. Canonical source special rule

When an actual draft migration is active, `ptsip.yaml` is migrated last and is not deleted immediately after its local source completion if global Final Point validation has not yet succeeded.

The final boundary is:

```text
all temporary sources completed/removed
canonical source Required Work Elements completed
Final Point validates against target draft/revision
no unresolved transition conflicts
expected snapshots still current
    -> canonical promotion allowed
```

Only then:

```text
remove old ptsip.yaml
rename Final Point -> ptsip.yaml
```

The implementation must minimize the interval where the canonical path is absent and must fail safely if filesystem operations cannot complete as planned.

For Tool `0.3.7`, this promotion path is exercised only in controlled fixtures unless a separate Project Profile migration decision is accepted.

## 6. Project-owner authority

Evidence and deterministic migration mechanics do not own architecture decisions.

WU-07 requires explicit project-owner confirmation for target deltas that change architecture intent, including ambiguous lifecycle reclassification, component split/merge, conflict replacement, or other semantics not already authorized by a prior accepted decision.

Purely mechanical application of an already accepted exact delta does not require inventing an additional architecture decision.

The decision to release Tool `0.3.7` is not itself authority to change `ptsip.version`, create `ptsip_0.3.7.yaml`, or promote a repository-local Final Point.

## 7. Stale-state and conflict rules

Before every mutation stage, validate the exact expected identities from WU-06.

Fail closed for at least:

- changed source profile;
- changed repository snapshot where the accepted analysis depends on it;
- changed Final Point content not represented by the accepted plan;
- target draft/revision mismatch;
- unresolved required obligation;
- conflict with accepted Final Point target state;
- source ordering mismatch;
- missing expected source;
- attempted promotion while another participating source remains;
- fixture mutation escaping its isolated repository root.

Automatic broad re-analysis after a stale-state failure must not silently apply a new plan. Re-analysis returns to preview/confirmation where target intent can change.

## 8. Optional Async additions

If the owner requested Asynchronous Work Targets after Required Work Elements were handled, WU-07 may apply those separately and visibly.

They must never:

- make an incomplete source appear complete;
- delay source completion without explicit owner choice;
- be required for source deletion;
- be silently inherited into another source's obligation taxonomy.

## 9. Work tracks

### WU-07A — preview and authorization contract

Render source order, required completion, accepted deltas, optional work, conflicts, and promotion effects deterministically.

### WU-07B — stale-state guard

Reuse existing snapshot/digest/CAS-style validation primitives where possible. Do not create a parallel weak write mechanism.

### WU-07C — accepted-delta Final Point apply

Apply only the accepted target delta. Avoid unrelated profile rewriting. Focused mutation verification occurs in isolated fixture repositories.

### WU-07D — source completion and deletion

Prove required completion before deleting a temporary source. Deletion tests operate only on fixture sources.

### WU-07E — canonical-last migration

Handle fixture canonical `ptsip.yaml` only after every fixture temporary source has converged when an actual migration is modeled.

### WU-07F — guarded promotion

Validate global completion and perform Final Point -> canonical promotion safely in controlled transition fixtures and future real migrations.

### WU-07G — interruption/retry behavior

Ensure interruption at any stage produces a discoverable, deterministic transition state on the next run instead of corrupting source ordering.

## 10. Required focused tests

All mutation tests below MUST use isolated fixture repositories:

- accepted simple transition apply;
- multi-source ordered apply;
- wrong source order rejected;
- incomplete Required Work Element blocks deletion;
- Removal element does not block deletion;
- Async work does not affect completion;
- stale source blocks mutation;
- stale Final Point blocks mutation;
- accepted target conflict blocks silent overwrite;
- append-only checkpoint integrity failure blocks resume;
- interruption after one temporary source deletion resumes from remaining sources;
- crash/recovery mismatch enters fail-closed recovery state;
- canonical source remains last;
- promotion blocked while any source remains;
- successful final promotion produces exactly one fixture canonical `ptsip.yaml` for the selected target draft;
- repository-local Tool `0.3.7` verification leaves `ptsip.yaml` and public example profiles unchanged;
- repository-local Tool `0.3.7` verification does not create `ptsip_0.3.7.yaml` without an independently accepted Project Profile migration target.

## 11. Non-goals

WU-07 does not define new lifecycle ontology, invent source obligation semantics, or bypass owner authority for convenience. It does not treat promotion success as release readiness; WU-08 owns full verification.

WU-07 does not force the PTSIP repository or public example profiles to adopt `0.3.7-draft` merely because the Tool version is `0.3.7`.

WU-07 focused verification does not mutate the PTSIP repository's actual Project Profile files.

## 12. Completion gate

WU-07 is complete only when the full ordered mutation path is deterministic, snapshot-guarded, owner-authorized where required, source-completion aware, interruption-safe, and incapable of promoting a partial Final Point.

Completion requires **fixture-based focused verification** of the write/deletion/recovery/promotion contract plus proof that the actual PTSIP Project Profile files remained unchanged during that verification.

Completion does **not** require creating, applying, deleting, or promoting `ptsip_0.3.7.yaml` in the PTSIP repository.

## 13. Entry discipline

WU-07 entered automatically under the project owner's standing successor-entry authorization after WU-06 completion was recorded and exact `dev/0.3.7` HEAD `c69f93eb19c6b7e227faaf6c9d38b563658f3b6a` was freshly revalidated.

This entry authorized WU-07 implementation only. It did **not** authorize applying any concrete target delta to the PTSIP repository, creating/modifying a repository Final Point, deleting a repository source profile, replacing repository canonical `ptsip.yaml`, or bypassing project-owner confirmation for architecture-changing decisions.

ADR-0017 additionally freezes the Tool `0.3.7` repository-specific boundary: no `ptsip_0.3.7.yaml` is created solely for this Tool release.

## 14. Closure and focused-verification evidence

WU-07 implementation and fixture-focused verification are closed against exact source SHA:

```text
68b29308e8531f93267acc2aec585f6e751021f1
```

The owner-dispatched self-hosted `tooling-test` workflow run `32825710974` checked out and verified that exact SHA before running the complete repository pytest suite.

Observed pytest result:

```text
437 passed
1 failed
```

The sole failure was:

```text
tests/ptsip/test_repository_self_profile_035.py::
test_repository_self_profile_is_valid_complete_and_revision_pinned
```

and was caused by the repository self-profile validator returning one warning:

```text
18 tracked file(s) are outside declared component and associated-artifact selectors;
this is not automatically a PTSIP violation.
```

No WU-07 migration execution, fixture apply/deletion/promotion, stale-state, CAS, Async, source-order, ledger-integrity, or recovery test failed. Therefore the WU-07 focused fixture scenarios are verified as passing at the exact SHA above.

The overall `tooling-test` run is **not** represented as successful: it correctly remains failed because the repository-wide self-profile regression is unresolved. That regression belongs to WU-08 repository self-analysis/full-regression closure and must be resolved there before release readiness can be claimed.

Repository Project Profile immutability was also rechecked after implementation:

```text
ptsip.yaml                                      = 0.3.6-draft
profiles/example.ptsip.yaml                     = 0.3.6-draft
profiles/hybrid-python-package.ptsip.yaml       = 0.3.6-draft
profiles/template-python-package.ptsip.yaml     = 0.3.6-draft
ptsip_0.3.7.yaml                                = absent
```

WU-07 is therefore `COMPLETE / FOCUSED TEST VERIFIED`. Full repository regression, package/distribution verification, exact-SHA workflow success, and release readiness remain WU-08 responsibilities.
