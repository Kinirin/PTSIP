# AGENTS.md

These instructions apply to coding agents working anywhere in this repository.

## Required context before work

Read these files before planning or modifying code:

1. `MEMORY.md`
2. `ptsip.yaml`
3. `src/ptsip/constants.py`
4. the Specification files under `spec/` for the bound `SPEC_VERSION` / `SPEC_REVISION`
5. the active Tool plan under `planning/`

`MEMORY.md` is repository-operational context only. Normative claims come from the bound Specification and project-owned machine-readable contracts.

## Repository-state discipline

- Re-read the remote branch HEAD immediately before any write, merge, release preparation, or evidence claim.
- Preserve maintainer commits and never force-update `main`.
- Do not claim a test, build, release, tag, or publication succeeded unless evidence for the exact relevant SHA was observed.
- Do not create a Tool release note early; it belongs at the explicit release boundary.
- Once a Tool tag is published, treat the tagged version document as immutable historical evidence.

## PTSIP and VPMS reasoning

PTSIP asks:

```text
What is this component?
```

VPMS asks:

```text
Why does this verification exist?
```

Rules:

- never infer PTSIP classification or VPMS verification purpose solely from path, filename, framework, language, or package location;
- keep PTSIP component classification and VPMS Verification Purpose as separate axes;
- current VPMS purposes are `PRODUCT` and `TOOLCHAIN`;
- preserve Verification Case identity and Purpose when reusing Formula logic;
- do not merge Product and Toolchain Policy merely because Formula implementation is shared;
- PTSIP core must not acquire a VPMS runtime dependency;
- VPMS PASS is not PTSIP CONFORMANT, and PTSIP CONFORMANT is not functional verification PASS.

## Tool 0.3.6 Responsibility Map direction

Tool `0.3.6` must preserve current explicit Responsibility Maps and add optional generalization.

Supported conceptual modes are:

```text
explicit
template
hybrid (template + repository overrides)
```

A migration must be preview-first and loss-preserving. Never silently replace a project-owned map. If a selected template cannot represent existing facts losslessly, report the conflict and stop.

Repository-owned explicit declarations outrank template defaults. Template selection must be explicit; never guess a template from repository layout.

See `planning/0.3.6.md` for the current implementation plan.

## Mandatory Specification work for releases

Starting with Tool `0.3.6`, release preparation must not proceed unless the Tool release has a new aligned draft Specification family.

For Tool `X.Y.Z`, the release contract requires:

```text
SPEC_VERSION = X.Y.Z-draft
```

and requires an immutable `SPEC_REVISION`, matching root `ptsip.yaml` binding, canonical Specification files at that revision, and `releasenote/spec-X.Y.Z-draft.md`.

Do not bypass or weaken the future release-contract gate to make a release pass. Fix the missing or inconsistent Specification work instead.

## Self-hosted verification

Full tests for Tool `0.3.6` work are to run through `.github/workflows/tooling-test.yml` after it is converted to manual self-hosted operation.

Expected runner:

```text
DESKTOP-5HCCQIR
```

The self-hosted workflow is **manual only**.

Before dispatching it:

1. tell the user that the self-hosted runner will be used;
2. wait for explicit confirmation that the host and PowerShell environment are ready;
3. dispatch only after that confirmation and explicit host-ready acknowledgement.

Never dispatch the self-hosted workflow automatically or merely because a test would be convenient. Do not substitute repeated GitHub-hosted full-suite runs when the self-hosted verification path is available.

For a release candidate, the self-hosted run must verify one exact source SHA. Release preparation must require that evidence before creating the Tool release draft.

## GitHub Actions cost boundary

- `tooling-test.yml`: must become the single manual self-hosted full-verification workflow before 0.3.6 verification begins.
- `release.yml`: keep as lightweight release preparation/orchestration and remove duplicate full-suite testing once the self-hosted gate is active.
- `tooling-release.yml`: keep as the release-triggered build and PyPI Trusted Publishing boundary.
- Avoid adding additional automatic test workflows without an explicit maintainer decision.
