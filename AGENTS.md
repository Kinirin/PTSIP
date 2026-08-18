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
- Do not create a Tool release note early; finalize and commit it at the explicit release boundary before the exact release-candidate SHA is verified.
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

### Exact merge-to-release gate sequence

A normal development merge is not automatically a release candidate. When `main` is ready for Tool `X.Y.Z`, first finalize and commit the reviewed `releasenote/X.Y.Z.md`; that commit must be part of the exact release-candidate SHA.

Agents must enforce this sequence:

```text
merged main + reviewed releasenote/X.Y.Z.md
    -> exact immutable main SHA
    -> Tool X.Y.Z
    -> Specification X.Y.Z-draft
    -> immutable SPEC_REVISION
    -> tooling-test.yml
         host_ready=true
         release_candidate=true
         source_sha=<exact SHA>
    -> self-hosted/release-verification on that exact SHA
    -> release.yml
         host_ready=true
         source_sha=<same exact SHA>
    -> draft release targets the same exact SHA without modifying main
    -> publish reviewed draft
    -> tooling-release.yml self-hosted build on the published tag
    -> minimal GitHub-hosted GNU/Linux Trusted Publishing job
```

Rules:

1. re-read remote `main` and identify the exact final SHA before release-candidate verification;
2. the Tool release note must already be committed in that SHA;
3. do not run release preparation if the Specification family, binding, release note, or immutable revision is missing or inconsistent;
4. the self-hosted release-candidate status must belong to the exact SHA supplied to `release.yml`;
5. `release.yml` must not create a new commit after that status is recorded;
6. `release.yml` must fail if the candidate is no longer the current `origin/main` HEAD;
7. never treat “we will fix the Specification after release” as an acceptable workaround.

## Self-hosted verification and release execution

Repository verification, release preparation, and release distribution builds use the self-hosted Windows runner.

Expected runner:

```text
DESKTOP-5HCCQIR
```

Before dispatching either `tooling-test.yml` or `release.yml`:

1. tell the user that the self-hosted runner will be used;
2. wait for explicit confirmation that the host and PowerShell environment are ready;
3. dispatch only after that confirmation with `host_ready=true`.

`tooling-test.yml` checks `RUNNER_NAME == DESKTOP-5HCCQIR` and has no automatic push, pull-request, tag, release, or schedule trigger.

Never dispatch the self-hosted verification workflow automatically or merely because a test would be convenient. Do not substitute repeated GitHub-hosted full-suite runs when the self-hosted verification path is available.

For a release candidate, `tooling-test.yml` requires an explicit full `source_sha`. A successful run records `self-hosted/release-verification` for the exact checked-out SHA. `release.yml` requires the same SHA and status, runs on the same approved self-hosted runner, rechecks `origin/main`, and creates the draft release without mutating `main`.

`tooling-release.yml` automatically starts when the reviewed Tool release is published. Its build and distribution verification job runs on `DESKTOP-5HCCQIR`; before publishing the draft release, ensure that runner is online and ready. If it is offline, the self-hosted build waits rather than falling back to a GitHub-hosted build.

## GitHub Actions cost boundary

- `tooling-test.yml`: manual self-hosted full verification and exact-SHA release-candidate status.
- `release.yml`: manual self-hosted release preparation; it verifies the exact status and creates a draft release without changing `main`.
- `tooling-release.yml` `build`: self-hosted Windows distribution build, exclusion checks, and `twine` verification.
- `tooling-release.yml` `publish`: the only current GitHub-hosted compute exception. It downloads already verified artifacts and runs PyPI Trusted Publishing on GNU/Linux because `pypa/gh-action-pypi-publish` is Docker-based and cannot run on the approved Windows self-hosted runner.
- New verification, regression, release preparation, or build-smoke workflows must default to self-hosted execution.
- Do not add GitHub-hosted push/PR/schedule test matrices or duplicate full-suite workflows without an explicit maintainer decision.
- Never move compilation, regression, package building, or other avoidable compute into the GitHub-hosted publish exception.
