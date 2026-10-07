# Root Family authority transition

This is an operational report. Policy records, admitted migration graphs, and the registered verification contract own authority.

## Canonical paths and source ownership

| Family | Developer migration owners | Support migration owners |
|---|---|---|
| NORM | [MPD-NORM-0001](../NORM/MPD-NORM-0001.yaml) ACTIVE | [SFP-NORM-0001](../../../src/policy/NORM/SFP-NORM-0001.yaml) ACTIVE<br>[SFP-NORM-0002](../../../src/policy/NORM/SFP-NORM-0002.yaml) DRAFT |
| GOV | [MPD-GOV-0001](../GOV/MPD-GOV-0001.yaml) ACTIVE<br>[MPD-GOV-0002](../GOV/MPD-GOV-0002.yaml) APPROVED | [SFP-GOV-0001](../../../src/policy/GOV/SFP-GOV-0001.yaml) ACTIVE |
| INTENT | [MPD-INTENT-0001](../INTENT/MPD-INTENT-0001.yaml) ACTIVE | [SFP-INTENT-0001](../../../src/policy/INTENT/SFP-INTENT-0001.yaml) ACTIVE |
| ARCH | [MPD-ARCH-0001](../ARCH/MPD-ARCH-0001.yaml) ACTIVE | [SFP-ARCH-0001](../../../src/policy/ARCH/SFP-ARCH-0001.yaml) ACTIVE<br>[SFP-ARCH-0002](../../../src/policy/ARCH/SFP-ARCH-0002.yaml) DRAFT<br>[SFP-ARCH-0003](../../../src/policy/ARCH/SFP-ARCH-0003.yaml) RETIRED |
| INFO | [MPD-INFO-0001](../INFO/MPD-INFO-0001.yaml) ACTIVE<br>[MPD-INFO-0002](../INFO/MPD-INFO-0002.yaml) APPROVED<br>[MPD-INFO-0003](../INFO/MPD-INFO-0003.yaml) DRAFT | [SFP-INFO-0001](../../../src/policy/INFO/SFP-INFO-0001.yaml) ACTIVE<br>[SFP-INFO-0002](../../../src/policy/INFO/SFP-INFO-0002.yaml) DRAFT<br>[SFP-INFO-0003](../../../src/policy/INFO/SFP-INFO-0003.yaml) RETIRED |
| CNTR | [MPD-CNTR-0001](../CNTR/MPD-CNTR-0001.yaml) ACTIVE<br>[MPD-CNTR-0002](../CNTR/MPD-CNTR-0002.yaml) DRAFT | [SFP-CNTR-0001](../../../src/policy/CNTR/SFP-CNTR-0001.yaml) ACTIVE<br>[SFP-CNTR-0002](../../../src/policy/CNTR/SFP-CNTR-0002.yaml) DRAFT<br>[SFP-CNTR-0003](../../../src/policy/CNTR/SFP-CNTR-0003.yaml) RETIRED |
| RISK | [MPD-RISK-0001](../RISK/MPD-RISK-0001.yaml) DRAFT | [SFP-RISK-0001](../../../src/policy/RISK/SFP-RISK-0001.yaml) DRAFT |
| SUPPLY | [MPD-SUPPLY-0001](../SUPPLY/MPD-SUPPLY-0001.yaml) DRAFT | [SFP-SUPPLY-0001](../../../src/policy/SUPPLY/SFP-SUPPLY-0001.yaml) DRAFT |
| REAL | [MPD-REAL-0001](../REAL/MPD-REAL-0001.yaml) ACTIVE<br>[MPD-REAL-0002](../REAL/MPD-REAL-0002.yaml) APPROVED | [SFP-REAL-0001](../../../src/policy/REAL/SFP-REAL-0001.yaml) ACTIVE |
| ASSURE | [MPD-ASSURE-0001](../ASSURE/MPD-ASSURE-0001.yaml) ACTIVE<br>[MPD-ASSURE-0002](../ASSURE/MPD-ASSURE-0002.yaml) APPROVED | [SFP-ASSURE-0001](../../../src/policy/ASSURE/SFP-ASSURE-0001.yaml) ACTIVE<br>[SFP-ASSURE-0002](../../../src/policy/ASSURE/SFP-ASSURE-0002.yaml) DRAFT<br>[SFP-ASSURE-0003](../../../src/policy/ASSURE/SFP-ASSURE-0003.yaml) RETIRED |
| CTRL | [MPD-CTRL-0001](../CTRL/MPD-CTRL-0001.yaml) ACTIVE | [SFP-CTRL-0001](../../../src/policy/CTRL/SFP-CTRL-0001.yaml) ACTIVE |
| CHANGE | [MPD-CHANGE-0001](../CHANGE/MPD-CHANGE-0001.yaml) ACTIVE<br>[MPD-CHANGE-0002](../CHANGE/MPD-CHANGE-0002.yaml) APPROVED<br>[MPD-CHANGE-0003](../CHANGE/MPD-CHANGE-0003.yaml) DRAFT | [SFP-CHANGE-0001](../../../src/policy/CHANGE/SFP-CHANGE-0001.yaml) ACTIVE |
| OPS | [MPD-OPS-0001](../OPS/MPD-OPS-0001.yaml) ACTIVE | [SFP-OPS-0001](../../../src/policy/OPS/SFP-OPS-0001.yaml) ACTIVE |
| RECORD | [MPD-RECORD-0001](../RECORD/MPD-RECORD-0001.yaml) ACTIVE | [SFP-RECORD-0001](../../../src/policy/RECORD/SFP-RECORD-0001.yaml) ACTIVE<br>[SFP-RECORD-0002](../../../src/policy/RECORD/SFP-RECORD-0002.yaml) DRAFT |

