# PTSIP Coding-Agent Contract

This is the concise operational contract for coding agents working with a PTSIP-governed Consumer Repository.

## Mandatory agent behavior

1. **Classify before modifying.** Before adding or relocating an SDK/package/module, determine whether it is `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT`.
2. **Purpose before reuse.** Do not create cross-plane sharing merely to remove duplicate code.
3. **No Product -> Toolchain runtime dependency.** Product code must not import/link/load Toolchain SDK code as a shipped dependency.
4. **Do not package Toolchain code with Product.** Development-only validators, migration tools, generators, build helpers, and their dependencies stay outside Product artifacts.
5. **Separate artifact owner from producer.** A Toolchain component may generate/package a Product Artifact. Do not classify the output by the producer's classification alone; inspect the artifact ownership, contents, derivation, and shipping scope.
6. **Prefer contract sharing.** When both planes need the same semantics, prefer schemas, registries, IDLs, test vectors, or generated contracts over a shared executable project-local SDK.
7. **Preserve independent build contexts.** Do not solve missing Product dependencies by installing or importing from the Toolchain environment, or vice versa.
8. **Preserve lifecycle independence.** A Toolchain-only refactor should not force Product compatibility/version/publication obligations unless a Product-facing artifact changes. A workflow trigger alone is not proof of release coupling.
9. **Never treat `common`, `shared`, `core`, or `utils` as ownerless.** Every such project-owned module must have explicit PTSIP ownership.
10. **Do not invent extra architecture classifications.** External libraries, standard-library/platform targets, and unresolved dependency targets use evidence-node scope/type, not a fourth PTSIP plane.
11. **Respect repository ownership.** Do not create PTSIP-specific `docs/`, `tools/`, `.ptsip/`, cache, or report directories solely to operate PTSIP tooling.
12. **Read the project profile when one exists.** Repository-specific declarations and exceptions override assumptions about directory names, but a profile declaration is not proof of conformance.
13. **Resolve the bound specification.** For automated conformance, use the specification source/version/revision declared by the project profile or by the PTSIP tooling build; do not silently substitute a different normative snapshot.
14. **Escalate boundary exceptions.** If a requested change requires violating a PTSIP `MUST`/`MUST NOT` rule, do not silently implement the violation. Create or request an architecture decision/exception record. An approved active PTSIP normative exception still blocks strict PTSIP conformance while the violation remains active.
15. **Preserve uncertainty.** `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` are decision statuses, not PTSIP classifications. Do not force a classification when evidence is insufficient.
16. **Preserve evidence provenance.** Keep `DECLARED`, `OBSERVED`, and `INFERRED` evidence distinguishable. Do not present inference as observation.
17. **Do not override observed evidence.** If profile declarations, dependency evidence, packaging evidence, lifecycle evidence, imported evidence, or agent reasoning conflict, report the conflict rather than choosing whichever source makes the repository appear conformant.
18. **Do not equate zero findings with conformance.** If a missing adapter, unresolved target, artifact gap, unstable snapshot, or other evidence gap can conceal an applicable mandatory-rule violation, treat the evaluation as `INCOMPLETE` unless a definite violation already establishes `NON_CONFORMANT`.
19. **Do not let project-local policy weaken PTSIP.** A project may define stricter same-plane component dependency constraints, but an explicit project allow does not authorize behavior prohibited by universal PTSIP rules.

## Constrained classification decisions

When acting as a PTSIP classification agent, return only one component decision at a time using the structure defined by `schemas/ptsip-agent-classification.schema.json`.

A resolved decision contains:

- `component_id`;
- `status: RESOLVED`;
- exactly one classification from `PRODUCT`, `TOOLCHAIN`, `NEUTRAL_CONTRACT`;
- `origin`;
- confidence in the range 0..1;
- one or more evidence IDs;
- rationale;
- counter-evidence.

When ownership cannot be responsibly resolved, use `status: UNKNOWN`, `CONFLICT`, or `INCOMPLETE` and `classification: null`. Do not invent a fourth classification.

A coding agent decision is review evidence. It MUST NOT silently mutate the project profile, approve an exception, or declare conformance unless a separate user-authorized workflow performs those actions.

A current consumer count is evidence, not a definition of Neutral Contract status. Do not reject or grant `NEUTRAL_CONTRACT` solely because one or both planes are or are not currently observed consuming it; evaluate non-executable contract semantics and lifecycle ownership.

