# WU-04I — Final Regression and WU-04 Completion

> **Status:** ACTIVE  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization + effective-map consumers  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04H — COMPLETE / VERIFIED  
> **Entry baseline:** `713dee236def879e00dadca894add98d65ffb754`  
> **WU-04H completion snapshot:** `9fac22d31333346dbe56a12dee890df1229d560b`  
> **Bound Specification:** `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea`  
> **Successor:** WU-05 — candidate-discovery evidence expansion; locked until WU-04I completion

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

WU-04I enters from development branch HEAD:

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

WU-04I does not authorize implementation of later roadmap work:

```text
WU-05 candidate-discovery evidence expansion
WU-06 evidence/provenance normalization redesign
WU-07 Tool 0.3.5 legacy reader
WU-08 lifecycle migration analyzer
WU-09 migration target proposals
WU-10 migration preview/confirmation/safe apply
WU-11 repository dogfood redesign
WU-12 release/package verification work
WU-13 final Specification freeze/release execution
```

It also does not authorize a broad rewrite of frozen Tool 0.3.5 compatibility tests simply to obtain a green repository-wide suite.

## 4. Required verification layers

### I1 — WU-04 focused integration

Run the focused and participating test families for WU-04D through WU-04H. The exact node/file set may be refined from repository ownership, but it must include the effective-map validation/conformance/clarification/VPMS contracts and architecture-isolation tests.

### I2 — declaration-mode equivalence matrix

Verify representative explicit, template, and hybrid profiles across the downstream consumers that depend on effective architecture.

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

WU-04 completion may be accepted only when remaining failures, if any, are demonstrated to belong to later roadmap or frozen historical compatibility responsibility rather than an unresolved WU-04 contract.

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
    -> WU-05 entry review
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

WU-04I is ACTIVE from the entry baseline above. WU-05 and later planning documents may exist for roadmap visibility, but they do not authorize implementation before this completion gate is satisfied.
