# 0.4.0 — Operational Rule Registry

> **Status:** DRAFT / DESIGN SOURCE  
> **Parent:** `planning/0.4.0.md`

## 1. Purpose

0.4.0 needs an executable bridge from normative Specification rules to supported remediation behavior without turning `ptsip.yaml` into a Tool-internal automation configuration file.

The intended separation is:

```text
Specification
    = normative correctness

Operational Rule Registry
    = Tool-internal executable support for selected rules

ptsip.yaml
    = project truth / intent / analysis contract
```

The registry operationalizes rules; it does not supersede Specification authority.

## 2. Rule identity

`rule_id` is the stable bridge across the remediation pipeline:

```text
Specification rule
    ↓
Operational Rule Registry
    ↓
Rule Evaluator
    ↓
Solution Space Engine
    ↓
Remediation Planner
    ↓
Postcondition Verifier
```

No internal registry metadata may redefine what the Specification means.

## 3. Candidate operational contract

Conceptually, a supported rule needs to answer four independent questions:

```text
Does this rule apply?
What is the current rule outcome?
What semantic remediation families can this Tool model?
How is a resulting target state verified?
```

Candidate protocol:

```python
class OperationalRule(Protocol):
    def applies(self, context): ...
    def evaluate(self, context): ...
    def propose_solutions(self, context): ...
    def verify(self, context, target): ...
```

The implementation may later split these responsibilities into narrower protocols if that produces stronger typing and focused testing.

## 4. Registry metadata

Potential internal metadata includes:

```text
rule_id
authority_requirement
automation_level
supported_remediation_families
verification_capability
```

The Tool should avoid a giant secondary policy DSL. Metadata exists to describe executable capability, not to create new project policy.

## 5. Remediation-family declaration

Each operationalized rule should declare which semantic remediation families the Tool knows how to model.

Conceptually:

```text
Rule
  ├─ detect applicability
  ├─ derive violation facts
  ├─ enumerate modeled semantic remediation families
  ├─ eliminate illegal families
  ├─ reduce equivalent/dominated families
  ├─ classify remaining cardinality
  └─ verify postconditions
```

This declaration is required for honest determinacy reporting. If a relevant legal family may exist outside the implemented model, PTSIP must expose `TOOL_CAPABILITY_GAP` rather than claiming global uniqueness.

## 6. Project Profile surface stability

New Tool remediation capabilities do not automatically justify new `ptsip.yaml` fields.

A new Project Profile field is justified only when the project itself must declare new architecture authority, intent, or analysis-contract information.

The following should generally remain Tool-internal unless a later approved design proves otherwise:

- remediation preference heuristics;
- confidence scores;
- AI provider selection/state;
- remediation history;
- optimization hints;
- Tool capability metadata;
- registry implementation details.

## 7. Capability matrix

A rule-support matrix may expose capabilities such as:

```text
detect / plan / apply / verify

detect / plan / owner-intent / verify

detect-only / capability-gap
```

This matrix reports Tool support. It is not a normative policy source.

## 8. Versioning and compatibility expectations

Operational-rule implementation must remain compatible with the independent identities established in 0.3.7:

```text
Tool Version
Project Profile Contract Version
Project Profile Instance Revision
Typed Specification Binding
```

The registry must not infer Specification support merely from Tool or PP version strings. It should consume the explicit Specification binding/capability contracts already established by 0.3.7.

## 9. Rule-evaluation boundary

A rule implementation may:

- read normalized evidence/facts;
- determine applicability from normative conditions;
- produce deterministic rule outcomes;
- enumerate modeled semantic families;
- define postconditions.

It may not:

- invent project lifecycle intent;
- convert observations into authority;
- weaken Specification constraints to produce a candidate;
- use an AI answer as rule authority;
- treat historical remediation choices as current normative constraints.

## 10. Verification expectations

Each operational rule should have focused tests for, as applicable:

- applicable / non-applicable states;
- conformant / non-conformant evaluation;
- expected derived facts;
- declared remediation families;
- unsupported-family capability gap;
- explicit authority requirements;
- verification of accepted target states;
- no hidden `ptsip.yaml` authority growth;
- no Tool-version-to-Specification inference.
