# 0.4.0 — Experimental Directions, Non-goals, and Open Decisions

> **Status:** DRAFT / PLANNING BOUNDARY SOURCE  
> **Parent:** `planning/0.4.0.md`

## 1. Experimental directions

The following ideas were discussed as experimental or future-facing. They must not become hidden dependencies of the 0.4.0 remediation core.

### Provider-neutral AI advisory

AI may assist with ambiguity reduction, explanation, candidate comparison, ranking, and owner-question drafting while remaining optional and non-authoritative.

### Autonomous LLM architect advisor

An autonomous architecture advisor may be explored only as an advisory surface. It cannot own project intent, normative authority, or mutation authorization.

### Natural-language Specification synthesis

Possible future flow:

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

The Tool cannot approve or activate its own normative change.

### Remediation history database

Historical remediation data may later provide optimization hints after Fresh Solve. It must never become current authority or an elimination basis.

### Self-modifying policy

Self-modifying policy is discouraged. PTSIP may propose a Specification/policy change but may not self-authorize it.

## 2. Explicit initial non-goals

The current 0.4.0 plan does not authorize or require:

- operationalizing every PTSIP rule;
- making AI mandatory runtime infrastructure;
- allowing AI to decide project architecture intent;
- expanding `ptsip.yaml` with confidence, provider, remediation-history, or optimization fields merely for Tool convenience;
- creating a giant secondary policy DSL;
- self-authorizing Specification changes;
- using historical remediation choices as authority;
- optimizing smallest textual diff over architecture correctness;
- presenting semantically equivalent physical edit orders as owner architecture choices;
- claiming global uniqueness without a closed remediation-family model;
- treating a capability gap as owner ambiguity;
- treating an external technical fact as owner intent;
- weakening the exact-SHA/full qualification meaning established before 0.4.0.

## 3. Backward-compatibility inheritance

0.4.0 must inherit rather than duplicate or weaken the following 0.3.7 foundations:

```text
Tool Version independence
Project Profile Contract Version independence
Project Profile Instance Revision independence
Typed Specification Binding
Evidence/provenance normalization
Source compatibility boundaries
Exact snapshot validation
Safe apply / recovery behavior
Architecture authority separation
```

Generic remediation should reuse these contracts where appropriate rather than create parallel identity/evidence/authority systems.

## 4. Open architecture decisions

The consolidated planning discussion intentionally left these decisions unresolved:

1. exact initial operational rule families for representative vertical slices;
2. exact package/module boundaries for remediation domain, operational rules, planner, authority gate, and execution planner;
3. exact machine-readable shape of `EscalationProof` and solution-space diagnostics;
4. exact authorization defaults for reversible mechanical changes;
5. whether 0.4.0 implements actual advisory providers or only a provider-neutral boundary;
6. whether remediation history is entirely deferred beyond 0.4.0;
7. exact release criteria for declaring the generic remediation framework sufficiently complete;
8. final WU decomposition and per-WU verification responsibility/Test Mode coverage;
9. whether workflow `scope` is removed in favor of one `mode` selector including `full`;
10. exact sub-branch names after WU responsibility approval.

These must be decided explicitly rather than silently filled in during implementation.

## 5. Planning decision discipline

A future WU may resolve one or more open decisions only when its plan records:

- the alternatives considered;
- the authority for the decision;
- the selected semantic contract;
- compatibility consequences;
- verification expectations;
- whether the decision changes Specification, Project Profile, Tool behavior, or only development implementation.

## 6. Tool capability gap as a safety feature

`TOOL_CAPABILITY_GAP` is not a failure to hide. It is an explicit safety result that prevents partial rule coverage from being mistaken for architecture determinacy.

A capability gap does not automatically mean:

```text
architecture ambiguous
owner must decide
repository unsatisfiable
```

It means only that the current Tool cannot complete the required modeled reasoning/remediation path.

## 7. Automation debt

0.4.0 should reduce repeated manual interpretation by coding agents. When a rule is operationalized, its deterministic reasoning should live in executable Tool behavior rather than informal memory.

At the same time, "automation" must not cross authority boundaries. Automating an unsupported semantic guess creates authority debt rather than reducing automation debt.

## 8. Profile surface stability

Before adding any new `ptsip.yaml` field, the design must answer:

```text
Does the project need to declare new truth/intent/authority?
```

If the answer is no and the information only controls Tool execution or optimization, prefer a Tool-internal contract/registry.

## 9. Criteria for promoting an experimental idea

An experimental direction should move into core 0.4.0 scope only if it can demonstrate:

- no new implicit architecture authority;
- no mandatory dependency on a specific AI/provider;
- deterministic fallback behavior;
- explicit provenance/authority boundary;
- focused verification;
- measurable reduction in user/agent reasoning burden;
- no unnecessary Project Profile surface expansion.

Until those conditions are approved, experimental features remain optional planning items.
