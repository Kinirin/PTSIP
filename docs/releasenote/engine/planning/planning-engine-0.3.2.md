# PTSIP Planning Engine 0.3.2

Engine ID: `urn:uuid:7a19982e-90ba-4388-9da0-90f57299e7d7`
Capability: `urn:uuid:b0c112bd-29b2-43c2-804e-c85ce1464369`

### Purpose

The registered Planning engine implements `planning.order`, `planning.impact`, `planning.select-installation`, `planning.prepare`. Its registered machine semantics include `EXPLICIT_DEPENDENCY_DAG_LEXICAL_TOPOLOGICAL_ORDER_FAIL_UNKNOWN_DUPLICATE_OR_CYCLE`, `VERIFY_CURRENT_TYPED_DEPENDENCY_DAG_AND_TWO_SIDED_EVIDENCE_THEN_FAIL_CLOSED_IMPACT_CLOSURE`. No test result, capability declaration, or installation step grants additional authority.

### Contract changes

Engine version: `0.3.2` (previous PTSIP `0.3.8b5` engine `0.3.1`).
Current capability contract: `3.0.0`.
The registered capability identity and operation authority remain unchanged. This release assigns a new independent engine version because the packaged source/producer boundary differs from the previously published artifact.

Version authority: `src/ptsip/engines/planning/version.yaml`. Published comparison source: `912a2b4956e856bec709cadc322cbd68588cdd5b`.

Required Windows x64 and Linux x64 verification must bind to the exact source SHA and component file hashes before publication. Existing published engine artifacts remain immutable; this note describes an upcoming version, not completed GitHub or PyPI publication.
