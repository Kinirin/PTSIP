# PTSIP 0.2.0-draft — Next Normative Snapshot Plan

> **Status:** Planning / non-normative  
> **Specification family:** `0.2.0-draft`  
> **Current Tool 0.2.2 normative baseline:** `895e12d27230af2bb99ad17a96e8df8ef41bc3e0`  
> **Next normative snapshot revision:** To be assigned by the merge commit that changes the normative specification assets  
> **Purpose:** Define the migration sequence and decision boundaries before changing normative PTSIP semantics.

## 1. Why this plan exists

PTSIP has now been exercised against more than one materially different Consumer Repository. The next specification change should therefore be driven by repeated evidence rather than by adding implementation features opportunistically.

This plan uses two Consumer Pilot evidence sources:

1. **turbo-system Pilot** — a large mixed-language repository with Python, .NET, GitHub Actions, multiple product artifacts, release workflows, dynamic dependencies, nested Product/Toolchain ownership, and lifecycle coupling pressure.
   - Pilot repository revision reported by the second Pilot: `2fac40340d75e28ee3c432ab448ee55fd052f02e`.
   - Tool: `0.2.0`.
   - Specification family: `0.2.0-draft`.
   - Bound specification revision: `895e12d27230af2bb99ad17a96e8df8ef41bc3e0`.
   - Result: evidence collection succeeded; strict conformance remained undetermined because coverage, artifact, phase, and ownership evidence were incomplete.

2. **Simple Connection Pilot feedback** — a JavaScript/TypeScript-, Node.js-, Electron-, SDK-, and Toolchain-oriented repository with explicit component-to-component dependency policy and repository-local governance tooling.
   - Repository: `kwaksinwoo01/Simple-Connection`.
   - The supplied Pilot feedback did not bind the analysis to an exact repository revision; this plan therefore records the feedback as a design input rather than pretending the current repository HEAD is the Pilot snapshot.
   - Important repository-specific governance such as `AGENTS.md`, `MEMORY.md`, `document_hub.md`, `General_ledger.json`, and the repository's own Governance Validator remain outside PTSIP ownership.

The two Pilots exercise different ecosystems but converge on several common PTSIP needs: explicit evidence coverage, artifact semantics, separation of declaration validation from conformance evaluation, stable diagnostics, and stronger dependency evidence semantics.

## 2. Version and identity policy

The next specification work **retains the family label `0.2.0-draft`**.

The draft family label is not the immutable normative identity. The exact merge commit containing the next complete normative change set becomes the new normative snapshot identity.

The Reference Tool remains independently versioned.

The normative snapshot migration MUST NOT silently change the meaning of current Tool `0.2.2`. Tool `0.2.2` remains bound to:

```text
Specification family: 0.2.0-draft
Specification revision: 895e12d27230af2bb99ad17a96e8df8ef41bc3e0
```

A later Tool release MUST explicitly bind whichever newer normative revision it implements.

## 3. Non-negotiable model constraints

The next snapshot MUST preserve the following architectural constraints unless a separate breaking decision explicitly changes them:

1. PTSIP has exactly three architectural classifications:
   - `PRODUCT`
   - `TOOLCHAIN`
   - `NEUTRAL_CONTRACT`
2. `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` are decision/evaluation states, not additional planes.
3. External, platform, standard-library, and unresolved dependency targets MUST NOT be turned into additional PTSIP architecture classifications. They belong to evidence-graph scope/type semantics.
4. A Project Profile is a declaration of intended architecture; it is not conformance truth.
5. Observed evidence MUST NOT be overwritten by declarations, agent output, transition metadata, or imported external evidence.
6. Transition/migration metadata and conformance state are distinct concepts; PTSIP does not define a mandatory-rule waiver that changes conformance.
7. External validator output is analysis input with provenance; it is not automatically trusted PTSIP conformance fact.
8. Project-specific policy MAY be stricter than the universal PTSIP minimum, but repository-specific governance MUST NOT become mandatory PTSIP core semantics.
9. Purpose and lifecycle ownership continue to precede code reuse.

## 4. Pilot findings accepted as normative design inputs

### 4.1 Evidence completeness and strict-claim gating — P0

Repeated Pilot evidence shows that absence of detected violations cannot imply conformance when applicable evidence is missing.

The next snapshot must define when an evidence gap is:

