# WU-04D — `ResolvedProfile`, Effective-Map Digest, and Derived Provenance

> **Status:** ACTIVE  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04C — declaration authority + Canonical Effective Responsibility Map boundary  
> **Entry baseline:** `d8713ac4e684852f3e6cf67a68165f82ae0b80aa`  
> **Bound Specification snapshot at entry:** `82abd09360df09a95fbbfb516855fa9ffb49f050`

## 1. Purpose

WU-04D introduces one runtime resolution abstraction that preserves the original project declaration while exposing the deterministic Canonical Effective Responsibility Map required by later consumers.

The stage owns exactly three concerns:

1. `ResolvedProfile` as the common source/effective view;
2. deterministic `effective_map_digest` calculation;
3. derived declaration provenance for materialized entities and explicit removals.

WU-04D does **not** move validation, conformance, clarification/adoption, or VPMS consumers onto the resolved view. Those integrations remain WU-04E through WU-04H.

## 2. Frozen responsibility boundary

WU-04D implements the already accepted WU-04C authority model:

```text
classification
    = lifecycle responsibility

source declaration / template selection / project override
    = architecture declaration authority

ResolvedProfile
    = non-authoritative runtime view over source + deterministic effective map
```

The runtime view MUST NOT mutate or rewrite project intent.

## 3. `ResolvedProfile` contract

`ResolvedProfile` must expose enough information for later stages without requiring downstream mode-specific logic.

Required information:

```text
source_payload
    original project declaration, preserved as a defensive copy

effective_payload
    deterministic explicit-form profile containing the effective map

source_mode
    explicit | template | hybrid

template_id
    exact selected template ID, or none for explicit

template_revision
    exact selected immutable template revision, or none for explicit

effective_map_digest
    deterministic SHA-256 identity of effective architecture semantics

provenance
    derived, non-authoritative origin metadata for effective entities and removals
```

The source and effective payloads remain distinguishable. `ResolvedProfile` must not make a materialized explicit payload look like the original project declaration.

## 4. Derived provenance vocabulary

WU-04C froze these values:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

WU-04D records provenance separately for:

- components;
- associated artifacts;
- relationships;
- explicit removals from hybrid materialization.

Rules:

### Explicit mode

Every effective project-declared entity has origin:

```text
PROJECT_EXPLICIT
```

### Template mode

Every effective template entity has origin:

```text
TEMPLATE
```

### Hybrid mode

For each stable-ID collection:

```text
unchanged template entity -> TEMPLATE
same-ID project replacement -> PROJECT_OVERRIDE
new project ID             -> PROJECT_EXTENSION
removed template ID         -> PROJECT_REMOVAL
```

Removed IDs do not appear in the effective map; their removal provenance is retained separately for explanation/review.

Derived provenance MUST NOT be serialized back into the canonical Project Profile merely to support resolution.

## 5. Effective-map digest boundary

The digest identifies the **effective architecture semantics**, not the source declaration mechanism.

Therefore digest input includes only:

```text
components
associated_artifacts
relationships
component_dependency_policy
policies
```

when present in the effective profile.

Digest input explicitly excludes:

```text
ptsip specification binding
responsibility_map source mode
template ID/revision
materialization provenance
source formatting / YAML key order
comments or other serialization-only details
```

Consequently two declarations that produce the same Canonical Effective Responsibility Map must produce the same digest even when one is `explicit`, one is `template`, and one is `hybrid`.

## 6. Canonical semantic normalization

Digest calculation must be deterministic without treating declaration order as architecture authority.

Normalization rules:

1. JSON object keys are sorted recursively by canonical JSON encoding.
2. Top-level stable-ID collections are sorted by `id`:
   - `components`;
   - `associated_artifacts`;
   - `relationships`.
3. Known set-valued schema fields are sorted before hashing:
   - component `roles`;
   - component/artifact `include` and `exclude` selectors;
   - component `manifests`, `consumers`, `analysis_inputs` where present;
   - dependency-policy `allow` and `deny` relations, ordered by `(from, to)`.
4. Unknown lists are preserved in declaration order rather than being guessed as set-valued.
5. The resulting semantic object is encoded as UTF-8 canonical JSON using stable separators and `ensure_ascii=False`.
6. The public digest string is:

```text
sha256:<lowercase hex>
```

The digest is an equivalence/reproducibility tool. It is not a project authority token and cannot authorize architecture mutation.

## 7. Existing WU-04B carrier transition

WU-04B introduced `MaterializedProfile` as a minimal transport object for materialization results.

WU-04D replaces that temporary carrier with the canonical `ResolvedProfile` abstraction rather than preserving two competing runtime models.

No deprecated alias is required on the unreleased Tool `0.3.6` development line. Existing callers of `materialize_profile()` continue to receive an object with the effective `payload` compatibility accessor only where required for the current internal transition, while the canonical fields become `source_payload` and `effective_payload`.

This compatibility accessor must not blur which payload is authoritative source declaration state.

## 8. Fail-closed behavior

WU-04D does not weaken WU-04B/C failure behavior.

Resolution must still fail when:

- template ID/revision is unknown;
- hybrid removal targets an unknown template entity;
- one ID is both replaced and removed;
- materialization cannot deterministically identify an entity origin;
- effective-map digest input contains malformed stable-ID collections that cannot be normalized safely.

WU-04D must not repair dangling endpoints, anchors, selector conflicts, or other semantic validation defects. Those remain validation concerns for WU-04E.

## 9. Implementation boundary

Primary implementation target:

```text
src/ptsip/validation/templates.py
```

Expected changes:

- introduce `ResolvedProfile`;
- introduce structured derived provenance;
- extend merge/materialization helpers to return origin/removal metadata;
- calculate deterministic effective-map digest;
- preserve source payload separately from effective payload;
- keep `materialize_profile()` as the single mode-resolution entry point for now.

Primary focused regression target:

```text
tests/ptsip/test_template_materialization_036.py
```

WU-04D tests should cover:

- source payload remains unchanged and independently available;
- explicit provenance;
- template provenance;
- hybrid override/extension/removal provenance;
- deterministic digest across repeated resolution;
- digest equality for semantically equivalent explicit/template declarations;
- digest insensitivity to stable-ID collection order and known set-valued ordering;
- digest changes when effective architecture semantics change;
- template identity/source mode do not alter equal effective-map digest.

No self-hosted workflow is dispatched as part of entering or implementing WU-04D unless the maintainer separately confirms the approved runner is ready.

## 10. WU-04D completion gate

WU-04D is complete only when:

- `ResolvedProfile` preserves source declaration and effective payload as separate views;
- exact source mode and template identity remain available;
- entity/removal provenance follows the WU-04C authority model;
- effective-map digest is deterministic and source-mode independent;
- digest normalization is bounded to known semantic collection rules rather than arbitrary list sorting;
- focused tests cover the new abstraction and digest/provenance semantics;
- WU-04E validation integration has **not** been entered early;
- WU-04E sub-document has not been created.

Until these conditions are met, WU-04E remains locked.
