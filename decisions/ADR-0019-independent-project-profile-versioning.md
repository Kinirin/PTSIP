# ADR-0019 — Independent Project Profile Versioning

**Status:** Accepted  
**Decision date:** 2026-08-26  
**Target Tool:** `0.3.7`  
**Governing work unit:** WU-08  
**Related decisions:** ADR-0008, ADR-0009, ADR-0017, ADR-0018  
**Bound development Specification:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`

## Context

PTSIP historically used Tool release numbers and Project Profile contract labels that shared the same numeric family, for example:

```text
PTSIP Tool 0.3.6
Project Profile 0.3.6-draft
```

That numeric alignment is not an architectural identity relationship. A Tool release may add implementation, diagnostics, compatibility readers, migration capability, or packaging behavior without changing the Project Profile contract. Conversely, the Project Profile contract may change in a way that requires profile migration while the same Tool release remains capable of reading, analyzing, or migrating both generations.

Treating the two numbers as if they must move together creates false migration pressure and makes compatibility analysis ambiguous. In particular, Tool `0.3.7` must not imply that a repository using a supported Project Profile must create a `0.3.7-draft` profile or rewrite its canonical `ptsip.yaml` merely to match the Tool number.

WU-08 therefore establishes a separate Project Profile version namespace.

## Decision

### 1. Tool version and Project Profile contract version are independent authorities

PTSIP SHALL treat these as independent version axes:

```text
Tool Version
    example: 0.3.7

Project Profile Contract Version
    example: pp.1.01

Project Profile Instance Revision
    immutable revision and/or content digest of one concrete project declaration
```

The following implication is invalid:

```text
Tool 0.3.7
    => Project Profile 0.3.7-draft
```

A Tool version change does not authorize a Project Profile contract migration. A Project Profile contract change does not require the Tool version to change if the current Tool already implements the new contract and its compatibility behavior.

### 2. Project Profile versions use the `pp` namespace

Project Profile contract versions use the PTSIP-specific identity form:

```text
pp.<major>.<minor>
```

`pp` means **Project Profile**.

This is a PTSIP contract-version grammar, not Semantic Versioning. It MUST NOT be interpreted as a Tool package version or Python distribution version.

The canonical textual representation uses an integer major and a minor rendered with a minimum width of two decimal digits:

```text
pp.0.00
pp.1.01
pp.1.02
pp.1.10
pp.2.00
pp.2.100
```

Implementations may compare parsed major/minor values numerically, but serialization MUST preserve the canonical `pp.` namespace and minimum two-digit minor representation.

### 3. Major-version responsibility

The governing definition is:

> **Project Profile major version은 governing lifecycle classification의 집합 또는 그 classification semantics가 변경되어 기존 프로젝트의 lifecycle 재분류를 요구할 수 있을 때 상승한다.**

Normatively, a Project Profile major version increases when the set of governing lifecycle classifications, or the semantics of those classifications, changes in a way that may require lifecycle reclassification of existing project declarations.

Major-version triggers include, but are not limited to:

- adding a governing lifecycle classification;
- removing a governing lifecycle classification;
- renaming a governing lifecycle classification when identity/meaning changes;
- changing the semantic boundary of an existing lifecycle classification;
- changing classification rules such that previously valid components may require lifecycle reclassification.

A mere increase in classification count is therefore not the only major trigger. Removal or semantic redefinition can also require a major increase.

### 4. Minor-version responsibility

Within one governing lifecycle generation, the Project Profile minor version increases when the **contract semantics or structure** of Project Profile declarations changes without redefining the governing lifecycle classification model itself.

Minor-version concerns include:

```text
components
relationships
associated_artifacts
policies
Responsibility Map declaration/authority semantics
other Project Profile contract fields or constraints
```

A change that crosses into governing lifecycle classification-set or classification-semantics redefinition is a major change instead.

### 5. Project instance changes do not bump the Project Profile contract version

The Project Profile contract version describes the declaration language and semantics, not the content revision of one repository's `ptsip.yaml`.

For example, under `pp.1.01`, a project may add or remove a concrete component, relationship, selector, associated artifact, or policy value that is already valid under the same contract. Such a project-specific declaration change does **not** by itself change `pp.1.01` to `pp.1.02`.

Project-specific state changes are identified by the Project Profile Instance Revision, including immutable source revision and/or deterministic content digest as defined by the applicable authority model.

Thus:

```text
Tool Version                  0.3.7    -> unchanged
Project Profile Contract      pp.1.01  -> unchanged
Project Profile Instance      digest A -> digest B
```

is a valid result of an ordinary repository declaration update.

### 6. Historical generation naming

`pp.0.00` is the compatibility identity for the legacy Project Profile governing model whose primary governing lifecycle boundary was `PRODUCT` versus `TOOLCHAIN`.

Historical schemas also contained `NEUTRAL_CONTRACT`; this ADR does not erase that historical vocabulary. For `pp.0.00` history, `NEUTRAL_CONTRACT` is treated as the neutral/non-governing contract classification alongside the legacy primary `PRODUCT` / `TOOLCHAIN` governing boundary.

The current governing lifecycle generation is based on:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

The intended first formally numbered current Project Profile contract under this ADR is:

```text
pp.1.01
```

This ADR does not retroactively assert that a published immutable `pp.1.00` contract existed. Any historical aliasing or compatibility mapping must be evidence-bound and separately documented rather than invented from numbering continuity.

### 7. Tool compatibility is explicit, not inferred from numeric equality

Each Tool generation must explicitly define which Project Profile contract versions it can perform relevant operations against.

Conceptually:

```text
Tool 0.3.7
    READ:       explicitly supported PP versions
    VALIDATE:   explicitly supported PP versions
    ANALYZE:    explicitly supported PP versions
    MIGRATE:    explicitly supported source -> target PP transitions
    CREATE:     explicitly supported target PP versions
