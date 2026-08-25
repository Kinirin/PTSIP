# WU-07 — Preview, Confirmation, Safe Sequential Apply, and Canonical Promotion

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-06 — target proposals and Final Point plan (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-07 exact entry baseline:** `c69f93eb19c6b7e227faaf6c9d38b563658f3b6a`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **Repository self-adoption boundary:** ADR-0017 — Tool Version and Project Profile Contract Independence  
> **Successor:** WU-08 — dogfood, regression, package and release readiness

## 0. Purpose

Own the only architecture-mutating boundary of the Tool `0.3.7` migration pipeline.

WU-07 applies accepted source-specific target deltas directly into a selected Final PTSIP Point File, removes completed temporary sources in the required order, migrates canonical `ptsip.yaml` last, and performs guarded final promotion **when an actual Project Profile draft migration exists**.

For the PTSIP repository's Tool `0.3.7` development itself, ADR-0017 explicitly determines that no `ptsip_0.3.7.yaml` target is created. The repository canonical `ptsip.yaml` remains `0.3.6-draft`. Therefore WU-07 implementation and mutation semantics are verified through controlled fixture repositories rather than by manufacturing a repository-local `0.3.6-draft -> 0.3.7-draft` migration.

## 1. Mutation pipeline

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

Mutation must stop at the first stale, conflicting, or incomplete boundary.

This pipeline is a capability contract. It is not a requirement that every Tool release create a new Final Point or migrate the repository's own Project Profile.

## 2. Source order enforcement

WU-07 must use the WU-01/WU-06 ordered source list and must not invent another sequence.

For Sequential Work:

```text
temporary sources: highest to lowest, excluding Final Point
canonical ptsip.yaml: always last
```

A removed temporary source may never become an intermediate migration destination or hop.

## 3. Per-source completion gate

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

## 4. Canonical source special rule

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

ADR-0017 establishes that this promotion path is **not executed against the PTSIP repository for Tool 0.3.7**, because no repository-local Project Profile target version has been selected.

## 5. Project-owner authority

Evidence and deterministic migration mechanics do not own architecture decisions.

WU-07 requires explicit project-owner confirmation for target deltas that change architecture intent, including ambiguous lifecycle reclassification, component split/merge, conflict replacement, or other semantics not already authorized by a prior accepted decision.

Purely mechanical application of an already accepted exact delta does not require inventing an additional architecture decision.

The decision to release Tool `0.3.7` is not itself authority to change `ptsip.version` or create `ptsip_0.3.7.yaml`.

## 6. Stale-state and conflict rules

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
- attempted promotion while another participating source remains.

Automatic broad re-analysis after a stale-state failure must not silently apply a new plan. Re-analysis returns to preview/confirmation where target intent can change.

## 7. Optional Async additions

If the owner requested Asynchronous Work Targets after Required Work Elements were handled, WU-07 may apply those separately and visibly.

They must never:

- make an incomplete source appear complete;
- delay source completion without explicit owner choice;
- be required for source deletion;
- be silently inherited into another source's obligation taxonomy.

## 8. Work tracks

### WU-07A — preview and authorization contract

Render source order, required completion, proposed deltas, optional work, conflicts, and promotion effects deterministically.

### WU-07B — stale-state guard

Reuse existing snapshot/digest/CAS-style validation primitives where possible. Do not create a parallel weak write mechanism.

### WU-07C — accepted-delta Final Point apply

Apply only the accepted target delta. Avoid unrelated profile rewriting.

### WU-07D — source completion and deletion

Prove required completion before deleting a temporary source.

### WU-07E — canonical-last migration

Handle root `ptsip.yaml` only after every temporary source has converged when an actual draft migration is active.

### WU-07F — guarded promotion

Validate global completion and perform Final Point -> canonical promotion safely in controlled transition fixtures and future real migrations.

### WU-07G — interruption/retry behavior

Ensure interruption at any stage produces a discoverable, deterministic transition state on the next run instead of corrupting source ordering.

## 9. Required tests

At minimum, controlled fixture repositories must cover:

- accepted simple transition apply;
- multi-source ordered apply;
- wrong source order rejected;
- incomplete Required Work Element blocks deletion;
- Removal element does not block deletion;
- Async work does not affect completion;
- stale source blocks mutation;
- stale Final Point blocks mutation;
- accepted target conflict blocks silent overwrite;
- interruption after one temporary source deletion resumes from remaining sources;
- canonical source remains last;
- promotion blocked while any source remains;
- successful final promotion produces exactly one canonical `ptsip.yaml` for the selected target draft;
- repository-local Tool `0.3.7` execution does not create `ptsip_0.3.7.yaml` without an independently accepted Project Profile migration target.

## 10. Non-goals

WU-07 does not define new lifecycle ontology, invent source obligation semantics, or bypass owner authority for convenience. It does not treat promotion success as release readiness; WU-08 owns full verification.

WU-07 also does not force the PTSIP repository or public example profiles to adopt `0.3.7-draft` merely because the Tool version is `0.3.7`.

## 11. Completion gate

WU-07 is complete only when the full ordered mutation path is deterministic, snapshot-guarded, owner-authorized where required, source-completion aware, interruption-safe, and incapable of promoting a partial Final Point.

Completion requires fixture-based proof of the mutation/promotion contract. It does **not** require creating, applying, or promoting `ptsip_0.3.7.yaml` in the PTSIP repository.

## 12. Entry discipline

WU-07 entered automatically under the project owner's standing successor-entry authorization after WU-06 completion was recorded and exact `dev/0.3.7` HEAD `c69f93eb19c6b7e227faaf6c9d38b563658f3b6a` was freshly revalidated.

This ACTIVE state authorizes WU-07 implementation only. It does **not** authorize applying any concrete target delta, creating/modifying a Final Point, deleting a source profile, replacing canonical `ptsip.yaml`, or bypassing project-owner confirmation for architecture-changing decisions.

ADR-0017 additionally freezes the Tool `0.3.7` repository-specific boundary: no `ptsip_0.3.7.yaml` is created solely for this Tool release.
