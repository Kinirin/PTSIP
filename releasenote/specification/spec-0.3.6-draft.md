# PTSIP Specification 0.3.6-draft

## Bound normative snapshot

The Tool `0.3.6` development line currently binds the `0.3.6-draft` Specification family to the immutable snapshot:

```text
d6995ed232e845b88d8235b851e80ab54b7804ea
```

Earlier normative checkpoints in this development line were:

```text
WU-03 canonical activation
12e2ccd15634ecb3d0a4195b0f61ac3f620e7540

WU-04C declaration-authority/materialization boundary
82abd09360df09a95fbbfb516855fa9ffb49f050
```

The binding first moved for WU-04C because that stage added new normative Responsibility Map declaration-authority and materialization rules. It moved again at WU-04G G0 because accepted project-owned clarification/adoption decisions required the additional normative safe-apply authority rule frozen as `PTSIP-RMAP-017`.

The binding does not move merely because implementation code, tests, planning documents, or regression-closure records change.

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

### WU-04C declaration authority and materialization boundary

WU-04C freezes declaration source as an axis separate from lifecycle ownership:

```text
classification
    = primary lifecycle ownership

source_mode
    = explicit | template | hybrid declaration-source structure

materializer
    = deterministic non-authoritative resolution
```

The project owns the complete declaration in `explicit` mode. In `template` mode, the project owns the decision to adopt one exact stable template ID plus immutable revision. In `hybrid` mode, the project additionally owns stable-ID replacement, extension, and removal decisions over the selected immutable template declaration.

Hybrid precedence is:

```text
project replacement / extension / removal
    > selected immutable template declaration
```

Tool `0.3.6` uses whole-entity stable-ID replacement rather than implicit field-level inheritance. Materialization must not infer missing architecture, auto-select templates, rewrite lifecycle ownership, silently repair dangling relationships, cascade-delete related declarations, or mutate the source profile merely to produce a valid result.

`explicit`, `template`, and `hybrid` source declarations resolve to one Canonical Effective Responsibility Map for downstream semantic evaluation while retaining the original declaration and exact template identity as separate provenance. A deterministic effective-map digest may be used for reproducibility/comparison but cannot become architecture authority.

These requirements are normative in `PTSIP-RMAP-013` through `PTSIP-RMAP-016` and are recorded architecturally by `ADR-0009`.

### WU-04G accepted-decision safe-apply authority

WU-04G G0 adds `PTSIP-RMAP-017`.

An **accepted project-owned clarification/adoption decision** may authorize only the exact project-owned declaration delta needed to represent that decision. When an immutable template declaration cannot represent that accepted decision without a project delta, the safe-apply layer may perform the source transition:

```text
template -> hybrid
```

while preserving the exact selected template ID and revision and writing only the minimum accepted project-owned override/extension/replacement.

Repository discovery, evidence collection, candidate generation, and deterministic materialization remain non-authoritative and cannot trigger that transition by themselves.

This normative addition is the reason the current Tool `0.3.6` binding is `d6995ed232e845b88d8235b851e80ab54b7804ea`.

## Compatibility boundary

Tool `0.3.6` does not treat obsolete Tool `0.3.5` ontology as canonical Tool `0.3.6` state.

The evidence-driven Tool `0.3.5 -> 0.3.6` migration system has been moved out of the Tool `0.3.6` release gate and continues as Tool `0.3.6.1` planning. Historical `TOOLCHAIN` input therefore remains migration input rather than an accepted canonical alias.

Legacy `TOOLCHAIN` must eventually be re-evaluated for `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, component split, associated-artifact representation, typed relationships, or unresolved clarification as supported by evidence and project-owner confirmation.

## VPMS boundary

PTSIP classification and VPMS Verification Purpose remain separate axes. Current VPMS purpose vocabulary remains `PRODUCT | TOOLCHAIN`; the VPMS `TOOLCHAIN` token is not a Tool `0.3.6` PTSIP classification.

## Canonical assets

The bound snapshot includes the `0.3.6-draft` Specification body, conformance and governance documents, terminology, the normative Responsibility Map companion, canonical schemas, registry vocabulary, embedded Specification resources, and the aligned coding-agent contract available at that immutable revision.
