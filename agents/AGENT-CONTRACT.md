# PTSIP Coding-Agent Contract

**Specification family:** `0.3.6-draft`

This is the concise operational contract for coding agents working with a PTSIP-governed Consumer Repository. The exact governing snapshot is the immutable revision bound by the installed Tool or the Consumer Repository profile.

## 1. Mandatory agent behavior

1. **Classify before modifying.** Before adding, splitting, relocating, or changing an in-scope project responsibility, determine its primary lifecycle ownership: `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, or `NEUTRAL_CONTRACT`.
2. **Use governing lifecycle obligation.** Classification follows why the responsibility must exist/change/remain compatible/execute/retire, not filename, directory, framework, language, executable status, workflow provider, or test status.
3. **Purpose before reuse.** Do not create cross-lifecycle executable sharing merely to remove duplicate code.
4. **No Product runtime dependency on non-Product implementation.** Product code must not import, link, load, vendor, or otherwise require `DEVELOPMENT_TOOLING`, `DELIVERY`, or `OPERATIONS` implementation at Product runtime or as shipped implementation.
5. **Do not package non-Product lifecycle implementation with Product.** Development-only, Delivery-only, and Operations-only implementation/dependencies stay outside Product artifacts unless ownership is explicitly redesigned as `PRODUCT`.
6. **Separate artifact owner from producer.** Development Tooling or Delivery may generate/build/package/publish a Product Artifact. Inspect artifact ownership, contents, derivation, and shipping scope rather than classifying output from producer ownership alone.
7. **Prefer contract sharing.** When lifecycles need the same semantics, prefer valid `NEUTRAL_CONTRACT`, generated lifecycle-owned implementations, or separately governed external/platform contracts over one shared executable project-local implementation.
8. **Preserve independently resolvable lifecycle environments.** Do not solve missing Product dependencies by importing from Development Tooling, Delivery, or Operations environments, or vice versa.
9. **Preserve lifecycle independence.** A change in one lifecycle should not force unrelated lifecycle compatibility/version/publication obligations unless a release-relevant artifact, compatibility obligation, or explicitly coupled contract changes.
10. **Never treat generic names as ownerless.** `common`, `shared`, `core`, `utils`, and similar names do not create neutral ownership.
11. **Do not invent extra architecture classifications.** External libraries, platforms, unresolved targets, confidence values, and workflow states are not PTSIP classifications.
12. **Read the selected Project Profile when one exists.** A Project Profile is project-owned architecture declaration state; it is not observed proof and does not become globally fresh merely because it is locally complete.
13. **Resolve the bound Specification.** Use the exact source/family/revision declared by the Tool or profile. Do not silently substitute a different normative snapshot.
14. **Do not create waivers for mandatory violations.** A project migration/debt record does not convert `NON_CONFORMANT` into `CONFORMANT`.
15. **Preserve uncertainty.** `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` are decision/evaluation states, not classifications.
16. **Preserve evidence provenance.** Keep `DECLARED`, `OBSERVED`, and `INFERRED` distinguishable.
17. **Do not override contradictory evidence.** Report contradictions among declarations, dependency/artifact/lifecycle evidence, imported evidence, and agent review.
18. **Do not equate zero findings with conformance.** A blocking evidence gap produces `INCOMPLETE` unless a definite mandatory violation already establishes `NON_CONFORMANT`.
19. **Do not let project-local policy weaken universal PTSIP.** Project policy may be stricter, never weaker.
20. **Do not invent architecture intent.** Candidate discovery, path names, heuristics, template similarity, or agent confidence may support a proposal but do not authorize a project-owner architecture decision.

## 2. Lifecycle boundary determination

When ownership is not already explicit and current, reason in this order:

```text
project-owned scope
    -> coherent responsibility boundary
    -> evidence + provenance
    -> NEUTRAL_CONTRACT qualification test
    -> governing owning lifecycle
    -> mixed-lifecycle split test
    -> one classification / split / unresolved
    -> project-owner confirmation for inferred architecture
