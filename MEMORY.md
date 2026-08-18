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

### Merge-to-release Specification gate

Normal development merges may continue before a Tool becomes a release candidate. However, once a merged `main` SHA is proposed for Tool `X.Y.Z` release, that exact SHA must satisfy the complete Specification contract before release preparation may proceed.

Required sequence:

```text
feature/development branches
        -> merge to main
        -> exact merged main SHA
        -> Tool X.Y.Z / Specification X.Y.Z-draft / immutable SPEC_REVISION check
        -> self-hosted release-candidate verification for that exact SHA
        -> release.yml
        -> tooling-release.yml re-check at the PyPI boundary
```

A missing or inconsistent Specification is a blocking release defect. Do not defer Specification creation, binding, release-note work, or `SPEC_REVISION` correction until after the Tool release.

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

GitHub-hosted Actions usage is constrained.

Full repository verification uses only `.github/workflows/tooling-test.yml`, which is manual and self-hosted.

Expected runner name:

```text
DESKTOP-5HCCQIR
```

Operational rule:

**Before dispatching the self-hosted workflow, tell the user that the runner will be used and wait for explicit confirmation that the host and PowerShell environment are ready.**

The workflow requires `host_ready=true`, checks `RUNNER_NAME == DESKTOP-5HCCQIR`, and has no push, pull-request, tag, release, or schedule trigger.

For release-candidate verification, dispatch it with `release_candidate=true`. A successful run records `self-hosted/release-verification` for the exact source SHA.

`release.yml` remains GitHub-hosted only for lightweight release preparation/orchestration and requires that exact self-hosted status instead of repeating full pytest. `tooling-release.yml` remains GitHub-hosted for distribution build checks and the PyPI Trusted Publishing boundary.

### Default runner rule

New repository verification, regression, build-smoke, or equivalent compute-heavy workflows **must default to self-hosted execution**. Do not introduce GitHub-hosted push/PR/schedule test matrices or duplicate full-suite workflows merely for convenience.

A GitHub-hosted runner is an exception that requires an explicit maintainer decision and must remain limited to a lightweight or platform-bound boundary such as release orchestration or Trusted Publishing. Existing `release.yml` and `tooling-release.yml` are the current approved exceptions; they must not absorb the full regression suite.

## Release-note discipline

Generated `git-cliff` output is a starting draft, not the final architectural explanation. New subsystems, policy changes, migrations, compatibility boundaries, and verification evidence must be reviewed and explained in human-written release notes before publication.
