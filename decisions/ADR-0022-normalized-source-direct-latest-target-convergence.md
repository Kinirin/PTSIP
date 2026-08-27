# ADR-0022 — Normalized Source to Direct Latest-Target Convergence

> **Status:** Accepted  
> **Date:** 2026-08-27  
> **Target Tool:** `0.3.7`  
> **Governing work unit:** WU-10  
> **Related authorities:** ADR-0012, ADR-0013, ADR-0014, ADR-0015, ADR-0016, ADR-0017, ADR-0019, ADR-0020, ADR-0021

## 1. Decision

PTSIP Project Profile migration SHALL use **direct latest-target convergence** rather than mandatory intermediate-version traversal.

A supported historical Project Profile source is read through its source-family compatibility reader, converted into the normalized semantic model, reconciled directly against the latest canonical Project Profile contract supported as a migration target by the running Tool, and then migrated to that target.

Conceptually:

```text
historical source
        ↓ source-specific compatibility reader
normalized source semantics
        ↓ direct reconciliation
latest supported canonical PP target
        ↓ authorized mutation
validated target profile
```

Intermediate Project Profile generations are compatibility/history knowledge. They are not mandatory execution hops and MUST NOT be materialized merely because they existed historically.

For example, if a repository still declares:

```text
0.2.0-draft
```

and a future Tool supports canonical target:

```text
pp.2.02
```

then the intended execution model is:

```text
0.2.0-draft
    ↓
pp.2.02
```

not:

```text
0.2.0-draft
↓
0.3.3-draft
↓
0.3.4-draft
↓
0.3.6-draft
↓
pp.1.01
↓
...
↓
pp.2.02
```

## 2. Intermediate history is knowledge, not execution state

Historical contract changes MAY contribute evidence and compatibility knowledge required to interpret an old source correctly or determine current target obligations.

However:

> **Historical changes may contribute compatibility knowledge, but they do not constitute mandatory execution hops.**

PTSIP MUST NOT require a project to successively create, validate, promote, or delete temporary profiles for every historical Project Profile generation between the source and current target.

Therefore:

- no synthetic intermediate PP target is created solely to replay version history;
- no migration authority is inferred merely from numeric adjacency between versions;
- no repository is required to occupy each historical PP generation before reaching the current supported target;
- historical source identity remains preserved as evidence/provenance even when the migration target skips many published generations.

## 3. Normalized semantic convergence

The central migration question is not:

```text
What is the next version after this source?
```

It is:

```text
What does this source declaration mean,
and what must change for it to satisfy the current target contract?
```

PTSIP SHALL therefore retain the source-family compatibility boundary established by ADR-0013 and use a normalized semantic representation as the convergence boundary.

Conceptually:

```text
0.2.0-draft ─┐
0.3.4-draft ─┤
0.3.6-draft ─┤
pp.1.01 ─────┤
pp.1.xx ─────┤
             ▼
    normalized project semantics
             │
             ▼
    current supported PP target
```

Source-specific readers remain responsible for preserving historical vocabulary and source meaning. Target-specific semantics remain responsible for defining the current canonical contract. The migration analyzer/reconciliation layer computes the delta between those two semantic states.

## 4. Direct migration does not mean unrestricted inference

Direct convergence MUST NOT be interpreted as permission for the Tool to guess ambiguous project meaning.

If a historical declaration cannot be deterministically mapped to the target semantics, the Tool must surface an unresolved/owner-decision requirement rather than invent a classification or policy choice.

For example, if historical `TOOLCHAIN` semantics could correspond to multiple current lifecycle classifications, direct migration remains:

```text
historical source
    ↓
latest target
```

but the plan may contain:

```text
OWNER_DECISION_REQUIRED
```

before the target can be completed.

Thus:

```text
direct latest-target convergence
    !=
unrestricted automatic semantic inference
```

Existing PTSIP authority boundaries remain in force:

```text
Inference != Authority
Finding != Decision
Proposal != Accepted Delta
Migration capability != Adoption authority
```

## 5. Transition classification remains explicit

Direct convergence does not collapse all transitions into semantic migration.

The Tool MUST continue to distinguish at least:

```text
IDENTITY_ONLY
SEMANTIC_MIGRATION
```

The accepted bridge:

```text
0.3.6-draft
    ↓ IDENTITY_ONLY
pp.1.01
```

remains an identity-only transition under ADR-0021 because the normalized source semantics already match the `pp.1.01` contract semantics covered by that bridge.

A genuinely older source may instead require semantic reconciliation directly against the latest supported target.

## 6. Target selection policy

Migration target selection is based on explicit Tool compatibility authority, not on Tool SemVer equality and not on graph pathfinding across every known PP generation.

For a given Tool release, the compatibility layer SHALL explicitly identify which canonical PP contract is the preferred/current migration target for supported historical sources.

