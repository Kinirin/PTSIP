# PTSIP Planning Engine 0.3.0

Engine ID: `urn:uuid:7a19982e-90ba-4388-9da0-90f57299e7d7`
Capability: `urn:uuid:b0c112bd-29b2-43c2-804e-c85ce1464369`

### Purpose

The registered Planning engine implements `planning.order`, `planning.impact`, `planning.select-installation`, `planning.prepare`. Its machine contract declares
`EXPLICIT_DEPENDENCY_DAG_LEXICAL_TOPOLOGICAL_ORDER_FAIL_UNKNOWN_DUPLICATE_OR_CYCLE`, `VERIFY_CURRENT_TYPED_DEPENDENCY_DAG_AND_TWO_SIDED_EVIDENCE_THEN_FAIL_CLOSED_IMPACT_CLOSURE`, `SELECT_SEVEN_CURRENT_INDEPENDENT_PACKAGE_ARTIFACTS_BY_EXPLICIT_HOST_AND_OWNER_VERIFICATION_EVIDENCE_WITHOUT_INSTALLATION`, `VERIFY_ALL_EXPLICIT_NATIVE_INVOCATION_SOURCE_REFS_BEFORE_ORDERING_PREPARED_DAG_WITHOUT_EXECUTION`. It consumes explicit inputs through its registered native interface.

### Contract changes

This release distributes the existing registered contract `urn:uuid:b0c112bd-29b2-43c2-804e-c85ce1464369`,
version `3.0.0`. The presentation correction introduces no engine
contract or implementation change. Existing public API and consumer authority
admission remain governed by their registered contracts.

Windows x64 and Linux x64 artifacts are included. macOS remains opt-in.
The original native artifacts are preserved from source
`58ff98ecc97769b5fb97631e2d8e89fcfdaf0243`; source and file hashes are recorded
in the Package manifest. GitHub artifact availability and PyPI publication are
verified separately.
