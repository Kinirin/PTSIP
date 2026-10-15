# PTSIP Planning Engine 0.3.1

Engine ID: `urn:uuid:7a19982e-90ba-4388-9da0-90f57299e7d7`
Capability: `urn:uuid:b0c112bd-29b2-43c2-804e-c85ce1464369`

### Purpose

The registered Planning engine implements `planning.order`, `planning.impact`, `planning.select-installation`, `planning.prepare`. Its machine contract declares
`EXPLICIT_DEPENDENCY_DAG_LEXICAL_TOPOLOGICAL_ORDER_FAIL_UNKNOWN_DUPLICATE_OR_CYCLE`, `VERIFY_CURRENT_TYPED_DEPENDENCY_DAG_AND_TWO_SIDED_EVIDENCE_THEN_FAIL_CLOSED_IMPACT_CLOSURE`, `SELECT_SEVEN_CURRENT_INDEPENDENT_PACKAGE_ARTIFACTS_BY_EXPLICIT_HOST_AND_OWNER_VERIFICATION_EVIDENCE_WITHOUT_INSTALLATION`, `VERIFY_ALL_EXPLICIT_NATIVE_INVOCATION_SOURCE_REFS_BEFORE_ORDERING_PREPARED_DAG_WITHOUT_EXECUTION`. It consumes explicit inputs through its registered native interface.

### Contract changes

This micro release increments the independent engine distribution version to `0.3.1`.
The capability contract and runtime authority boundary are unchanged.
Windows x64 and Linux x64 native wheels are rebuilt from the exact release source.
macOS is optional and is not included in this release.

The wheel archive uses explicit ZIP headers accepted by PyPI. Engine binaries
and member hashes are preserved during archive normalization. All seven engines
are uploaded sequentially through one publication job before the complete
PTSIP `0.3.8.b3` Package is published.
