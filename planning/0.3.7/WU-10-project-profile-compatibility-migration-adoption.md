# WU-10 — Project Profile Compatibility, Migration, and Adoption

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-09 — Independent Project Profile Identity Core (`COMPLETE / VERIFIED`)  
> **Architecture authorities:** ADR-0010, ADR-0017, ADR-0019, ADR-0020, ADR-0021, ADR-0022  
> **Exact entry baseline:** `f31b317af91a7b5aea83aa16aef289cca81d289d`  
> **Predecessor verification authority:** `565a28fa1789206d3168408c906524f647f85d24` via `tooling-test` run `33031350756`  
> **Successor:** WU-11 — Tool 0.3.7 Final Regression, Specification Freeze, and Release Readiness

## 0. Purpose

Integrate the independent Project Profile identity layer from WU-09 with historical source compatibility, normalized semantic reconciliation, direct latest-target migration, controlled execution, and explicitly authorized adoption.

WU-10 owns semantic compatibility, historical identity bridging, direct target convergence, migration continuity, and adoption. It does not own Tool package release identity or final release readiness.

The governing migration architecture is ADR-0022:

```text
supported historical source
        ↓ source-specific compatibility reader
normalized source semantics
        ↓ direct reconciliation
current/latest canonical PP target supported by the Tool
        ↓ authorized execution
validated target profile
```

Intermediate Project Profile generations may provide compatibility/history knowledge, but they are not mandatory execution hops and are not materialized merely to replay version history.

WU-10 entered ACTIVE state under the standing successor-entry rule after WU-09 closure. This entry state does not by itself authorize implementation beyond the accepted architecture and WU-10 boundaries.

## 1. Historical generation mapping

Define explicit source-generation mappings for historical Tool-numbered Project Profile labels without rewriting their history.

At minimum:

```text
legacy source labels
0.3.4-draft
0.3.6-draft
        ↓ explicit compatibility interpretation
normalized source semantics / compatibility generation
        ↓ direct reconciliation
current canonical PP contract
pp.1.01
```

The compatibility-generation identity is metadata used to interpret the source correctly; it is not a required intermediate repository state.

The mapping must preserve source vocabulary and immutable source revision. Historical labels must not be represented as though they were originally published under the `pp.*` namespace.

Historical source identity and current target PP identity must both remain visible in migration evidence.

## 2. `0.3.6-draft -> pp.1.01` identity-only bridge

ADR-0021 defines the specific bridge:

```text
0.3.6-draft
    ↓ IDENTITY_ONLY
pp.1.01
```

For this bridge:

```text
components delta:               NONE
relationships delta:            NONE
associated_artifacts delta:     NONE
policies delta:                 NONE
Responsibility Map delta:       NONE
lifecycle classification delta: NONE
```

Therefore the Tool MUST NOT require users to reclassify or redesign an otherwise valid `0.3.6-draft` Project Profile merely because its canonical contract identity becomes `pp.1.01`.

Identity/schema validation after the rewrite remains required.

This bridge is a direct convergence case whose semantic delta is zero.

## 3. Compatibility semantics

Build on WU-09 identity compatibility to answer semantic questions:

- which historical source families can be read;
- which source families can be normalized for migration analysis;
- which canonical PP contract is the current migration target supported by the Tool;
- which lifecycle classifications require explicit owner decisions;
- which source declarations have no deterministic current equivalent;
- when a transition is `IDENTITY_ONLY` rather than semantic migration;
- when migration is unsupported and must fail closed.

Historical `TOOLCHAIN` semantics must remain source semantics and must never auto-convert to a current lifecycle classification merely from name similarity or Tool version.

Compatibility knowledge from intermediate historical generations may inform source interpretation or target obligations, but the runtime MUST NOT convert those generations into mandatory execution hops.

## 4. Direct latest-target policy

WU-10 SHALL implement ADR-0022 direct latest-target convergence.

The migration question is:

```text
What does this supported source mean,
and what must change for it to satisfy the current target PP contract?
```

not:

```text
What historical Project Profile version must the repository enter next?
```

For Tool `0.3.7`, the current canonical target is `pp.1.01`. Future Tool generations may select a later canonical target such as `pp.2.02` through explicit Tool/PP compatibility authority without requiring repositories to materialize every intervening PP generation.

Requirements:

- supported historical sources are read through source-family-specific readers;
- source semantics are represented in the normalized/common migration model;
- the analyzer compares that normalized source directly with the selected current target semantics;
- no synthetic intermediate PP profile is created merely because intermediate versions existed historically;
- pairwise historical migration edges are not automatically composed into migration authority;
- ambiguity in source meaning or target selection fails closed or produces an owner-decision requirement;
- migration capability remains distinct from adoption authority.

## 5. PP-aware target identity and temporary-profile naming

Canonical in-document PP identity:

```text
pp.1.01
```

New-generation filename token:

```text
pp1.01
```

New-generation temporary path, when a temporary target is actually required:

```text
ptsip_pp1.01.yaml
```