```

Compatibility MUST NOT be inferred from matching numbers between Tool and Project Profile identities.

This enables the Tool to detect:

- a supported Project Profile requiring no migration;
- an older Project Profile that can be analyzed and migrated;
- a newer or unknown Project Profile that must fail closed or request a compatible Tool;
- a source/target pair for which no authorized migration path exists.

### 8. Tool `0.3.7` remains Tool `0.3.7`

The target Tool release remains:

```text
PTSIP Tool 0.3.7
```

Adopting `pp.1.01` as the Project Profile contract namespace does not create a Tool `pp.1.01` version and does not alter Python package Semantic Versioning.

The Tool release identity and Project Profile contract identity must be surfaced separately in CLI, package, validation, compatibility, and migration evidence where both are relevant.

### 9. This ADR does not immediately rewrite the repository canonical profile

The current implementation still parses Project Profile generations using historical `<major>.<minor>.<micro>-draft` labels and temporary filenames such as `ptsip_<major>.<minor>.<micro>.yaml`. The current profile schema likewise binds the existing `0.3.6-draft` contract label.

Therefore this architecture decision alone MUST NOT directly rewrite:

```text
ptsip.yaml
profiles/example.ptsip.yaml
profiles/hybrid-python-package.ptsip.yaml
profiles/template-python-package.ptsip.yaml
```

to `pp.1.01`.

The `pp.1.01` contract transition requires synchronized implementation of version parsing, schema identity, source compatibility, migration identity/ordering, temporary-profile naming or equivalent generation identity, diagnostics, compatibility declarations, tests, and normative documentation.

Until that transition is implemented and verified, existing historical labels remain valid source identities rather than being silently relabeled.

### 10. Migration responsibility follows Project Profile contract changes

A new Project Profile contract version creates migration responsibility only when a concrete project is on a different source contract and the Tool determines that a transition is required and supported.

If a Tool is upgraded while the project's Project Profile contract remains compatible and unchanged, no Project Profile migration is required solely because of the Tool upgrade.

If the Project Profile contract changes while the Tool version remains unchanged, the Tool may still perform the migration if that exact source/target compatibility is explicitly implemented and authorized.

## Consequences

- Tool releases can evolve without forcing repository-wide Project Profile rewrites.
- Project Profile contract changes become explicit architecture events with their own compatibility and migration responsibility.
- Numeric equality can no longer be mistaken for compatibility authority.
- lifecycle-reclassification-scale changes are isolated in the Project Profile major axis.
- non-lifecycle contract evolution is isolated in the Project Profile minor axis.
- ordinary project declaration edits remain instance revisions rather than global contract-version bumps.
- Tool `0.3.7` can be designed to support legacy and current Project Profile generations simultaneously.
- transition/migration code must stop assuming that Project Profile identity is Tool-style three-part SemVer with `-draft`.

## WU-08 implementation gate

Before WU-08 may represent `pp.1.01` as an active Project Profile contract, it must define and verify at least:

- canonical `pp` version parser/serializer and ordering;
- schema/runtime identity for `pp.1.01`;
- explicit mapping of supported historical source families to PP compatibility identities;
- Tool `0.3.7` Project Profile compatibility matrix;
- migration source/target identity behavior that does not depend on Tool SemVer;
- temporary/Final Point naming and identity rules for PP versions;
- fail-closed diagnostics for unknown/unsupported PP versions;
- tests proving Tool-version changes do not force PP migration;
- tests proving PP-version changes can require migration while Tool version remains unchanged;
- documentation that exposes Tool version and PP contract version as separate fields.

No repository-local Project Profile promotion to `pp.1.01` is authorized merely by accepting this ADR.

## Non-authority statement

This ADR establishes Project Profile version semantics and the intended `pp.1.01` current-generation identity. It does not by itself authorize canonical profile mutation, temporary profile creation, source profile deletion, Final Point promotion, Tool release publication, or a fabricated historical `pp.1.00` release.