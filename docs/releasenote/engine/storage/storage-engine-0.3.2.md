# PTSIP Storage Engine 0.3.2

Engine ID: `urn:uuid:d2b72bd9-63e1-4816-b06e-f516f01aee2e`
Capability: `urn:uuid:6c16fe73-53a0-4d39-8cd4-3fc2bc96ca3b`

### Purpose

The registered Storage engine implements `storage.mediate`, `storage.invoke`. Its registered machine semantics include `VERIFY_STORAGE_OWNER_PROVIDER_AND_CURRENT_TWO_SIDED_COMPATIBILITY_EVIDENCE_THEN_MEDIATION_PLAN`, `MEDIATE_REGISTERED_PROVIDER_AND_CONSUME_CURRENT_CALLER_ADMITTED_AUTHORITY_BEFORE_ISOLATED_NATIVE_PROVIDER_INVOCATION`. No test result, capability declaration, or installation step grants additional authority.

### Contract changes

Engine version: `0.3.2` (previous PTSIP `0.3.8b5` engine `0.3.1`).
Current capability contract: `3.0.0`.
The registered capability identity and operation authority remain unchanged. This release assigns a new independent engine version because the packaged source/producer boundary differs from the previously published artifact.

Version authority: `src/ptsip/engines/storage/version.yaml`. Published comparison source: `912a2b4956e856bec709cadc322cbd68588cdd5b`.

Required Windows x64 and Linux x64 verification must bind to the exact source SHA and component file hashes before publication. Existing published engine artifacts remain immutable; this note describes an upcoming version, not completed GitHub or PyPI publication.
