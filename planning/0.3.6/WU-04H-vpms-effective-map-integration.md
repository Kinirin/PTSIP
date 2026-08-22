# WU-04H — VPMS Narrow Read-Only Effective-Map Integration

> **Status:** COMPLETE / VERIFIED  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization + effective-map consumers  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04G — COMPLETE / EXACT-SHA VERIFIED  
> **Entry baseline:** `ab05051ee9fd5f10a4ca3aa17020ad314ee82722`  
> **Completion snapshot:** `9fac22d31333346dbe56a12dee890df1229d560b`  
> **WU-04G verification snapshot:** `3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667`  
> **Bound Specification:** `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea`  
> **Successor:** WU-04I — final regression + WU-04 completion, NEXT / NOT ENTERED

## 0. Purpose

WU-04H connects VPMS to the canonical Tool 0.3.6 Effective Responsibility Map without giving VPMS responsibility for PTSIP profile resolution, template materialization, lifecycle classification authority, conformance, adoption, clarification, or decision control.

WU-04D through WU-04G established the authoritative PTSIP read chain:

```text
selected profile
    -> validate_profile()
    -> ValidationResult.resolved_profile
    -> ResolvedProfile.effective_payload
```

WU-04H extends that already-resolved view only far enough for VPMS target metadata consumption:

```text
PTSIP selected profile
    -> validate_profile()
    -> ResolvedProfile.effective_payload
    -> narrow immutable target-metadata snapshot
    -> VPMS target lookup / verification execution
```

VPMS MUST NOT independently interpret:

```text
explicit
template
hybrid
overrides
removals
selector precedence
template revisions
PTSIP architecture authority
```

Those remain exclusively PTSIP responsibilities.

WU-04H is therefore a narrow downstream-consumer integration stage. It is not a VPMS redesign stage and it is not a second implementation of the PTSIP Responsibility Map.

## 1. Predecessor authority and entry conditions

WU-04G is complete and exact-SHA verified for its declared stage scope. Its canonical result establishes that clarification/adoption and the local/remote decision path consume validated effective architecture, preserve the exact selected profile path, and fail closed on invalid or unresolved source state.

WU-04H MUST preserve all predecessor guarantees. In particular, it MUST NOT reintroduce raw profile interpretation as an alternate architecture authority merely because VPMS historically read `ptsip.yaml` directly.

The WU-04H entry baseline is:

```text
ab05051ee9fd5f10a4ca3aa17020ad314ee82722
```

No WU-04I implementation or test migration may be entered before WU-04H completion is reviewed.

## 2. Non-negotiable architecture boundary

PTSIP and VPMS remain sibling systems with different semantic questions:

```text
PTSIP
    What is this component?

VPMS
    Why does this verification exist?
```

The permitted data/dependency direction remains:

```text
PTSIP resolved architecture
          |
          | narrow read-only metadata
          v
         VPMS
```

The following are prohibited:

```text
PTSIP core -> VPMS runtime dependency

VPMS -> PTSIP profile mutation

VPMS -> PTSIP Decision Authority mutation

VPMS -> template/hybrid materialization

VPMS -> PTSIP classification inference

PTSIP classification -> automatic VPMS purpose mapping

VPMS PASS/FAIL -> PTSIP conformance result
```

Existing package-isolation behavior remains authoritative:

```text
import ptsip
    succeeds when VPMS is unavailable

import vpms
    does not implicitly import ptsip

src/ptsip/**
    contains no vpms runtime dependency
```

WU-04H MUST preserve that separation.

## 3. Existing integration state

The existing VPMS bridge is located at:

```text
src/vpms/integration/ptsip_bridge.py
```

Its current compatibility-oriented behavior reads a PTSIP profile file directly, extracts minimal component metadata, and refuses template/hybrid profiles because VPMS does not own PTSIP materialization.

That historical boundary was appropriate before Tool 0.3.6 established `ResolvedProfile.effective_payload`, but it is no longer sufficient as the canonical Tool 0.3.6 integration path.

