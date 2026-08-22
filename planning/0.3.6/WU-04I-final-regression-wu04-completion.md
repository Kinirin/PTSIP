# WU-04I — Final Regression and WU-04 Completion

> **Status:** ACTIVE — VERIFICATION CANDIDATE PREPARED / EXACT-SHA EXECUTION PENDING  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization + effective-map consumers  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04H — COMPLETE / VERIFIED  
> **Entry baseline:** `713dee236def879e00dadca894add98d65ffb754`  
> **WU-04H completion snapshot:** `9fac22d31333346dbe56a12dee890df1229d560b`  
> **Bound Specification:** `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea`  
> **Regression-remediation snapshot:** `5f7bfdce120650db30d247a686d50a93f91bb3e4`  
> **Exact-SHA verification:** PENDING  
> **Successor:** WU-05 — repository dogfood / self-evaluation; locked until WU-04I completion

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

WU-04I MUST NOT introduce a new architecture model. Its job is to prove that the completed WU-04 pipeline behaves coherently across explicit, template, and hybrid declaration modes and to classify any remaining repository-wide failures before WU-04 is marked complete.

The evidence-driven Tool `0.3.5` migration roadmap has been moved out of Tool `0.3.6` to Tool `0.3.6.1`; it is not part of WU-04I or the Tool `0.3.6` release gate.

## 1. Entry authority

WU-04H completed at exact implementation verification snapshot:

```text
9fac22d31333346dbe56a12dee890df1229d560b
```

Maintainer-provided WU-04H completion evidence:

```text
H focused verification                          35 passed / exit=0
VPMS integration + architecture isolation       41 passed / exit=0
repository self-profile                         4 passed / exit=0
```

WU-04I entered from development branch HEAD:

```text
713dee236def879e00dadca894add98d65ffb754
```

Commits after the H verification snapshot are preserved and form part of the I entry tree; I must not rewrite or discard them merely to reproduce the earlier H exact SHA.

## 2. Scope

WU-04I owns final WU-04 integration/regression verification for:

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

WU-04I does not authorize Tool `0.3.6.1` migration-continuation implementation:

```text
0.3.6.1 WU-01 candidate-discovery evidence expansion
0.3.6.1 WU-02 evidence/provenance normalization
0.3.6.1 WU-03 Tool 0.3.5 legacy reader
0.3.6.1 WU-04 lifecycle migration analyzer
0.3.6.1 WU-05 migration target proposals
0.3.6.1 WU-06 migration preview/confirmation/safe apply
```

Those plans live under `planning/0.3.6.1.md` and `planning/0.3.6.1/`.

WU-04I also does not enter later Tool `0.3.6` release-closure work early:

```text
WU-05 repository dogfood / self-evaluation
WU-06 full regression / package verification
WU-07 final Specification freeze / release preparation
```

It does not authorize a broad rewrite of frozen Tool `0.3.5` compatibility tests simply to obtain a green repository-wide suite.

## 4. Required verification layers

### I1 — WU-04 focused integration

Run the focused and participating test families for WU-04D through WU-04H. The exact node/file set may be refined from repository ownership, but it must include the effective-map validation/conformance/clarification/VPMS contracts and architecture-isolation tests.

### I2 — declaration-mode equivalence matrix

Verify representative explicit, template, and hybrid profiles across downstream consumers that depend on effective architecture.

Required invariants include:

```text
same effective architecture
    -> same downstream semantic result

source declaration differences
    -> preserved as provenance
    -> not reinterpreted independently by downstream consumers
```

### I3 — fail-closed and mutation boundaries

Verify that invalid/unresolved architecture does not produce partial canonical downstream state and that read-only consumers do not mutate profile, authority, or effective-map state.

### I4 — repository self-profile

The repository's own `ptsip.yaml` must validate cleanly under the current Tool 0.3.6 bound Specification and remain free of undeclared tracked-file coverage warnings required by the self-profile contract.

### I5 — full repository regression

Run the complete repository test suite at an exact candidate SHA. Zero failures is preferred, but any remaining failure must be exhaustively classified by responsibility and must not be hidden by narrowing the test command.

WU-04 completion may be accepted only when remaining failures, if any, are demonstrated to belong outside WU-04 responsibility rather than an unresolved WU-04 contract.

## 5. Verification evidence discipline

The final WU-04 completion record must capture:

```text
candidate exact SHA
working-tree cleanliness
Python/runtime identity where material
focused WU-04 result
repository self-profile result
full repository regression result
remaining failure count
classification for every remaining failure
```

A failed full-repository workflow MUST NOT be described as a successful workflow. Stage completion and release readiness are separate claims.

## 6. Specification / ADR handling

Default I decision: no new `SPEC_REVISION` solely for regression closure.

A new immutable Specification revision is required only if I discovers a genuinely missing or incorrect normative PTSIP rule. Test expectation drift or implementation cleanup alone is not sufficient reason to move the Specification.

## 7. Completion gate

