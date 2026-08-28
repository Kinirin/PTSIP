# ADR-0017 — Tool Version and Project Profile Contract Independence

**Status:** Accepted  
**Decision date:** 2026-08-25  
**Target Tool:** `0.3.7`  
**Governing work units:** WU-07 / WU-08  
**Related decisions:** ADR-0010, ADR-0011, ADR-0016  
**Bound development Specification:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`

## Context

Tool `0.3.7` adds versioned draft-profile transition, source-specific migration obligations, deterministic target planning, and guarded sequential application. The active `0.3.7-draft` Specification family is additive over the Tool `0.3.6` lifecycle and Responsibility Map semantics; it does not introduce a new Project Profile lifecycle ontology or a new canonical Project Profile schema merely because the Tool version is `0.3.7`.

The PTSIP repository currently adopts:

```text
ptsip.yaml = 0.3.6-draft
```

The public profile examples also remain `0.3.6-draft`:

```text
profiles/example.ptsip.yaml
profiles/hybrid-python-package.ptsip.yaml
profiles/template-python-package.ptsip.yaml
```

Earlier Tool `0.3.7` planning assumed that the PTSIP repository itself would dogfood a concrete `0.3.6-draft -> 0.3.7-draft` Project Profile migration by creating `ptsip_0.3.7.yaml`. That assumption conflates the Tool release version with the Project Profile contract version.

## Decision

### 1. Tool version and Project Profile version are independent axes

```text
PTSIP Tool version
    !=
Project Profile Specification version
```

Tool `0.3.7` may implement and ship transition/migration capabilities while the PTSIP repository and compatible consumer profiles continue to declare `ptsip.version: 0.3.6-draft`.

A Tool release does not by itself require a Project Profile draft-family migration.

### 2. No repository `ptsip_0.3.7.yaml` for Tool 0.3.7

The PTSIP repository will **not** create `ptsip_0.3.7.yaml` merely because Tool `0.3.7` is being developed or released.

The current canonical repository profile remains:

```text
ptsip.yaml = 0.3.6-draft
```

unless a later, independently justified Project Profile contract change selects a newer target draft family.

### 3. Existing profile examples remain 0.3.6-draft

The three existing public example profiles remain `0.3.6-draft` unless their Project Profile contract semantics actually change.

They must not be version-bumped solely to mirror the Tool package version.

### 4. The migration engine remains fully implemented

WU-01 through WU-07 are not cancelled or reduced to dead code. Their transition, compatibility, analyzer, proposal, reconciliation, execution, recovery, deletion, and promotion behavior remains the Tool `0.3.7` feature set.

Concrete mutation behavior is verified through controlled fixture repositories representing real draft-family transitions, including simple, sequential, stale, interrupted, and promotion cases.

The production PTSIP repository itself does not need an artificial target generation to prove those mechanics.

### 5. Repository dogfood changes from mutating migration to controlled self-analysis

WU-08 repository dogfood will verify that Tool `0.3.7` can inspect/analyze the PTSIP repository without forcing a Project Profile version change. Mutation/promotion behavior is proven in isolated fixture repositories.

The PTSIP repository's canonical profile must remain unchanged unless a separately accepted Project Profile migration decision exists.

### 6. `0.3.7-draft` remains a Tool-development normative family

This decision does not repeal ADR-0011 or the immutable `0.3.7-draft` normative snapshot.

`0.3.7-draft` continues to define the new transition/migration behavior for Tool `0.3.7` development and release. It is not interpreted as a blanket requirement that every Project Profile, including the PTSIP repository's own profile, declare `ptsip.version: 0.3.7-draft`.

### 7. Superseded self-adoption assumption

This ADR supersedes only the earlier planning/ADR-0011 consequence that expected WU-01/WU-08 to create/manage a repository-local `ptsip_0.3.7.yaml` and promote it into `ptsip.yaml` during Tool `0.3.7` development.

All general transition rules from ADR-0010 and the `0.3.7-draft` companion remain valid when an actual Project Profile draft migration is selected.

## Consequences

- no `ptsip_0.3.7.yaml` is created for the Tool `0.3.7` release;
- root `ptsip.yaml` remains `0.3.6-draft` unless separately migrated;
- existing profile examples remain `0.3.6-draft`;
- WU-07 verifies mutation/promotion through isolated transition fixtures rather than mutating the PTSIP repository;
- WU-08 performs repository self-analysis/dogfood without artificial profile promotion;
- future real Project Profile draft migrations still use Temporary PTSIP Profile Files, Final Point planning, source-specific completion, and guarded promotion exactly as specified.

## Non-authority statement

This ADR does not select a future Project Profile target version, classification change, component split/merge, relationship change, or other architecture delta. Any future repository profile migration requires its own accepted target decision and exact migration plan.
