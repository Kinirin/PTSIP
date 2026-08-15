# PTSIP Coding-Agent Contract

**Specification family:** `0.3.4-draft`

This is the concise operational contract for coding agents working with a PTSIP-governed Consumer Repository. The exact governing snapshot is the immutable revision bound by the installed Tool or the Consumer Repository profile.

## 1. Mandatory agent behavior

1. **Classify before modifying.** Before adding or relocating an SDK/package/module, determine whether it is `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT`.
2. **Purpose before reuse.** Do not create cross-plane sharing merely to remove duplicate code.
3. **No Product -> Toolchain runtime dependency.** Product code must not import, link, load, vendor, or otherwise require Toolchain implementation at Product runtime or as a shipped dependency.
4. **Do not package Toolchain implementation with Product.** Development-only validators, migration tools, generators, build helpers, and Toolchain-only dependencies stay outside Product artifacts.
5. **Separate artifact owner from producer.** A Toolchain component may generate/package a Product Artifact. Inspect artifact ownership, contents, derivation, and shipping scope rather than classifying the output from the producer alone.
6. **Prefer contract sharing.** When both planes need the same semantics, prefer schemas, registries, IDLs, test vectors, generated immutable contracts, or separate generated implementations over one shared executable project-local SDK.
7. **Preserve independent build contexts.** Do not solve missing Product dependencies by importing from the Toolchain environment, or vice versa.
8. **Preserve lifecycle independence.** A Toolchain-only change should not force Product compatibility/version/publication obligations unless a Product-facing artifact or Product obligation actually changes. A workflow trigger alone is not proof of release coupling.
9. **Never treat `common`, `shared`, `core`, or `utils` as ownerless.** Every project-owned component remains subject to explicit PTSIP ownership.
10. **Do not invent extra architecture classifications.** External libraries, platform targets, and unresolved dependency targets are evidence-node scope/type, not a fourth PTSIP plane.
11. **Respect Consumer Repository ownership.** Do not create PTSIP-specific documentation, tooling, cache, report, or hidden directories solely so external PTSIP tooling can operate.
12. **Read the selected Project Profile when one exists.** A Project Profile is project-owned architecture declaration state. It is not proof of conformance and it does not become globally fresh merely because it is complete locally.
13. **Resolve the bound Specification.** Use the exact source/family/revision declared by the Tool or profile. Do not silently substitute a different normative snapshot.
14. **Do not create waivers for mandatory violations.** If requested work would violate a PTSIP `MUST`/`MUST NOT`, report the rule and propose remediation. A project migration/debt record does not convert `NON_CONFORMANT` into `CONFORMANT`.
15. **Preserve uncertainty.** `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` are decision/evaluation states, not architecture classifications.
16. **Preserve evidence provenance.** Keep `DECLARED`, `OBSERVED`, and `INFERRED` evidence distinguishable. Do not present inference as observation.
17. **Do not override contradictory evidence.** Report contradictions among profile declarations, dependency evidence, artifact evidence, lifecycle evidence, imported evidence, and agent review.
18. **Do not equate zero findings with conformance.** A blocking evidence gap produces `INCOMPLETE` unless a definite mandatory violation already establishes `NON_CONFORMANT`.
19. **Do not let project-local policy weaken universal PTSIP.** Project policy may be stricter, never weaker.
20. **Do not invent architecture intent.** Candidate discovery, directory naming, heuristics, or agent confidence do not authorize a project-owner classification decision.

## 2. Durable explicit adoption facts

When an explicit adoption/resolution workflow records a component architecture decision, preserve the structured fact set losslessly when those facts are supplied:

- `classification`;
- `purpose`;
- `shipped`;
- `runtime_required`;
- `lifecycle_owner`;
- `executable`.

Canonical `lifecycle_owner` values are:

- `PRODUCT`;
- `DEVELOPMENT_TOOLING`;
- `INDEPENDENT`.

Required relationships include:

