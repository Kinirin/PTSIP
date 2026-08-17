# ADR-0006 — Establish VPMS Verification Purpose Management

**Status:** Accepted  
**Decision:** Establish `VPMS — Verification Purpose Management System` as an independent verification-purpose management subsystem for Tool `0.3.5`  
**Design lineage:** PTSIP purpose-first architecture ownership + Tool `0.3.5` VPMS introduction plan

## Context

PTSIP determines the architectural responsibility of repository components. It answers the question:

```text
What is this component?
```

Long-lived repositories also need a separate answer for verification work. Product behavior tests, Toolchain verification, migration checks, repository automation checks, compatibility checks, and policy assertions can accumulate under one broad test surface even though they protect different responsibilities.

Treating test location, framework, implementation ownership, or reuse alone as the verification purpose creates ambiguity. A verification implementation may itself be development-time tooling and therefore commonly be classified by PTSIP as `TOOLCHAIN`, while the correctness obligation it protects may belong to Product behavior.

Tool `0.3.5` therefore needs an explicit verification-purpose boundary without changing the established PTSIP architecture classifications or making verification execution equivalent to PTSIP conformance.

## Decision

PTSIP establishes the name:

```text
VPMS
Verification Purpose Management System
```

The former working name `TVFM (Test Verification Function Management)` is retired before implementation.

VPMS governs the purpose, composition, selection, and execution of verification work. Its defining question is:

```text
Why does this verification exist?
```

The responsibility split is:

```text
PTSIP
    What is this component?

VPMS
    Why does this verification exist?
```

The systems share a purpose-first principle but govern different semantic responsibilities:

```text
PTSIP
    Purpose precedes reuse.

VPMS
    Verification purpose precedes reuse and execution.
```

VPMS is not a fourth PTSIP Plane or architecture classification. Tool `0.3.5` initially defines exactly two VPMS verification purposes:

```text
PRODUCT
TOOLCHAIN
```

These values describe the responsibility whose correctness the verification protects; they do not reclassify the verification implementation itself.

For example:

```text
verification implementation
    PTSIP classification = TOOLCHAIN

verification purpose
    VPMS purpose = PRODUCT

meaning
    a Toolchain-owned verifier protects Product behavior
```

A second verifier may remain PTSIP `TOOLCHAIN` while also having VPMS purpose `TOOLCHAIN` because it protects development tooling correctness.

## Verification-purpose rule

VPMS purpose is determined by the responsibility whose correctness would be lost if the verification were absent or failing.

Purpose classification should therefore answer questions such as:

```text
Why was this verification created?
What changed that requires this verification?
If this verification disappears, whose correctness can no longer be established?
Does failure indicate a Product behavior defect or a Toolchain/development-process defect?
```

Directory names, compilation boundaries, package inclusion, or test frameworks may provide evidence, but none is sufficient as the sole authority for verification purpose.

## Verification Case ownership

A `Verification Case`, not a source test file, is the smallest purpose-bound VPMS execution unit.

A case binds the independent semantic units required to execute verification while preserving purpose:

```text
Verification Purpose
Target
Formula
Variables
Policy
Runner
```

Their responsibilities remain distinct:

```text
Formula   = how to verify
Variables = mutable verification data
Policy    = what outcome or contract is intentionally governed
Case      = how purpose + target + formula + variables + policy + runner are bound
Runner    = how the selected case is executed
```

This distinction allows multiple cases to reuse implementation without merging their verification obligations.

## Reuse rule

VPMS exists to permit reuse without erasing purpose.

The default cross-purpose reuse boundary is the `Formula` layer. A Formula is a candidate for reuse across Product-purpose and Toolchain-purpose cases when it expresses a purpose-neutral invariant that remains meaningful without knowing which purpose consumes it.

A reusable Formula should not require `PRODUCT` or `TOOLCHAIN` identity, hard-code a specific project component merely to establish purpose, embed mutable expected values, or silently contain purpose-owned Policy.

The governing decision test is:

> Would this verification logic still make sense if all Product and Toolchain names were removed from the repository context?