The WU-04H goal is not to teach VPMS how to materialize templates. The goal is to stop requiring VPMS to interpret PTSIP source declarations at all for canonical 0.3.6 consumption.

The canonical 0.3.6 path becomes:

```text
PTSIP resolves architecture
    -> VPMS consumes already-resolved metadata
```

not:

```text
VPMS reads raw ptsip.yaml
    -> VPMS inspects source mode
    -> VPMS decides whether materialization is required
```

## 4. Accepted WU-04H decision package

### H1-B — Effective Map is the canonical Tool 0.3.6 VPMS metadata source

Canonical Tool 0.3.6 VPMS integration MUST consume metadata derived from:

```text
ResolvedProfile.effective_payload
```

and MUST NOT use raw `ptsip.yaml` as a second architecture interpretation.

This gives explicit, template, and hybrid source declarations one downstream VPMS semantic model.

### H2-B — Preserve package isolation

`vpms` MUST NOT import unstable `ptsip` runtime internals merely to call `validate_profile()` itself as the canonical integration mechanism.

Canonical orchestration is conceptually:

```python
validation = validate_profile(...)
resolved = validation.resolved_profile
metadata = metadata_from_effective_map(resolved.effective_payload)
```

The caller or integration layer may compose the two sibling packages, but neither subsystem acquires bidirectional runtime ownership of the other.

The preferred architecture therefore keeps the VPMS bridge as a consumer of an already-resolved mapping rather than as an orchestrator of PTSIP validation.

### H3-B — Keep the VPMS-facing metadata contract narrow

The initial H contract remains intentionally minimal:

```text
component_id
classification
```

WU-04H MUST NOT add PTSIP fields merely because they are available in the Effective Map.

The following remain outside the initial VPMS metadata contract unless an actual VPMS runtime requirement is demonstrated:

```text
purpose
roles
relationships
associated_artifacts
shipped
runtime_required
executable
release_owner
compatibility_owner
source_mode
template identity
template revision
materialization provenance
```

The narrow contract minimizes semantic coupling and keeps VPMS dependent only on the stable target metadata it actually consumes.

### H4-B — Explicit/template/hybrid equivalence

If explicit, template, and hybrid profiles resolve to semantically equivalent effective component architecture, VPMS MUST receive equivalent target metadata.

VPMS does not need to know which source declaration mode produced the effective architecture.

The required downstream property is:

```text
same effective component identity + classification
    -> same VPMS target metadata
```

### H5-B — Invalid or unresolved PTSIP architecture blocks VPMS metadata

If PTSIP cannot produce a valid `ResolvedProfile`:

```text
validation/materialization failure
    -> no canonical VPMS metadata snapshot
    -> no raw-profile fallback
    -> no partial target list
    -> no architecture guess
```

WU-04H MUST preserve the fail-closed behavior established by WU-04G.

A failed PTSIP resolution MUST NOT cause VPMS to inspect raw `components` or another partial source representation to continue execution.

### H6-B — PTSIP classification and VPMS Verification Purpose remain independent

PTSIP Tool 0.3.6 classifications remain exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

VPMS Verification Purpose remains exactly:

```text
PRODUCT
TOOLCHAIN
```

No mapping such as:

```text
PTSIP DEVELOPMENT_TOOLING
    -> VPMS TOOLCHAIN
```

is automatic or authoritative.

For example, the following remains a valid combination:

```text
PTSIP classification:
    DEVELOPMENT_TOOLING

VPMS Verification Purpose:
    PRODUCT
```

because implementation ownership and protected verification purpose answer different questions.

`VerificationPurpose.TOOLCHAIN` is VPMS vocabulary and MUST NOT be renamed merely because Tool 0.3.6 PTSIP replaced historical PTSIP `TOOLCHAIN` classification with the five-classification lifecycle model.

### H7-B — Integration remains read-only

VPMS execution and metadata consumption MUST NOT change:

```text
ptsip.yaml
selected profile path
source_mode
template identity
effective map
DecisionStore
GitHub authority
clarification decisions
classification
```

