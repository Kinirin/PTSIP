# PTSIP Projection Engine 0.2.2

Engine ID: `urn:uuid:befe7a8f-786c-4851-babe-8a154a2324d0`
Capability: `urn:uuid:6a4077b3-184f-4975-bb6b-425c02584d45`

### Purpose

The registered Projection engine implements `projection.select`, `projection.render`. Its registered machine semantics include `EXPLICIT_JSON_POINTER_SELECTION_WITHOUT_SOURCE_MUTATION_OR_LEGACY_OUTPUT_PARITY`, `VERIFY_CURRENT_CONTEXT_SNAPSHOT_AND_SOURCE_THEN_APPLY_REGISTERED_PROJECTION_FIELDS_AND_OUTPUT_SCHEMA`. No test result, capability declaration, or installation step grants additional authority.

### Contract changes

Engine version: `0.2.2` (previous PTSIP `0.3.8b5` engine `0.2.1`).
Current capability contract: `2.0.1`.
The registered capability identity and operation authority remain unchanged. This release assigns a new independent engine version because the packaged source/producer boundary differs from the previously published artifact.

Version authority: `src/ptsip/engines/projection/version.yaml`. Published comparison source: `912a2b4956e856bec709cadc322cbd68588cdd5b`.

Required Windows x64 and Linux x64 verification must bind to the exact source SHA and component file hashes before publication. Existing published engine artifacts remain immutable; this note describes an upcoming version, not completed GitHub or PyPI publication.
