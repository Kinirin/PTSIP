# PTSIP Conformance

**Version:** 0.1.0-draft

PTSIP conformance is defined so that a human reviewer or automated agent can make a reproducible claim about a project.

## 1. Conformance levels

### 1.1 PTSIP Core Conformant

A project is **PTSIP Core Conformant** when it satisfies all applicable normative MUST/MUST NOT requirements in `PTSIP-SPEC.md`, including:

- SDK classification;
- Product-to-Toolchain runtime dependency prohibition;
- packaging isolation;
- independently resolvable build environments;
- lifecycle ownership separation;
- explicit exception governance.

### 1.2 PTSIP Enforced Conformant

A project is **PTSIP Enforced Conformant** when it is Core Conformant and additionally provides:

- a machine-readable PTSIP project profile;
- automated dependency-boundary validation;
- product artifact inspection or an equivalent packaging check;
- diagnostics that report PTSIP rule IDs;
- CI or equivalent repeatable validation.

## 2. Required evidence

A conformance claim SHOULD identify:

- PTSIP specification version;
- project commit or release;
- project profile path;
- validation result;
- active exceptions.

Example:

```text
PTSIP Conformance: Enforced
Specification: 0.1.0
Project revision: <commit>
Profile: /ptsip.yaml
Active exceptions: none
```

## 3. Active violations

A project with an unresolved violation of a mandatory rule MUST NOT claim strict conformance.

A project MAY state `PTSIP-adopting` or `PTSIP-transitioning` while remediation is in progress.

## 4. Validator independence

The PTSIP validator belongs to the Toolchain plane. A Product build MUST NOT require the validator at runtime.

## 5. False-positive handling

A validator suppression MUST NOT silently disable a rule.

Suppressions SHOULD reference an exception record or a documented false-positive classification.
