# 0.4.0 — Verification and Test Mode Strategy

> **Status:** DRAFT / DEVELOPMENT INFRASTRUCTURE SOURCE  
> **Parent:** `planning/0.4.0.md`

## 1. Inherited verification architecture

0.4.0 inherits the verification-responsibility normalization and Test Mode control plane completed before the branch opened.

The governing separation is:

```text
Verification Responsibility
    ≠ Physical Test Location
    ≠ CI Execution Mode
```

Responsibilities remain:

```text
ptsip.yaml
    = verification responsibility / architecture authority

tests/**
    = physical test implementation

.github/test_modes.yaml
    = execution declaration

resolve_test_modes.py
    = deterministic selection mechanism

tooling-test.yml
    = execution mechanism
```

A physical test path must never become architecture authority by inference.

## 2. Declared responsibility to execution

The supported direction is:

```text
Declared verification responsibility
        ↓
component_ref
        ↓
Test Mode
        ↓
pytest target / execution settings
```

Watch paths are execution-selection evidence only.

## 3. Initial Test Modes at branch baseline

```text
ptsip-migration
    → ptsip-migration-verification
    → tests/ptsip/migration

ptsip-evidence
    → ptsip-evidence-verification
    → tests/ptsip/evidence

ptsip-source-compat
    → ptsip-source-compat-verification
    → tests/ptsip/source_compat

vpms
    → vpms-verification
    → tests/vpms
```

Broad/core modes should be added only when a durable verification responsibility justifies them, not merely because a folder exists.

## 4. Resolver contract

The existing selection resolver supports:

- path normalization;
- watch-glob matching;
- automatic changed-file selection;
- dependency-sensitive multi-mode selection;
- manual exact/all selection;
- control-plane changes selecting all registered modes;
- fail-closed unknown manual modes;
- actual Git diff input;
- execution plans containing only `id`, `component_ref`, and pytest targets.

The resolver must not copy classification, roles, or purpose from Project Profile into the execution registry as a second architecture authority.

## 5. Selective versus full qualification

The invariant is:

```text
Selective Test Mode success
    ≠ full repository qualification

Full qualification success
    = complete repository regression
      + release contract checks
      + package build/distribution checks
      + artifact verification
      + exact-SHA self-hosted/tooling-test status
```

The meaning of the existing exact-SHA success status must not be weakened by selective tests.

## 6. 0.4.0 iteration strategy

For ordinary WU development:

```text
focused unit/contract tests
        ↓
relevant Test Mode(s)
        ↓
WU/integration checkpoint
        ↓
full exact-SHA qualification only at meaningful boundaries
```

The objective is to reduce feedback time and GitHub Actions usage while preserving strong release/checkpoint verification.

## 7. New remediation tests

As 0.4.0 capabilities are added, tests should be grouped by durable verification responsibility rather than implementation history or version number.

Potential future responsibilities include, subject to WU approval:

- remediation-domain contract verification;
- operational-rule verification;
- solution-space/planner verification;
- authority-gate/safe-apply verification;
- remediation integration verification.

This document does not authorize those new Test Modes yet. The correct sequence is:

```text
new durable verification responsibility
    ↓
Project Profile declaration
    ↓
component_ref
    ↓
Test Mode registration if selective execution is useful
```

## 8. Test organization principles

- do not derive Test Modes from directories;
- do not create a new test folder for one isolated file unless the responsibility is durable;
- integration is a valid responsibility when it verifies a real cross-boundary contract;
- version suffixes remain only when version compatibility itself is under test;
- shared helpers must remain neutral support and not masquerade as verification authority;
- full release qualification remains broader than selective mode coverage.

## 9. Workflow simplification note

The current pre-0.4.0 workflow introduced separate `scope` and `mode` inputs as an initial vertical slice. The planning discussion subsequently identified a simpler likely interface:

```text
mode:
  all
  ptsip-migration
  ptsip-evidence
  ptsip-source-compat
  vpms
  full
```

where `full` alone performs exact-SHA full qualification.

This simplification remains a workflow-maintenance decision and should not be confused with 0.4.0 remediation architecture authority.

## 10. Verification expectations for remediation features

Each operationalized remediation family should cover, as applicable:

- applicable / non-applicable repository states;
- deterministic fact derivation;
- violation and conformant outcomes;
- candidate generation;
- elimination reasons;
- survivor cardinality;
- owner-intent / external-fact distinction;
- Authority Gate result;
- stale-snapshot rejection;
- Repository Change Plan generation;
- guarded apply/refusal;
- postcondition verification;
- no evidence-to-authority conversion;
- no AI/heuristic elimination of legal candidates.
