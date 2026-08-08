# PTSIP Governance

**Version:** 0.2.0-draft

## 1. Purpose

This document governs changes to the PTSIP specification itself and exceptions made by projects adopting PTSIP.

## 2. Canonical source

The public repository `https://github.com/kwaksinwoo01/ptsip-spec` is the canonical source for PTSIP terminology, normative rules, registry IDs, schemas, and conformance definitions.

Copies, blog posts, examples, agent prompts, and tooling snapshots are informative unless explicitly bound to a PTSIP specification version. Tooling SHOULD additionally record an immutable specification revision when practical.

## 3. Versioning

PTSIP uses semantic-version-like specification versioning:

- **MAJOR** — incompatible change to normative meaning or conformance.
- **MINOR** — backward-compatible addition of rules, profiles, or capabilities.
- **PATCH** — clarification, editorial correction, or non-semantic repair.

Draft suffixes MAY be used before stable publication.

## 4. Rule identity

Normative rules have stable IDs such as `PTSIP-DEP-001`.

A published rule ID MUST NOT be silently reused for a different semantic obligation.

If a rule's meaning changes incompatibly, governance SHOULD introduce a new rule or a new major specification version.

## 5. Change categories

Every specification change SHOULD be classified as:

- `EDITORIAL`
- `CLARIFICATION`
- `NORMATIVE_ADDITION`
- `NORMATIVE_BREAKING`
- `SCHEMA_CHANGE`
- `CONFORMANCE_CHANGE`

## 6. Decision records

A normative architecture change SHOULD have an ADR or equivalent decision record describing:

- problem;
- alternatives;
- decision;
- consequences;
- compatibility impact;
- affected rule IDs.

## 7. Project exceptions

Projects MAY record exceptions, but exceptions do not erase the original rule.

An exception MUST identify the exact violated rule and MUST be reviewable.

Permanent exceptions SHOULD be treated as evidence that either:

1. the project is not strictly conformant, or
2. the specification may require a future extension/profile.

## 8. Tooling relationship

The PTSIP specification and an implementation of PTSIP tooling have independent release lifecycles.

A PTSIP tooling implementation MUST identify which specification version it supports and MUST NOT present its own implementation behavior as a normative PTSIP rule unless that behavior is grounded in the canonical specification.

External tooling SHOULD preserve Consumer Repository Non-Intrusion and SHOULD keep tool-owned state outside the Consumer Repository by default.

## 9. Stability policy

The 0.x series is experimental. Terminology and schemas may change.

A 1.0 release SHOULD NOT occur until:

- core rule meanings are stable;
- at least one real repository has adopted the profile;
- conformance checks have been exercised;
- agent instructions have been tested against real change tasks;
- reference tooling or equivalent enforcement has been exercised.

The repository is licensed under Apache License 2.0.

## 10. Non-claim policy

Specification changes MUST NOT present PTSIP as an externally standardized industry term unless an external standards body actually adopts it.

Documentation SHOULD distinguish:

- the project-defined name and rule set;
- the pre-existing software-engineering concepts that influenced it.
