# PTSIP Context Engine 0.2.0

Engine ID: `urn:uuid:0ca335ff-48b2-4358-99e4-3eeb48e94af3`
Capability: `urn:uuid:965ee856-a9cd-4326-bfd6-d23638b9cb2c`

### Purpose

The registered Context engine implements `context.snapshot`, `context.load`. Its machine contract declares
`SNAPSHOT_EXPLICIT_SOURCE_WITHOUT_REINTERPRETATION`, `VERIFY_BOUND_JSON_SOURCE_AND_INLINE_CONSTRAINTS_THEN_PRESERVE_CONTEXT_CONTENT`. It consumes explicit inputs through its registered native interface.

### Contract changes

This release distributes the existing registered contract `urn:uuid:965ee856-a9cd-4326-bfd6-d23638b9cb2c`,
version `2.0.0`. The presentation correction introduces no engine
contract or implementation change. Existing public API and consumer authority
admission remain governed by their registered contracts.

Windows x64 and Linux x64 artifacts are included. macOS remains opt-in.
The original native artifacts are preserved from source
`58ff98ecc97769b5fb97631e2d8e89fcfdaf0243`; source and file hashes are recorded
in the Package manifest. GitHub artifact availability and PyPI publication are
verified separately.