Requirements:

- canonical profile remains `ptsip.yaml`;
- temporary profile identity, if used, must encode the actual selected PP target rather than a Tool target or an obsolete historical intermediate;
- instance revision and immutable Specification binding remain separate from PP contract identity;
- legacy alias paths are resolved through typed PP equivalence rather than raw filename/string equality;
- duplicate logical PP targets fail closed;
- if multiple target contracts are intentionally supported, explicit compatibility/adoption policy must resolve which target applies; the Tool must not infer a target from numeric ordering alone.

WU-10 must not recreate the old false implication that Tool `0.3.7` requires a `ptsip_0.3.7.yaml` file.

## 6. In-progress migration continuity

Direct latest-target convergence must preserve valid migration work already in progress. Continuity preservation is an exception for existing accepted state, not a requirement to replay historical versions.

### 6.1 Existing `ptsip_0.3.6.yaml`

If an active migration already contains:

```text
ptsip_0.3.6.yaml
```

it remains the active target file where ADR-0021 declares it equivalent to the current `pp.1.01` target. Its internal identity may be rewritten:

```text
0.3.6-draft -> pp.1.01
```

without creating `ptsip_pp1.01.yaml`.

The existing legacy path is treated as an alias for the equivalent canonical PP target for that migration lineage.

### 6.2 No existing `ptsip_0.3.6.yaml`

If migration has not already created a `0.3.6-draft` target, the Tool must not fabricate that obsolete intermediate solely to preserve historical numbering.

The actual current target may be created directly as:

```text
ptsip_pp1.01.yaml
```

subject to normal owner authority, snapshot validation, and migration planning.

### 6.3 Canonical `ptsip.yaml` already at `0.3.6-draft`

If canonical `ptsip.yaml` is already semantically valid under `0.3.6-draft`, transition to `pp.1.01` is an in-place identity rewrite:

```text
ptsip.version: 0.3.6-draft
        ↓
ptsip.version: pp.1.01
```

Expected transition properties:

```text
semantic migration required: false
owner reclassification decision required: false
temporary PP target required: false
identity rewrite required: true
post-write validation required: true
```

## 7. Equivalent-target collision handling

If both equivalent target paths exist:

```text
ptsip_0.3.6.yaml
ptsip_pp1.01.yaml
```

the Tool must not choose automatically.

Because the ADR-0021 bridge maps both to the same logical PP destination, discovery must fail closed with the stable equivalent-target ambiguity diagnostic:

```text
DUPLICATE_EQUIVALENT_TARGET
```

Resolution requires explicit repository-state correction/authority before migration continues.

## 8. Normalized analyzer/planner integration

Adapt the existing WU-05/WU-06 migration layers to PP identities and direct convergence while preserving their established boundaries:

```text
historical source profile
+ source-family compatibility reader
+ exact repository snapshot
        ↓
normalized source semantics
+ selected current target PP semantics
        ↓
semantic / identity difference
        ↓
Required Work Elements
Removal Migration Elements
Asynchronous Work Targets
        ↓
ProposalBundle
AcceptedDeltaBundle
UnresolvedBundle
        ↓
Deterministic current-target convergence plan
```

The analyzer must recognize `IDENTITY_ONLY` bridges and avoid generating artificial semantic obligations for unchanged Project Profile contract content.

For genuine older-generation migrations, the analyzer computes source-to-current-target obligations directly. It does not require the repository to be migrated to each historical intermediate generation first.

Historical compatibility knowledge may explain why a target obligation exists, but the resulting plan must express the actual source-to-current-target work.

No compatibility mapping may bypass project-owner authority for genuine lifecycle reclassification or target declaration changes.

## 9. Execution integration

Adapt WU-07 execution state, ledger, semantic CAS, deletion gates, recovery, and promotion to PP generation identities and direct target convergence.

Preserve:

- typed state machine;
- append-only checkpoint ledger;
- exact source/final/repository bindings;
- Required-before-Async semantics;
- source completion before deletion;
- canonical-last promotion for genuine semantic migration;
- fail-closed recovery;
- accepted delta only mutation.

Execution applies the selected source-to-current-target plan. It MUST NOT promote the repository successively through intermediate PP generations merely to replay version history.

For identity-only canonical rewrites, execution must not invent a synthetic semantic migration sequence merely to reuse the older state machine path.

Execution must consume WU-09/WU-10 PP identities rather than derive migration authority from Tool version.

## 10. Adoption authority

Migration capability and repository adoption are distinct authorities.

WU-10 may prove direct migration through controlled fixture repositories without migrating the real PTSIP repository.

Real repository or consumer adoption of `pp.1.01` requires the appropriate supported source identity, exact repository state, and owner authorization where adoption mutates a real project.

Tool `0.3.7` alone is never adoption authority.

The accepted `IDENTITY_ONLY` equivalence removes semantic reclassification work; it does not erase the need for an authorized real-project write.

## 11. Fixture verification

Minimum controlled fixtures:

- canonical `0.3.6-draft` -> in-place `pp.1.01` identity-only rewrite;
- identity-only rewrite produces no component/relationship/artifact/policy/lifecycle delta;
- existing `ptsip_0.3.6.yaml` reused without creating `ptsip_pp1.01.yaml`;
- missing legacy `0.3.6` target -> direct `ptsip_pp1.01.yaml` creation;
- simultaneous `ptsip_0.3.6.yaml` and `ptsip_pp1.01.yaml` -> fail-closed equivalent-target collision;
- genuinely older supported historical source -> direct `pp.1.01` semantic migration where required;
- sufficiently old supported source converges directly to the current target without materializing historical intermediate profiles;
- historical intermediate-version knowledge may affect compatibility analysis but does not produce intermediate execution/profile files;
- future-shaped fixture proving a historical source can target a later canonical PP contract directly without mandatory `pp.1.01 -> pp.1.02 -> ...` repository traversal;
- lifecycle reclassification requiring owner decision;
- historical vocabulary with no automatic current mapping;
- unsupported historical source;
- unsupported PP target;
- ambiguous target selection fails closed;
- stale source/final/repository state;
- semantic CAS mismatch;
- interruption/recovery before checkpoint;
- interruption/recovery after source deletion;
- guarded final promotion;
- no Tool-numbered `ptsip_0.3.7.yaml` target creation;
- no synthetic historical PP temporary targets created solely for version traversal.

Fixtures must remain isolated from the real repository Project Profile unless separately authorized adoption is performed.

## 12. User-facing transition disclosure

The Project Profile contract note for `pp.1.01` must make the identity-only bridge explicit so users do not spend time re-reviewing unchanged declarations.

It must state that `0.3.6-draft` and `pp.1.01` are equivalent in the Project Profile contract semantics covered by ADR-0021 and that the namespace change itself introduces no required changes to:

```text
components
relationships
associated_artifacts
policies
lifecycle classifications
```

Migration documentation must also establish the general direct-convergence rule:

> Historical/intermediate Project Profile generations may inform compatibility analysis, but a supported project is migrated directly from its actual source declaration to the current supported canonical PP target. Intermediate versions are not mandatory user-visible migration steps.

Historical labels remain historical facts and are not retroactively renamed.

## 13. Long-term maintainability boundary

WU-10 should centralize source-family compatibility, historical identity interpretation, identity-equivalence bridges, target selection, filename alias resolution, and target-contract semantics in explicit typed adapters/registries rather than distribute historical-version conditionals across analyzer, planner, and executor modules.

The long-term architecture is:

```text
historical source readers
        ↓
normalized semantic model
        ↓
current target contract semantics
        ↓
deterministic reconciliation
```

This avoids both:

- an N-by-M collection of handwritten converters from every old source directly to every new target; and
- repository execution that replays every historical Project Profile generation in sequence.

Source reading, normalized semantics, target semantics, proposal generation, execution, and adoption authority remain separate layers even when implemented in the same WU.

## 14. Non-goals

WU-10 does not own:

- PP grammar/parser fundamentals already owned by WU-09;
- a universal Project Profile shortest-path / migration-route engine;
- mandatory intermediate-version traversal;
- moving historical release-note files into new namespace directories;
- final Tool `0.3.7` package version bump;
- final documentation freeze;
- final immutable Tool Specification revision;
- final exact-SHA release handoff.

## 15. Completion gate

WU-10 is complete only when:

- historical source families map explicitly into reusable normalized compatibility semantics without historical relabeling;
- the current canonical migration target is selected through explicit Tool/PP compatibility authority;
- supported historical sources reconcile directly against the current target rather than requiring intermediate Project Profile traversal;
- no synthetic intermediate PP profile is materialized solely to replay version history;
- `0.3.6-draft -> pp.1.01` is implemented as an `IDENTITY_ONLY` bridge with no artificial semantic migration obligations;
- an existing `ptsip_0.3.6.yaml` can continue as the equivalent `pp.1.01` target without duplicate target creation;
- repositories without the legacy target can create/use `ptsip_pp1.01.yaml` directly;
- canonical `0.3.6-draft` can transition in place to `pp.1.01` with post-write validation;
- duplicate equivalent legacy/new targets fail closed;
- analyzer/planner/executor operate without Tool-version-as-profile-authority assumptions;
- historical intermediate changes can inform compatibility analysis without becoming mandatory execution hops;
- unsupported/ambiguous source or target compatibility fails closed;
- genuine lifecycle reclassification remains owner-authorized;
- controlled historical-source -> current-target direct migration fixtures pass;
- interruption/recovery/promotion remain safe under PP identities and direct convergence;
- user-facing PP transition documentation states both the identity-only equivalence and direct latest-target rule clearly;
- real repository adoption, if not separately authorized, remains non-mutating;
- focused compatibility/migration tests pass at an exact source SHA.

Completion of WU-10 authorizes automatic entry state into WU-11 under the standing successor-entry rule, but does not authorize WU-11 implementation or release by itself.