The metadata snapshot presented to VPMS should remain immutable.

### H8-B — WU-04H is not Tool 0.3.5 architecture migration

Existing raw bridge behavior may remain as a narrow historical/compatibility boundary where required, but it is not the canonical Tool 0.3.6 effective-map integration path.

WU-04H MUST NOT become:

```text
Tool 0.3.5 legacy profile reader
TOOLCHAIN -> DEVELOPMENT_TOOLING translator
legacy boundaries migration
historical profile repair
```

Those concerns belong to later Tool 0.3.5 migration work units.

## 5. Canonical runtime flow

The required canonical H flow is:

```text
repository
    |
    v
selected PTSIP profile
    |
    v
validate_profile()
    |
    +-- invalid --------------------------+
    |                                     |
    |                                     v
    |                         block VPMS metadata consumption
    |
    v
ValidationResult.resolved_profile
    |
    v
ResolvedProfile.effective_payload
    |
    v
narrow VPMS-facing metadata projection
    |
    +--> component_id
    +--> classification
    |
    v
immutable PtsipMetadataSnapshot
    |
    v
resolve_target_metadata(TargetRef)
    |
    v
VPMS VerificationCase
```

The following flow is prohibited:

```text
VPMS
    -> read raw ptsip.yaml
    -> inspect responsibility_map.mode
    -> resolve template identity
    -> materialize template
    -> merge hybrid overrides/removals
    -> infer effective component set
```

That would create a second PTSIP implementation inside VPMS.

## 6. Production scope

Primary production target:

```text
src/vpms/integration/ptsip_bridge.py
```

Likely export surface:

```text
src/vpms/integration/__init__.py
```

Recommended canonical addition:

```python
metadata_from_effective_map(effective_payload)
    -> PtsipMetadataSnapshot
```

or an equivalent narrowly named API.

The new effective-map projection API should:

1. accept an already-resolved read-only mapping;
2. read only effective `components` required for VPMS target metadata;
3. validate only the narrow VPMS metadata shape;
4. preserve deterministic component ordering;
5. reject duplicate or malformed target identities explicitly;
6. avoid PTSIP architecture validation or selector interpretation;
7. avoid template materialization;
8. avoid importing `ptsip` runtime internals;
9. expose no write path.

The existing:

```python
load_ptsip_metadata(profile_path)
```

must not remain the canonical Tool 0.3.6 integration path.

Whether it remains as a compatibility API, is narrowed, or is explicitly marked historical should be decided during implementation without using WU-04H as an excuse for unrelated breaking cleanup.

## 7. Test migration scope

WU-04H now owns the narrow VPMS/PTSIP Effective Map integration boundary.

Authoritative H structural scope:

```text
tests/vpms/integration/test_ptsip_bridge_035.py
tests/vpms/integration/test_ptsip_boundary_035.py
tests/vpms/architecture/test_isolation_035.py
```

New focused Tool 0.3.6 file authorized by this plan:

```text
tests/vpms/integration/test_ptsip_effective_map_036.py
```

`tests/vpms/integration/test_repository_self_adoption_035.py` remains outside automatic H migration unless the H implementation genuinely changes its target-binding contract. That file primarily owns VPMS registry/purpose selection and real repository case execution rather than Effective Map projection.

WU-04H MUST NOT use this stage as permission to restructure all of:

```text
tests/vpms/**
```

Domain, execution, registry, Formula, Variables, Policy, and Runner behavior remain outside H unless a direct H compatibility issue is discovered and explicitly recorded.

## 8. Required focused H contracts

The new focused file should cover at least the following contracts:

```text
test_explicit_effective_map_projects_vpms_target_metadata

test_template_effective_map_projects_same_vpms_target_metadata

test_hybrid_effective_map_projects_effective_override_metadata

test_hybrid_removal_is_absent_from_vpms_target_metadata

test_invalid_profile_produces_no_vpms_metadata_snapshot

test_vpms_metadata_projection_does_not_fallback_to_raw_profile

test_ptsip_classification_does_not_determine_vpms_verification_purpose

test_effective_metadata_consumption_does_not_mutate_ptsip_source

test_effective_metadata_consumption_does_not_mutate_decision_authority

test_vpms_bridge_does_not_import_ptsip_runtime

test_ptsip_core_does_not_import_vpms
```

The explicit/template/hybrid equivalence cases may share a parametrized assertion where doing so does not hide declaration-mode-specific setup.

Stateful mutation/authority cases MUST continue to use fresh isolated state. Read-only metadata fixtures may be shared only when the input Effective Map is immutable and no freshness/staleness behavior is under test.

## 9. Existing VPMS 0.3.5 compatibility-test rule

Current VPMS tests include historical PTSIP `TOOLCHAIN` classification examples because VPMS was introduced under Tool 0.3.5.

WU-04H MUST NOT blindly replace every historical occurrence of:

```text
TOOLCHAIN
```

with:

```text
DEVELOPMENT_TOOLING
```

Instead, tests must be classified by responsibility:

```text
historical Tool 0.3.5 / raw bridge compatibility contract
    -> preserve historical vocabulary where that is the contract

canonical Tool 0.3.6 Effective Map integration contract
    -> use PRODUCT / DEVELOPMENT_TOOLING / DELIVERY /
       OPERATIONS / NEUTRAL_CONTRACT as appropriate
```

VPMS `VerificationPurpose.TOOLCHAIN` remains valid VPMS vocabulary and is never renamed under WU-04H.

## 10. Isolation requirements

After WU-04H, all of the following MUST remain true:

```text
PTSIP imports without VPMS installed

VPMS imports without implicitly importing PTSIP

PTSIP source does not import vpms

VPMS ordinary execution has no PTSIP write path

VPMS outcome does not become PTSIP conformance

VPMS Verification Purpose does not become PTSIP classification

PTSIP classification does not determine VPMS Verification Purpose
```

The legacy raw-file bridge must not become an alternate canonical Tool 0.3.6 architecture authority.

## 11. Specification and ADR handling

Default WU-04H decision: no new `SPEC_REVISION` is required at entry.

ADR-0006 already establishes the intended dependency/data boundary:

```text
PTSIP
    -> explicit stable project/component data contract
    -> VPMS
```

and already prohibits PTSIP core from depending on VPMS and prohibits ordinary VPMS verification execution from obtaining a write path into PTSIP profile or authority state.

WU-04D through WU-04G subsequently established the missing Tool 0.3.6 resolved/effective-map mechanism.

Therefore WU-04H is initially treated as implementation of an already-approved architecture boundary rather than a new normative PTSIP rule.

A new Specification revision is required only if implementation discovers a genuinely new normative PTSIP meaning that is not already covered by the current Specification/ADR authority.

Implementation convenience alone is not sufficient reason to move `SPEC_REVISION`.

## 12. Verification strategy

WU-04H verification is narrower than WU-04I final WU-04 regression.

Recommended H verification sequence:

```text
1. architecture isolation tests
2. existing VPMS PTSIP bridge/boundary tests
3. new H focused Effective Map tests
4. relevant VPMS integration family
5. repository self-profile validation
```

Representative focused invocation:

```powershell
.\.venv\Scripts\python.exe -m pytest -q `
    tests/vpms/architecture/test_isolation_035.py `
    tests/vpms/integration/test_ptsip_bridge_035.py `
    tests/vpms/integration/test_ptsip_boundary_035.py `
    tests/vpms/integration/test_ptsip_effective_map_036.py
```

The H implementation should prefer the repository checkout source over stale installed package state during verification, following the lesson captured during WU-04G.

A complete repository regression MUST NOT replace focused H verification. Conversely, H focused success MUST NOT be described as Tool 0.3.6 release readiness.

The authoritative final WU-04-wide complete regression remains WU-04I responsibility.

## 13. Completion gate

WU-04H is complete only when all of the following are reviewed and satisfied:

