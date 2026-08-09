# PTSIP Coding-Agent Contract

This is the concise operational contract for coding agents working with a PTSIP-governed Consumer Repository.

## Mandatory agent behavior

1. **Classify before modifying.** Before adding or relocating an SDK/package/module, determine whether it is `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT`.
2. **Purpose before reuse.** Do not create cross-plane sharing merely to remove duplicate code.
3. **No Product -> Toolchain runtime dependency.** Product code must not import/link/load Toolchain SDK code as a shipped dependency.
4. **Do not package Toolchain code with Product.** Development-only validators, migration tools, generators, build helpers, and their dependencies stay outside Product artifacts.
5. **Prefer contract sharing.** When both planes need the same semantics, prefer schemas, registries, IDLs, test vectors, or generated contracts over a shared executable project-local SDK.
6. **Preserve independent build contexts.** Do not solve missing Product dependencies by installing or importing from the Toolchain environment, or vice versa.
7. **Preserve lifecycle independence.** A Toolchain-only refactor should not force Product compatibility constraints unless a Product-facing artifact changes.
8. **Never treat `common`, `shared`, `core`, or `utils` as ownerless.** Every such module must have explicit PTSIP ownership.
9. **Respect repository ownership.** Do not create PTSIP-specific `docs/`, `tools/`, `.ptsip/`, cache, or report directories solely to operate PTSIP tooling.
10. **Read the project profile when one exists.** Repository-specific declarations and exceptions override assumptions about directory names, but a profile declaration is not proof of conformance.
11. **Resolve the bound specification.** For automated conformance, use the specification source/version/revision declared by the project profile or by the PTSIP tooling build; do not silently substitute a different normative snapshot.
12. **Escalate boundary exceptions.** If a requested change requires violating a PTSIP MUST/MUST NOT rule, do not silently implement the violation. Create or request an architecture decision/exception record.
13. **Preserve uncertainty.** `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` are decision statuses, not PTSIP classifications. Do not force a classification when evidence is insufficient.
14. **Do not override observed evidence.** If profile declarations, dependency evidence, packaging evidence, or lifecycle evidence conflict, report the conflict rather than choosing whichever source makes the repository appear conformant.

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

## Read-only default for pilot and inspection

When operating as an external PTSIP inspection, pilot, or validation agent, treat the Consumer Repository as read-only unless the user explicitly authorizes a repository mutation.

Tool-owned caches and generated pilot reports belong outside the Consumer Repository by default. If the user requests a report inside the repository, use the location the user specifies or the repository's own established convention rather than inventing a PTSIP directory hierarchy.

If repository HEAD or observable tracked state changes during evidence collection, mark the evidence snapshot invalidated and rerun or ask for investigation. Do not combine evidence from different revisions into one stable conformance claim.

## Required pre-change questions

Before a boundary-affecting change, answer:

- What is the component's purpose?
- Which plane owns its release lifecycle?
- Is it shipped with the Product?
- Which direction do new dependency edges point?
- What lifecycle phase does each relevant edge serve?
- Can shared semantics be represented as a Neutral Contract Artifact?
- Does the change introduce Product/Toolchain release coupling?

## Required post-change checks

After a boundary-affecting change:

- validate the declared PTSIP profile when one exists;
- inspect cross-plane dependency edges;
- verify Product packaging excludes Toolchain-only artifacts;
- report unresolved evidence/coverage gaps;
- report any PTSIP rule IDs affected by the change.

## Instruction priority

The canonical order is:

1. the bound canonical `PTSIP-SPEC.md` draft family and immutable revision;
2. repository-specific PTSIP Project Profile, when present, as the project's declaration;
3. observed repository/dependency/artifact evidence;
4. approved PTSIP exception/ADR records;
5. this Agent Contract from the same specification revision;
6. informal examples.

A profile declaration has authority over intended ownership but does not override contradictory observed evidence when evaluating conformance. If two sources conflict, preserve and report the conflict rather than silently rewriting either source.
