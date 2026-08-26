# Project Profile pp.1.01

> **State:** WU-09 identity implementation / pre-adoption
> **Authority:** ADR-0019, ADR-0021
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
components:             unchanged
relationships:          unchanged
associated_artifacts:   unchanged
policies:               unchanged
Responsibility Map:     unchanged
lifecycle classifications: unchanged
```

An otherwise valid `0.3.6-draft` Project Profile does **not** need lifecycle redesign, component reclassification, or a fresh architecture review merely because the canonical identity becomes `pp.1.01`.

Post-rewrite identity/schema validation is still required. `IDENTITY_ONLY` does not mean validation is skipped.

## Historical identity remains historical

PTSIP does not rewrite release history by claiming that existing `0.3.6-draft` files were originally published as `pp.1.01`.

`0.3.6-draft` remains a historical source identity. WU-10 owns the explicit compatibility bridge and migration-continuity behavior between historical labels and PP identities.

## Migration continuity

The PP namespace must not cause valid migration work already targeting `0.3.6-draft` to restart.

The accepted WU-10 behavior is:

- if `ptsip_0.3.6.yaml` already exists, continue with that file and do not create a duplicate `ptsip_pp1.01.yaml`;
- if no `ptsip_0.3.6.yaml` target exists yet, a new current target may use `ptsip_pp1.01.yaml` directly;
- if canonical `ptsip.yaml` is already valid at `0.3.6-draft`, conversion to `pp.1.01` is an in-place identity rewrite rather than a semantic migration;
- simultaneous equivalent targets fail closed rather than being selected implicitly.

Those execution rules are documented here for user clarity but are implemented and verified by WU-10, not WU-09.

## WU-09 boundary

WU-09 implements the typed PP identity, canonical parser/serializer, filename token, operation-specific Tool/PP support declaration, schema/runtime identity surfaces, diagnostics, and maintained example identity surfaces.

WU-09 does not authorize real-project profile migration or canonical promotion.
