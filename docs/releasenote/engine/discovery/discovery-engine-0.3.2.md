# PTSIP Discovery Engine 0.3.2

Engine ID: `urn:uuid:e51f2248-0db7-4f5c-aa4b-e278a133bc50`
Capability: `urn:uuid:6cc343f5-a3af-404e-a22f-8d57f861bb97`

### Purpose

The registered Discovery engine implements `discovery.resolve`, `discovery.verify-registry`, `discovery.verify-invocation`. Its registered machine semantics include `VERIFY_CURRENT_OWNER_SOURCE_HASH_CLOSURE_THEN_EXACT_REGISTERED_CAPABILITY`, `VERIFY_CURRENT_OWNER_SOURCE_HASH_CLOSURE_WITHOUT_AUTHORIZATION`. No test result, capability declaration, or installation step grants additional authority.

### Contract changes

Engine version: `0.3.2` (previous PTSIP `0.3.8b5` engine `0.3.1`).
Current capability contract: `3.0.0`.
The registered capability identity and operation authority remain unchanged. This release assigns a new independent engine version because the packaged source/producer boundary differs from the previously published artifact.

Version authority: `src/ptsip/engines/discovery/version.yaml`. Published comparison source: `912a2b4956e856bec709cadc322cbd68588cdd5b`.

Required Windows x64 and Linux x64 verification must bind to the exact source SHA and component file hashes before publication. Existing published engine artifacts remain immutable; this note describes an upcoming version, not completed GitHub or PyPI publication.