- **blocking** — the gap can hide the presence or absence of an applicable `MUST`/`MUST NOT` violation;
- **non-blocking** — the unsupported evidence cannot materially affect the evaluated normative claim; or
- **out of scope** — the subject is not governed by the evaluated PTSIP claim.

The design MUST NOT use a universal rule such as `unresolved_count > 0 => fail`.

The intended principle is:

```text
Evidence gap
    ↓
Can this gap conceal the result of an applicable mandatory rule?
    ├─ yes → strict claim blocked / INCOMPLETE
    └─ no  → reportable, non-blocking gap
```

### 4.2 Product Artifact model — P0

Both Pilots require a model that separates the producer of an artifact from the architectural owner of the produced artifact.

The next snapshot should define at least:

- artifact identity;
- architectural owner;
- producer component;
- format/type;
- contained components or content evidence;
- derivation/generation relationship;
- shipping/distribution scope;
- evidence source and confidence/provenance.

The specification must support relationships such as:

```text
TOOLCHAIN build component
        └─ GENERATES / PACKAGES → PRODUCT artifact
```

without reclassifying the Toolchain producer as Product or the Product artifact as Toolchain.

### 4.3 Mandatory violations and remediation semantics — P0

Pilot review initially exposed inconsistency between exception text and machine-readable schema. The subsequent governance decision is simpler: PTSIP does not define a waiver path for mandatory-rule violations.

The next snapshot must establish:

- sufficient evidence of a `MUST`/`MUST NOT` violation -> `NON_CONFORMANT`;
- project governance records do not alter that result;
- remediation/migration changes the architecture, after which evaluation is rerun;
- historical `PTSIP-EXC-001` remains interpretable only under earlier immutable revisions and is retired/superseded in the new snapshot.

### 4.4 Mandatory component split criterion — P1

PTSIP already defines components and deterministic selector precedence. The remaining normative gap is not the absence of a component model; it is the absence of objective guidance for when a broad component MUST be split because it no longer represents coherent ownership.

The next snapshot should evaluate candidate split triggers including differences in:

- shipped state;
- executable purpose;
- release owner;
- compatibility owner;
- manifest/build ownership;
- product/toolchain lifecycle responsibility.

The design must avoid requiring one physical directory per component.

### 4.5 Typed edge, lifecycle phase, and provenance semantics — P1

Dependency evidence needs adapter-independent meaning before more adapters are added.

The next snapshot should define:

- normative meanings of edge types such as `IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, and `PUBLISHES`;
- whether multiple edge types may validly describe one observed relationship;
- lifecycle phase semantics such as `RUNTIME`, `BUILD`, `TEST`, `RELEASE`, and `INSPECTION`;
- multi-phase edges;
- resolution status;
- provenance such as `DECLARED`, `OBSERVED`, and `INFERRED`;
- uncertainty/confidence behavior without converting guesses into conformance facts.

### 4.6 Profile validation versus conformance evaluation — P1

The next snapshot should formally separate:

```text
Profile Validation
    = Is the declaration structurally and semantically valid?

Conformance Evaluation
    = Does observed evidence satisfy the applicable PTSIP rules and declaration?
```

The intended CLI architecture for a later Tool implementation is conceptually:

```text
ptsip validate .
ptsip conform .
```

The specification design should define conformance outcomes such as:

- `CONFORMANT`
- `NON_CONFORMANT`
- `INCOMPLETE`

A `CONFORMANT` result requires sufficient applicable evidence coverage in addition to no observed violation.

### 4.7 Stable diagnostic/finding contract — P1

Conformance and evidence results need a stable machine-readable diagnostic contract usable by CI systems, developer tooling, automation agents, and external reporters.

The next snapshot should define the semantics needed for a versioned diagnostic format, including at least:

- unique finding/diagnostic instance ID;
- stable PTSIP rule ID;
- outcome/status;
- severity;
- source and target component when applicable;
- evidence references;
- message;
- evaluator/provenance metadata.

A rule ID and a diagnostic instance ID must remain distinct because one rule may produce many findings.

### 4.8 Optional component-to-component dependency policy — P1 / profile extension

Simple Connection demonstrates a valid need for stricter project-local dependency topology inside one PTSIP plane, for example a Toolchain default-deny policy with an allowlist of permitted SDK edges.

PTSIP should be able to **express and validate** such stricter policy without making the policy a universal PTSIP architectural rule.

Intended layering:

```text
PTSIP universal normative minimum
        +
