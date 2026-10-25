# PTSIP Execution Engine 0.3.2

Engine ID: `urn:uuid:b66de0f9-c454-4dbc-b43c-a2e695184671`
Capability: `urn:uuid:0a525fa7-7ce8-443c-b7d8-7885fadaf8e2`

### Purpose

The registered Execution engine implements `execution.assess-attempt`, `execution.assess-stop`, `execution.invoke`. Its registered machine semantics include `VERIFY_CURRENT_REFERENCED_EFFECT_PROVENANCE_WITHOUT_INFERRING_ABSENCE_FROM_ATTEMPT_FAILURE`, `ACK_NOT_STOP_CONFIRMATION_AND_STOP_NOT_ORIGINAL_SUCCESS_OR_EFFECT_REVERSAL`. No test result, capability declaration, or installation step grants additional authority.

### Contract changes

Engine version: `0.3.2` (previous PTSIP `0.3.8b5` engine `0.3.1`).
Current capability contract: `3.0.0`.
The registered capability identity and operation authority remain unchanged. This release assigns a new independent engine version because the packaged source/producer boundary differs from the previously published artifact.

Version authority: `src/ptsip/engines/execution/version.yaml`. Published comparison source: `912a2b4956e856bec709cadc322cbd68588cdd5b`.

Required Windows x64 and Linux x64 verification must bind to the exact source SHA and component file hashes before publication. Existing published engine artifacts remain immutable; this note describes an upcoming version, not completed GitHub or PyPI publication.
