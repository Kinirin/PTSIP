# PTSIP Specification 0.3.6-draft

## Bound normative snapshot

The Tool `0.3.6` development line currently binds the `0.3.6-draft` Specification family to the immutable snapshot:

```text
82abd09360df09a95fbbfb516855fa9ffb49f050
```

The earlier WU-03 canonical-activation snapshot was:

```text
12e2ccd15634ecb3d0a4195b0f61ac3f620e7540
```

The binding moved because WU-04C added new normative Responsibility Map declaration-authority and materialization rules. It did not move merely because later implementation code existed.

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

## Compatibility boundary

Tool `0.3.6` compatibility with Tool `0.3.5` means understanding valid legacy profiles through a dedicated migration path and producing evidence-backed proposals. It does not mean accepting obsolete ontology as canonical new-schema state.

Legacy `TOOLCHAIN` must be re-evaluated for `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, component split, associated-artifact representation, typed relationships, or unresolved clarification as supported by evidence and project-owner confirmation.

## VPMS boundary

PTSIP classification and VPMS Verification Purpose remain separate axes. Current VPMS purpose vocabulary remains `PRODUCT | TOOLCHAIN`; the VPMS `TOOLCHAIN` token is not a Tool `0.3.6` PTSIP classification.

## Canonical assets

The bound snapshot includes the `0.3.6-draft` Specification body, conformance and governance documents, terminology, the normative Responsibility Map companion, canonical schemas, registry vocabulary, embedded Specification resources, and the aligned coding-agent contract available at that immutable revision.
