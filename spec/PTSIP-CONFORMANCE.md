# PTSIP Conformance

**Version:** 0.2.0-draft

PTSIP conformance is defined so that a human reviewer or automated agent can make a reproducible claim about a project.

## 1. Conformance levels

### 1.1 PTSIP Core Conformant

A project is **PTSIP Core Conformant** when it satisfies all applicable normative MUST/MUST NOT requirements in `PTSIP-SPEC.md`, including:

- SDK classification;
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
- a Specification Binding identifying canonical source and specification version;
- automated dependency-boundary validation;
- product artifact inspection or an equivalent packaging check;
- diagnostics that report PTSIP rule IDs;
- CI or equivalent repeatable validation.

## 2. Required evidence

A conformance claim SHOULD identify:

- PTSIP canonical specification source;
- PTSIP specification version;
- exact immutable specification revision when available;
- project commit or release;
- project profile path or configuration source for Enforced Conformance;
- validation result;
- active exceptions.

Example:

```text
PTSIP Conformance: Enforced
Specification source: https://github.com/kwaksinwoo01/ptsip-spec
Specification version: 0.2.0-draft
Specification revision: <commit-or-release>
Project revision: <commit>
Profile: <project-defined location>
Active exceptions: none
```

## 3. Active violations

A project with an unresolved violation of a mandatory rule MUST NOT claim strict conformance.

A project MAY state `PTSIP-adopting` or `PTSIP-transitioning` while remediation is in progress.

## 4. External validator independence

An external PTSIP validator is architecture-governance tooling and is not part of the Consumer Repository's Product or project-owned Toolchain plane merely because it is installed in a developer virtual environment, user-level tool environment, CI image, or equivalent external environment.

If a project vendors or takes lifecycle ownership of the validator, that copy becomes subject to normal PTSIP classification.

A Product build MUST NOT require a PTSIP validator at runtime.

## 5. Non-intrusion evidence

For external PTSIP tooling, inspection and pilot validation SHOULD demonstrate that the Consumer Repository was not modified unless the user explicitly selected a write-enabled operation.

Tool-owned caches and pilot reports SHOULD be placed outside the Consumer Repository by default.

## 6. False-positive handling

A validator suppression MUST NOT silently disable a rule.

Suppressions SHOULD reference an exception record or a documented false-positive classification.
