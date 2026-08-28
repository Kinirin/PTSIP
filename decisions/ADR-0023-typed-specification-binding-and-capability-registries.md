# ADR-0023 — Typed Specification Binding and Independent Capability Registries

**Status:** Accepted  
**Decision:** Adopt the long-term-maintainability architecture for Specification binding and move implementation to WU-12  
**Target Tool:** `0.3.7`  
**Related authorities:** ADR-0017, ADR-0019, ADR-0021, ADR-0022

## Context

WU-11 release-readiness review exposed a coupling inherited from Tool `0.3.6`: generic Project Profile validation compares the profile's `ptsip.specification.revision` directly with the installed Tool's single `SPEC_REVISION` constant.

That coupling happened to work while the repository Tool, Specification binding, and canonical Project Profile all described the same historical generation. It is no longer valid after PTSIP separated:

```text
Tool Version
Project Profile Contract Version
Project Profile Instance Revision
Specification family + immutable revision
```

A proposed workaround was to let the generic validator consult the WU-10 historical Project Profile bridge whenever a profile's Specification revision differs from the current Tool `SPEC_REVISION`.

That workaround is rejected as a long-term architecture. Historical migration compatibility would become a central validation bottleneck: every otherwise valid historical or concurrently supported Specification binding would require a migration bridge entry even when no migration operation is being performed.

The project owner selected the long-term-maintainability option and explicitly increased the Tool `0.3.7` work sequence by adding WU-12 rather than forcing the additional architecture into WU-11.

## Decision

PTSIP SHALL introduce a typed Specification binding model and independent capability registries.

Canonical identity is represented conceptually as:

```text
Tool Version
    independent package/runtime identity

Project Profile Contract Version
    independent PP contract identity

SpecificationBinding
    family
    source
    immutable revision

Project Profile Instance Revision
    immutable identity of one concrete project declaration
```

### 1. Typed `SpecificationBinding`

A canonical current-generation Project Profile SHALL represent its Specification binding explicitly, including at least:

```yaml
ptsip:
  version: "pp.1.01"
  specification:
    family: "0.3.7-draft"
    source: "https://github.com/Kinirin/PTSIP"
    revision: "<immutable-specification-revision>"
```

`ptsip.version` is Project Profile contract identity. It MUST NOT be used as an implicit Specification family field.

### 2. Independent capability registries

PTSIP SHALL keep at least these authorities conceptually separate:

```text
Tool -> Project Profile capability
Tool -> Specification capability
historical source -> migration compatibility bridge
```

The existing PP support registry remains responsible for which Project Profile contracts the Tool can identify, validate, analyze, or create.

A new Specification capability registry SHALL be responsible for which Specification families/revisions the Tool can use for defined operations.

The historical Project Profile compatibility bridge remains responsible for migration interpretation only. It MUST NOT become the generic validator's registry for otherwise valid Specification bindings.

### 3. Composition instead of Cartesian registration

Tool/PP/Specification support MUST NOT require a permanently duplicated Cartesian registry entry for every possible tuple unless an operation has a genuine cross-contract restriction.

The normal validation shape is:

```text
validate PP capability independently
        +
validate Specification capability independently
        +
apply a narrow explicit compatibility constraint only where semantics require one
```

An available historical migration bridge MUST NOT by itself authorize a current Specification binding, and a supported Specification binding MUST NOT by itself authorize Project Profile migration.

### 4. Generic validator responsibility

Generic Project Profile validation SHALL NOT require:

```text
profile.specification.revision == Tool.SPEC_REVISION
```

and SHALL NOT route ordinary validation through historical migration compatibility.

Instead, validation SHALL:

1. validate the declared PP contract through PP capability authority;
2. validate the declared `SpecificationBinding` through Specification capability authority;
3. validate schema and Responsibility Map semantics;
4. apply explicit cross-contract constraints only when such constraints are normative;
5. remain fail closed for unsupported or malformed bindings.

### 5. Historical compatibility responsibility

Historical readers and bridges remain revision-bound and explicit for migration operations such as:

```text
historical Tool-numbered Project Profile
    -> normalized source semantics
    -> current canonical PP target
```

They preserve historical vocabulary and migration meaning, but they are not a general Specification lookup service.

### 6. PTSIP repository adoption

After the WU-12 binding architecture is implemented and verified, the PTSIP repository itself is intended to adopt the current identities:

```text
Tool:                  0.3.7
Project Profile:       pp.1.01
Specification family: 0.3.7-draft
Specification revision: final WU-12 immutable normative snapshot
```

The previously created commit `555af435a4bb68140c2c869efa34d12c624d51a4` remains the WU-11 PP-aware transition Specification baseline. Because WU-12 changes normative binding/schema surfaces, it is not forced to remain the final Tool `0.3.7` release `SPEC_REVISION`.

WU-12 SHALL create/select the final immutable normative snapshot after its normative surfaces are complete and bind Tool `0.3.7` to that exact revision.

## Responsibility boundary

```text
PP capability registry
    -> Project Profile contract operations

Specification capability registry
    -> Specification family/revision operations

historical compatibility bridge
    -> historical migration interpretation

generic validator
    -> compose current declared capabilities without migration inference

release gate
    -> verify the PTSIP repository's exact release identities and final immutable snapshot
```

No layer may silently promote inference from another layer into authority.

## Work-unit consequence

WU-11 is reduced from final release authority to PP-aware normative/release-surface preparation and handoff to WU-12.

WU-12 becomes the final Tool `0.3.7` integration and release-readiness authority and owns:

- typed `SpecificationBinding`;
- Specification capability registry;
- PP/Specification capability composition;
- canonical schema/specdata binding changes;
- generic validator responsibility separation;
- PTSIP root `pp.1.01` + `0.3.7-draft` adoption;
- final immutable `SPEC_REVISION` selection;
- full regression/package/workflow verification;
- final release handoff.

## Rejected alternatives

### Keep exact equality with one Tool `SPEC_REVISION`

Rejected. It re-couples Tool release identity to every Project Profile's Specification revision and prevents durable multi-family compatibility.

### Use historical migration bridges as generic validation authority

Rejected. It creates a central responsibility bottleneck and makes migration mappings mandatory for non-migration validation.

### Implement the full architecture inside WU-11

Rejected. The architecture is intentionally assigned its own WU so implementation, testing, and regression ownership remain narrow and diagnosable.