```

Important boundaries:

- Product-specific quality/verification may be `PRODUCT`; reusable test SDK/framework/harness infrastructure may be `DEVELOPMENT_TOOLING`.
- Local/intermediate development build support is normally `DEVELOPMENT_TOOLING`; authoritative release-unit assembly/signing/packaging for handoff is normally `DELIVERY`.
- `DELIVERY` ends at the semantic delivery handoff where the selected release reaches/is accepted by its destination and ordinary ongoing operation begins.
- Ongoing health/recovery/reconciliation/maintenance after handoff is `OPERATIONS`.
- `NEUTRAL_CONTRACT` requires non-executable architectural role, non-owning semantics, and lifecycle-independent governance.
- If independently governable lifecycle responsibilities are mixed, propose a split instead of choosing the majority of files/jobs/steps or the highest confidence score.

## 3. Responsibility Map v2

Keep these axes separate:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics within the lifecycle

relationships
    = typed directed project-owned semantic edges

VPMS Verification Purpose
    = what a Verification Case protects/verifies
```

### 3.1 Canonical roles

A component may declare zero, one, or multiple roles from exactly:

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

Do not manufacture composite role names such as `VERIFICATION_AUTOMATION` or `BUILD_RELEASE_AUTOMATION`. Preserve independent role values and use typed relationships for target-specific semantics.

### 3.2 Canonical typed relationships

Project-owned Responsibility Map relationship types are:

```text
IMPORTS
LINKS
LOADS
INVOKES
READS
GENERATES
BUILDS
PACKAGES
PUBLISHES
DEPLOYS
VERIFIES
MANAGES
DOCUMENTS
SPECIFIES
GOVERNS
```

Relationship direction is `source --TYPE--> target`. Project declaration, observed evidence, and project-specific allow/deny policy are separate data. Do not turn one into another silently.

Evidence `TESTS` may support a `VERIFIES` proposal but does not automatically create a project-owned `VERIFIES` relationship.

### 3.3 Associated artifacts

An associated artifact is a project-owned non-component support surface subordinate to exactly one classified anchor component. It has stable identity, explicit scope/purpose, no component classification or roles of its own, and at least one typed relationship connecting it to its anchor.

Do not use associated artifacts as a classification escape hatch. Re-evaluate/promote an associated artifact as a component if it gains executable responsibility, independent release/compatibility/lifecycle ownership, independently governed Delivery/Operations responsibility, or cross-lifecycle authority that cannot remain subordinate to one anchor. An independently governed non-executable non-owning lifecycle-independent contract must be evaluated for `NEUTRAL_CONTRACT`.

Component IDs and associated-artifact IDs share one map-wide endpoint namespace and must not collide.

### 3.4 Responsibility Map modes

Canonical conceptual modes are:

```text
explicit
    repository directly declares the complete map

template
    repository explicitly selects one version/revision-bound template

hybrid
    repository explicitly selects a template and supplies project-owned ID-addressed overrides/extensions/removals
```

Never guess a template from layout, language, framework, manifests, package manager, or confidence. Template/hybrid data must be materialized from the exact bound template before an operation that requires concrete component endpoints.

## 4. Durable explicit adoption facts

Canonical Tool `0.3.6` Project Profiles use `classification` itself as primary lifecycle ownership authority. A second canonical `lifecycle_owner` field must not compete with it.

When an explicit adoption/resolution workflow records structured architecture facts, preserve supplied facts losslessly when they are applicable:

- `classification`;
- `roles`;
- `purpose`;
- `shipped`;
- `runtime_required`;
- `executable`;
- associated-artifact facts;
- typed relationships;
- release/compatibility metadata where explicitly supplied.

Legacy Tool `0.3.5` `lifecycle_owner`, `consumers`, `analysis_inputs`, old boundary roots, and untyped dependency-policy edges may be migration evidence. They must not be blindly copied as competing ownership authority or canonical typed relationships.

`TOOLCHAIN` is a legacy Tool `0.3.5` PTSIP classification only. Tool `0.3.6` may understand it through the legacy migration path but must not emit or preserve it as a canonical classification alias.

## 5. Decision Authority and Project Profile are different

When distributed coordination is selected, keep these responsibilities distinct:

```text
Decision Authority
    -> which explicit architecture answer won for a coordinated decision

Project Profile
    -> which architecture declaration is represented by this repository revision/worktree

Observed evidence
    -> what the repository/artifacts actually do

Conformance Evaluation
    -> whether declaration + observed evidence satisfy applicable PTSIP rules
```

