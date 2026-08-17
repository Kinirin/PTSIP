# PTSIP Status

## Current release state

- Canonical repository: `Kinirin/PTSIP`
- Maturity: Experimental
- Current published Tool/package version: **`0.3.5`**
- Current Tool release tag: **`tool-v0.3.5`**
- Tool `0.3.5` release commit: `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1`
- Tool `0.3.5` GitHub Release: **PUBLISHED** on 2026-08-17
- Tool `0.3.5` PyPI publication: **COMPLETE** through Trusted Publishing
- Tool release workflow: `tooling-release` run `32011194245` — **SUCCESS**
- Release-preparation workflow: `prepare-tool-release` run `32007134610` — **SUCCESS**
- Bound Specification: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Supported Python metadata: Python 3.11–3.14
- Routine hosted Tool CI: Python 3.14
- Tool release namespace: `tool-v*`
- Specification release/design namespace: `spec-v*`
- License: Apache License 2.0

Tool `0.3.5` is the first published PTSIP Tool release that includes **VPMS — Verification Purpose Management System**. VPMS is an independent sibling subsystem that manages why verification exists, while PTSIP continues to govern what repository components are and how Product, Toolchain, and Neutral Contract responsibilities are separated.

The defining boundary remains:

```text
PTSIP
    What is this component?

VPMS
    Why does this verification exist?
```

VPMS is not a fourth PTSIP Plane, and a passing VPMS Verification Case is not a PTSIP conformance result.

## Tool 0.3.5 delivered state

Tool `0.3.5` includes:

- independent `src/vpms/` package ownership alongside `src/ptsip/`;
- explicit verification purposes `PRODUCT` and `TOOLCHAIN`;
- Verification Case, Formula, Variables, Policy, Target, and Runner semantics;
- Formula reuse without purpose collapse;
- purpose-based selection and framework-neutral command execution;
- a narrow read-only PTSIP metadata bridge;
- no `ptsip -> vpms` runtime dependency;
- no VPMS CLI in this release;
- representative repository self-adoption for Product and Toolchain verification purposes;
- root `ptsip.yaml` repository self-profile bound to the active `0.3.4-draft` revision;
- packaging configuration that includes both `ptsip*` and `vpms*` in distributions.

The repository self-profile is a project-owned responsibility declaration, not conformance proof. It is intentionally strict enough to detect tracked files that are outside all declared component selectors.

## Verification and publication evidence

### Repository regression after self-profile correction

The final maintainer-local full regression after assigning `.github/ISSUE_TEMPLATE/**` to repository maintenance completed successfully:

```text
python -m pytest -q
244 passed in 571.90s
```

The self-profile regression remained strict; the test was not weakened to ignore unassigned tracked files.

### Package and installed-wheel verification

Earlier Tool `0.3.5` implementation-completion evidence from GitHub Actions run `31930110185` established:

- `240 passed in 10.25s` on Python 3.14.7 at the pre-self-profile package-verification checkpoint;
- Tool identity `0.3.5`;
- Specification identity `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`;
- successful wheel and sdist build;
- successful `twine check`;
- VPMS package files present in both distributions;
- built-wheel reinstall and installed import smoke PASS;
- no unintended VPMS CLI surface.

That checkpoint predates the repository self-profile extension, so it is retained as package-boundary evidence rather than misreported as the final repository regression.

### Release preparation and publication

Release preparation run `32007134610` completed successfully and created the versioned release-note commit and GitHub Release draft after the Tool test/CLI gate passed.

The reviewed Tool `0.3.5` release note was then finalized at commit:

```text
79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1
```

Publishing `tool-v0.3.5` triggered `tooling-release` run `32011194245`. Both jobs completed successfully:

- `build` — package version/tag match, distribution build, package-content checks, `twine check`, artifact upload;
- `publish` — artifact download and `pypa/gh-action-pypi-publish@release/v1` Trusted Publishing.

This is the canonical Tool `0.3.5` publication boundary.

## Specification state

Tool `0.3.5` intentionally remains bound to:

```text
Specification family:   0.3.4-draft
Specification revision: b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e
```

The existing `spec-v0.3.4-draft` design-release tag is not moved. Tool-version progression and Specification-version progression remain independent.

## Tool lineage

- Tool `0.3.0`: published
- Tool `0.3.1`: published
- Tool `0.3.2`: source-only migration
- Tool `0.3.3`: permanently source-only
- Tool `0.3.4`: published historical Tool release
- Tool `0.3.5`: **published; first VPMS-capable Tool release**

Completed Tool `0.3.5` planning documents are removed from the active `planning/` directory after publication. Their history remains available through Git history and the immutable `tool-v0.3.5` release point; current future planning remains under `planning/0.4.0.md`.