- `PRODUCT` requires `lifecycle_owner: PRODUCT`;
- `TOOLCHAIN` requires `lifecycle_owner: DEVELOPMENT_TOOLING`;
- `TOOLCHAIN` must not be `shipped: true`;
- `TOOLCHAIN` must not be `runtime_required: true`;
- `NEUTRAL_CONTRACT` must be non-executable;
- `NEUTRAL_CONTRACT`, when lifecycle ownership is represented, requires `lifecycle_owner: INDEPENDENT`.

`release_owner` and `compatibility_owner` are separate project metadata and must not be used as lossy aliases for canonical `lifecycle_owner`.

Boundary-root shorthand can remain useful for simple declarations, but it cannot represent the complete adoption fact set. A write-enabled adoption/resolution workflow must not silently discard those facts merely to preserve shorthand. It should require component declarations or another lossless representation before mutation.

## 3. Decision Authority and Project Profile are different

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

## 4. Distributed decision gate

When an active task reaches an architecture-sensitive boundary and distributed coordination is selected:

1. determine the stable coordination domain and normalized component scope;
2. read the relevant current authority state before relying on local declaration freshness;
3. do not create authority history merely to check whether authority state exists;
4. if no local declaration and no authority decision exist, create/reuse a pending decision only when the active task actually requires one;
5. if a valid remote winner exists and the local declaration is missing, validate and safely project the winner locally;
6. if local and remote declarations are semantically equivalent, report a resolved/consistent result without rewriting merely equivalent profile formatting;
7. if local and remote declarations conflict, stop the affected operation with an explicit authority/profile conflict and do not overwrite either side silently;
8. if repository/profile state changed after evidence or projection preparation, refuse stale application and re-analyze;
9. if required authority freshness or safe mutation cannot be established, fail closed rather than creating an isolated Local winner.

A complete local Project Profile is not sufficient reason to skip the relevant distributed authority read.

Semantic equivalence is based on architecture meaning, not YAML key order, whitespace, or Tool-generated formatting.

## 5. First-valid-resolution-wins

For one distributed decision identity:

- the first valid accepted resolution wins;
- a later contradictory resolution must not replace it;
- writes must use ordered conditional mutation such as CAS, transaction, consensus, or an equivalent stale-writer-safe mechanism;
- a stale writer must reread current authority after conditional-write failure;
- if the same-scope winner is already resolved, accept that winner rather than retrying a contradictory answer;
- independent decisions for different scopes may retry/rebase safely when backend semantics permit it.

A coding agent may execute an explicit user decision. It must not manufacture the first answer itself.

## 6. Global decision state is not local projection state

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

A global `RESOLVED` state means one architecture answer won. It does not mean every clone has applied that declaration.

A clone-local application receipt cannot redefine, replace, or reopen the global winner.

## 7. Action-time synchronization

PTSIP does not require continuous background polling. A coding agent should consult distributed authority at the operation boundary where authoritative coordinated architecture state matters.

Do not poll on a timer merely because a decision exists. Do not create reminders when no active task depends on the decision.

If a gate returns `DECISION_REQUIRED`, stop only the affected boundary-sensitive work and ask the user for the missing explicit architecture facts.

## 8. Explicit resolution

When the user supplies the architecture decision in the active chat or another authorized structured interface:

1. record the explicit facts without inference;
2. validate the answer against the bound Specification;
3. resolve the stable distributed decision identity when distributed coordination is selected;
4. preserve the accepted first winner;
5. reconcile the accepted winner into the selected local Project Profile only after freshness and concurrent-content checks;
6. never overwrite a conflicting pre-existing local declaration silently.

A stale Issue/chat answer must not be applied to a changed repository/profile snapshot without re-analysis.

## 9. Constrained classification review

When acting only as a classification-review agent, return one component decision using `schemas/ptsip-agent-classification.schema.json`.

A resolved review decision contains:

- `component_id`;
- `status: RESOLVED`;
- exactly one of `PRODUCT`, `TOOLCHAIN`, `NEUTRAL_CONTRACT`;
- origin/confidence;
- evidence IDs;
- rationale;
- counter-evidence.

When ownership cannot be responsibly resolved, use `UNKNOWN`, `CONFLICT`, or `INCOMPLETE` with no architecture classification.

Agent review evidence must not silently mutate the Project Profile or Decision Authority. Mutation requires a separate explicit adoption/resolution workflow authorized by the project owner/user.

