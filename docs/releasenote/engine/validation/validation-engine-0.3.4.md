# PTSIP Validation Engine 0.3.4

Engine ID: `urn:uuid:44258a79-af23-4ce9-94c9-6d23de29bd21`
Capability: `urn:uuid:450659b1-1e89-4138-91d3-8dcbf6d6dc1a`

### Purpose

The registered Validation engine implements `validation.contract`, `validation.schema`, `validation.typed-io`, `validation.precondition`, `validation.case`, `validation.compose`, `validation.verify-handoff`, `validation.verify-context-handoff`, `validation.prepare-cases`, `validation.select-cases`, `validation.prepare-composition`. Its registered machine semantics include `INDEPENDENT_JSON_SCHEMA_VALIDATION_WITH_SCOPED_REFERENCE_RESOLUTION`, `EXPLICIT_OPTIONAL_INDEPENDENT_VALIDATION_CHECK_DAG_WITHOUT_IMPLICIT_VPMS_EQUIVALENCE`. No test result, capability declaration, or installation step grants additional authority.

### Contract changes

Engine version: `0.3.4` (previous PTSIP `0.3.8b5` engine `0.3.3`).
Current capability contract: `4.0.1`.
The Validation capability contract is updated to `4.0.1`; it retains the published `4.0.0` scoped JSON Schema reference-resolution behavior, including explicit local resources and opt-in bounded network acquisition.

Version authority: `src/ptsip/engines/validation/version.yaml`. Published comparison source: `912a2b4956e856bec709cadc322cbd68588cdd5b`.

Required Windows x64 and Linux x64 verification must bind to the exact source SHA and component file hashes before publication. Existing published engine artifacts remain immutable; this note describes an upcoming version, not completed GitHub or PyPI publication.
