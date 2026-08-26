# WU-10 — Project Profile Compatibility, Migration, and Adoption

> **Status:** PLANNED  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-09 — Independent Project Profile Identity Core  
> **Architecture authorities:** ADR-0010, ADR-0017, ADR-0019, ADR-0020  
> **Successor:** WU-11 — Tool 0.3.7 Final Regression, Specification Freeze, and Release Readiness

## 0. Purpose

Integrate the independent Project Profile identity layer from WU-09 with historical source compatibility, migration analysis, controlled transition execution, and explicitly authorized adoption.

WU-10 owns semantic compatibility and migration. It does not own Tool package release identity or final release readiness.

## 1. Historical generation mapping

Define explicit source-generation mappings for historical Tool-numbered Project Profile labels without rewriting their history.

At minimum:

```text
legacy source labels
0.3.4-draft
0.3.6-draft
        ↓ explicit compatibility mapping
historical PP generations
including pp.0.00 where applicable
        ↓
current target PP contract
pp.1.01
```

The mapping must preserve source vocabulary and immutable source revision. Historical labels must not be represented as though they were originally published under the `pp.*` namespace.

## 2. Compatibility semantics

Build on WU-09 identity compatibility to answer semantic questions:

- which source families can be read;
- which can be analyzed as migration sources;
- which target PP contracts are valid destinations;
- which lifecycle classifications require explicit owner decisions;
- which source declarations have no direct current equivalent;
- when migration is unsupported and must fail closed.

Historical `TOOLCHAIN` semantics must remain source semantics and must never auto-convert to a current lifecycle classification merely from name similarity or Tool version.

## 3. PP-aware transition identity

Refactor transition discovery so active target/source generations use Project Profile contract identity rather than Tool SemVer identity.

Temporary/Final Point naming and identity rules must be designed around PP contracts, not around Tool release numbers.

Requirements:

- canonical profile remains `ptsip.yaml`;
- temporary profile identity, if used, must encode an actual PP target rather than a Tool target;
- filename/internal identity must agree;
- target ordering must compare PP major/minor numerically;
- duplicate logical PP targets fail closed;
- non-monotonic or ambiguous target selection fails closed;
- instance revision and immutable Specification binding remain separate from PP version ordering.

WU-10 must not recreate the old false implication that Tool `0.3.7` requires a `ptsip_0.3.7.yaml` file.

## 4. Migration analyzer/planner integration

Adapt the existing WU-05/WU-06 migration layers to PP identities while preserving their established boundaries:

```text
source profile
+ exact repository snapshot
+ normalized evidence
+ explicit target PP semantics
        ↓
Required Work Elements
Removal Migration Elements
Asynchronous Work Targets
        ↓
ProposalBundle
AcceptedDeltaBundle
UnresolvedBundle
        ↓
Deterministic Final Point plan
```

No compatibility mapping may bypass project-owner authority for lifecycle reclassification or target declaration changes.

## 5. Execution integration

Adapt WU-07 execution state, ledger, semantic CAS, deletion gates, recovery, and promotion to PP generation identities.

Preserve:

- typed state machine;
- append-only checkpoint ledger;
- exact source/final/repository bindings;
- Required-before-Async semantics;
- source completion before deletion;
- canonical-last promotion;
- fail-closed recovery;
- accepted delta only mutation.

Execution must consume WU-09/WU-10 PP identities rather than derive migration authority from Tool version.

## 6. Adoption authority

Migration capability and repository adoption are distinct authorities.

WU-10 may prove migration through controlled fixture repositories without migrating the real PTSIP repository.

Real repository or consumer adoption of `pp.1.01` requires:

```text
supported source identity
+ explicit target pp.1.01 contract
+ completed compatibility analysis
+ exact accepted migration decisions
+ stale-state validation
+ project-owner authorization
```

Tool `0.3.7` alone is never adoption authority.

## 7. Fixture verification

Minimum controlled fixtures:

- legacy source -> `pp.1.01` simple migration;
- sequential PP migration with multiple actual target generations;
- sparse target history without synthetic intermediate PP versions;
- lifecycle reclassification requiring owner decision;
- historical vocabulary with no automatic current mapping;
- unsupported PP source;
- unsupported PP target;
- duplicate PP target identity;
- stale source/final/repository state;
- semantic CAS mismatch;
- interruption/recovery before checkpoint;
- interruption/recovery after source deletion;
- guarded final promotion;
- no Tool-numbered temporary target creation.

Fixtures must remain isolated from the real repository Project Profile.

## 8. Long-term maintainability boundary

WU-10 should centralize source-family compatibility and PP migration mapping in explicit typed adapters/registries rather than distribute historical-version conditionals across analyzer, planner, and executor modules.

Source reading, semantic mapping, proposal generation, execution, and adoption authority remain separate layers even when implemented in the same WU.

## 9. Non-goals

WU-10 does not own:

- PP grammar/parser fundamentals already owned by WU-09;
- final Tool `0.3.7` package version bump;
- final documentation freeze;
- final immutable Tool Specification revision;
- final exact-SHA release handoff.

## 10. Completion gate

WU-10 is complete only when:

- historical source families map explicitly to PP compatibility generations;
- `pp.1.01` is represented as a semantic migration target independently from Tool `0.3.7`;
- transition discovery/order uses PP identities;
- analyzer/planner/executor operate without Tool-version-as-profile-authority assumptions;
- unsupported/ambiguous PP compatibility fails closed;
- lifecycle reclassification remains owner-authorized;
- controlled legacy -> `pp.1.01` migration fixtures pass;
- interruption/recovery/promotion remain safe under PP identities;
- real repository adoption, if not separately authorized, remains non-mutating;
- focused compatibility/migration tests pass at an exact source SHA.

Completion of WU-10 authorizes automatic entry state into WU-11 under the standing successor-entry rule, but does not authorize WU-11 implementation or release by itself.
