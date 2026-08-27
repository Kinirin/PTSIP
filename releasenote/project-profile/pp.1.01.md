# Project Profile pp.1.01

> **State:** WU-10 compatibility / migration implementation
> **Authority:** ADR-0019, ADR-0021, ADR-0022
> **Tool target:** 0.3.7

## What pp.1.01 means

`pp.1.01` is the first formally numbered current-generation **Project Profile Contract Version** under the independent PP namespace.

It is not a PTSIP Tool package version.

```text
Tool Version                     example: 0.3.7
Project Profile Contract Version example: pp.1.01
Project Profile Instance Revision immutable identity of one concrete declaration
```

## Important compatibility notice for 0.3.6-draft users

For the ADR-0021 bridge, historical Project Profile identity `0.3.6-draft` and canonical Project Profile identity `pp.1.01` have the same Project Profile contract semantics.

The transition is classified as:

```text
IDENTITY_ONLY
```

There is no contract-content delta solely because of this identity change:

```text
components:                unchanged
relationships:             unchanged
associated_artifacts:      unchanged
policies:                  unchanged
Responsibility Map:        unchanged
lifecycle classifications: unchanged
```

An otherwise valid `0.3.6-draft` Project Profile does **not** need lifecycle redesign, component reclassification, or a fresh architecture review merely because the canonical identity becomes `pp.1.01`.

Post-rewrite identity/schema validation is still required. `IDENTITY_ONLY` does not mean validation is skipped.

## Historical identity remains historical

PTSIP does not rewrite release history by claiming that existing `0.3.6-draft` files were originally published as `pp.1.01`.

`0.3.6-draft` remains a historical source identity. Compatibility metadata explains how that historical source is interpreted and which current PP target it may converge to; it does not retroactively rename the source.

## Direct latest-target convergence

ADR-0022 defines Project Profile migration as **source-to-current-target convergence**, not historical version traversal.

A supported project is migrated from the Project Profile it actually has to the current canonical Project Profile target selected by the installed Tool compatibility authority.

For Tool `0.3.7`, that target is `pp.1.01`.

Conceptually:

```text
actual supported source
        ↓ source-family reader
normalized source semantics
        ↓ direct reconciliation
current canonical PP target
        ↓ authorized execution
validated target profile
```

Historical or intermediate Project Profile generations may be used as compatibility knowledge when interpreting the source or explaining a current target obligation. They are **not mandatory user-visible execution steps** and are not materialized merely to replay version history.

For example, a future Tool that explicitly selects `pp.2.02` as its current supported target may directly plan:

```text
old supported source
        ↓
pp.2.02
```

without requiring the repository to enter `pp.1.01`, `pp.1.02`, `pp.1.03`, and every later intermediate contract first.

Direct convergence does not grant unrestricted inference authority. If historical semantics cannot be mapped deterministically to the current target, PTSIP must fail closed or require an explicit owner decision rather than guess.

## Migration continuity

The PP namespace must not cause valid migration work already targeting `0.3.6-draft` to restart.

The accepted WU-10 behavior is:

- if `ptsip_0.3.6.yaml` already exists, continue with that physical file and do not create a duplicate `ptsip_pp1.01.yaml`;
- the existing legacy target path is treated as an equivalent physical alias for logical target `pp.1.01`, and its internal historical identity is normalized to `pp.1.01` before PP-aware semantic execution is bound;
- if no `ptsip_0.3.6.yaml` target exists yet, a new current target may use `ptsip_pp1.01.yaml` directly;
- if canonical `ptsip.yaml` is already valid at `0.3.6-draft`, conversion to `pp.1.01` is an in-place identity rewrite rather than a semantic migration;
- simultaneous equivalent targets fail closed rather than being selected implicitly;
- no `ptsip_0.3.7.yaml` or other synthetic historical target is created solely because the Tool version or an intermediate historical generation existed.

## Adoption authority

Migration capability and real-project adoption are separate authorities.

Tool `0.3.7` supporting `pp.1.01` does not by itself authorize mutation of a repository's canonical `ptsip.yaml`. Real adoption still requires the exact supported source state and the applicable project-owner authorization.

The `IDENTITY_ONLY` equivalence removes unnecessary semantic reclassification work. It does not remove write authorization, snapshot validation, or post-write validation requirements.

## WU boundaries

WU-09 established typed PP identity, parser/serializer behavior, filename tokens, Tool/PP support declarations, schema/runtime identity surfaces, diagnostics, and maintained example identity surfaces.

WU-10 owns historical compatibility interpretation, direct current-target analysis/planning, legacy target continuity, guarded identity rewrite, PP-aware semantic execution/promotion, and adoption authority boundaries.

Final Tool `0.3.7` package identity, Specification freeze, and release-readiness handoff remain WU-11 responsibilities.