## Evidence and conformance behavior

When reviewing dependency or artifact evidence:

- preserve relationship type such as `IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, or `PUBLISHES`;
- preserve lifecycle phase such as `RUNTIME`, `BUILD`, `TEST`, `RELEASE`, or `INSPECTION` when evidence supports it;
- preserve multiple phases when applicable;
- leave unknown phase/target unresolved rather than guessing;
- distinguish `PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, and `UNRESOLVED_TARGET` evidence scope from PTSIP architectural classification;
- distinguish Product Artifact owner/classification from the Toolchain/Product component that produced it.

When contributing to a conformance evaluation, use these outcomes only:

- `CONFORMANT` — applicable evidence is sufficient and no blocking mandatory violation/active PTSIP normative exception exists;
- `NON_CONFORMANT` — sufficient evidence establishes an applicable `MUST`/`MUST NOT` violation;
- `INCOMPLETE` — blocking evidence or unresolved ownership prevents a conformant conclusion and no definite violation has already settled the result.

`NOT_EVALUATED` is an execution state, not a conformance outcome.

## Profile Validation versus Conformance Evaluation

Do not confuse a valid Project Profile with a conformant repository.

**Profile Validation** checks declaration quality: schema, IDs, selectors, references, binding syntax, exception structure, and optional project-policy consistency.

**Conformance Evaluation** combines declaration with observed dependency, artifact, lifecycle, coverage, exception, and other evidence against PTSIP rules.

If a profile uses the reference ownership model, it uses either boundary-root shorthand or component declarations, not both simultaneously.

## Read-only default for pilot and inspection

When operating as an external PTSIP inspection, pilot, or validation agent, treat the Consumer Repository as read-only unless the user explicitly authorizes a repository mutation.

Tool-owned caches and generated pilot reports belong outside the Consumer Repository by default. If the user requests a report inside the repository, use the location the user specifies or the repository's own established convention rather than inventing a PTSIP directory hierarchy.

If repository HEAD or observable tracked state changes during evidence collection, mark the evidence snapshot invalidated and rerun or ask for investigation. Do not combine evidence from different revisions into one stable conformance claim.

## External evidence

External validator output may be used as PTSIP analysis input when its producer, subject repository/revision, scope, and integrity/provenance are known.

External evidence MUST NOT silently override contradictory native observed evidence, project declaration, or applicable PTSIP rules. Repository-specific governance systems remain repository-owned unless PTSIP explicitly adopts a general interoperable evidence contract.

## Required pre-change questions

Before a boundary-affecting change, answer:

- What is the component's purpose?
- Which plane owns its release lifecycle?
- Is it shipped with the Product?
- Is the declared component boundary coherent, or does the change introduce a split trigger?
- Which direction do new dependency edges point?
- What relationship type and lifecycle phase does each relevant edge serve?
- Is the target a project-owned component, external dependency, platform target, or unresolved target?
- Can shared semantics be represented as a Neutral Contract Artifact?
- Does the change introduce Product/Toolchain release coupling?
- Does a Toolchain producer generate a Product Artifact, and if so what evidence describes the artifact contents and derivation?

## Required post-change checks

After a boundary-affecting change:

- validate the declared PTSIP profile when one exists;
- inspect cross-plane and project-specific constrained dependency edges;
- verify Product packaging excludes Toolchain-only artifacts;
- distinguish artifact owner from producer;
- report blocking and non-blocking unresolved evidence/coverage gaps;
- report any PTSIP rule IDs affected by the change;
- do not declare `CONFORMANT` unless applicable mandatory evidence is sufficient.

## Instruction priority

The canonical order is:

1. the bound canonical `PTSIP-SPEC.md` draft family and immutable revision;
2. repository-specific PTSIP Project Profile, when present, as the project's declaration;
3. observed repository/dependency/artifact evidence;
4. imported external evidence with explicit provenance;
5. approved PTSIP exception/ADR records;
6. this Agent Contract from the same specification revision;
7. informal examples.

A profile declaration has authority over intended ownership but does not override contradictory observed evidence when evaluating conformance. An exception documents an active deviation but does not erase the violated PTSIP rule. If sources conflict, preserve and report the conflict rather than silently rewriting either source.
