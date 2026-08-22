# WU-07 — Tool 0.3.5 Legacy Reader

> **Status:** PRE-CREATED / LOCKED  
> **Roadmap predecessor:** WU-06 — evidence/provenance normalization  
> **Planning baseline:** `ac73ba300ded36805255441c71491194e5264ed3`  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** WU-08 — lifecycle migration analyzer

## 0. Purpose

Introduce an explicit read-only compatibility boundary for Tool `0.3.5` project/profile architecture so migration analysis can understand historical PTSIP declarations without teaching the canonical Tool `0.3.6` runtime to treat legacy semantics as current architecture.

## 1. Canonical rule

Tool `0.3.5` historical PTSIP classifications remain:

```text
PRODUCT
TOOLCHAIN
NEUTRAL_CONTRACT
```

Tool `0.3.6` canonical classifications remain:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

The legacy reader MUST preserve historical meaning exactly. `TOOLCHAIN` MUST NOT be silently translated to `DEVELOPMENT_TOOLING` at read time.

## 2. Scope

- identify supported Tool 0.3.5 profile/schema forms and historical boundary declarations;
- parse them into a dedicated legacy-read representation;
- preserve legacy classification vocabulary and source locations;
- preserve enough historical structure for later migration analysis;
- distinguish legacy-read objects from canonical Tool 0.3.6 `ResolvedProfile`/Effective Map objects;
- reject malformed or unsupported historical input explicitly rather than repairing it while reading.

## 3. Work tracks

### WU-07A — legacy format inventory
Freeze exactly which published 0.3.5-era forms are supported.

### WU-07B — dedicated compatibility model
Create a narrow model that makes historical state inspectable without making it canonical runtime state.

### WU-07C — strict reader
Implement deterministic parsing, source identity, and diagnostics.

### WU-07D — historical fixture suite
Use frozen 0.3.5 examples to verify vocabulary preservation and non-mutation.

### WU-07E — migration handoff
Expose a stable read-only input contract for WU-08 without performing migration decisions itself.

## 4. Non-goals

WU-07 does not:

- decide target lifecycle classification;
- rewrite `ptsip.yaml`;
- infer split components or relationships;
- auto-select Tool 0.3.6 templates;
- translate `TOOLCHAIN` automatically;
- mutate historical evidence;
- broaden canonical 0.3.6 validation to accept legacy profiles as if they were current.

## 5. Completion gate

WU-07 is complete when supported 0.3.5 profiles can be read deterministically and non-destructively, historical vocabulary is preserved, unsupported/malformed cases fail explicitly, canonical 0.3.6 runtime authority is not weakened, and WU-08 receives a stable migration-analysis input contract.

## 6. Entry discipline

Pre-created planning document only. Assign actual entry SHA and status `ACTIVE` only after WU-06 completion.
