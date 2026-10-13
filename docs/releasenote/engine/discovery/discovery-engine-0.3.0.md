# PTSIP Discovery Engine 0.3.0

Engine ID: `urn:uuid:e51f2248-0db7-4f5c-aa4b-e278a133bc50`
Capability: `urn:uuid:6cc343f5-a3af-404e-a22f-8d57f861bb97`

### Purpose

The registered Discovery engine implements `discovery.resolve`, `discovery.verify-registry`, `discovery.verify-invocation`. Its machine contract declares
`VERIFY_CURRENT_OWNER_SOURCE_HASH_CLOSURE_THEN_EXACT_REGISTERED_CAPABILITY`, `VERIFY_CURRENT_OWNER_SOURCE_HASH_CLOSURE_WITHOUT_AUTHORIZATION`, `VERIFY_REGISTERED_CURRENT_ENGINE_BINDING_REQUEST_AND_NATIVE_ARTIFACT_WITHOUT_GRANT`. It consumes explicit inputs through its registered native interface.

### Contract changes

This release distributes the existing registered contract `urn:uuid:6cc343f5-a3af-404e-a22f-8d57f861bb97`,
version `3.0.0`. The presentation correction introduces no engine
contract or implementation change. Existing public API and consumer authority
admission remain governed by their registered contracts.

Windows x64 and Linux x64 artifacts are included. macOS remains opt-in.
The original native artifacts are preserved from source
`58ff98ecc97769b5fb97631e2d8e89fcfdaf0243`; source and file hashes are recorded
in the Package manifest. GitHub artifact availability and PyPI publication are
verified separately.
