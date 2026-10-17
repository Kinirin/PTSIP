# PTSIP Discovery Engine 0.3.1

Engine ID: `urn:uuid:e51f2248-0db7-4f5c-aa4b-e278a133bc50`
Capability: `urn:uuid:6cc343f5-a3af-404e-a22f-8d57f861bb97`

### Purpose

The registered Discovery engine implements `discovery.resolve`, `discovery.verify-registry`, `discovery.verify-invocation`. Its machine contract declares
`VERIFY_CURRENT_OWNER_SOURCE_HASH_CLOSURE_THEN_EXACT_REGISTERED_CAPABILITY`, `VERIFY_CURRENT_OWNER_SOURCE_HASH_CLOSURE_WITHOUT_AUTHORIZATION`, `VERIFY_REGISTERED_CURRENT_ENGINE_BINDING_REQUEST_AND_NATIVE_ARTIFACT_WITHOUT_GRANT`. It consumes explicit inputs through its registered native interface.

### Contract changes

This micro release increments the independent engine distribution version to `0.3.1`.
The capability contract and runtime authority boundary are unchanged.
Windows x64 and Linux x64 native artifacts retain their verified original source and hashes.
macOS is optional and is not included in this release.

The wheel archive uses explicit ZIP headers accepted by PyPI. Engine binaries
and member hashes are preserved during archive normalization. All seven engines
are embedded in each platform-specific PTSIP `0.3.8.b3` wheel. Only the PTSIP
project is uploaded to PyPI; independent engine versions and original artifacts
remain recorded in the complete Package composition.
