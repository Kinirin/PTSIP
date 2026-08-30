# 0.4.0 — Solution Space and Remediation Planning

> **Status:** DRAFT / DESIGN SOURCE  
> **Parent:** `planning/0.4.0.md`

## 1. Purpose

The Solution Space Engine is the core mechanism by which PTSIP reduces architecture questions before asking a project owner. It must reason over legal semantic target states, not merely generate a list of plausible edits.

## 2. No confidence-as-authority

The decision flow is:

```text
Evidence
  ↓
Derived facts
  ↓
Legal semantic solution space
  ↓
Hard-constraint elimination
  ↓
Remaining semantic solutions
```

A confidence score cannot substitute for authority. A high-confidence guess cannot remove an otherwise legal semantic option.

Canonical planning statement:

> **판별이 명확할수록 옵션은 적어지고 판별이 불분명하면 옵션은 늘어납니다.**

## 3. Cardinality contract

After all provable elimination and semantic reduction:

```text
0 remaining
→ UNSATISFIABLE

1 remaining
→ DETERMINISTIC_REMEDIATION
  (authorization remains separate)

2+ remaining
→ ARCHITECTURE_CHOICE_REQUIRED / OWNER_INTENT_REQUIRED
```

Cardinality is evaluated only within the Tool's declared modeled remediation families.

## 4. Candidate pipeline

```text
candidate generation
    ↓
normative elimination
    ↓
architecture-preservation checks
    ↓
semantic equivalence / dominance reduction
    ↓
cost / blast-radius ordering among valid survivors
```

Candidate elimination and ranking are different operations.

### Elimination is proof-based

A candidate may be removed only when PTSIP can establish a reason such as:

- violates an applicable normative constraint;
- conflicts with explicit current authority;
- fails a required precondition;
- breaks a required architecture invariant;
- is provably semantically equivalent to another retained candidate;
- is provably dominated under the modeled semantic objective.

### Ranking is non-authoritative

Among remaining valid candidates, PTSIP may order options using factors such as:

- reversibility;
- blast radius;
- future consistency;
- implementation cost;
- advisory heuristics.

Ranking must not silently become elimination.

## 5. Uniqueness must be scoped

PTSIP must not claim global uniqueness unless the remediation-family model for that rule is demonstrably closed and complete.

Preferred language:

```text
unique among modeled remediation families
```

Not:

```text
the only possible architecture
```

If the Tool knows that its remediation-family model is incomplete, the correct outcome is `TOOL_CAPABILITY_GAP`, not overclaimed determinism.

## 6. Semantic target versus physical mutation

Semantic architecture decisions and repository-edit mechanics are separate:

```text
SemanticRemediationPlan
        ↓
RepositoryChangePlan
```

Multiple edit orders or mechanical strategies that produce the same authorized semantic target are not multiple owner choices.

Example:

```text
one semantic target
    ↓
move A then edit profile
or
edit profile then move A
```

If both are semantically equivalent and safe, the Execution Planner chooses among them without escalating a fake architecture decision.

## 7. Planner priority order

Optimization begins only after correctness and authority constraints are satisfied.

```text
1. normative correctness
2. authority preservation
3. architecture intent preservation
4. lifecycle separation
5. future consistency
6. reversibility
7. blast radius
8. implementation cost
9. diff size
```

Key invariant:

```text
minimal diff ≠ best architecture
```

A smaller textual patch must not outrank a more correct lifecycle or authority structure.

## 8. Option reduction objective

PTSIP should reduce questions, not create them. Therefore the planner should:

- eliminate provably invalid options before presentation;
- merge semantically equivalent options where safe;
- avoid exposing physical edit-order alternatives as semantic choices;
- explain why survivors remain distinct;
- ask the owner only about the unresolved semantic dimension.

## 9. Expected planner outputs

A semantic planning result should be inspectable enough to expose:

```text
applicable rule(s)
established facts
modeled remediation families
eliminated candidates + reasons
surviving semantic candidates
cardinality classification
required authority
```

The exact serialization remains a later WU decision.

## 10. Verification expectations

Focused tests should cover:

- no candidate elimination by confidence alone;
- deterministic one-survivor case;
- multiple legal-survivor case;
- zero-survivor unsatisfiable case;
- incomplete family model → capability gap;
- semantic-equivalence reduction;
- ranking without elimination;
- architecture preservation over coarse repository evidence;
- planner priority where smaller diff is rejected in favor of stronger architecture.
