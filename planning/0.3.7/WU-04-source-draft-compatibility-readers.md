# WU-04 — Source-Draft Compatibility Readers

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-03 — evidence/provenance normalization (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-04 exact entry baseline:** `84946457f31156e47c15454498711afa526cf19d`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
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

The result must remain tied to that source identity so Sequential Work can evaluate each source independently.

## 3. Scope

- inventory supported historical/current source profile families needed by 0.3.7 migration;
- parse each supported source into a dedicated compatibility representation;
- preserve source vocabulary, stable IDs, selectors, relationships, associated artifacts, declaration modes, and source locations where present;
- distinguish source-read objects from canonical 0.3.7 resolved/effective objects;
- reject malformed/unsupported historical input explicitly;
- expose a stable read-only input contract for WU-05;
- retain exact source draft/revision identity.

## 4. Work tracks

### WU-04A — source-family inventory

Freeze exactly which 0.3.5/0.3.6-era forms 0.3.7 promises to migrate.

### WU-04B — compatibility models

Prefer narrowly scoped adapters/models per genuinely distinct semantics. Do not create duplicate models where existing 0.3.6 structures can safely represent a source without reinterpretation.

### WU-04C — strict readers

Implement deterministic parsing and diagnostics with source identity preserved.

### WU-04D — frozen fixtures

Use historical examples and explicit multi-generation fixtures to verify preservation and non-mutation.

### WU-04E — migration handoff

Expose source architecture plus identity to WU-05 without assigning target obligations in the reader.

## 5. Non-goals

WU-04 does not:

- decide target lifecycle classification;
- assign Required/Removal/Async status;
- rewrite `ptsip.yaml` or temporary profiles;
- infer split components or relationships;
- auto-select templates;
- broaden canonical validation to accept legacy profiles as if they were current;
- mark a source migration complete.

## 6. Completion gate

WU-04 is complete when supported source profiles can be read deterministically and non-destructively, historical meaning remains intact, malformed/unsupported cases fail explicitly, source identity remains bound to the read result, canonical runtime authority is not weakened, and WU-05 receives a stable source-specific analysis input.

## 7. Entry discipline

WU-04 entered automatically under the project owner's standing successor-entry authorization after WU-03 completion was recorded and exact `dev/0.3.7` HEAD `84946457f31156e47c15454498711afa526cf19d` was freshly revalidated.

This ACTIVE state authorizes WU-04 implementation only. It does not authorize WU-05 implementation, Required/Removal/Async categorization, Temporary PTSIP Profile mutation, Final Point delta application, or bypass of any project-owner decision gate.
