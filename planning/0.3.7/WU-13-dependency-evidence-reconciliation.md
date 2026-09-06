# WU-13 — Dependency Evidence Reconciliation

**Status:** ACTIVE  
**Entry baseline:** `f2fbbb5175eafde0ff8a3b7c5e7ca31a8224189d`  
**Branch:** `dev/0.3.7-wu13-dependency-reconciliation`

## Purpose

Reduce large fail-closed dependency evidence sets into mechanically reconciled evidence before human review, without weakening PTSIP lifecycle rules or manufacturing architecture authority.

WU-13 is a Tool-side evidence/reconciliation improvement. It does not change the bound `0.3.7-draft` normative Specification in its initial tranche.

## Problem demonstrated by turbo-system

The consumer repository exposed several direct Product imports whose runtime distributions were already declared in a Product-owned runtime requirements manifest, but PTSIP still emitted unresolved `dependency-target:*` blockers because native Python declaration discovery only considered repository-root requirement files.

The same audit also exposed import/distribution naming mismatches such as:

- `PIL` -> `Pillow`
- `yaml` -> `PyYAML`

and showed that flat blocker output forces manual package-by-package review.

## Tranche 1 — Product runtime declaration reconciliation

Implement a deterministic reconciliation layer that:

1. inspects requirement manifests already assigned to declared components;
2. recognizes `requirements*.txt/.in` and names such as `runtime-requirements.txt`;
3. treats same-component declarations as external dependency evidence;
4. permits a Product component with `runtime_required: true` to use a Product-owned runtime requirements manifest as external dependency supply evidence;
5. supports only explicit conservative import/distribution aliases;
6. records reconciliation provenance and declaration paths;
7. suppresses only the unresolved dependency coverage gap proven by this declaration evidence;
8. leaves dynamic imports, unresolved relative imports, verification-purpose separation, and ambiguous namespace distributions unresolved for later tranches.

## Non-goals

WU-13 must not:

- infer lifecycle classification or project-owned relationships;
- convert dynamic imports into static architecture;
- treat arbitrary Product requirement files as global architecture authority;
- use the active Python environment as dependency authority;
- silently clear unresolved edges without provenance;
- weaken `PTSIP-DEP-001` or `PTSIP-EVD-003`;
- move the immutable Specification revision unless a later tranche identifies a genuine normative defect.

## Planned follow-up tranches

- component/role-aware verification dependency separation;
- repository-local relative and namespace-package reconciliation;
- deterministic dynamic-import classification summaries;
- actionable buckets (`REPOSITORY_DEFECT`, `REVIEW_REQUIRED`, `EVIDENCE_INCOMPLETE`, `RESOLVER_LIMITATION`);
- concise human summary output while preserving full machine JSON;
- batch/component-level reconciliation reporting.

## Local-agent cost-control bootstrap

WU-13 uses the machine-readable operational policy:

```text
planning/0.3.7/WU-13-local-agent-cost-control.yaml
```

The local coding agent MUST use machine-first evidence handling before broad source reasoning. Until the planned Actionability Classifier and Review Pack surfaces exist, each AI review cycle is bounded to the configured item/context budget, full conformance JSON stays out of the prompt by default, and validation is batched by change/tranche rather than repeated per dependency.

Implementation priority is intentionally reordered so cost-control infrastructure comes before large-scale dependency cleanup:

```text
Actionability Classifier
    -> Review Pack
    -> concise summary
    -> incremental cache
    -> validation-plan generation
    -> remaining dependency reconciliation tranches
```

This policy is operational planning metadata only. It does not change PTSIP lifecycle authority, Specification semantics, or Consumer Repository architecture.

## Verification boundary

Focused tests must prove declaration matching, alias matching, and fail-closed behavior for an undeclared dependency. Full repository regression and exact-SHA self-hosted verification remain required before WU-13 completion.
