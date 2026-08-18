# PTSIP Repository Working Memory

This file is durable repository-operational context for maintainers and coding agents. It is **not** a normative Specification and must not replace `ptsip.yaml`, `spec/`, schemas, registry data, or ADRs.

## Current published baseline

- Canonical repository: `Kinirin/PTSIP`
- Current published Tool: `0.3.5`
- Current Tool tag: `tool-v0.3.5`
- Tool `0.3.5` release commit: `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1`
- Current bound Specification: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Tool `0.3.5` is the first published Tool release containing VPMS.
- Tool `0.3.5` intentionally retained the existing explicit repository Responsibility Map instead of introducing a generalized map schema.

## Next Tool direction: 0.3.6

Tool `0.3.6` is planned as a compatibility-focused Responsibility Map generalization release.

Required behavior:

1. keep current explicit Responsibility Maps valid;
2. add optional supported Responsibility Map templates;
3. allow repositories to choose a built-in template, keep a fully custom map, or use a template plus explicit overrides;
4. provide preview-first, loss-preserving migration;
5. support either template-backed output or a fully materialized explicit migrated map;
6. never infer template choice or responsibility purpose from paths alone;
7. make VPMS visible in the new normative Specification.

The detailed plan is `planning/0.3.6.md`.

## Specification release rule

From Tool `0.3.6` onward, a Tool release is not allowed to reuse an older draft Specification family merely because the Tool code is otherwise ready.

Release preparation requires:

```text
Tool X.Y.Z
    ->
Specification X.Y.Z-draft
    ->
immutable SPEC_REVISION
```

The root `ptsip.yaml`, Tool constants, canonical Specification files, and `releasenote/spec-<family>.md` must agree. `.github/scripts/verify_release_contract.py` is the fail-closed release gate used by release preparation and the PyPI build boundary.

### Exact merge-to-release Specification gate

Normal development merges may continue before a Tool becomes a release candidate. At the explicit release boundary, finalize and commit the reviewed `releasenote/X.Y.Z.md` first. The resulting final `main` commit becomes the only SHA eligible for release-candidate verification.

Required sequence:

```text
feature/development branches
        -> merge to main
        -> finalize and commit releasenote/X.Y.Z.md
        -> exact final main SHA
        -> Tool X.Y.Z / Specification X.Y.Z-draft / immutable SPEC_REVISION check
        -> tooling-test.yml on self-hosted Windows
             release_candidate=true
             source_sha=<exact SHA>
        -> self-hosted/release-verification on that exact SHA
        -> release.yml on self-hosted Windows with the same source_sha
        -> draft release targets the same SHA and does not mutate main
        -> publish reviewed draft
        -> tooling-release.yml self-hosted build on the published tag
        -> minimal GNU/Linux Trusted Publishing boundary
```

A missing or inconsistent Specification is a blocking release defect. Do not defer Specification creation, binding, release-note work, or `SPEC_REVISION` correction until after the Tool release.

The exact-SHA invariant is deliberate: no release-note commit, generated file, or other repository mutation may be inserted between successful release-candidate verification and creation of the draft release.

## PTSIP / VPMS boundary

PTSIP answers:

```text
What is this component?
```

VPMS answers:

```text
Why does this verification exist?
```

VPMS verification purposes are currently `PRODUCT` and `TOOLCHAIN`.

The PTSIP classification of verifier implementation code and the VPMS purpose of a Verification Case are separate axes. PTSIP core must not depend on VPMS. VPMS PASS is not PTSIP CONFORMANT.

## Coding-agent read order

Before repository changes, read:

1. `AGENTS.md`
2. this `MEMORY.md`
3. `ptsip.yaml`
4. `src/ptsip/constants.py`
5. the bound Specification under `spec/`
6. the active version plan under `planning/`

Re-read the remote branch HEAD immediately before writes. Do not rely on a previously observed SHA when another maintainer may have committed.

## Workflow resource policy

GitHub-hosted Actions usage is constrained. Self-hosted execution is the default for repository verification, release preparation, and distribution building.

Approved Windows self-hosted runner:

```text
DESKTOP-5HCCQIR
```

Operational rules:

- Before dispatching `tooling-test.yml` or `release.yml`, tell the user that the self-hosted runner will be used and wait for explicit confirmation that the host and PowerShell environment are ready.
- Both manual workflows require `host_ready=true` and check `RUNNER_NAME == DESKTOP-5HCCQIR`.
- For release-candidate verification, `tooling-test.yml` also requires the exact full `source_sha`.
- A successful release-candidate run records `self-hosted/release-verification` for the exact checked-out SHA.
- `release.yml` requires the same SHA, requires it still to be `origin/main`, and creates the draft release without committing or pushing anything.
- `tooling-release.yml` uses the self-hosted Windows runner for its build job. Before publishing a draft Tool release, ensure the runner is online; if it is offline, the build should wait rather than fall back to GitHub-hosted compute.

### Minimal GitHub-hosted exception

The `tooling-release.yml` PyPI `publish` job remains on GitHub-hosted GNU/Linux only because `pypa/gh-action-pypi-publish` is a Docker-based action and therefore cannot execute on the approved Windows self-hosted runner.

That job may only:

```text
download already verified distributions
        ->
perform PyPI Trusted Publishing
```

It must not absorb regression tests, package building, release preparation, or other avoidable compute.

### Default runner rule

New repository verification, regression, release preparation, build-smoke, or equivalent compute-heavy workflows **must default to self-hosted execution**. Do not introduce GitHub-hosted push/PR/schedule test matrices or duplicate full-suite workflows merely for convenience.

A GitHub-hosted runner is an exception requiring an explicit maintainer decision. The current approved exception is only the narrow GNU/Linux PyPI Trusted Publishing job described above.

## Release-note discipline

Generated `git-cliff` output is a starting draft, not the final architectural explanation. New subsystems, policy changes, migrations, compatibility boundaries, and verification evidence must be reviewed and explained in human-written release notes before publication.

For Tool releases, the reviewed `releasenote/X.Y.Z.md` must be committed before the final release-candidate SHA is sent through self-hosted verification.
