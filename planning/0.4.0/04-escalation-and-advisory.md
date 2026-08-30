# 0.4.0 — Escalation and Advisory Boundary

> **Status:** DRAFT / DESIGN SOURCE  
> **Parent:** `planning/0.4.0.md`

## 1. Escalation is a proof boundary

PTSIP should escalate only after deterministic reduction has exhausted what can be decided from current Specification, current repository facts, and current explicit authority.

Escalation is not the default path and must not be triggered merely because a case is complex.

## 2. Escalation Proof

When owner input is required, the Tool should emit an explicit proof containing at least:

- the exact unresolved semantic dimension;
- surviving semantic candidates;
- eliminated candidates and the reason for each elimination;
- established repository facts;
- applicable normative constraints;
- authority already available;
- the exact missing authority/intent required to continue.

The following are not valid escalation reasons by themselves:

```text
complex
low confidence
AI needed
```

Escalation should demonstrate missing authority rather than model uncertainty.

## 3. Owner-question minimization

Bad interaction:

```text
Which PTSIP rule applies here and what should we do?
```

Desired interaction:

```text
PTSIP establishes facts and constraints,
eliminates illegal/redundant options,
then asks only for the remaining project-intent dimension.
```

This implements the 0.4.0 product objective that developers and coding agents should not need to relearn PTSIP Specification reasoning for supported cases.

## 4. Owner intent versus external fact

Not every unresolved input is architecture intent.

```text
missing project intent
    → OWNER_INTENT_REQUIRED

missing externally knowable fact
    → EXTERNAL_FACT_REQUIRED
```

External facts enter the system as evidence/facts with provenance. They do not become project intent merely because the owner supplied them.

Conversely, when the owner explicitly chooses among remaining architecture-valid semantic targets, that answer may be materialized as project authority through the normal authority path.

## 5. AI role

AI is optional advisory infrastructure only.

AI must not become:

- Specification authority;
- normative rule evaluator authority;
- architecture-intent authority;
- final decision maker;
- a mechanism that silently deletes legal semantic candidates;
- mutation authorization.

Acceptable advisory responsibilities include:

```text
ambiguity explanation
candidate comparison
survivor ranking
owner-question drafting
human-readable rationale
```

## 6. Provider-neutral advisory contract

A provider-neutral boundary was discussed:

```python
class AdvisoryResolver(Protocol):
    def resolve(self, request: AdvisoryRequest) -> AdvisoryResponse: ...
```

Potential providers may include:

```text
human
OpenAI
Claude
Gemini
offline/local advisor
```

The provider list is illustrative, not an implementation commitment.

`AdvisoryResponse` must never automatically become `Authority`.

## 7. Core must work without AI

The deterministic remediation core must remain functional with no AI provider configured.

Core behavior includes:

- evidence collection;
- fact derivation;
- rule evaluation;
- solution-space generation;
- hard-constraint elimination;
- cardinality classification;
- Escalation Proof generation;
- Authority Gate decisions from explicit authority;
- postcondition verification.

AI may improve explanation or ambiguity reduction, but cannot be a hidden dependency for correctness.

## 8. Advisory ranking boundary

Advisory or heuristic systems may rank surviving valid candidates. They cannot erase a candidate unless a separate proof/hard constraint establishes elimination.

```text
legal survivors
    ↓
optional advisory ranking
    ↓
ordered legal survivors
```

Not:

```text
AI preference
    ↓
silent candidate deletion
```

## 9. Natural-language policy/specification ideas

A future-facing flow may be explored:

```text
natural-language intent
    ↓
draft machine-readable policy/specification change
    ↓
validate
    ↓
project-owner approval
    ↓
immutable Specification revision
```

The Tool must never self-authorize its own normative changes. A proposed Specification change is a separate governance action, not an ordinary remediation step.

## 10. Verification expectations

Focused verification should include:

- owner escalation only when semantic authority is missing;
- external fact classified separately from owner intent;
- exact unresolved dimension present in Escalation Proof;
- elimination reasons retained;
- AI/advisory response cannot become Authority automatically;
- advisory ranking cannot remove legal candidates;
- deterministic core works with no advisory provider;
- owner answer can be materialized only through an explicit authority path.