- canonical Tool 0.3.6 VPMS metadata is derived from validated `ResolvedProfile.effective_payload`;
- explicit/template/hybrid effective architecture produces equivalent VPMS metadata when semantically equivalent;
- hybrid effective overrides are visible in projected metadata;
- hybrid effective removals are absent from projected metadata;
- invalid/unresolved PTSIP architecture produces no canonical VPMS metadata snapshot;
- no raw-profile fallback acts as canonical architecture authority;
- VPMS does not implement template or hybrid materialization semantics;
- VPMS does not infer or mutate PTSIP architecture;
- VPMS does not mutate profile or Decision Authority state;
- PTSIP core has no VPMS dependency;
- VPMS package isolation remains intact;
- PTSIP Tool 0.3.6 classifications remain the five canonical values;
- VPMS Verification Purpose remains exactly `PRODUCT | TOOLCHAIN`;
- PTSIP classification remains independent from VPMS Verification Purpose;
- H focused tests pass;
- H-participating existing tests pass or every remaining failure is explicitly classified;
- no WU-04I implementation or test migration was entered early.

After this gate is reviewed:

```text
WU-04H COMPLETE
    -> fresh branch HEAD
    -> create WU-04I document
    -> enter WU-04I
```

### Completion record

WU-04H completion was reviewed against exact implementation snapshot:

```text
9fac22d31333346dbe56a12dee890df1229d560b
```

Maintainer-provided repository-local verification on that exact SHA completed with all three process exit codes equal to `0`:

```text
H focused verification
    35 passed in 5.29s
    exit=0

VPMS integration + architecture isolation
    41 passed in 4.78s
    exit=0

Repository self-profile verification
    4 passed in 2.70s
    exit=0
```

The first two pytest invocations emitted a Windows `%TEMP%\pytest-of-rhkrt\pytest-current` cleanup `PermissionError` from an ignored `atexit` callback after pytest had already reported all selected tests passed. Because both processes returned exit code `0`, this is classified as post-test temporary-directory cleanup noise rather than an H semantic/test failure.

Completion review confirms H1-B through H8-B, including explicit/template/hybrid equivalence, effective hybrid override/removal behavior, fail-closed unresolved handoff, no canonical raw fallback, package isolation, classification/purpose independence, read-only PTSIP/Decision Authority behavior, and preservation of Tool 0.3.5 compatibility boundaries. No WU-04I implementation or test migration was entered before this H completion review.

WU-04H is therefore **COMPLETE / VERIFIED** at snapshot `9fac22d31333346dbe56a12dee890df1229d560b`. This stage-scoped verification does not claim Tool 0.3.6 release readiness or replace WU-04I final WU-04-wide regression.

## 14. Explicitly out of scope

WU-04H does not authorize:

```text
VPMS VerificationPurpose vocabulary redesign
VPMS TOOLCHAIN rename
VPMS registry redesign
VPMS Formula/Variables/Policy model redesign
VPMS Runner redesign
PTSIP conformance changes
PTSIP clarification/adoption changes
PTSIP safe-apply changes
Decision Authority changes
GitHub CAS changes
template catalog changes
materializer redesign
Tool 0.3.5 architecture migration
blind TOOLCHAIN translation
repository-wide tests/vpms migration
release workflow changes
WU-04I final regression implementation
```

If implementation discovers that one of these boundaries must change, WU-04H must be amended before that work begins.

## 15. Expected WU-04 pipeline after H

The intended WU-04 sequence remains:

```text
WU-04A  template catalog
WU-04B  deterministic materializer
WU-04C  declaration authority
WU-04D  ResolvedProfile
WU-04E  validation consumes effective map
WU-04F  conformance consumes effective map
WU-04G  clarification/adoption consumes effective map
WU-04H  VPMS narrow read-only effective-map consumer
WU-04I  final regression + WU-04 completion
```

WU-04H is the final downstream runtime integration stage before WU-04I performs the final WU-04 regression/completion gate.