## 10. Evidence and conformance behavior

Preserve dependency relationship type where supported:

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

Distributed-coordination implementation rules (`PTSIP-AUT-*`) govern a backend/tool claiming distributed authority semantics. They are not additional Consumer Repository architecture planes and do not mean a repository becomes non-conformant merely because it does not use distributed coordination.

## 11. Profile Validation versus Conformance Evaluation

**Profile Validation** checks declaration structure and semantics: schema, IDs, selectors, binding syntax, project policy, and represented architecture facts.

**Authority reconciliation** checks whether the local declaration is consistent with the relevant coordinated winner when distributed coordination is selected.

**Conformance Evaluation** combines the declaration with observed dependency, artifact, lifecycle, snapshot, and coverage evidence against PTSIP rules.

These operations must not be collapsed into one another.

## 12. Read-only default and snapshot integrity

External inspection/pilot behavior is read-only by default. Tool-owned caches, reports, local decision databases, and operational state belong outside the Consumer Repository unless the user explicitly chooses a repository path.

If repository HEAD or observed tracked content changes during evidence collection or before a prepared profile write, mark the operation stale and re-run. Do not combine evidence from different revisions into a stable conformance claim.

## 13. External evidence

External validator output may be used when producer, subject repository/revision, scope, and provenance/integrity are known.

External evidence must not silently override contradictory native observed evidence, project declaration, Decision Authority state, or applicable PTSIP rules.

## 14. Required pre-change questions

Before a boundary-affecting change, establish:

- What is the component's primary purpose?
- Which lifecycle owns it?
- Is it shipped with the Product?
- Is Product runtime dependent on it?
- Is it executable or a neutral contract?
- Is the declared component boundary coherent?
- Which dependency direction, relationship type, and lifecycle phase are introduced?
- Is the target project-owned, external, platform, or unresolved?
- Can shared semantics be represented as a Neutral Contract Artifact?
- Does the active coordination mode require an authority freshness check before proceeding?

## 15. Required post-change checks

After a boundary-affecting change:

- validate the selected Project Profile;
- when distributed coordination is selected, account for relevant current authority state;
- inspect cross-plane and project-specific constrained dependency edges;
- verify Product packaging excludes Toolchain-only implementation;
- distinguish artifact owner from producer;
- report blocking/non-blocking evidence gaps;
- report affected PTSIP rule IDs;
- do not declare `CONFORMANT` unless applicable mandatory evidence is sufficient.

## 16. Instruction priority

Use this order:

1. bound canonical `PTSIP-SPEC.md` family + immutable revision;
2. relevant Decision Authority winner for coordinated unresolved/resolved architecture decisions when distributed coordination is selected;
3. repository-specific Project Profile as durable project declaration;
4. observed repository/dependency/artifact evidence;
5. imported external evidence with provenance;
6. project architecture decision records/history;
7. this Agent Contract from the same Specification revision;
8. informal examples.

This ordering does not make Decision Authority a conformance oracle. Authority governs which explicit architecture answer won; observed evidence still governs what the repository actually does, and conformance still requires deterministic evaluation against the bound Specification.

## 17. VPMS verification-purpose obligations

VPMS (`Verification Purpose Management System`) is an optional sibling subsystem. It governs why verification exists and how purpose-bound Verification Cases are selected and executed; it does not replace PTSIP architecture classification, Decision Authority, or Conformance Evaluation.

When an agent creates, modifies, reuses, selects, or executes VPMS verification, follow these rules:

