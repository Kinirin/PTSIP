# PTSIP Execution Engine 0.3.0

Engine ID: `urn:uuid:b66de0f9-c454-4dbc-b43c-a2e695184671`
Capability: `urn:uuid:0a525fa7-7ce8-443c-b7d8-7885fadaf8e2`

### Purpose

The registered Execution engine implements `execution.assess-attempt`, `execution.assess-stop`, `execution.invoke`. Its machine contract declares
`VERIFY_CURRENT_REFERENCED_EFFECT_PROVENANCE_WITHOUT_INFERRING_ABSENCE_FROM_ATTEMPT_FAILURE`, `ACK_NOT_STOP_CONFIRMATION_AND_STOP_NOT_ORIGINAL_SUCCESS_OR_EFFECT_REVERSAL`, `CONSUME_CURRENT_CALLER_ADMITTED_LOCAL_AUTHORITY_AND_INVOKE_EXACT_BOUND_NATIVE_REQUEST_WITH_BOUNDED_IO_REVOCATION_AND_DIRECT_CHILD_STOP`. It consumes explicit inputs through its registered native interface.

### Contract changes

This release distributes the existing registered contract `urn:uuid:0a525fa7-7ce8-443c-b7d8-7885fadaf8e2`,
version `3.0.0`. The presentation correction introduces no engine
contract or implementation change. Existing public API and consumer authority
admission remain governed by their registered contracts.

Windows x64 and Linux x64 artifacts are included. macOS remains opt-in.
The original native artifacts are preserved from source
`58ff98ecc97769b5fb97631e2d8e89fcfdaf0243`; source and file hashes are recorded
in the Package manifest. GitHub artifact availability and PyPI publication are
verified separately.
