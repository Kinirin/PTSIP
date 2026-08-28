# ADR-0020 — Responsibility-Segmented Work Unit Sequence for Project Profile Transition

**Status:** Accepted  
**Decision date:** 2026-08-26  
**Target Tool:** `0.3.7`  
**Related decisions:** ADR-0017, ADR-0018, ADR-0019  
**Governing planning family:** `planning/0.3.7.md`

## Context

WU-08 originally accumulated repository self-analysis, package verification, Project Profile identity implementation, compatibility/migration work, and final release-readiness responsibilities. After ADR-0019 introduced an independent `pp.<major>.<minor>` Project Profile contract namespace, keeping all of those responsibilities inside WU-08 would make failures harder to localize and would make exact-SHA verification boundaries ambiguous.

The project owner prefers responsibility separation even when it increases the number of Work Units, because the narrower responsibility boundaries make debugging and regression ownership clearer. At the same time, the long-term-maintainability properties of the broader architecture option must be preserved inside each separated Work Unit.

## Decision

Tool `0.3.7` work after WU-07 SHALL be divided into four responsibility-specific Work Units:

```text
WU-08
Repository Self-Analysis and Package Baseline
        ↓
WU-09
Independent Project Profile Identity Core
        ↓
WU-10
Project Profile Compatibility, Migration, and Adoption
        ↓
WU-11
Tool 0.3.7 Final Regression, Specification Freeze, and Release Readiness
```

### 1. WU-08 — Repository Self-Analysis and Package Baseline

WU-08 owns repository-level baseline stabilization only. It may identify and repair repository self-profile ownership gaps, verify controlled migration fixtures, synchronize package artifact ownership evidence, and accept architecture decisions required by later work.

WU-08 MUST NOT implement the new `pp.<major>.<minor>` runtime identity model, perform real Project Profile migration to `pp.1.01`, freeze the final Tool `0.3.7` Specification, or claim final release readiness.

ADR-0019 may be accepted during WU-08 as architecture authority without activating `pp.1.01` in repository profiles.

### 2. WU-09 — Independent Project Profile Identity Core

WU-09 owns the machine-readable Project Profile contract identity layer independently from Tool SemVer.

Its responsibility includes:

- the `pp.<major>.<minor>` parser and canonical serializer;
- numeric major/minor comparison with canonical textual preservation;
- the three-axis identity model: Tool Version, Project Profile Contract Version, and Project Profile Instance Revision;
- schema/runtime identity representation;
- supported/unsupported Project Profile contract diagnostics;
- explicit Tool-to-PP compatibility declarations at the identity layer;
- tests proving Tool-version changes do not imply PP-version changes and PP-version changes do not require Tool-version changes when the Tool already supports them.

WU-09 MUST NOT perform repository Project Profile migration or canonical promotion. It produces a stable identity/compatibility foundation consumed by WU-10.

### 3. WU-10 — Project Profile Compatibility, Migration, and Adoption

WU-10 owns semantic compatibility and migration between historical Project Profile generations and the independently versioned PP contract model.

Its responsibility includes:

- historical source-family mapping, including legacy Tool-numbered source labels;
- `pp.0.00` compatibility semantics and the current `pp.1.01` target contract;
- PP-aware transition discovery and ordering;
- PP-aware Temporary Profile / Final Point identity rules where temporary artifacts are required;
- migration analyzer/planner/executor integration against PP identities;
- controlled fixture migration such as legacy generation -> `pp.1.01`;
- fail-closed handling of unsupported, ambiguous, stale, or incompatible PP generations;
- explicit authorization boundaries for real repository or consumer-profile adoption.

WU-10 MUST NOT treat Tool `0.3.7` as authority to migrate a Project Profile. Actual repository adoption is permitted only through independently justified and authorized PP migration.

### 4. WU-11 — Tool 0.3.7 Final Regression, Specification Freeze, and Release Readiness

WU-11 is the only post-WU-10 Work Unit that may declare final Tool `0.3.7` release readiness.

It owns:

- final Tool `0.3.7` package/version identity;
- final documentation/schema/specdata synchronization;
- complete regression after WU-09/WU-10 changes;
- final package/distribution verification;
- final exact-SHA workflow evidence;
- immutable Tool `0.3.7` Specification binding;
- explicit supported Project Profile compatibility declaration;
- release handoff.

Earlier successful workflow runs remain valid evidence for the exact SHA and responsibility they verified, but they are not final release-readiness authority after later PP identity/migration changes.

## Long-Term Maintainability Rules Applied to Every Work Unit

Responsibility segmentation MUST NOT become a collection of loosely coupled patches. Each WU SHALL preserve the following long-term-maintainability properties:

1. **Typed boundaries:** inputs, outputs, identities, and state transitions are explicit rather than inferred from filenames or Tool versions.
2. **Stable ownership:** each responsibility has one primary WU and one architectural owner; later WUs consume earlier outputs rather than duplicating their logic.
3. **Independent verification:** each WU has focused tests for its own contract so failures identify the responsible layer before full integration testing.
4. **Fail-closed compatibility:** unknown or ambiguous PP identities do not silently fall back to the current contract.
5. **Extensible registries/contracts:** supported PP families and Tool-to-PP compatibility are explicit data/contracts rather than scattered hard-coded assumptions where a registry or typed model is appropriate.
6. **No Tool/PP coupling:** Tool SemVer, PP contract version, and profile instance revision remain independent identities throughout all layers.
7. **No premature adoption:** architecture acceptance, identity implementation, migration capability, repository adoption, and release readiness are separate authorities.

## Consequences

- more Work Units are created, but each failure has a narrower debugging and regression ownership boundary;
- PP identity mechanics can evolve without forcing migration code or release code to change in the same unit;
- migration/adoption defects are distinguishable from parser/schema/identity defects;
- final release evidence necessarily postdates all PP identity and migration integration changes;
- long-term maintainability is preserved through typed contracts and explicit compatibility ownership rather than by keeping responsibilities in one large WU.
