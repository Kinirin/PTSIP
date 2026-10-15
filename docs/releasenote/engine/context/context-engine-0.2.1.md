# PTSIP Context Engine 0.2.1

Engine ID: `urn:uuid:0ca335ff-48b2-4358-99e4-3eeb48e94af3`
Capability: `urn:uuid:965ee856-a9cd-4326-bfd6-d23638b9cb2c`

### Purpose

The registered Context engine implements `context.snapshot`, `context.load`. Its machine contract declares
`SNAPSHOT_EXPLICIT_SOURCE_WITHOUT_REINTERPRETATION`, `VERIFY_BOUND_JSON_SOURCE_AND_INLINE_CONSTRAINTS_THEN_PRESERVE_CONTEXT_CONTENT`. It consumes explicit inputs through its registered native interface.

### Contract changes

This micro release increments the independent engine distribution version to `0.2.1`.
The capability contract and runtime authority boundary are unchanged.
Windows x64 and Linux x64 native wheels are rebuilt from the exact release source.
macOS is optional and is not included in this release.

The wheel archive uses explicit ZIP headers accepted by PyPI. Engine binaries
and member hashes are preserved during archive normalization. All seven engines
are uploaded sequentially through one publication job before the complete
PTSIP `0.3.8.b3` Package is published.
