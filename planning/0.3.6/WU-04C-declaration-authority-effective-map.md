# WU-04C — Declaration Authority, `source_mode`, and Canonical Effective Map Boundary

> **Status:** COMPLETE  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04A catalog identity + WU-04B materializer core  
> **Entry baseline:** `5be7623fa1b750a1c11e349fd7f00073233d9595`  
> **Accepted decision:** `decisions/ADR-0009-responsibility-map-declaration-authority.md`  
> **Bound normative snapshot after WU-04C:** `82abd09360df09a95fbbfb516855fa9ffb49f050`

## 1. Purpose

WU-04C freezes the responsibility boundary between:

1. lifecycle ownership;
2. project/template declaration authority; and
3. Tool materialization responsibility.

The purpose is to ensure that `explicit`, `template`, and `hybrid` declarations can resolve into the same Canonical Effective Responsibility Map without allowing source mode, template logic, or materialization code to become lifecycle-classification authority.

## 2. Three independent responsibility boundaries

### A. Lifecycle responsibility

`classification` answers:

```text
Which lifecycle primarily owns this architectural responsibility?
```

Canonical values remain:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

Lifecycle ownership is independent from declaration source.

A `DELIVERY` component remains `DELIVERY` whether it came from an explicit declaration, an adopted template, or a hybrid project override.

### B. Declaration authority

`source_mode` and derived provenance answer:

```text
Where did this architecture declaration come from, and which declaration authority controls it?
```

`source_mode` is exactly one of:

```text
explicit
template
hybrid
```

It is not a PTSIP classification, role, relationship type, VPMS Verification Purpose, or evidence provenance value.

### C. Materialization responsibility

The PTSIP materializer answers:

```text
How is already-declared authority reproduced as one deterministic effective map?
```

The materializer is non-authoritative. It MUST NOT:

- infer missing architecture;
- select a template from repository evidence;
- silently repair broken relationships;
- silently remove dangling declarations;
- change a lifecycle classification to make validation pass;
- invent components, associated artifacts, roles, or relationships;
- reinterpret project-owned architecture based on confidence or path heuristics.

Its responsibility is deterministic resolution only.

## 3. Mode-specific authority boundary

### `explicit`

```text
Project declaration
    -> complete Responsibility Map
    -> Canonical Effective Responsibility Map
```

The project owns the entire declaration.

Materialization is a semantic copy/normalization operation and does not create architecture intent.

### `template`

```text
Project
    -> explicitly selects exact template ID + immutable revision
    -> template provides selected declaration
    -> Canonical Effective Responsibility Map
```

The project owns the decision to adopt the exact template identity.

The selected template revision supplies the default declaration content that the project explicitly adopted.

Template selection MUST NOT be inferred from repository shape, language, framework, manifest, package manager, candidate confidence, or naming conventions.

A later template revision MUST NOT silently change the effective architecture of a project still bound to an earlier revision.

### `hybrid`

```text
Project selects exact template ID + revision
        +
Project-owned ID-addressed override/add/remove declarations
        |
        v
Deterministic materialization
        |
        v
Canonical Effective Responsibility Map
```

Authority precedence is:

```text
project explicit override / removal / extension
    > selected template declaration
```

The materializer only applies that precedence. It does not decide what the project should override.

## 4. Hybrid entity replacement boundary

Tool `0.3.6` uses **whole-entity replacement by stable ID**, not field-level inheritance, for hybrid component, associated-artifact, and relationship overrides.

Example:

```text
template component id = package
        |
project supplies component id = package
        |
        v
project replacement owns the entire resulting `package` entity
```

This avoids field-level authority fragmentation such as:

```text
classification -> template?
roles          -> template?
include        -> project?
purpose        -> template?
```

If a project replaces one entity ID, the replacement declaration is project-owned as one coherent entity.

A project extension is a new stable ID that does not exist in the selected template.

A project removal is an explicit project decision not to adopt a selected template entity.

## 5. Derived declaration provenance

Runtime materialization MAY expose derived provenance for review, diagnostics, migration comparison, and safe apply.

Canonical planning vocabulary for derived entity origin is:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

`PROJECT_REMOVAL` describes a project decision in materialization provenance; removed entities are not present in the effective map.

Derived provenance is runtime/explanatory metadata. It MUST NOT be written into the canonical Project Profile merely to make materialization work.

For example:

```yaml
source_mode: hybrid
provenance:
  components:
    product: TEMPLATE
    verification: TEMPLATE
    delivery: PROJECT_OVERRIDE
    monitoring: PROJECT_EXTENSION
```

