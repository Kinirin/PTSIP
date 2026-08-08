# PTSIP Coding-Agent Contract

This is the concise operational contract for coding agents working in a PTSIP-governed repository.

## Mandatory agent behavior

1. **Classify before modifying.** Before adding or relocating an SDK/package/module, determine whether it is `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT`.
2. **Purpose before reuse.** Do not create cross-plane sharing merely to remove duplicate code.
3. **No Product -> Toolchain runtime dependency.** Product code must not import/link/load Toolchain SDK code as a shipped dependency.
4. **Do not package Toolchain code with Product.** Development-only validators, migration tools, generators, build helpers, and their dependencies stay outside Product artifacts.
5. **Prefer contract sharing.** When both planes need the same semantics, prefer schemas, registries, IDLs, test vectors, or generated contracts over a shared executable project-local SDK.
6. **Preserve independent build contexts.** Do not solve missing Product dependencies by installing or importing from the Toolchain environment, or vice versa.
7. **Preserve lifecycle independence.** A Toolchain-only refactor should not force Product compatibility constraints unless a Product-facing artifact changes.
8. **Never treat `common`, `shared`, `core`, or `utils` as ownerless.** Every such module must have explicit PTSIP ownership.
9. **Read the project profile.** Repository-specific PTSIP paths and exceptions override assumptions about directory names.
10. **Escalate boundary exceptions.** If a requested change requires violating a PTSIP MUST/MUST NOT rule, do not silently implement the violation. Create or request an architecture decision/exception record.

## Required pre-change questions

Before a boundary-affecting change, answer:

- What is the component's purpose?
- Which plane owns its release lifecycle?
- Is it shipped with the Product?
- Which direction do new dependency edges point?
- Can shared semantics be represented as a Neutral Contract Artifact?
- Does the change introduce Product/Toolchain release coupling?

## Required post-change checks

After a boundary-affecting change:

- validate the declared PTSIP profile;
- inspect cross-plane dependency edges;
- verify Product packaging excludes Toolchain-only artifacts;
- report any PTSIP rule IDs affected by the change.

## Instruction priority

The canonical order is:

1. `spec/PTSIP-SPEC.md`
2. repository-specific `ptsip.yaml`
3. approved PTSIP exception/ADR records
4. this Agent Contract
5. informal examples

If two sources conflict, use the higher-priority source and report the conflict.
