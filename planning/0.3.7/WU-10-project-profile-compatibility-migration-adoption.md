# WU-10 — Project Profile Compatibility, Migration, and Adoption

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-09 — Independent Project Profile Identity Core (`COMPLETE / VERIFIED`)  
> **Architecture authorities:** ADR-0010, ADR-0017, ADR-0019, ADR-0020, ADR-0021  
> **Exact entry baseline:** `f31b317af91a7b5aea83aa16aef289cca81d289d`  
> **Predecessor verification authority:** `565a28fa1789206d3168408c906524f647f85d24` via `tooling-test` run `33031350756`  
> **Successor:** WU-11 — Tool 0.3.7 Final Regression, Specification Freeze, and Release Readiness

## 0. Purpose

Integrate the independent Project Profile identity layer from WU-09 with historical source compatibility, migration analysis, controlled transition execution, and explicitly authorized adoption.

WU-10 owns semantic compatibility, historical identity bridging, migration continuity, and adoption. It does not own Tool package release identity or final release readiness.

WU-10 entered ACTIVE state under the standing successor-entry rule after WU-09 closure. This entry state does not by itself authorize implementation beyond the accepted architecture and WU-10 boundaries.

## 1. Historical generation mapping

Define explicit source-generation mappings for historical Tool-numbered Project Profile labels without rewriting their history.

At minimum:

```text
legacy source labels
0.3.4-draft
0.3.6-draft
        ↓ explicit compatibility mapping
historical/current PP generations
        ↓
current canonical PP contract
pp.1.01
```

The mapping must preserve source vocabulary and immutable source revision. Historical labels must not be represented as though they were originally published under the `pp.*` namespace.

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

## 3. Compatibility semantics

Build on WU-09 identity compatibility to answer semantic questions:

- which source families can be read;
- which can be analyzed as migration sources;
- which target PP contracts are valid destinations;
- which lifecycle classifications require explicit owner decisions;
- which source declarations have no direct current equivalent;
- when a transition is `IDENTITY_ONLY` rather than semantic migration;
- when migration is unsupported and must fail closed.

Historical `TOOLCHAIN` semantics must remain source semantics and must never auto-convert to a current lifecycle classification merely from name similarity or Tool version.

## 4. PP-aware transition identity

Refactor transition discovery so active target/source generations use Project Profile contract identity rather than Tool SemVer identity.

Canonical in-document identity:

```text
pp.1.01
```

New-generation filename token:

```text
pp1.01
```

New-generation temporary path, when needed:

```text
ptsip_pp1.01.yaml
```

Requirements:

- canonical profile remains `ptsip.yaml`;
- temporary profile identity, if used, must encode an actual PP target rather than a Tool target;
- target ordering must compare PP major/minor numerically;
- duplicate logical PP targets fail closed;
- non-monotonic or ambiguous target selection fails closed;
- instance revision and immutable Specification binding remain separate from PP version ordering;
- legacy alias paths are resolved through typed PP equivalence rather than raw filename/string equality.

WU-10 must not recreate the old false implication that Tool `0.3.7` requires a `ptsip_0.3.7.yaml` file.

## 5. In-progress migration continuity

The identity transition must preserve valid migration work already in progress.

### 5.1 Existing `ptsip_0.3.6.yaml`

If an active migration already contains:

```text
ptsip_0.3.6.yaml
```

it remains the active target file. Its internal identity may be rewritten:

```text
0.3.6-draft -> pp.1.01
```

without creating `ptsip_pp1.01.yaml`.

The existing legacy path is treated as an alias for the equivalent canonical PP target for that migration lineage.

### 5.2 No existing `ptsip_0.3.6.yaml`

If the migration has not yet created a `0.3.6-draft` target, the Tool must not fabricate that obsolete intermediate solely to preserve historical numbering.

The actual current target may be created directly as:

```text
ptsip_pp1.01.yaml
```

subject to normal owner authority, snapshot validation, and migration planning.

### 5.3 Canonical `ptsip.yaml` already at `0.3.6-draft`

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

## 6. Equivalent-target collision handling

If both equivalent target paths exist:

```text
ptsip_0.3.6.yaml
ptsip_pp1.01.yaml
```

the Tool must not choose automatically.

Because the ADR-0021 bridge maps both to the same logical PP destination, discovery must fail closed with a stable equivalent-target ambiguity diagnostic such as:

```text
DUPLICATE_EQUIVALENT_TARGET
```

Resolution requires explicit repository-state correction/authority before migration continues.

## 7. Migration analyzer/planner integration

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

The analyzer must recognize `IDENTITY_ONLY` bridges and avoid generating artificial semantic obligations for unchanged Project Profile contract content.

No compatibility mapping may bypass project-owner authority for genuine lifecycle reclassification or target declaration changes.

## 8. Execution integration

