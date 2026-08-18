# PTSIP Governance

**Version:** 0.3.6-draft

## 1. Purpose

This document governs changes to the PTSIP Specification itself, immutable draft-snapshot identity, rule/schema evolution, and project remediation/governance records related to PTSIP adoption.

## 2. Canonical source

The public repository `https://github.com/Kinirin/PTSIP` is the canonical source for PTSIP terminology, normative rules, registry IDs, schemas, conformance definitions, normative companion specifications, agent contract, ADRs, and Reference Tool source maintained by this project.

Copies, blog posts, examples, prompts, and tooling snapshots are informative unless explicitly bound to a PTSIP Specification family and immutable revision.

## 3. Versioning

PTSIP uses semantic-version-like Specification versioning for stable releases:

- **MAJOR** — incompatible change to normative meaning or conformance;
- **MINOR** — backward-compatible addition of rules/profiles/capabilities;
- **PATCH** — clarification/editorial/non-semantic repair.

Draft suffixes MAY be used before stable publication.

### 3.1 Draft-family identity

A label such as `0.3.6-draft` identifies a mutable draft family, not a unique immutable snapshot.

Every machine-evaluated use of a mutable draft SHOULD bind an immutable Git revision. For Enforced Conformance and Reference Tool binding, the immutable revision is required.

A Reference Tool implementing a mutable draft MUST identify both:

- draft family; and
- exact immutable revision implemented.

Tool versioning remains independent from Specification family versioning.

### 3.2 Coherent normative migration

A draft-family migration that changes normative text, companion specifications, schema, registry semantics, agent behavior, or embedded Specification resources SHOULD update all affected canonical assets coherently before an immutable normative snapshot is selected.

The immutable snapshot commit does not need to contain a literal self-SHA. A subsequent Tool binding commit MAY point backward to that already-created snapshot. This is the preferred pattern when the Tool stores a literal `SPEC_REVISION` constant.

For `0.3.6-draft`, `spec/PTSIP-RESPONSIBILITY-MAP.md` is part of the normative Specification family and MUST be present at the bound immutable revision together with the other required canonical Specification documents.

## 4. Rule identity

Normative rules use stable IDs such as `PTSIP-DEP-001`, `PTSIP-RMAP-004`, `PTSIP-ADP-001`, and `PTSIP-AUT-001`.

A published rule ID MUST NOT be silently reused for incompatible meaning. Incompatible semantics SHOULD receive a new rule ID or a new major Specification version.

## 5. Change categories

Specification changes SHOULD be classified as one or more of:

- `EDITORIAL`;
- `CLARIFICATION`;
- `NORMATIVE_ADDITION`;
- `NORMATIVE_BREAKING`;
- `SCHEMA_CHANGE`;
- `CONFORMANCE_CHANGE`.

Draft-family evolution still records change category and immutable revision even when the family label remains unchanged.

## 6. Decision records

A normative architecture change SHOULD have an ADR or equivalent record describing problem, alternatives, decision, consequences, compatibility impact, and affected rule IDs.

Implementation evidence MUST be distinguished from normative Specification semantics. A Tool behavior becomes normative only when the bound canonical Specification says so.

## 7. Violations and remediation

PTSIP defines no waiver mechanism authorizing violation of a PTSIP `MUST`/`MUST NOT` rule.

Projects MAY track architectural debt, approval history, ownership, target state, review conditions, or migration work in their own governance systems. Those records MUST NOT change the PTSIP conformance result produced from applicable rules and evidence.

A confirmed mandatory-rule violation remains `NON_CONFORMANT` until remediated and reevaluated.

`PTSIP-EXC-001` is retired; historical exception records remain interpretable only under the immutable snapshot that defined that rule.

## 8. Distributed coordination governance

Distributed Decision Authority is an implementation capability governed by `PTSIP-AUT-*` rules. It does not create a new Consumer Repository architecture classification.

A backend-specific storage format (for example Git refs or a hosted database) is not universal PTSIP semantics unless separately standardized as an interoperability contract.

A distributed backend claiming PTSIP conformance MUST document its coordination domain, stable decision identity, ordered state/conditional mutation model, authority freshness behavior, reconciliation behavior, and fail-closed semantics.

## 9. Tooling relationship

Specification and Tool releases have independent lifecycles.

A PTSIP Tool MUST identify the family/revision it supports and MUST NOT present implementation-only behavior as normative unless grounded in that bound revision.

External Tooling SHOULD preserve Consumer Repository Non-Intrusion and keep Tool-owned state outside the Consumer Repository by default.

Missing adapter, template-materialization, or evidence coverage is a Tool/evidence-coverage condition unless the Specification itself lacks the required adapter-independent semantics.

## 10. Stability policy

The 0.x series is experimental. Terminology, classification ontology, Responsibility Map schema, and migration behavior may change through explicit coherent migrations.

A 1.0 release SHOULD NOT occur until core rule meaning is stable, multiple materially different repositories have exercised the lifecycle-ownership model, Project Profile adoption/migration has been tested, rule-relative conformance has been exercised, Product Artifact checks have been validated, agent behavior has been tested on real change tasks, and distributed/local coordination semantics have enough operational evidence.

## 11. Non-claim policy

Specification changes MUST NOT present PTSIP as an externally standardized industry term unless an external standards body actually adopts it.

Documentation SHOULD distinguish:

- the project-defined PTSIP name/rule set; and
- the pre-existing software-engineering concepts that influenced it.

The repository is licensed under Apache License 2.0.
