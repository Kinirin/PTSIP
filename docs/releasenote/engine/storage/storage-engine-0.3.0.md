# PTSIP Storage Engine 0.3.0

Engine ID: `urn:uuid:d2b72bd9-63e1-4816-b06e-f516f01aee2e`
Capability: `urn:uuid:6c16fe73-53a0-4d39-8cd4-3fc2bc96ca3b`

### Purpose

The registered Storage engine implements `storage.mediate`, `storage.invoke`. Its machine contract declares
`VERIFY_STORAGE_OWNER_PROVIDER_AND_CURRENT_TWO_SIDED_COMPATIBILITY_EVIDENCE_THEN_MEDIATION_PLAN`, `MEDIATE_REGISTERED_PROVIDER_AND_CONSUME_CURRENT_CALLER_ADMITTED_AUTHORITY_BEFORE_ISOLATED_NATIVE_PROVIDER_INVOCATION`. It consumes explicit inputs through its registered native interface.

### Contract changes

This release distributes the existing registered contract `urn:uuid:6c16fe73-53a0-4d39-8cd4-3fc2bc96ca3b`,
version `3.0.0`. The presentation correction introduces no engine
contract or implementation change. Existing public API and consumer authority
admission remain governed by their registered contracts.

Windows x64 and Linux x64 artifacts are included. macOS remains opt-in.
The original native artifacts are preserved from source
`58ff98ecc97769b5fb97631e2d8e89fcfdaf0243`; source and file hashes are recorded
in the Package manifest. GitHub artifact availability and PyPI publication are
verified separately.
