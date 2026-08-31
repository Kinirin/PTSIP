# OP-01 — Experimental and Deferred Directions

> **Status:** DRAFT / OPTIONAL PLANNING  
> **Classification:** `OPTIONAL / NON-BLOCKING` unless an item is explicitly marked `DEFERRED / OUT-OF-RELEASE`  
> **CORE planning:** `planning/0.4.0/`

## 1. Purpose

This document retains 0.4.0-cycle ideas that may improve explanation, usability, experimentation, optimization, or future extensibility but are not required to release the approved generic remediation CORE.

Nothing here may become a hidden dependency of the CORE.

## 2. Optional advisory direction

### Provider-neutral advisory surface

An advisory extension may assist with:

```text
ambiguity explanation
candidate comparison
survivor ranking
owner-question drafting
```

It must remain non-authoritative. An advisory result cannot:

- become Specification authority;
- become Project Profile architecture intent;
- eliminate a legal semantic candidate without a deterministic proof/hard constraint;
- authorize repository mutation;
- bypass Fresh Solve.

A provider-neutral boundary may be reconsidered for CORE only through explicit Promotion Review. Concrete providers remain optional unless separately reclassified.

### Concrete advisory providers

Potential optional adapters may include:

```text
human-assisted adapter
OpenAI
Claude
Gemini
offline / local advisor
```

No specific provider may become required runtime infrastructure for generic remediation.

## 3. Autonomous LLM architecture advisor

**Classification:** `DEFERRED / OUT-OF-RELEASE` by default.

An autonomous architecture advisor may be explored only as an advisory surface. It cannot own project intent, normative authority, candidate legality, or mutation authorization.

## 4. Natural-language Specification synthesis

**Classification:** `DEFERRED / OUT-OF-RELEASE`.

Possible future flow:

```text
natural-language intent
    ↓
draft machine-readable policy / Specification change
    ↓
validate
    ↓
project-owner approval
    ↓
immutable Specification revision
```

The Tool cannot approve or activate its own normative change.

## 5. Remediation history and optimization

Persistent remediation-history storage is not required for 0.4.0 CORE.

If explored later, history may be used only after a Fresh Solve as a non-authoritative optimization hint. Historical outcomes must never become current authority or a reason to remove a currently legal candidate.

A persistent remediation-history database is therefore `DEFERRED / OUT-OF-RELEASE` unless separately approved.

## 6. Advanced ranking and UX

Potential optional improvements include:

- richer cost/blast-radius ranking among already legal survivors;
- enhanced remediation explanation and visualization;
- advanced rollback/recovery UX beyond the CORE safety contract;
- convenience commands for reviewing modeled candidate spaces;
- additional representative or production rule families beyond minimum release proof.

These may improve the Tool but do not change the meaning of semantic legality or authority.

## 7. Workflow convenience cleanup

Simplifying PTSIP's own development workflow input surface, such as replacing separate `scope + mode` controls with one selector containing `full`, may be explored independently.

This is not required for the generic remediation CORE and must not weaken:

```text
Selective Test Mode success
    ≠ full repository qualification
```

or the meaning of exact-SHA `self-hosted/tooling-test` qualification.

Productizing reusable PTSIP-backed Test Mode orchestration for Consumer Repositories is tracked separately in:

```text
OP-02-ptsip-backed-test-mode-productization.md
```

## 8. Self-modifying policy

**Classification:** `DEFERRED / OUT-OF-RELEASE` and discouraged.

PTSIP may propose a policy/Specification change but cannot approve or activate that normative change by itself.

## 9. Universal rule operationalization

Operationalizing every PTSIP rule in 0.4.0 is explicitly out of scope.

The CORE proves the framework using representative complete rule families. Additional rule families may be optional extensions; universal coverage is deferred.

## 10. Promotion criteria

An optional idea may move into CORE only if it can demonstrate:

- no new implicit architecture authority;
- deterministic fallback behavior;
- explicit provenance and authority boundaries;
- focused verification;
- no unnecessary Project Profile surface expansion;
- a concrete approved 0.4.0 objective/invariant/release proof that fails if the work is deferred;
- project-owner approval through the Pre-WU-00A Promotion Review process.

Until then it stays outside the release dependency graph.
