# WU-04I — Final Regression and WU-04 Completion

> **Status:** COMPLETE / EXACT-SHA VERIFIED  
> **Parent work unit:** WU-04 — COMPLETE  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04H — COMPLETE / VERIFIED  
> **Entry baseline:** `713dee236def879e00dadca894add98d65ffb754`  
> **WU-04H completion snapshot:** `9fac22d31333346dbe56a12dee890df1229d560b`  
> **Bound Specification:** `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea`  
> **Regression-remediation snapshot:** `5f7bfdce120650db30d247a686d50a93f91bb3e4`  
> **Exact-SHA verification / WU-04 completion snapshot:** `eadb07f78f9690ca0180bfbba194c5c9602e1838`  
> **GitHub Actions run:** `32588090216`  
> **GitHub Actions job:** `97067421324`  
> **Successor:** WU-05 — repository dogfood / self-evaluation; NEXT / ENTRY REVIEW PENDING

## 0. Purpose

WU-04I is the final verification and closure stage for the complete WU-04 Responsibility Map declaration/materialization/effective-map pipeline.

WU-04A through WU-04H established, in order:

```text
template catalog identity
    -> deterministic materialization
    -> declaration authority / source_mode
    -> ResolvedProfile + digest + provenance
    -> validation consumes effective map
    -> conformance consumes effective map
    -> clarification/adoption consumes effective map
    -> VPMS consumes narrow read-only effective metadata
```

WU-04I did not introduce a new architecture model. It closed the regression debt exposed by the earlier WU-04G full regression, then verified the complete WU-04 pipeline at one exact candidate SHA.

The evidence-driven Tool `0.3.5` migration roadmap remains deferred to Tool `0.3.6.1`; it is not part of WU-04I or the Tool `0.3.6` release gate.

## 1. Entry authority

WU-04H completed at exact implementation verification snapshot:

```text
9fac22d31333346dbe56a12dee890df1229d560b
```

WU-04H completion evidence:

```text
H focused verification                          35 passed / exit=0
VPMS integration + architecture isolation       41 passed / exit=0
repository self-profile                         4 passed / exit=0
```

WU-04I entered from development branch HEAD:

```text
713dee236def879e00dadca894add98d65ffb754
```

Commits after the H verification snapshot were preserved and became part of the I candidate tree.

## 2. Scope

WU-04I owned final WU-04 integration/regression verification for:

- explicit / template / hybrid declaration equivalence where effective architecture is semantically equivalent;
- template revision binding and deterministic materialization;
- hybrid stable-ID override, extension, and removal semantics;
- source/effective view separation and provenance;
- validation and conformance effective-map consumption;
- clarification/adoption effective-map consumption and fail-closed behavior;
- exact selected-profile-path decision/CAS behavior already established by WU-04G;
- VPMS narrow read-only effective-map consumption and package isolation;
- repository self-profile validity under the canonical Tool 0.3.6 model;
- full repository regression review sufficient to classify remaining failures.

## 3. Non-goals

WU-04I did not authorize Tool `0.3.6.1` migration-continuation implementation:

```text
0.3.6.1 WU-01 candidate-discovery evidence expansion
0.3.6.1 WU-02 evidence/provenance normalization
0.3.6.1 WU-03 Tool 0.3.5 legacy reader
0.3.6.1 WU-04 lifecycle migration analyzer
0.3.6.1 WU-05 migration target proposals
0.3.6.1 WU-06 migration preview/confirmation/safe apply
```

Those plans remain under `planning/0.3.6.1.md` and `planning/0.3.6.1/`.

WU-04I also did not enter Tool `0.3.6` WU-05/WU-06/WU-07 implementation early. WU-05 becomes eligible for a fresh entry review only after this closure record.

## 4. Required verification layers

### I1 — WU-04 focused integration

The exact-SHA full repository regression executed every participating WU-04D through WU-04H test family as a strict superset of the focused integration set. No narrowed command was used to hide unrelated failures.

The predecessor H-focused verification remains separately recorded at `9fac22d...`; WU-04I's final authority is the complete exact-SHA regression at `eadb07f...`.

### I2 — declaration-mode equivalence matrix

The complete suite includes the dedicated template/materialization, validation-effective-map, conformance-effective-map, clarification/adoption-effective-map, and VPMS effective-map tests. All passed at the exact candidate SHA.

The required invariant remains:

```text
same effective architecture
    -> same downstream semantic result

source declaration differences
    -> preserved as provenance
    -> not reinterpreted independently by downstream consumers
```

### I3 — fail-closed and mutation boundaries

The complete suite includes stale-state, profile-path, hybrid mutation, invalid legacy input, decision-authority, conformance, and VPMS isolation contracts. All passed.

The workflow additionally ran `ptsip clarify . --json` against the repository and recorded:

```text
status: NO_CLARIFICATION_REQUIRED
repository dirty: false
snapshot comparison: STABLE
```

It then ran `ptsip gate . --coordination local --json` and recorded:

```text
status: NO_DECISION_REQUIRED
backend: LOCAL
repository dirty: false
```

These are direct repository-level checks that the read-only clarification/gate path did not mutate the evaluated repository.

### I4 — repository self-profile

`ptsip validate . --json` passed on the exact SHA with:

```text
valid: true
errors: []
warnings: []
source_mode: explicit
materialized: true
responsibility_map_coverage.unassigned_count: 0
```

The Effective Responsibility Map digest recorded by the run was:

```text
sha256:c009d4fb98daa8a3dc0fd2dfbdd974b467cf107c8c1e7814e460059f4c0c4a01
```