is an internal/review representation, not a required canonical profile serialization.

## 6. Canonical Effective Responsibility Map

All three declaration modes must resolve to the same downstream semantic shape:

```text
explicit ----+
template ----+--> Canonical Effective Responsibility Map
hybrid ------+
                    |
                    +--> components
                    +--> associated_artifacts
                    +--> relationships
                    +--> component dependency policy
                    +--> project policies
```

Downstream evaluators should consume this effective view rather than implement mode-specific architecture semantics.

The source declaration remains available separately so the Tool can explain where the effective map came from without rewriting user intent.

## 7. Effective-map identity

WU-04D will introduce a deterministic semantic digest for the materialized map:

```text
effective_map_digest = sha256(canonical semantic representation)
```

The digest is intended to support:

- reproducibility checks;
- migration preview comparison;
- semantic-equivalence checks;
- stale/concurrent state detection inputs;
- diagnostics explaining whether two differently serialized declarations produce the same effective architecture.

The digest MUST NOT become architecture authority and MUST NOT replace the underlying declaration provenance.

## 8. No automatic conflict repair

Materialization must fail closed when project declarations create an invalid effective map.

Examples:

- removing a component while leaving a relationship that targets it;
- overriding an associated artifact so its anchor no longer exists;
- introducing component/artifact identity collision;
- removing an unknown template ID;
- replacing and removing the same ID;
- producing selector conflicts that violate canonical validation.

The Tool MUST report the inconsistency. It MUST NOT silently delete or rewrite related entities merely to produce a valid map.

## 9. Frozen normative implementation contract

WU-04C promoted the approved authority boundary into the canonical `0.3.6-draft` Responsibility Map companion as:

```text
PTSIP-RMAP-013
    Declaration source does not alter lifecycle ownership

PTSIP-RMAP-014
    Exact template selection is project-owned architecture authority

PTSIP-RMAP-015
    Hybrid precedence is stable-ID whole-entity authority

PTSIP-RMAP-016
    Materialization is deterministic and non-authoritative
```

The canonical and embedded registry copies were aligned to those rules and to the declaration-source/effective-map concepts.

The immutable Specification snapshot selected after the WU-04C normative change is:

```text
82abd09360df09a95fbbfb516855fa9ffb49f050
```

Tool constants and root `ptsip.yaml` are bound to this snapshot after the snapshot commit, preserving the rule that an immutable normative revision is selected first and consumers bind to it afterward.

## 10. WU-04 stage transition

WU-04C is complete. WU-04D is the next stage but has **not been entered yet**.

```text
WU-04A  template catalog identity                         COMPLETE
   |
WU-04B  deterministic materializer core                  COMPLETE
   |
WU-04C  declaration authority + source_mode boundary     COMPLETE
   |
WU-04D  ResolvedProfile + digest + provenance            NEXT / NOT ENTERED
   |
WU-04E  validation consumes effective map                LOCKED
   |
WU-04F  conformance consumes effective map               LOCKED
   |
WU-04G  clarification/adoption read paths consume view   LOCKED
   |
WU-04H  VPMS narrow read-only bridge consumes view       LOCKED
   |
WU-04I  regression + WU-04 completion                    LOCKED
```

No WU-04D sub-document exists yet by design.

When WU-04D is actually entered, the required sequence is:

1. freshly read the development branch HEAD;
2. verify WU-04C remains complete and the bound Specification is coherent;
3. create the WU-04D sub-document with that exact entry baseline;
4. mark WU-04D ACTIVE;
5. only then implement the `ResolvedProfile`, digest, and derived provenance boundary.

## 11. Completion record

WU-04C completion conditions are satisfied:

- the approved three-boundary model is recorded in accepted `ADR-0009`;
- `source_mode` is explicitly declaration source, not lifecycle ownership;
- exact template selection is project-owned architecture authority;
- hybrid whole-entity replacement/add/remove precedence is frozen;
- materialization is explicitly non-authoritative and fail-closed;
- Canonical Effective Responsibility Map semantics are fixed for downstream consumers;
- `PTSIP-RMAP-013` through `PTSIP-RMAP-016` are normative;
- canonical and embedded registry contracts are aligned;
- a new immutable Specification snapshot was selected and the Tool/root profile binding was advanced to it;
- no future WU-04 sub-stage document was created early.

WU-04C completion does not claim full repository regression execution. Full regression remains subject to the approved self-hosted verification boundary.
