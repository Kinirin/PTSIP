# Contributing to PTSIP

PTSIP is an experimental, project-defined architecture policy. Current Tool `0.3.6` is bound to Specification `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea`.

## Contribution categories

Changes should identify one primary category:

- editorial/documentation;
- clarification;
- normative addition;
- normative breaking change;
- schema/registry change;
- conformance change;
- Tool implementation change;
- release/verification infrastructure change.

## Normative changes

A normative change should:

1. identify affected rule IDs and normative assets;
2. explain the architecture problem and lifecycle boundary involved;
3. describe compatibility impact;
4. update the registry when rule metadata changes;
5. update schemas and embedded machine-readable copies when required;
6. update conformance requirements when necessary;
7. add or update an ADR for material architecture decisions;
8. select a new immutable `SPEC_REVISION` only after the normative change is complete and reviewed.

Documentation, tests, workflows, planning, release notes, and Tool implementation changes do **not** move `SPEC_REVISION` by themselves.

## Tool 0.3.6 terminology

Use the current full name **Primary Lifecycle Ownership and Responsibility Isolation Policy** when expanding `PTSIP` in current Tool `0.3.6` documentation.

Canonical Tool `0.3.6` lifecycle classifications are exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is historical Tool `0.3.5` migration input only and must not be introduced as a canonical Tool `0.3.6` alias.

Keep lifecycle classification, responsibility roles, typed relationships, declaration/materialization provenance, and VPMS Verification Purpose as separate axes.

## Architecture authority

Do not infer project-owned architecture from path names, frameworks, workflow names, templates, heuristics, or confidence. Repository evidence may support review or a proposal but does not replace explicit Project Profile/Responsibility Map authority.

When modifying architecture-sensitive behavior, preserve the distinction between:

```text
Specification
Decision Authority
Project Profile / Responsibility Map
Observed evidence
Conformance Evaluation
```

## Examples

Examples must not silently weaken normative rules or imply that a particular repository layout has universal lifecycle ownership. If an example uses an exception, migration state, or unresolved evidence, label it explicitly.

## Release-facing changes

Before a Tool release, keep release-facing documents consistent with the exact Tool/package version and bound Specification revision. Historical release notes and ADRs are immutable history and must not be rewritten merely to make current-version wording uniform.