Adapt WU-07 execution state, ledger, semantic CAS, deletion gates, recovery, and promotion to PP generation identities.

Preserve:

- typed state machine;
- append-only checkpoint ledger;
- exact source/final/repository bindings;
- Required-before-Async semantics;
- source completion before deletion;
- canonical-last promotion for genuine semantic migration;
- fail-closed recovery;
- accepted delta only mutation.

For identity-only canonical rewrites, execution must not invent a synthetic semantic migration sequence merely to reuse the older state machine path.

Execution must consume WU-09/WU-10 PP identities rather than derive migration authority from Tool version.

## 9. Adoption authority

Migration capability and repository adoption are distinct authorities.

WU-10 may prove migration through controlled fixture repositories without migrating the real PTSIP repository.

Real repository or consumer adoption of `pp.1.01` requires the appropriate supported source identity, exact repository state, and owner authorization where adoption mutates a real project.

Tool `0.3.7` alone is never adoption authority.

The accepted `IDENTITY_ONLY` equivalence removes semantic reclassification work; it does not erase the need for an authorized real-project write.

## 10. Fixture verification

Minimum controlled fixtures:

- canonical `0.3.6-draft` -> in-place `pp.1.01` identity-only rewrite;
- identity-only rewrite produces no component/relationship/artifact/policy/lifecycle delta;
- existing `ptsip_0.3.6.yaml` reused without creating `ptsip_pp1.01.yaml`;
- missing legacy `0.3.6` target -> direct `ptsip_pp1.01.yaml` creation;
- simultaneous `ptsip_0.3.6.yaml` and `ptsip_pp1.01.yaml` -> fail-closed equivalent-target collision;
- legacy source -> `pp.1.01` genuine semantic migration for older generations where required;
- sequential PP migration with multiple actual target generations;
- sparse target history without synthetic intermediate PP versions;
- lifecycle reclassification requiring owner decision;
- historical vocabulary with no automatic current mapping;
- unsupported PP source;
- unsupported PP target;
- stale source/final/repository state;
- semantic CAS mismatch;
- interruption/recovery before checkpoint;
- interruption/recovery after source deletion;
- guarded final promotion;
- no Tool-numbered `ptsip_0.3.7.yaml` target creation.

Fixtures must remain isolated from the real repository Project Profile unless separately authorized adoption is performed.

## 11. User-facing transition disclosure

The Project Profile contract note for `pp.1.01` must make the identity-only bridge explicit so users do not spend time re-reviewing unchanged declarations.

It must state that `0.3.6-draft` and `pp.1.01` are equivalent in the Project Profile contract semantics covered by ADR-0021 and that the namespace change itself introduces no required changes to:

```text
components
relationships
associated_artifacts
policies
lifecycle classifications
```

Historical labels remain historical facts and are not retroactively renamed.

## 12. Long-term maintainability boundary

WU-10 should centralize source-family compatibility, identity-equivalence bridges, filename alias resolution, and PP migration mapping in explicit typed adapters/registries rather than distribute historical-version conditionals across analyzer, planner, and executor modules.

Source reading, identity equivalence, semantic mapping, proposal generation, execution, and adoption authority remain separate layers even when implemented in the same WU.

## 13. Non-goals

WU-10 does not own:

- PP grammar/parser fundamentals already owned by WU-09;
- moving historical release-note files into new namespace directories;
- final Tool `0.3.7` package version bump;
- final documentation freeze;
- final immutable Tool Specification revision;
- final exact-SHA release handoff.

## 14. Completion gate

WU-10 is complete only when:

- historical source families map explicitly to PP compatibility generations;
- `0.3.6-draft -> pp.1.01` is implemented as an `IDENTITY_ONLY` bridge with no artificial semantic migration obligations;
- an existing `ptsip_0.3.6.yaml` can continue as the equivalent `pp.1.01` target without duplicate target creation;
- repositories without the legacy target can create/use `ptsip_pp1.01.yaml` directly;
- canonical `0.3.6-draft` can transition in place to `pp.1.01` with post-write validation;
- duplicate equivalent legacy/new targets fail closed;
- transition discovery/order uses PP identities and typed equivalence rather than Tool-version/string identity assumptions;
- analyzer/planner/executor operate without Tool-version-as-profile-authority assumptions;
- unsupported/ambiguous PP compatibility fails closed;
- genuine lifecycle reclassification remains owner-authorized;
- controlled legacy -> `pp.1.01` migration fixtures pass;
- interruption/recovery/promotion remain safe under PP identities;
- user-facing PP transition documentation states the identity-only equivalence clearly;
- real repository adoption, if not separately authorized, remains non-mutating;
- focused compatibility/migration tests pass at an exact source SHA.

Completion of WU-10 authorizes automatic entry state into WU-11 under the standing successor-entry rule, but does not authorize WU-11 implementation or release by itself.