The Tool MUST NOT construct a new migration authority merely because multiple pairwise historical transitions could theoretically be chained.

The architecture therefore does not require a universal shortest-path or route-selection engine for Project Profile migration.

If multiple current targets are intentionally supported for different purposes, the applicable target must be resolved by explicit compatibility/adoption policy. Ambiguity fails closed.

## 7. Temporary profile behavior

For a new migration that requires a temporary target, the Tool creates only the actual selected canonical target.

Example for a future target:

```text
ptsip.yaml
version: 0.2.0-draft

        ↓ direct convergence

ptsip_pp2.02.yaml
version: pp.2.02
```

The Tool MUST NOT fabricate:

```text
ptsip_0.3.4.yaml
ptsip_0.3.6.yaml
ptsip_pp1.01.yaml
...
ptsip_pp2.01.yaml
```

solely to replay historical generations.

## 8. Existing in-progress migration continuity

This ADR does not invalidate ADR-0021 continuity rules.

If a valid migration was already in progress before the new PP identity/target policy and contains an accepted legacy temporary target such as:

```text
ptsip_0.3.6.yaml
```

that existing work must not be discarded merely to enforce a new filename convention.

Where ADR-0021 defines the legacy target as equivalent to the selected canonical PP target, the existing file may continue as the lineage target and may receive the authorized identity rewrite.

This is continuity preservation, not mandatory intermediate traversal.

If no such in-progress target exists, obsolete intermediate targets MUST NOT be created.

## 9. Analyzer and planner responsibility

WU-05/WU-06 migration analysis and reconciliation SHALL be adapted around the direct-convergence model:

```text
normalized historical source
+ exact repository snapshot
+ current target PP semantics
        ↓
semantic/identity difference
        ↓
Required Work Elements
Removal Migration Elements
Asynchronous Work Targets
        ↓
ProposalBundle
AcceptedDeltaBundle
UnresolvedBundle
        ↓
Deterministic target-state convergence plan
```

A plan may internally cite historical compatibility knowledge explaining why a target obligation exists, but the plan MUST describe the actual source-to-current-target work rather than exposing every historical contract generation as an execution step.

## 10. Execution responsibility

WU-07 safety properties remain applicable to genuine semantic migration:

- exact snapshot binding;
- typed execution state;
- append-only checkpoint ledger;
- semantic CAS;
- accepted-delta-only mutation;
- Required-before-Async behavior;
- source completion gates;
- fail-closed recovery;
- guarded promotion.

The executor applies the direct source-to-selected-target plan. It does not successively promote through historical PP generations.

For `IDENTITY_ONLY` canonical rewrites, the Tool must use the smallest correctly authorized identity-write path with post-write schema/identity validation rather than inventing a synthetic semantic migration sequence.

## 11. Long-term maintenance model

This architecture intentionally avoids an N-by-M collection of direct handwritten converters such as:

```text
0.2.0 -> pp.2.02
0.3.4 -> pp.2.02
0.3.6 -> pp.2.02
pp.1.01 -> pp.2.02
...
```

and also avoids repository execution that accumulates and applies every historical delta one generation at a time.

Instead it separates:

```text
historical source readers
        ↓
normalized semantic model
        ↓
current target contract semantics
        ↓
reconciliation
```

When a new PP target is introduced, existing source readers should remain reusable unless the historical interpretation itself was incomplete or incorrect. Target evolution should primarily affect target semantics/reconciliation rather than require rewriting every old-source converter.

## 12. Verification requirements

WU-10 verification MUST include direct-convergence fixtures proving at least:

- a sufficiently old supported historical source can converge directly to the current canonical PP target without intermediate profile materialization;
- historical intermediate generations are not created as temporary files;
- intermediate-version history may inform compatibility analysis without becoming execution hops;
- ambiguous historical semantics produce owner-decision/unresolved output rather than guessed mappings;
- `0.3.6-draft -> pp.1.01` remains `IDENTITY_ONLY` with zero semantic delta;
- existing ADR-0021 legacy temporary target continuity remains valid;
- unsupported source/target compatibility fails closed;
- stale state, CAS, recovery, deletion, and promotion guarantees remain correct under direct convergence.

## 13. Superseded planning assumptions

Any WU-10 planning language that implies mandatory sequential migration through multiple actual PP generations is superseded by this ADR.

Sequential execution may still exist for multiple source obligations or for previously existing in-progress migration state, but **Project Profile version history itself is not a mandatory sequential execution path**.

## 14. Non-authority statement

This ADR selects the migration architecture and target-convergence semantics. It does not by itself authorize mutation of the real PTSIP repository `ptsip.yaml`, publication of a Tool release, creation of a future PP contract, or automatic resolution of project-owner decisions.
