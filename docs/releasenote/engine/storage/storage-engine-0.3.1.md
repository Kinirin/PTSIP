# PTSIP Storage Engine 0.3.1

Engine ID: `urn:uuid:d2b72bd9-63e1-4816-b06e-f516f01aee2e`
Capability: `urn:uuid:6c16fe73-53a0-4d39-8cd4-3fc2bc96ca3b`

### Purpose

The registered Storage engine implements `storage.mediate`, `storage.invoke`. Its machine contract declares
`VERIFY_STORAGE_OWNER_PROVIDER_AND_CURRENT_TWO_SIDED_COMPATIBILITY_EVIDENCE_THEN_MEDIATION_PLAN`, `MEDIATE_REGISTERED_PROVIDER_AND_CONSUME_CURRENT_CALLER_ADMITTED_AUTHORITY_BEFORE_ISOLATED_NATIVE_PROVIDER_INVOCATION`. It consumes explicit inputs through its registered native interface.

### Contract changes

This micro release increments the independent engine distribution version to `0.3.1`.
The capability contract and runtime authority boundary are unchanged.
Windows x64 and Linux x64 native wheels are rebuilt from the exact release source.
macOS is optional and is not included in this release.

The wheel archive uses explicit ZIP headers accepted by PyPI. Engine binaries
and member hashes are preserved during archive normalization. All seven engines
are uploaded sequentially through one publication job before the complete
PTSIP `0.3.8.b3` Package is published.
