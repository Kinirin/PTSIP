# PTSIP Context Engine 0.2.2

Engine ID: `urn:uuid:0ca335ff-48b2-4358-99e4-3eeb48e94af3`
Capability: `urn:uuid:965ee856-a9cd-4326-bfd6-d23638b9cb2c`

### Purpose

The registered Context engine implements `context.snapshot`, `context.load`. Its registered machine semantics include `SNAPSHOT_EXPLICIT_SOURCE_WITHOUT_REINTERPRETATION`, `VERIFY_BOUND_JSON_SOURCE_AND_INLINE_CONSTRAINTS_THEN_PRESERVE_CONTEXT_CONTENT`. No test result, capability declaration, or installation step grants additional authority.

### Contract changes

Engine version: `0.2.2` (previous PTSIP `0.3.8b5` engine `0.2.1`).
Current capability contract: `2.0.0`.
The registered capability identity and operation authority remain unchanged. This release assigns a new independent engine version because the packaged source/producer boundary differs from the previously published artifact.

Version authority: `src/ptsip/engines/context/version.yaml`. Published comparison source: `912a2b4956e856bec709cadc322cbd68588cdd5b`.

Required Windows x64 and Linux x64 verification must bind to the exact source SHA and component file hashes before publication. Existing published engine artifacts remain immutable; this note describes an upcoming version, not completed GitHub or PyPI publication.