1. **Determine verification purpose first.** Identify the responsibility whose correctness would be lost if the verification disappeared or failed, then record exactly `PRODUCT` or `TOOLCHAIN` in the Verification Case.
2. **Do not infer purpose from physical placement or mechanism.** A `tests/` path, directory name, filename, framework, file extension, compilation status, package inclusion, or runner command may be evidence but is not purpose authority.
3. **Prefer validated VPMS metadata over heuristics.** Once Verification Cases are registered, use `VerificationCase.purpose` as the selection authority. `PRODUCT`, `TOOLCHAIN`, and `FULL` are explicit selection scopes; `FULL` is not a third verification purpose.
4. **Keep PTSIP classification and VPMS purpose separate.** A verifier may be PTSIP `TOOLCHAIN` while its VPMS purpose is `PRODUCT`. Do not convert VPMS purpose into a PTSIP component classification or the reverse.
5. **Preserve Verification Case identity.** Purpose, target, Formula, Variables, Policy, Runner, and result identity remain distinct even when cases share implementation.
6. **Reuse Formula only when it is genuinely purpose-neutral.** Cross-purpose Formula reuse is allowed only when the verification rule remains meaningful without knowing whether PRODUCT or TOOLCHAIN consumes it.
7. **Do not silently merge or normalize Policy.** Shared Formula or Runner implementation does not authorize Product and Toolchain obligations to share Policy. Treat Policy changes as governed expectation/contract changes, not ordinary fixture cleanup.
8. **Never edit Policy merely to make a failing verification pass.** If a governed expectation is wrong, surface the proposed Policy change and its reason explicitly instead of repairing the expected value silently.
9. **Keep Policy names responsibility-oriented.** Do not duplicate Case purpose or execution framework in Policy identifiers merely for convenience. The repository examples `distribution.contract-integrity`, `distribution.package-integrity`, `release.workflow-integrity`, and `ci.verification-boundary` demonstrate this separation; they are not mandatory global names.
10. **Preserve purpose during selective execution.** PRODUCT selection must not pull in TOOLCHAIN-purpose cases and TOOLCHAIN selection must not pull in PRODUCT-purpose cases merely because cases share a Formula, Runner mechanism, source module, or physical directory.
11. **Surface ambiguity instead of guessing.** Missing, malformed, unknown, or unresolved VPMS definition state must remain explicit. Do not manufacture a purpose or reference to make Registry loading succeed.
12. **Do not create PTSIP authority history for VPMS purpose ambiguity.** A VPMS purpose decision is not a PTSIP architecture-classification decision and ordinary VPMS execution must not create or resolve PTSIP Decision Authority records.
13. **Do not reorganize the repository solely to encode purpose.** Reference layouts such as `tests/formula`, `tests/product`, and `tests/toolchain` are optional organization. A mixed physical test module may contain independent PRODUCT-purpose and TOOLCHAIN-purpose Verification Cases.
14. **Keep VPMS results separate from PTSIP conformance.** `PASS`, `FAIL`, `ERROR`, and `SKIPPED` are VPMS execution outcomes. They must not be reported as `CONFORMANT`, `NON_CONFORMANT`, or `INCOMPLETE` without a separate PTSIP Conformance Evaluation.

### 17.1 Registry diagnostics are authoritative validation evidence

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

When Registry loading returns diagnostics, preserve the diagnostic `code`, `location`, `message`, and any supplied `case_id`, `reference_kind`, or `reference`. Do not replace these with path-based guesses or silently rewrite definitions to suppress the diagnostic.

In particular:

- `UNKNOWN_PURPOSE` requires an explicit supported purpose rather than inference from path/framework;
- `UNRESOLVED_REFERENCE` requires the missing target/Formula/Variables/Policy/Runner reference to be resolved explicitly rather than fabricated;
- `DUPLICATE_CASE_ID` requires distinct Case identity rather than silent overwrite or merge.

### 17.2 Selection and execution boundaries

Selection operates on validated Registry Cases and explicit `SelectionScope`. The agent must not discover files and bypass Registry purpose metadata merely to approximate PRODUCT or TOOLCHAIN selection.

Runner execution preserves the selected Case's identity, purpose, and target in `VerificationResult`. Runner diagnostics such as `VPMS-RUN-CONTRACT-ERROR` and `VPMS-RUN-EXECUTION-ERROR` describe VPMS execution failure; they are not PTSIP classification or conformance decisions.

If a selected Case references a Runner without an executor registration, surface the missing registration before execution rather than substituting another Runner silently.

VPMS consumption of PTSIP target metadata remains read-oriented. Ordinary VPMS selection or execution must not rewrite `ptsip.yaml`, mutate PTSIP classification, or alter Decision Authority state.
