# PTSIP Specification 0.3.6-draft

## Bound normative snapshot

The Tool `0.3.6` development line binds the `0.3.6-draft` Specification family to the immutable snapshot:

```text
12e2ccd15634ecb3d0a4195b0f61ac3f620e7540
```

This snapshot is the first WU-03 canonical activation snapshot after WU-01 lifecycle-boundary rules and WU-02 Responsibility Map semantics were frozen.

## Normative changes

`0.3.6-draft` changes PTSIP classification from the Tool `0.3.5` three-classification model into primary lifecycle ownership with exactly five canonical classifications:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` remains historical Tool `0.3.5` input vocabulary only. It is not a canonical Tool `0.3.6` classification alias.

The family defines repeatable lifecycle-boundary determination using governing lifecycle obligation, including Product versus Development Tooling, development build versus Delivery build, Delivery handoff versus Operations, Neutral Contract qualification, mixed-lifecycle split detection, and fail-closed unresolved classification behavior.

Responsibility Map v2 adds orthogonal component roles, typed project-owned relationships, and associated artifacts. Canonical roles are:

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

Canonical project-declared relationship types are:

```text
IMPORTS
LINKS
LOADS
INVOKES
READS
GENERATES
BUILDS
PACKAGES
PUBLISHES
DEPLOYS
VERIFIES
MANAGES
DOCUMENTS
SPECIFIES
GOVERNS
```

Associated artifacts are project-owned non-component support surfaces subordinate to exactly one classified anchor component. They do not receive or inherit a PTSIP classification and must be promoted/re-evaluated if independent lifecycle ownership emerges.

Responsibility Map declaration modes are `explicit`, `template`, and `hybrid`. Template selection is explicit and revision-bound; candidate discovery or repository layout does not select architecture authority.

## Compatibility boundary

Tool `0.3.6` compatibility with Tool `0.3.5` means understanding valid legacy profiles through a dedicated migration path and producing evidence-backed proposals. It does not mean accepting obsolete ontology as canonical new-schema state.

Legacy `TOOLCHAIN` must be re-evaluated for `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, component split, associated-artifact representation, typed relationships, or unresolved clarification as supported by evidence and project-owner confirmation.

## VPMS boundary

PTSIP classification and VPMS Verification Purpose remain separate axes. Current VPMS purpose vocabulary remains `PRODUCT | TOOLCHAIN`; the VPMS `TOOLCHAIN` token is not a Tool `0.3.6` PTSIP classification.

## Canonical assets

The bound snapshot includes the `0.3.6-draft` Specification body, conformance and governance documents, terminology, the normative Responsibility Map companion, canonical schemas, registry vocabulary, embedded Specification resources, and the aligned coding-agent contract.