optional project-specific component dependency constraints
```

Initial profile semantics should remain small. Edge type and phase constraints may be added only when the underlying edge/phase model is normative and stable enough.

## 5. Accepted Tool implementation inputs that are not normative changes by themselves

The following are valid Tool roadmap items but MUST NOT be mistaken for new PTSIP architectural obligations merely because a Pilot requested them:

### Tool 0.2.3 correctness candidates

- Python BOM-aware source decoding;
- deterministic relative-import resolution where repository evidence permits it;
- distinction among project component, external dependency, platform/stdlib, and unresolved targets;
- GitHub Actions working-directory and false-positive handling;
- explicit evaluator-run/evaluator-blocked state instead of interpreting an empty finding list as zero violations.

### Tool 0.3.0 capability candidates

- JavaScript/TypeScript source dependency adapter;
- npm manifest adapter;
- agent decision ingestion with provenance;
- optional component dependency policy evaluation;
- artifact evidence adapter framework;
- stable diagnostic contract implementation;
- evidence-coverage gate and `conform` command;
- external evidence envelope/import;
- extended .NET adapter;
- Go adapter.

Tool implementation follows the normative model. Tool behavior does not define normative semantics by itself.

## 6. Deferred or deliberately non-core proposals

### 6.1 Component role

A `role` or `roles` field may be useful as project-defined metadata such as `validation`, `test`, `schema`, `migration`, `build`, or `operation`.

For the next snapshot, role SHOULD remain contextual metadata rather than a closed normative PTSIP classification taxonomy.

### 6.2 Transitional / migration architecture

Migration metadata is useful for describing `current` and `target` architecture, but it MUST NOT rewrite current observed conformance.

The design must preserve:

```text
Current architecture ≠ Target architecture ≠ Conformance
```

Transition metadata belongs primarily to adoption/migration planning unless a later ADR establishes normative semantics.

### 6.3 External evidence import

External evidence is a promising capability, but any future format must carry provenance such as producer, producer version, repository/revision subject, generation time, evidence type, scope, claims, and integrity information.

Imported evidence is input to PTSIP evaluation; it does not automatically override native observed evidence or determine PTSIP conformance.

### 6.4 Profile composition / organization baseline

Profile composition is deferred until immutable binding and deterministic composition semantics are designed.

Any future `extends` mechanism needs at least:

- immutable baseline identity;
- merge/override semantics;
- conflict behavior;
- resolution order;
- cycle detection;
- reproducibility guarantees.

Mutable unpinned organizational baselines must not make historical conformance results change retroactively.

## 7. Proposed normative design work package

The first implementation work after this planning document is a **single coherent normative snapshot design**, not Tool feature implementation.

The design PR should update all affected normative/machine-readable assets together so the merge commit is a reproducible snapshot.

Expected assets include:

- `spec/PTSIP-SPEC.md`;
- `spec/PTSIP-CONFORMANCE.md`;
- `spec/PTSIP-TERMINOLOGY.md` where new canonical terms are required;
- `spec/PTSIP-GOVERNANCE.md` only where change/conformance governance needs clarification;
- `schemas/ptsip-profile.schema.json`;
- any new normative/interoperability schema introduced for artifacts, evidence, or diagnostics;
- `registry/ptsip-registry.yaml`;
- `agents/AGENT-CONTRACT.md` only for semantics that agents must obey;
- an ADR describing the final normative decisions;
- `CHANGELOG.md`.

The normative change must be internally consistent at one merge revision. A later Tool release then binds that immutable revision explicitly.

## 8. Normative design questions to resolve before merge

The next snapshot design is not complete until these questions have explicit answers:

1. What exact evidence condition turns an evaluation from `INCOMPLETE` into eligible-for-conformance?
2. How is blocking coverage determined per applicable mandatory rule without relying on arbitrary global percentages?
3. What is the canonical Product Artifact model and how are owner, producer, derivation, and contents separated?
4. What objective conditions require one broad component to be split into multiple ownership components?
5. What are the canonical dependency edge and lifecycle phase semantics?
6. How are `DECLARED`, `OBSERVED`, and `INFERRED` evidence distinguished and prioritized without treating declaration as truth?
7. What exact criteria distinguish Toolchain inspection of Product implementation from prohibited or risky executable implementation reuse?
8. What is the relationship between workflow trigger, actual artifact change, version change, publication, and lifecycle coupling?
9. What are the minimum neutrality criteria for `NEUTRAL_CONTRACT` without requiring an arbitrary number of current consumers?
10. How are remediation/migration records kept separate from conformance, and how is retired `PTSIP-EXC-001` represented without a waiver path?
11. Must a profile choose exactly one ownership declaration mode (`boundaries` XOR `components`), or is a deterministic combined model justified?
12. What information is required in a stable diagnostic contract?
13. How are stricter project-local component dependency constraints represented without expanding PTSIP into a generic dependency-policy language?
14. Which semantics belong in the core specification and which belong only in an implementation/interoperability schema?

## 9. Change classification and compatibility discipline

Every normative edit in the next snapshot must be labelled using existing governance categories:

- `CLARIFICATION`;
- `NORMATIVE_ADDITION`;
- `SCHEMA_CHANGE`;
- `CONFORMANCE_CHANGE`;
- `NORMATIVE_BREAKING` only when unavoidable and explicitly justified.

Existing rule IDs must not be silently repurposed for incompatible meaning.

Candidate new rule families or IDs may be proposed in the ADR, but they are not accepted merely by appearing in this planning document.

## 10. Acceptance criteria for the next normative snapshot

The normative design PR is ready to merge only when all of the following are true:

- the specification family remains `0.2.0-draft`;
- all changed normative assets agree at the same revision;
- exactly three architectural classifications remain;
- evidence node scope/type is separated from architecture classification;
- Profile declaration and conformance evaluation are explicitly distinct;
- `CONFORMANT` cannot be reached through missing applicable evidence;
- artifact owner and artifact producer are distinct concepts;
- exception semantics are machine-expressible and consistent with strict-conformance text;
- component split semantics are objective enough for independent implementations to reach compatible conclusions;
- dependency edge/phase/provenance meanings are explicit enough for different language adapters to produce interoperable evidence;
- diagnostic semantics distinguish rule identity from finding identity;
- project-specific intra-plane dependency constraints remain optional and do not become universal PTSIP rules;
- transition metadata cannot hide present non-conformance;
- imported external evidence cannot silently override native evidence;
- no Simple Connection-specific or turbo-system-specific file layout is made normative;
- compatibility consequences are documented;
- the final merge commit is recorded as the new immutable normative snapshot identity.

## 11. Migration sequence

### Phase 0 — Planning

This document only. No normative semantics change.

### Phase 1 — New `0.2.0-draft` normative snapshot — first priority

Create one design/ADR-driven PR that resolves the normative questions above and updates all dependent schemas/registry/conformance assets together.

No Tool feature implementation is mixed into this PR except unavoidable embedded normative resource synchronization needed to preserve repository consistency; published Tool 0.2.0 remains bound to its existing revision.

### Phase 2 — Tool 0.2.3 evidence correctness

Implement low-risk correctness improvements that make existing evidence collection more accurate without pretending to deliver the full new conformance pipeline.

### Phase 3 — Tool 0.3.0 conformance capability

Implement the new normative snapshot's evidence/evaluation contracts, including `conform`, stable diagnostics, agent decision ingestion, artifact framework, component policy evaluation, and selected ecosystem adapters.

### Phase 4 — Additional ecosystem/adoption features

Evaluate extended .NET, Go, additional packaging adapters, transition metadata, profile composition, and organization baselines using further Consumer Pilots.

## 12. Explicit non-goals of this migration plan

This plan does not:

- modify either Consumer Repository;
- copy Simple Connection governance rules into PTSIP;
- declare either Pilot repository PTSIP conformant;
- introduce a fourth architectural classification;
- auto-approve exceptions;
- make AI classification authoritative;
- require Consumer Repositories to adopt PTSIP-specific directory topology;
- require Tool 0.2.0 to implement a specification revision it was not released against;
- define all future language or packaging adapters in the core specification.

## 13. Completion signal

Phase 1 is complete when the normative design PR is merged and its merge SHA is recorded as the new exact normative identity for the `0.2.0-draft` family.

Only after that identity exists should Tool `0.2.3` or Tool `0.3.0` be bound to and implemented against the new snapshot.