The table contains 22 Developer and 24 Support migration owners. State-specific siblings preserve the original responsibility state. Empty source families stay DRAFT.

The separately registered ACTIVE policies `MPD-REAL-0003` and `MPD-ASSURE-0003` own the directly authorized Go test choice and its verification contract. They are new scoped policies, not source migration owners. The neutral module policies `MPD-CNTR-0003`, `MPD-REAL-0004`, and `SFP-CNTR-0004` are also separately admitted ACTIVE policies. The final catalog contains 26 new Developer Root policies and 25 new Support Root policies.

The 50 Developer and 24 Support source files moved without content changes to each plane's `legacy/` directory. The graphs map 263 Developer and 344 Support responsibility units to exact new policy IDs, sections and JSON pointers. Archived sources have catalog role `MIGRATION_SOURCE`. They do not independently own the current Root responsibility.

Old source reads are explicitly registered frozen interfaces reconstructed from Root-owned values. They expose their canonical owners. Missing owners, class/state drift, overlapping pointers, escaping paths, altered source bytes and changed values fail closed. No filename search or semantic-similarity fallback exists.

New Root policy reads and Developer resolver bindings select new IDs and exact unit sections. Support resolves its own family records, schema/role registries and `SUP_` subjects. Root containers have role `ROOT_FAMILY_CONTRACT_AUTHORITY`; they do not aggregate the original project-scoped effects.

## Lifecycle evidence

Developer migration owners were individually registered at DRAFT 0.0. Direct project-owner approval and status preflight then applied DRAFT 0.0 -> APPROVED 1.0 -> ACTIVE 2.0 where required. APPROVED/DRAFT sources retain those states. Support ACTIVE/DRAFT/RETIRED source states are preserved under its class-local lifecycle. Original source approvals and states remain unchanged.

The registration analysis and lifecycle results are recorded in `PRA-20261006-ROOT-FAMILY-TRANSITION.yaml`, `root-family-transition-lifecycle.json`, `PRA-20261006-ROOT-FAMILY-GO-VERIFICATION.yaml`, and `root-family-go-lifecycle.json`. Approval `recorded_at` records when this run captured the human decision, not an invented original approval date.

## Verification entry

`developer/policy/contracts/root-family-transition-verification.v1.json` is the language-neutral versioned acceptance contract. The Go module `developer/tests/rootfamily` interprets it and exercises the current Python consumer implementation at its JSON command boundary. The test language does not own normative meaning. Existing Python regression tests retain their semantic assertions; path/binding fixtures now follow exact source and Root routes.

```powershell
go -C developer/tests/rootfamily test -count=1 -v ./...
```

The `repository-architecture` Test Mode registers the Go module alongside its existing pytest targets. Deterministic mode selection and CI consume the same execution plan. The complete Python corpus must still be evaluated independently; a focused Go pass is not a full repository PASS.

`root-family-transition-bindings.jsonl` preserves the base binding inputs solely as non-authoritative regression evidence. It is never a runtime selection source. Runtime bindings live under `policy-resolver-bindings/`.

## Neutral module creation

`MPD-REAL-0004.rules.neutral_module_creation` owns the project-owner-directed default for new direct repository-root modules and all new modules under `src/`. Their canonical definitions must be language-neutral JSON/YAML machine programs with opaque versioned identity, explicit semantics, exact authority owner, and a registered adapter. Handwritten language-specific new modules and embedded language-specific source wrappers are forbidden in this scope. Existing runtime adapters may interpret admitted neutral modules. Go test selection remains scoped to `developer/tests/rootfamily`; existing Python tests remain preserved. MODIFY bindings expose this conditional creation rule at the root and every registered src entry.

`MPD-CNTR-0003` and `SFP-CNTR-0004` independently own each plane's `registries/root-family-projection.module.json`. Index admission, owner status, schema, class, identity and exact program SHA256 are checked before execution. `PTSIP_JSON_PROGRAM_V1` expresses reconstruction steps, identity/state/kernel predicates and frozen-interface digest validation as neutral data. Its operator and instruction semantics are explicit in `language_contract`. `src/ptsip/governance/authority.py` is the existing registered interpreter adapter; the newly introduced `src/ptsip/policy_projection.py` was removed. The same programs execute through an independent Go interpreter for all 74 source policies. Support installed assets do not import or inherit Developer authority.

`neutral-module-lifecycle.json` records resumed Developer transitions and observed previously applied phases. Support's class-local direct-owner admission is recorded in `src/policy/registries/root-family-projection-module-admission.json`. No source migration policy was promoted by these additional admissions.