WU-04I is complete only when all of the following are reviewed:

- WU-04A through WU-04H completion records remain internally consistent;
- explicit/template/hybrid effective-map behavior is covered end-to-end;
- hybrid override/removal semantics remain deterministic;
- invalid/unresolved source remains fail-closed;
- downstream consumers use the resolved effective architecture rather than independent raw-profile interpretations;
- package/dependency isolation remains intact;
- repository self-profile passes;
- complete repository regression is executed on an exact SHA;
- every remaining failure is either fixed in WU-04I or explicitly classified outside WU-04 responsibility;
- no WU-05 implementation is entered early;
- the final WU-04 completion snapshot and evidence are recorded in this document, `planning/0.3.6.md`, and `STATUS.md`.

After this gate:

```text
WU-04 COMPLETE
    -> fresh branch HEAD
    -> WU-05 repository-dogfood entry review
    -> assign WU-05 exact entry baseline
    -> WU-05 ACTIVE
```

## 8. Initial execution sequence

Recommended order:

```text
1. inventory WU-04 participating tests and known historical failures
2. run WU-04 focused integration set
3. run repository self-profile verification
4. run full repository regression at exact SHA
5. classify/fix only WU-04-owned failures
6. repeat exact-SHA regression as required
7. record completion snapshot and close WU-04
```

## 9. Regression candidate preparation

WU-04G's exact-SHA full regression at `3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667` reported:

```text
303 passed / 8 failed
```

WU-04I reviewed all eight failures against the current Tool `0.3.6` architecture rather than treating the historical expectations as automatically authoritative.

The remediation snapshot is:

```text
5f7bfdce120650db30d247a686d50a93f91bb3e4
```

No production implementation or canonical architecture rule was changed by this remediation tranche. The changes are bounded to regression fixtures/expectations and the `0.3.6-draft` release-note binding record.

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

The release note now records the WU-03 and WU-04C snapshots as earlier checkpoints and records `d6995ed...` as the current bound snapshot selected by the WU-04G G0 `PTSIP-RMAP-017` normative addition.

### I-R2 — Agent classification conflict fixture

The historical conflict fixture supplied `TOOLCHAIN` as the agent classification. Tool `0.3.6` correctly rejects that token at the current agent-classification schema boundary, so the fixture never reached the declaration-conflict behavior it intended to test.

The fixture now uses canonical `DEVELOPMENT_TOOLING` against a declared `PRODUCT` component. This preserves the actual contract under test:

```text
valid conflicting agent review evidence
    -> blocking declaration-conflict evidence
    -> affects_declared_classification = false
    -> Project Profile remains authoritative
```

### I-R3 — Evidence evaluator fixture

The historical evidence test constructed an obsolete `0.3.4-draft` / `TOOLCHAIN` profile and therefore failed current profile validation before reaching the evaluator behavior named by the test.

The fixture was migrated to a canonical `0.3.6-draft` explicit Responsibility Map using `DEVELOPMENT_TOOLING` and the current policy keys. The test still checks its original semantic contract: an evaluator that actually ran with zero findings reports `RAN`, rather than inferring execution state from an empty finding list.

### I-R4 — Unscoped release workflow expectation

The previous test expected an unscoped release-like GitHub workflow to block lifecycle evaluation automatically. That contradicts the current Tool `0.3.6` evidence-authority boundary: workflow names and trigger scope are evidence, not lifecycle classification authority.

The migrated test now requires the workflow to remain observable and reviewable while verifying that lack of a positive path scope alone does not create an architecture failure.

### I-R5 — Legacy topology boundary fixture

The remaining historical topology test expected Tool `0.3.6` topology migration to accept an obsolete `0.3.4-draft` `boundaries.product/toolchain` profile and preserve `TOOLCHAIN` semantics.

That behavior is no longer a Tool `0.3.6` canonical contract. The dedicated Tool `0.3.5` legacy-reader and assisted migration system has moved to Tool `0.3.6.1`.

The Tool `0.3.6` test now freezes the correct current boundary:

```text
legacy boundary profile
    -> canonical 0.3.6 topology validation rejects it
    -> fail closed
    -> source profile remains unchanged
    -> repository root is not moved
```

This does not implement the deferred `0.3.6.1` legacy reader.

## 10. Exact-SHA verification still required

The remediation review above is not execution evidence.

WU-04I remains ACTIVE until a current exact-SHA verification run establishes I1-I5, including the complete repository regression and repository self-profile result. No success result is inferred from the fact that all eight previously observed failures have been remediated in source.

After the exact-SHA run, record its run/job identity, Python/runtime identity, focused/full results, self-profile result, any remaining failure classification, and the final completion snapshot here before changing WU-04I to COMPLETE.

WU-04I is ACTIVE from the entry baseline above. Tool `0.3.6.1` roadmap documents may exist for planning visibility, but they do not authorize migration implementation before Tool `0.3.6` release. Tool `0.3.6` WU-05 and later release-closure stages likewise remain locked until this completion gate is satisfied.
