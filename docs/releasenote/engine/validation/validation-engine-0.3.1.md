# PTSIP Validation Engine 0.3.1

Engine ID: `urn:uuid:44258a79-af23-4ce9-94c9-6d23de29bd21`
Capability: `urn:uuid:450659b1-1e89-4138-91d3-8dcbf6d6dc1a`

### Purpose

The registered Validation engine implements `validation.contract`, `validation.schema`, `validation.typed-io`, `validation.precondition`, `validation.case`, `validation.compose`, `validation.verify-handoff`, `validation.verify-context-handoff`, `validation.prepare-cases`, `validation.select-cases`, `validation.prepare-composition`. Its machine contract declares
`INDEPENDENT_INLINE_JSON_SCHEMA_2020_12_VALIDATION_NO_EXTERNAL_REFERENCE`, `EXPLICIT_OPTIONAL_INDEPENDENT_VALIDATION_CHECK_DAG_WITHOUT_IMPLICIT_VPMS_EQUIVALENCE`, `VERIFY_CURRENT_VERSIONED_VPMS_RELATIONS_AND_FULL_RESPONSIBILITY_CLASSIFICATION_WITHOUT_PUBLIC_API_CUTOVER`, `VERIFY_EXPLICIT_CONTEXT_SOURCE_AND_PROJECTION_OUTPUT_OWNERS_WITH_PRESERVED_PUBLIC_API`, `VERIFY_CURRENT_CASE_CATALOG_REFERENCE_CLOSURE_AND_INDEPENDENT_INLINE_CASE_CONTRACTS`, `SELECT_EXPLICIT_CASE_IDS_OR_CURRENT_REGISTERED_SELECTOR_WITH_NO_EXECUTION`, `VERIFY_ALL_SELECTED_CASE_ADAPTERS_AND_CONTRACT_INPUTS_BEFORE_ANY_INVOCATION`. It consumes explicit inputs through its registered native interface.

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
