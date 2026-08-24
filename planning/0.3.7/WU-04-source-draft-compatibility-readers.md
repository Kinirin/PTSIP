# WU-04 — Source-Draft Compatibility Readers

> **Status:** COMPLETE / FOCUSED TEST VERIFIED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-03 — evidence/provenance normalization (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-04 exact entry baseline:** `84946457f31156e47c15454498711afa526cf19d`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **Accepted architecture decision:** `ADR-0013 — Common Source Model with Family-Specific Semantics`  
> **WU-04 implementation content SHA:** `fbd243fd7ce340c66f8ded31e418eedbeec9d506`  
> **Focused verification:** `14 passed / 0 failed` in the isolated compatibility-reader harness  
> **Successor:** WU-05 — migration analyzer and obligation evaluation

## 0. Purpose

Provide explicit read-only compatibility boundaries for source PTSIP profiles that participate in migration, including historical Tool `0.3.5` semantics and Tool `0.3.6` draft profiles, without teaching the canonical Tool `0.3.7` runtime to reinterpret historical semantics as current architecture.

This generalizes the cancelled Tool `0.3.6.1` WU-03 legacy-reader plan.

## 1. Canonical compatibility rule

Historical source vocabulary must be preserved exactly while reading.

Tool `0.3.5` historical classifications:

```text
PRODUCT
TOOLCHAIN
NEUTRAL_CONTRACT
```

Tool `0.3.6` canonical lifecycle classifications:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

A reader MUST NOT silently translate `TOOLCHAIN -> DEVELOPMENT_TOOLING`, silently normalize an obsolete draft into 0.3.7 semantics, or mutate source declarations while reading.

## 2. Source-generation identity

A source reader receives the exact generation identity from WU-01:

```text
path
source draft version
source specification revision
content/snapshot identity
```

The result remains tied to that source identity so Sequential Work can evaluate each source independently.

The implemented reader verifies the exact profile bytes, declared version, specification source, and immutable revision against `ProfileGenerationIdentity` before compatibility projection. Post-read byte changes are reported as stale source context.

## 3. Frozen supported source families

Tool `0.3.7` migration readers support exactly:

```text
Tool 0.3.5 source semantics:
0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e

Tool 0.3.6 source semantics:
0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
```

Support is revision-bound rather than version-label-only. A recognized draft label at another revision fails as `UNSUPPORTED_SOURCE_REVISION`; an unrecognized family fails as `UNSUPPORTED_SOURCE_FAMILY`.

This prevents a mutable historical draft label from silently acquiring semantics that were never reviewed for Tool 0.3.7 migration.

## 4. Accepted architecture — balanced common model

The project owner selected the balanced option recorded by ADR-0013:

```text
source profile bytes
    -> family/revision validation
    -> source-local integrity validation
    -> common CompatibilitySourceProfile
       + family-specific semantics
    -> WU-05
```

Common source fields cover genuinely shared concepts only:

- generation identity;
- component ID;
- source classification token;
- include/exclude selectors;
- purpose;
- declaration scope;
- source location pointer;
- preserved source attributes;
- policies;
- associated artifacts and relationships when explicitly declared by that family;
- immutable representation of the validated raw source payload.

Source-read objects serialize with:

```text
authority = SOURCE_DECLARATION_ONLY
```

They are not canonical effective profiles.

Family-specific semantics remain separate:

- `V034SourceSemantics` preserves component-vs-boundary declaration form and historical boundary roots;
- `V036SourceSemantics` preserves Responsibility Map mode, template identity, hybrid declaration scope, and remove-ID sets.

## 5. Work tracks — completion record

### WU-04A — source-family inventory — COMPLETE

Frozen the exact Tool 0.3.5 / Tool 0.3.6 source families and immutable Specification revisions required by Tool 0.3.7.

Historical Tool `0.3.5` is represented by the Project Profile family it actually shipped against: `0.3.4-draft @ b5b17...`.

### WU-04B — compatibility models — COMPLETE

Added:

```text
src/ptsip/source_compat/model.py
```

The common model deliberately stores historical classification as `source_classification` rather than the current canonical `Classification` enum. Historical fields such as Tool 0.3.5 `lifecycle_owner`, `consumers`, and `analysis_inputs`, and Tool 0.3.6 `roles`, are retained as source attributes rather than promoted into current semantics.

The complete validated source payload is recursively frozen so the handoff is read-only and information-preserving.

### WU-04C — strict readers — COMPLETE

Added:

```text
src/ptsip/source_compat/reader.py
src/ptsip/source_compat/integrity.py
src/ptsip/source_compat/__init__.py
src/ptsip/specdata/ptsip-source-profile-0.3.4.schema.json
src/ptsip/specdata/ptsip-source-profile-0.3.6.schema.json
```

The compatibility boundary is separate from canonical `validate_profile()` and canonical template materialization.

Strict reading performs:

1. exact WU-01 generation byte binding;
2. source version/revision/source identity matching;
3. frozen historical schema validation;
4. source-local historical integrity validation;
5. non-destructive compatibility projection.

Source-local integrity includes historical constraints that do not require evaluation of the current repository:

- Tool 0.3.5 boundary overlap rejection;
- duplicate source IDs;
- component dependency-policy source references/conflicts;
- Tool 0.3.6 hybrid override/remove conflicts;
- Tool 0.3.6 explicit endpoint namespace/reference integrity;
- duplicate semantic relationships.

Repository-current selector coverage, file existence, and migration obligation evaluation are intentionally not performed here; they belong to WU-05.

### WU-04D — frozen fixtures — COMPLETE / FOCUSED VERIFIED

Added role-scoped compatibility tests:

```text
tests/ptsip/source_compat/test_reader_037.py
tests/ptsip/source_compat/test_reader_integrity_037.py
```

Focused scenarios cover:

- exact supported family/revision inventory;
- Tool 0.3.5 `TOOLCHAIN` preservation;
- Tool 0.3.5 component attributes preservation;
- Tool 0.3.5 boundary-form preservation without component inference;
- Tool 0.3.6 explicit roles/relationships/artifacts preservation;
- Tool 0.3.6 hybrid override scope and removal-ID preservation;
- Tool 0.3.6 template source read without current-template materialization;
- recognized-version/unknown-revision fail-closed behavior;
- unsupported family fail-closed behavior;
- WU-01 generation identity mismatch rejection;
- `TOOLCHAIN` rejection as a Tool 0.3.6 classification;
- post-read source-byte stale detection;
- deterministic, non-mutating read output;
- Tool 0.3.5 boundary-overlap rejection;
- Tool 0.3.6 hybrid override/remove conflict rejection.

Focused isolated verification result:

```text
14 passed / 0 failed
```

This is focused implementation verification, not the WU-08 full repository/self-hosted exact-SHA regression gate.

### WU-04E — migration handoff — COMPLETE

Public compatibility API:

```text
ptsip.source_compat.read_source_profile(...)
ptsip.source_compat.validate_source_read_binding(...)
ptsip.source_compat.supported_source_families()
```

The handoff gives WU-05 source architecture plus exact source-generation identity, but assigns no target classification and no Required/Removal/Async obligation category.

## 6. Explicit non-goals preserved

WU-04 did not:

- decide target lifecycle classification;
- translate `TOOLCHAIN -> DEVELOPMENT_TOOLING`;
- assign Required/Removal/Async status;
- rewrite `ptsip.yaml` or temporary profiles;
- infer split components or relationships;
- auto-select or materialize a current template for a historical source;
- broaden canonical validation to accept legacy profiles as current runtime input;
- mark a source migration complete;
- create `ptsip_0.3.7.yaml`;
- create or mutate a Final PTSIP Point File.

## 7. Completion gate assessment

WU-04 completion requirements are satisfied at focused-test scope:

- supported source families read deterministically and non-destructively;
- historical vocabulary remains intact;
- malformed, unsupported, identity-mismatched, and stale source input fails explicitly;
- exact source-generation identity remains bound to the read result;
- canonical runtime authority is not weakened;
- WU-05 receives a stable source-specific analysis input contract.

Full repository regression, package verification, self-hosted workflow evidence, and end-to-end repository dogfood remain WU-08 responsibilities.

## 8. Entry discipline

WU-04 entered automatically under the project owner's standing successor-entry authorization after WU-03 completion was recorded and exact `dev/0.3.7` HEAD `84946457f31156e47c15454498711afa526cf19d` was freshly revalidated.

Implementation began only after the project owner selected **Option C — Common Source Model + Family-specific Semantic Extensions**. ADR-0013 records that accepted architecture choice.
