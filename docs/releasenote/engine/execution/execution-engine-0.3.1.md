# PTSIP Execution Engine 0.3.1

Engine ID: `urn:uuid:b66de0f9-c454-4dbc-b43c-a2e695184671`
Capability: `urn:uuid:0a525fa7-7ce8-443c-b7d8-7885fadaf8e2`

### Purpose

The registered Execution engine implements `execution.assess-attempt`, `execution.assess-stop`, `execution.invoke`. Its machine contract declares
`VERIFY_CURRENT_REFERENCED_EFFECT_PROVENANCE_WITHOUT_INFERRING_ABSENCE_FROM_ATTEMPT_FAILURE`, `ACK_NOT_STOP_CONFIRMATION_AND_STOP_NOT_ORIGINAL_SUCCESS_OR_EFFECT_REVERSAL`, `CONSUME_CURRENT_CALLER_ADMITTED_LOCAL_AUTHORITY_AND_INVOKE_EXACT_BOUND_NATIVE_REQUEST_WITH_BOUNDED_IO_REVOCATION_AND_DIRECT_CHILD_STOP`. It consumes explicit inputs through its registered native interface.

### Contract changes

This micro release increments the independent engine distribution version to `0.3.1`.
The capability contract and runtime authority boundary are unchanged.
Windows x64 and Linux x64 native wheels are rebuilt from the exact release source.
macOS is optional and is not included in this release.

The wheel archive uses explicit ZIP headers accepted by PyPI. Engine binaries
and member hashes are preserved during archive normalization. All seven engines
are uploaded sequentially through one publication job before the complete
PTSIP `0.3.8.b3` Package is published.