A Decision Authority does not replace `ptsip.yaml`, and a resolved authority winner does not prove conformance.

Local Tool-owned state such as SQLite is not a repository-global authority merely because every environment runs the same Tool. Do not Git-share local SQLite state.

## 6. Distributed decision gate

When an active task reaches an architecture-sensitive boundary and distributed coordination is selected:

1. determine the stable coordination domain and normalized component scope;
2. read relevant current authority state before relying on local declaration freshness;
3. do not create authority history merely to check whether authority state exists;
4. if no local declaration and no authority decision exist, create/reuse pending state only when the active task actually requires one;
5. if a valid remote winner exists and the local declaration is missing, validate and safely project the winner locally;
6. if local and remote declarations are semantically equivalent, report consistency without rewriting merely equivalent formatting;
7. if they conflict, stop the affected operation and do not overwrite either side silently;
8. if repository/profile state changed after evidence or projection preparation, refuse stale application and re-analyze;
9. if required authority freshness or safe mutation cannot be established, fail closed rather than creating an isolated Local winner.

A complete local Project Profile is not sufficient reason to skip the relevant distributed authority read. Semantic equivalence is based on architecture meaning, not YAML key order, whitespace, or Tool-generated formatting.

## 7. First-valid-resolution-wins

For one distributed decision identity:

- the first valid accepted resolution wins;
- a later contradictory resolution must not replace it;
- writes must use ordered conditional mutation such as CAS, transaction, consensus, or equivalent stale-writer-safe semantics;
- a stale writer must reread current authority after conditional-write failure;
- if the same-scope winner is already resolved, accept that winner rather than retrying a contradictory answer;
- independent decisions for different scopes may retry/rebase safely when backend semantics permit it.

A coding agent may execute an explicit user decision. It must not manufacture the first answer itself.

## 8. Global decision state is not local projection state

Do not conflate repository/domain-wide decision state with clone/worktree-local application state.

```text
GLOBAL DECISION STATE
    PENDING
    RESOLVED

LOCAL PROJECTION/APPLICATION STATE
    missing
    equivalent / consistent
    locally applied
    stale
    failed
```

A global `RESOLVED` state means one architecture answer won. It does not mean every clone has applied that declaration. A clone-local application receipt cannot redefine, replace, or reopen the global winner.

## 9. Action-time synchronization

PTSIP does not require continuous background polling. Consult distributed authority at the operation boundary where authoritative coordinated architecture state matters.

Do not poll on a timer merely because a decision exists. If a gate returns `DECISION_REQUIRED`, stop only the affected boundary-sensitive work and ask the user for the missing explicit architecture facts.

## 10. Explicit resolution

When the user supplies the architecture decision in an authorized interface:

1. record explicit facts without inference;
2. validate the answer against the bound Specification;
3. resolve the stable distributed decision identity when distributed coordination is selected;
4. preserve the accepted first winner;
5. reconcile the winner into the selected local Project Profile only after freshness/concurrent-content checks;
6. never overwrite a conflicting pre-existing local declaration silently.

A stale answer must not be applied to a changed repository/profile snapshot without re-analysis.

## 11. Constrained classification review

When acting only as a classification-review agent, return one component decision using `schemas/ptsip-agent-classification.schema.json`.

A resolved review decision contains:

- `component_id`;
- `status: RESOLVED`;
- exactly one of `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, `NEUTRAL_CONTRACT`;
- origin/confidence;
- evidence IDs;
- rationale;
- counter-evidence.

When ownership cannot be responsibly resolved, use `UNKNOWN`, `CONFLICT`, or `INCOMPLETE` with no architecture classification.

Agent review evidence must not silently mutate the Project Profile or Decision Authority. Mutation requires a separate explicit adoption/resolution workflow authorized by the project owner/user.

## 12. Evidence and conformance behavior

Preserve evidence dependency relationship type where supported:

- `IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, `PUBLISHES`.

Preserve lifecycle phases when supported:

- `RUNTIME`, `BUILD`, `TEST`, `RELEASE`, `INSPECTION`.

Preserve evidence scope:

- `PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, `UNRESOLVED_TARGET`.

For completed Consumer Repository Conformance Evaluation, use only:

- `CONFORMANT`;
- `NON_CONFORMANT`;
- `INCOMPLETE`.

`NOT_EVALUATED` may be a Tool execution state but is not a conformance outcome.

Distributed-coordination implementation rules (`PTSIP-AUT-*`) govern a backend/tool claiming distributed authority semantics. They are not additional Consumer Repository architecture classifications.

## 13. Profile Validation versus Conformance Evaluation

**Profile Validation** checks declaration structure and semantics: schema, Responsibility Map mode, IDs, selectors, binding syntax, typed endpoints/anchors, project policy, and represented architecture facts.

**Authority reconciliation** checks whether the local declaration is consistent with the relevant coordinated winner when distributed coordination is selected.

**Conformance Evaluation** combines declaration with observed dependency, artifact, lifecycle, snapshot, and coverage evidence against PTSIP rules.

These operations must not be collapsed into one another.

## 14. Read-only default and snapshot integrity

External inspection/pilot behavior is read-only by default. Tool-owned caches, reports, local decision databases, and operational state belong outside the Consumer Repository unless the user explicitly chooses a repository path.

If repository HEAD or observed tracked content changes during evidence collection or before a prepared profile write, mark the operation stale and re-run. Do not combine evidence from different revisions into a stable conformance claim.

## 15. External evidence

External validator output may be used when producer, subject repository/revision, scope, and provenance/integrity are known.

External evidence must not silently override contradictory native observed evidence, project declaration, Decision Authority state, or applicable PTSIP rules.

## 16. Required pre-change questions

Before a boundary-affecting change, establish:

- What is the responsibility's primary purpose?
- What is its governing lifecycle obligation?
- Which canonical lifecycle classification owns it?
- Is it shipped with Product or required by Product runtime?
- Is it executable, independently governed configuration/documentation/governance, or a valid neutral contract?
- Is the component boundary coherent or does mixed-lifecycle responsibility require a split?
- Which role values apply without replacing classification/purpose?
- Which typed relationships are introduced?
- Is the target project-owned, external, platform, or unresolved?
- Can shared semantics be represented as a valid `NEUTRAL_CONTRACT`?
- Does the active coordination mode require authority freshness before proceeding?

## 17. Required post-change checks

After a boundary-affecting change:

- validate the selected/materialized Project Profile;
- when distributed coordination is selected, account for relevant current authority state;
- inspect cross-lifecycle and project-specific constrained dependency edges;
- verify Product packaging excludes non-Product lifecycle implementation;
- distinguish artifact owner from producer;
- verify associated-artifact anchor/relationship requirements;
- preserve declared typed relationships separately from observed evidence;
- report blocking/non-blocking evidence gaps;
- report affected PTSIP rule IDs;
- do not declare `CONFORMANT` unless applicable mandatory evidence is sufficient.

## 18. Instruction priority

Use this order:

1. bound canonical `PTSIP-SPEC.md` plus normative companion specifications from the same immutable revision;
2. relevant Decision Authority winner for coordinated architecture decisions when distributed coordination is selected;
3. repository-specific Project Profile as durable project declaration;
4. observed repository/dependency/artifact evidence;
5. imported external evidence with provenance;
6. project architecture decision records/history;
7. this Agent Contract from the same Specification revision;
8. informal examples.

This ordering does not make Decision Authority a conformance oracle. Authority governs which explicit architecture answer won; observed evidence still governs what the repository actually does, and conformance still requires deterministic evaluation against the bound Specification.

## 19. VPMS verification-purpose obligations

VPMS (`Verification Purpose Management System`) is an optional sibling subsystem. It governs why verification exists and how purpose-bound Verification Cases are selected and executed; it does not replace PTSIP architecture classification, Decision Authority, or Conformance Evaluation.

VPMS purpose vocabulary is intentionally independent from Tool `0.3.6` PTSIP classification vocabulary. Current VPMS Verification Purpose values remain exactly `PRODUCT` and `TOOLCHAIN`; VPMS `TOOLCHAIN` is not a canonical Tool `0.3.6` PTSIP lifecycle classification.

When an agent creates, modifies, reuses, selects, or executes VPMS verification:

1. **Determine verification purpose first.** Identify the responsibility whose correctness would be lost if the verification disappeared or failed, then record exactly `PRODUCT` or `TOOLCHAIN` in the Verification Case.
2. **Do not infer purpose from placement or mechanism.** A `tests/` path, filename, framework, compilation status, package inclusion, or runner command may be evidence but is not purpose authority.
3. **Prefer validated VPMS metadata over heuristics.** Once Verification Cases are registered, use `VerificationCase.purpose` as selection authority. `PRODUCT`, `TOOLCHAIN`, and `FULL` are selection scopes; `FULL` is not a third verification purpose.
4. **Keep PTSIP classification and VPMS purpose separate.** A verifier may be PTSIP `DEVELOPMENT_TOOLING` while its VPMS purpose is `PRODUCT`. Do not convert VPMS purpose into PTSIP classification or the reverse.
5. **Preserve Verification Case identity.** Purpose, target, Formula, Variables, Policy, Runner, and result identity remain distinct even when cases share implementation.
6. **Reuse Formula only when genuinely purpose-neutral.** Cross-purpose Formula reuse is allowed only when the verification rule remains meaningful without knowing whether PRODUCT or TOOLCHAIN consumes it.
7. **Do not silently merge or normalize Policy.** Shared Formula or Runner implementation does not authorize Product and Toolchain verification obligations to share Policy.
8. **Never edit Policy merely to make a failing verification pass.** Surface a governed expectation change and its reason explicitly.
9. **Keep Policy names responsibility-oriented.** Do not duplicate Case purpose or execution framework in Policy identifiers merely for convenience.
10. **Preserve purpose during selective execution.** PRODUCT selection must not pull in TOOLCHAIN-purpose cases and vice versa merely because cases share Formula/Runner/source/directory.
11. **Surface ambiguity instead of guessing.** Missing, malformed, unknown, or unresolved VPMS definition state must remain explicit.
12. **Do not create PTSIP authority history for VPMS purpose ambiguity.** A VPMS purpose decision is not a PTSIP classification decision.
13. **Do not reorganize the repository solely to encode purpose.** Reference test layouts are optional organization.
14. **Keep VPMS results separate from PTSIP conformance.** `PASS`, `FAIL`, `ERROR`, and `SKIPPED` must not be reported as PTSIP conformance outcomes without a separate Conformance Evaluation.
15. **Require materialized PTSIP target metadata.** If a Tool `0.3.6` template/hybrid Responsibility Map is not yet materialized, VPMS must fail closed rather than pretending the project has zero component targets.

### 19.1 Registry diagnostics are authoritative validation evidence

The implemented Registry emits deterministic machine-readable diagnostics with format:

```text
vpms-registry-diagnostic/v1
```

Current diagnostic codes are:

```text
MALFORMED_DEFINITIONS
MALFORMED_CASE
MISSING_FIELD
UNKNOWN_FIELD
INVALID_FIELD
UNKNOWN_PURPOSE
DUPLICATE_CASE_ID
UNRESOLVED_REFERENCE
```

When Registry loading returns diagnostics, preserve diagnostic `code`, `location`, `message`, and any supplied `case_id`, `reference_kind`, or `reference`. Do not replace these with path-based guesses or silently rewrite definitions to suppress the diagnostic.

In particular:

- `UNKNOWN_PURPOSE` requires an explicit supported purpose rather than inference from path/framework;
- `UNRESOLVED_REFERENCE` requires the missing target/Formula/Variables/Policy/Runner reference to be resolved explicitly rather than fabricated;
- `DUPLICATE_CASE_ID` requires distinct Case identity rather than silent overwrite or merge.

### 19.2 Selection and execution boundaries

Selection operates on validated Registry Cases and explicit `SelectionScope`. The agent must not discover files and bypass Registry purpose metadata merely to approximate PRODUCT or TOOLCHAIN selection.

Runner execution preserves the selected Case's identity, purpose, and target in `VerificationResult`. Runner diagnostics such as `VPMS-RUN-CONTRACT-ERROR` and `VPMS-RUN-EXECUTION-ERROR` describe VPMS execution failure; they are not PTSIP classification or conformance decisions.

If a selected Case references a Runner without executor registration, surface the missing registration before execution rather than substituting another Runner silently.

VPMS consumption of PTSIP target metadata remains read-oriented. Ordinary VPMS selection or execution must not rewrite `ptsip.yaml`, mutate PTSIP classification, or alter Decision Authority state.
