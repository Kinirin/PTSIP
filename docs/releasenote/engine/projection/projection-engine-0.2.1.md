# PTSIP Projection Engine 0.2.1

Engine ID: `urn:uuid:befe7a8f-786c-4851-babe-8a154a2324d0`
Capability: `urn:uuid:6a4077b3-184f-4975-bb6b-425c02584d45`

### Purpose

The registered Projection engine implements `projection.select`, `projection.render`. Its machine contract declares
`EXPLICIT_JSON_POINTER_SELECTION_WITHOUT_SOURCE_MUTATION_OR_LEGACY_OUTPUT_PARITY`, `VERIFY_CURRENT_CONTEXT_SNAPSHOT_AND_SOURCE_THEN_APPLY_REGISTERED_PROJECTION_FIELDS_AND_OUTPUT_SCHEMA`. It consumes explicit inputs through its registered native interface.

### Contract changes

This micro release increments the independent engine distribution version to `0.2.1`.
The capability contract and runtime authority boundary are unchanged.
Windows x64 and Linux x64 native wheels are rebuilt from the exact release source.
macOS is optional and is not included in this release.

The wheel archive uses explicit ZIP headers accepted by PyPI. Engine binaries
and member hashes are preserved during archive normalization. All seven engines
are uploaded sequentially through one publication job before the complete
PTSIP `0.3.8.b3` Package is published.