### I5 — full repository regression

The complete repository suite was executed without narrowing:

```text
python -m pytest -q
327 passed in 259.67s (0:04:19)
remaining failures: 0
```

There are therefore no residual failures requiring an outside-WU-04 classification at closure.

## 5. Verification evidence

Final verification authority:

```text
GitHub Actions workflow: tooling-test
run:                     32588090216
job:                     97067421324
candidate exact SHA:     eadb07f78f9690ca0180bfbba194c5c9602e1838
runner:                   self-hosted Windows X64
runner name observed:     DESKTOP-5HCCQIR
Python:                   3.14.6
full pytest:              327 passed / 0 failed
pytest wall time:         259.67s
job conclusion:           success
status context:           self-hosted/tooling-test
status state:             success
```

The workflow explicitly checked out `eadb07f...` and then compared `git rev-parse HEAD` against the dispatched SHA before verification.

Tool / Specification identity also passed:

```text
PTSIP Tool 0.3.6
Specification: 0.3.6-draft
SPEC_REVISION: d6995ed232e845b88d8235b851e80ab54b7804ea
```

Distribution verification passed after the repository regression:

```text
python -m build                              PASS
ptsip-0.3.6-py3-none-any.whl                built
ptsip-0.3.6.tar.gz                           built
twine check wheel                            PASS
twine check sdist                            PASS
force reinstall built wheel                  PASS
ptsip --version from built wheel              PASS
ptsip spec from built wheel                   PASS
ptsip conform --help from built wheel         PASS
VPMS public compatibility smoke               PASS
```

The workflow then recorded the successful exact-SHA commit status:

```text
context:     self-hosted/tooling-test
state:       success
description: Self-hosted Windows verification passed for the exact dispatched SHA
target:      https://github.com/Kinirin/PTSIP/actions/runs/32588090216
```

## 6. Specification / ADR handling

No new `SPEC_REVISION` was created for WU-04I closure.

The regression review found stale test/release-note expectations, not a missing normative architecture rule. The canonical Tool binding therefore remains:

```text
0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
```

No new ADR was required.

## 7. Regression-remediation record

WU-04G's exact-SHA full regression at `3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667` reported:

```text
303 passed / 8 failed
```

WU-04I reviewed all eight failures against the current Tool `0.3.6` architecture rather than treating historical expectations as automatically authoritative.

The remediation snapshot was:

```text
5f7bfdce120650db30d247a686d50a93f91bb3e4
```

No production implementation or canonical architecture rule was changed by this remediation tranche. The changes were bounded to regression fixtures/expectations and the `0.3.6-draft` release-note binding record.

### I-R1 — Specification-binding fixtures

Four failures were caused by fixtures still expecting the earlier WU-04C-bound revision `82abd09360df09a95fbbfb516855fa9ffb49f050` after WU-04G G0 had normatively advanced Tool `0.3.6` to:

```text
d6995ed232e845b88d8235b851e80ab54b7804ea
```

Updated surfaces:

```text
tests/ptsip/test_release_readiness_030.py
tests/ptsip/test_tool_identity_035.py
tests/ptsip/test_tooling.py
releasenote/spec-0.3.6-draft.md
```

### I-R2 — Agent classification conflict fixture

The historical conflict fixture supplied `TOOLCHAIN` as the agent classification. Tool `0.3.6` correctly rejects that token at the current agent-classification schema boundary, so the fixture never reached the declaration-conflict behavior it intended to test.

The fixture now uses canonical `DEVELOPMENT_TOOLING` against a declared `PRODUCT` component, preserving the actual conflict/authority contract.

### I-R3 — Evidence evaluator fixture

The historical evidence test constructed an obsolete `0.3.4-draft` / `TOOLCHAIN` profile and failed current profile validation before reaching the evaluator behavior named by the test.

The fixture was migrated to a canonical `0.3.6-draft` explicit Responsibility Map using `DEVELOPMENT_TOOLING` and current policy keys.

### I-R4 — Unscoped release workflow expectation

The previous test expected an unscoped release-like GitHub workflow to block lifecycle evaluation automatically. Current Tool `0.3.6` treats workflow names/trigger scope as evidence, not lifecycle classification authority.

The migrated test therefore requires the workflow to remain observable and reviewable without turning absence of a positive path scope into architecture authority.

### I-R5 — Legacy topology boundary fixture

The historical topology test expected Tool `0.3.6` topology migration to accept an obsolete `0.3.4-draft` `boundaries.product/toolchain` profile and preserve `TOOLCHAIN` semantics.

That compatibility behavior belongs to the deferred Tool `0.3.6.1` legacy-reader/migration system. Tool `0.3.6` now freezes the correct boundary:

```text
legacy boundary profile
    -> canonical 0.3.6 topology validation rejects it
    -> fail closed
    -> source profile remains unchanged
    -> repository root is not moved
```

## 8. Completion decision

All WU-04I completion-gate conditions are satisfied at exact verification snapshot:

```text
eadb07f78f9690ca0180bfbba194c5c9602e1838
```

Closure decision:

```text
WU-04I COMPLETE / EXACT-SHA VERIFIED
WU-04  COMPLETE
remaining regression failures: 0
next stage: WU-05 ENTRY REVIEW PENDING
```

The commits that update this document, `planning/0.3.6.md`, and `STATUS.md` after the successful run are closure-record documentation commits. They do not replace the exact implementation verification authority `eadb07f...` and do not constitute WU-05 entry.

WU-05 must still begin with a fresh branch-HEAD read and an explicitly recorded entry baseline before it becomes ACTIVE.
