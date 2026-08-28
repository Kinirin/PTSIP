# ADR-0018 — Separate Product Components for Migration Subsystems

**Status:** Accepted  
**Decision date:** 2026-08-25  
**Target Tool:** `0.3.7`  
**Governing work unit:** WU-08  
**Related decisions:** ADR-0008, ADR-0009, ADR-0012, ADR-0013, ADR-0014, ADR-0015, ADR-0016, ADR-0017  
**Bound development Specification:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`

## Context

WU-08 repository self-analysis found that the current root `ptsip.yaml` remains valid but reports 18 tracked files outside all declared component and associated-artifact selectors.

The 18 files are exactly the Tool `0.3.7` product subpackages introduced by the migration pipeline:

```text
src/ptsip/evidence/**        # 4 files
src/ptsip/source_compat/**   # 4 files
src/ptsip/migration/**       # 10 files
```

They are packaged by the existing `ptsip*` distribution rule and are therefore shipped Tool code rather than planning, fixture, or repository-only support files.

The existing `ptsip-core` component intentionally enumerates owned subpackages rather than using a broad `src/ptsip/**` selector. The new subpackages were therefore left uncovered until an explicit repository architecture decision was made.

Three architecture choices were reviewed:

1. broaden `ptsip-core` to a recursive `src/ptsip/**` ownership rule;
2. declare `evidence`, `source_compat`, and `migration` as separate long-lived PRODUCT components;
3. keep one `ptsip-core` component and add the three selectors explicitly.

The project owner selected option 2, prioritizing long-term maintainability and explicit subsystem architecture.

## Decision

### 1. Establish three independent PRODUCT components

The repository Responsibility Map will declare:

```text
ptsip-evidence
ptsip-source-compat
ptsip-migration
```

All three are Tool product implementation components with:

```text
classification: PRODUCT
roles: [IMPLEMENTATION]
shipped: true
runtime_required: false
executable: false
release_owner: tool
compatibility_owner: tool
```

Their selectors are exact subsystem roots:

```text
ptsip-evidence       -> src/ptsip/evidence/**
ptsip-source-compat  -> src/ptsip/source_compat/**
ptsip-migration      -> src/ptsip/migration/**
```

This keeps future new `src/ptsip/<subsystem>/` directories uncovered until explicitly classified instead of silently inheriting PRODUCT ownership.

### 2. Preserve existing `ptsip-core` ownership

`ptsip-core` keeps its current explicit selectors and does not broaden to `src/ptsip/**`.

The new components are not aliases or nested selector overrides of `ptsip-core`; each owns a disjoint tracked-file scope.

### 3. Record actual typed dependency directions

The Responsibility Map will represent the current code architecture with typed relationships:

```text
ptsip-evidence       --IMPORTS--> ptsip-core
ptsip-source-compat  --IMPORTS--> ptsip-core
ptsip-source-compat  --READS-----> ptsip-embedded-contracts
ptsip-migration      --IMPORTS--> ptsip-core
ptsip-migration      --IMPORTS--> ptsip-evidence
ptsip-migration      --IMPORTS--> ptsip-source-compat
```

The canonical contracts specify the three new product subsystems, and repository verification verifies them:

```text
ptsip-canonical-contracts --SPECIFIES--> ptsip-evidence
ptsip-canonical-contracts --SPECIFIES--> ptsip-source-compat
ptsip-canonical-contracts --SPECIFIES--> ptsip-migration

repository-verification --VERIFIES--> ptsip-evidence
repository-verification --VERIFIES--> ptsip-source-compat
repository-verification --VERIFIES--> ptsip-migration
```

These relationships describe existing implementation and verification dependencies; they do not grant architecture authority to inferred future dependencies.

### 4. No Project Profile version migration

This is a repository-owned explicit Responsibility Map architecture update under the already adopted `0.3.6-draft` Project Profile contract.

It does **not** change:

```text
ptsip.version = 0.3.6-draft
ptsip.specification.revision = d6995ed232e845b88d8235b851e80ab54b7804ea
```

It does not create `ptsip_0.3.7.yaml` and does not supersede ADR-0017.

Tool `0.3.7` and Project Profile contract version remain independent axes.

### 5. Verification requirements

WU-08 must verify that after applying this repository architecture declaration:

- all 18 previously uncovered product files are assigned to exactly one of the three new components;
- repository Responsibility Map coverage returns `unassigned_count == 0`;
- there are no equal-specificity component conflicts;
- no existing selector becomes unmatched;
- the three new components are classified as PRODUCT;
- their typed relationships resolve to declared endpoints;
- actual root `ptsip.yaml` remains `0.3.6-draft` at the same immutable Specification revision;
- no `ptsip_0.3.7.yaml` is created.

## Consequences

- the migration architecture is visible as three independently evolvable Tool product components;
- future subsystem-specific roles, dependency policy, compatibility boundaries, and verification can evolve without overloading the monolithic `ptsip-core` declaration;
- introducing another Tool subpackage does not automatically classify it as PRODUCT;
- repository self-profile completeness remains a deliberate architectural invariant rather than a warning that tests ignore;
- the change is an explicit project-owned architecture declaration, not an inference from file paths or Tool version numbering.

## Non-authority statement

This decision classifies only the three existing Tool `0.3.7` subsystem roots and the dependency relationships supported by their current implementation. It does not authorize future component creation, lifecycle reclassification, selector broadening, Project Profile version migration, or unrelated Responsibility Map changes.