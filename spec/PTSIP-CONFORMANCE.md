# PTSIP Conformance

**Version:** 0.2.0-draft

PTSIP conformance is defined so that a human reviewer or automated agent can make a reproducible claim about a project.

## 1. Conformance levels

### 1.1 PTSIP Core Conformant

A project is **PTSIP Core Conformant** when it satisfies all applicable normative MUST/MUST NOT requirements in `PTSIP-SPEC.md`, including:

- SDK/component classification;
- Product-to-Toolchain runtime dependency prohibition;
- packaging isolation;
- independently resolvable build environments;
- lifecycle ownership separation;
- Consumer Repository Non-Intrusion for external PTSIP tooling;
- explicit exception governance.

A project does not need to add PTSIP-specific documentation or tooling directories merely to claim Core Conformance.

### 1.2 PTSIP Enforced Conformant

A project is **PTSIP Enforced Conformant** when it is Core Conformant and additionally provides:

- a machine-readable PTSIP Project Profile or equivalent declaration;
- a Specification Binding identifying canonical source, draft family/version, and immutable specification revision when the specification is mutable;
- automated dependency-boundary validation;
- product artifact inspection or an equivalent packaging check;
- diagnostics that report PTSIP rule IDs;
- stable evidence tied to one repository snapshot;
- CI or equivalent repeatable validation.

## 2. Required evidence

A conformance claim SHOULD identify:

- PTSIP canonical specification source;
- PTSIP specification version/family;
- exact immutable specification revision when available;
- project commit or release;
- project profile path or configuration source for Enforced Conformance;
- validation result;
- active exceptions;
- evidence snapshot status and unresolved coverage gaps.

Example:

```text
PTSIP Conformance: Enforced
Specification source: https://github.com/kwaksinwoo01/ptsip
Specification version: 0.2.0-draft
Specification revision: <commit-or-release>
Project revision: <commit>
Profile: <project-defined location>
Evidence snapshot: STABLE
Active exceptions: none
```

A report whose repository snapshot changed during collection MUST NOT be used as strict Enforced Conformance evidence without being rerun on a stable snapshot.

## 3. Active violations and unresolved decisions

A project with an unresolved violation of a mandatory rule MUST NOT claim strict conformance.

A component decision status of `UNKNOWN`, `CONFLICT`, or `INCOMPLETE` is not itself a fourth classification. Such unresolved status MAY prevent a strict conformance claim when it affects a boundary relevant to an applicable MUST/MUST NOT rule.

A project MAY state `PTSIP-adopting` or `PTSIP-transitioning` while remediation or ownership resolution is in progress.

## 4. Declaration versus observed evidence

A Project Profile records intended ownership and policy. It does not prove that dependencies, artifacts, or build behavior comply with the declaration.

Automated conformance evidence SHOULD distinguish:

- declared component ownership;
- agent or heuristic candidate decisions;
- observed dependency/build/package evidence; and
- deterministic rule findings.

A contradiction between declaration and observed behavior MUST NOT be hidden by treating the declaration as authoritative proof of compliance.

## 5. External validator independence

An external PTSIP validator is architecture-governance tooling and is not part of the Consumer Repository's Product or project-owned Toolchain plane merely because it is installed in a developer virtual environment, user-level tool environment, CI image, or equivalent external environment.

If a project vendors or takes lifecycle ownership of the validator, that copy becomes subject to normal PTSIP classification.

A Product build MUST NOT require a PTSIP validator at runtime.

## 6. Non-intrusion evidence

For external PTSIP tooling, inspection and pilot validation SHOULD compare observable repository state before and after analysis rather than emit a constant assertion.

At minimum, a tool SHOULD identify which observation methods were used. Examples include repository revision, Git status/index state, tracked-content fingerprints, and untracked/ignored state.

A change observed during analysis does not by itself prove that PTSIP caused the change. The report SHOULD mark the evidence unstable or indeterminate and require rerun or investigation.

Tool-owned caches and pilot reports SHOULD be placed outside the Consumer Repository by default. A user-selected output path inside the repository is an explicit write and SHOULD be reported as such rather than described as non-intrusive.

## 7. Coverage and unresolved evidence

A validator SHOULD report inaccessible files, parser failures, unresolved dynamic dependencies, unsupported languages/adapters, and other evidence gaps.

Unresolved evidence MUST NOT be silently converted into an absence-of-violation claim.

## 8. False-positive handling

A validator suppression MUST NOT silently disable a rule.

Suppressions SHOULD reference an exception record or a documented false-positive classification.
