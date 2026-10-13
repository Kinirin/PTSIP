# PTSIP Projection Engine 0.2.0

Engine ID: `urn:uuid:befe7a8f-786c-4851-babe-8a154a2324d0`
Capability: `urn:uuid:6a4077b3-184f-4975-bb6b-425c02584d45`

### Purpose

The registered Projection engine implements `projection.select`, `projection.render`. Its machine contract declares
`EXPLICIT_JSON_POINTER_SELECTION_WITHOUT_SOURCE_MUTATION_OR_LEGACY_OUTPUT_PARITY`, `VERIFY_CURRENT_CONTEXT_SNAPSHOT_AND_SOURCE_THEN_APPLY_REGISTERED_PROJECTION_FIELDS_AND_OUTPUT_SCHEMA`. It consumes explicit inputs through its registered native interface.

### Contract changes

This release distributes the existing registered contract `urn:uuid:6a4077b3-184f-4975-bb6b-425c02584d45`,
version `2.0.0`. The presentation correction introduces no engine
contract or implementation change. Existing public API and consumer authority
admission remain governed by their registered contracts.

Windows x64 and Linux x64 artifacts are included. macOS remains opt-in.
The original native artifacts are preserved from source
`58ff98ecc97769b5fb97631e2d8e89fcfdaf0243`; source and file hashes are recorded
in the Package manifest. GitHub artifact availability and PyPI publication are
verified separately.