If yes, the logic may belong in a Formula. If no, it belongs in purpose-owned Variables, Policy, Case configuration, or purpose-specific execution logic.

Cross-purpose Formula reuse is permitted. Cross-purpose Policy reuse is not assumed by default because Policy carries governed intent. Shared implementation MUST NOT collapse two distinct verification obligations into one implicit purpose.

## Source ownership

VPMS is implemented as a sibling subsystem, not as a nested PTSIP package.

The intended source ownership is:

```text
src/
├─ ptsip/
│  └─ PTSIP implementation
│
└─ vpms/
   └─ VPMS implementation
```

This separation records that PTSIP architecture ownership and VPMS verification-purpose management are independent responsibilities even when VPMS consumes stable PTSIP-owned target metadata.

Consumer Repositories are not required to mirror this implementation topology for their own verification files. Repository-layout neutrality remains intact.

## Dependency direction

PTSIP core MUST NOT depend on VPMS.

The prohibited architecture is:

```text
PTSIP <----> VPMS
```

The permitted relationship is one-way consumption of an explicit, stable PTSIP-owned data contract by VPMS when target metadata is required:

```text
PTSIP
  |
  | stable project/component data contract
  v
VPMS
```

This arrow represents data ownership and permitted dependency direction, not a requirement that PTSIP invoke VPMS.

PTSIP classification, conformance, adoption, gate, resolve, and authority behavior must remain usable when VPMS is absent or disabled. VPMS should not import unstable PTSIP internals merely for convenience, and it must not obtain a write path into PTSIP authority or project-profile state as part of ordinary verification execution.

## Conformance boundary

VPMS execution results and PTSIP conformance outcomes are separate claims.

```text
VPMS PRODUCT verification PASS
    !=
PTSIP CONFORMANT
```

and:

```text
PTSIP CONFORMANT
    !=
Product functional verification PASS
```

VPMS may later provide verification evidence to an explicit integration boundary, but ordinary VPMS case success or failure does not create, change, or replace a PTSIP architecture classification or conformance decision.

## Consequences

1. Tool `0.3.5` can add verification-purpose management without expanding the PTSIP classification set.
2. Product-purpose and Toolchain-purpose verification can remain independently selectable even when they reuse the same Formula implementation.
3. Verification implementation ownership and verification purpose remain separate axes.
4. VPMS can evolve under `src/vpms/` without requiring PTSIP core to depend on it.
5. A stable read-only PTSIP metadata boundary may be introduced later when concrete VPMS target metadata requires it.
6. Runner implementation remains framework-neutral; VPMS is not defined as a pytest-specific manager.
7. Consumer Repository test directory structure remains optional organizational guidance rather than a conformance requirement.
8. Later adoption, agent-contract, packaging, CLI, and specification work must reflect implemented VPMS behavior rather than reinterpret this ADR as already-delivered runtime functionality.

## Rejected alternatives

### Add VPMS as a fourth PTSIP Plane

Rejected. Verification purpose and component architecture classification answer different questions. Conflating them would make verification metadata alter PTSIP architecture meaning.

### Infer verification purpose from the test directory

Rejected. Repository paths are organizational evidence, not sufficient purpose authority, and PTSIP/VPMS must remain usable across heterogeneous repository layouts.

### Treat the verification implementation classification as its purpose

Rejected. A PTSIP `TOOLCHAIN` verifier can protect either Product or Toolchain correctness; implementation ownership does not determine the protected responsibility.

### Make reuse the primary ownership rule

Rejected. Shared implementation does not imply shared purpose. Formula reuse is allowed only while purpose, Variables, Policy, and Case identity remain independently governed.

### Nest VPMS under `ptsip`

Rejected. VPMS and PTSIP own different semantic boundaries. A nested ownership model would unnecessarily enlarge PTSIP core coupling and the source-level conflict surface.

### Let PTSIP core depend on VPMS

Rejected. PTSIP must remain independently usable and must not acquire a runtime dependency on verification-purpose management.

### Treat VPMS PASS as PTSIP conformance

Rejected. Functional or tooling verification success and architectural conformance are distinct claims and require distinct evaluation semantics.
