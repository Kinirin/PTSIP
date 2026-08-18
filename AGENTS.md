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

Do not bypass or weaken `.github/scripts/verify_release_contract.py` to make a release pass. Fix the missing or inconsistent Specification work instead.

### Merge-to-release gate sequence

A normal development merge is not automatically a release candidate. When a merged `main` SHA is selected as the Tool `X.Y.Z` release candidate, agents must enforce this sequence:

```text
merged main SHA
    -> Tool X.Y.Z
    -> Specification X.Y.Z-draft
    -> immutable SPEC_REVISION
    -> tooling-test.yml with release_candidate=true on that exact SHA
    -> release.yml
    -> tooling-release.yml re-check before PyPI publication
```

Rules:

1. re-read remote `main` and identify the exact merged SHA before release-candidate verification;
2. do not run release preparation if the Specification family, binding, release note, or immutable revision is missing or inconsistent;
3. the self-hosted release-candidate status must belong to the exact SHA used for release preparation;
4. never treat “we will fix the Specification after release” as an acceptable workaround.

## Self-hosted verification

Full tests run through the single manual `.github/workflows/tooling-test.yml` workflow on the self-hosted Windows runner.

Expected runner:

```text
DESKTOP-5HCCQIR
```

Before dispatching it:

1. tell the user that the self-hosted runner will be used;
2. wait for explicit confirmation that the host and PowerShell environment are ready;
3. dispatch only after that confirmation with `host_ready=true`.

The workflow checks `RUNNER_NAME == DESKTOP-5HCCQIR` and has no automatic push, pull-request, tag, release, or schedule trigger.

Never dispatch the self-hosted workflow automatically or merely because a test would be convenient. Do not substitute repeated GitHub-hosted full-suite runs when the self-hosted verification path is available.

For a release candidate, run the workflow with `release_candidate=true`. A successful run records `self-hosted/release-verification` for the exact source SHA. `release.yml` requires that status before it can create the Tool release draft.

## GitHub Actions cost boundary

- `tooling-test.yml`: single manual self-hosted full-verification workflow.
- `release.yml`: lightweight GitHub-hosted release preparation/orchestration; no duplicate full pytest run.
- `tooling-release.yml`: release-triggered GitHub-hosted distribution build checks and PyPI Trusted Publishing boundary.
- New verification, regression, build-smoke, or equivalent compute-heavy workflows must default to self-hosted execution.
- Do not add GitHub-hosted push/PR/schedule test matrices or duplicate full-suite workflows without an explicit maintainer decision.
- A GitHub-hosted runner is an exception, not the default. Keep approved exceptions lightweight or platform-bound and do not move the full regression suite into them.
