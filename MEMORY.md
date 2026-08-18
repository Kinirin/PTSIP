# PTSIP Repository Working Memory

This file is durable repository-operational context for maintainers and coding agents. It is **not** a normative Specification and must not replace `ptsip.yaml`, `spec/`, schemas, registry data, or ADRs.

## Current published baseline

- Canonical repository: `Kinirin/PTSIP`
- Current published Tool: `0.3.5`
- Current Tool tag: `tool-v0.3.5`
- Tool `0.3.5` release commit: `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1`
- Published Tool `0.3.5` bound Specification: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Tool `0.3.5` is the first published Tool release containing VPMS.
- Tool `0.3.5` uses canonical PTSIP classifications `PRODUCT | TOOLCHAIN | NEUTRAL_CONTRACT`.

Do not rewrite the historical meaning of the published `0.3.5` release. Tool `0.3.6` migration support must understand it as legacy input.

## Active Tool 0.3.6 development

Development branch:

```text
tool-0.3.6-lifecycle-ownership
```

Active plan:

```text
planning/0.3.6.md
```

The first `0.3.6-draft` normative baseline containing both updated Specification body and terminology exists at immutable revision:

```text
654e41d49600fc091f9a6cb6b1c60bbc7da4e301
```

This revision is the WU-00 baseline snapshot. The development Tool/runtime/profile/schema is not yet fully activated to that family merely because the normative baseline exists. Later work units must update schema, implementation, embedded specdata, Tool constants, repository self-adoption, and final release binding consistently.

WU-01 lifecycle boundary determination is now frozen in the active `0.3.6-draft` Specification and `decisions/ADR-0007-primary-lifecycle-boundary-determination.md`. The next sequential ontology-dependent work unit is WU-02 (role + typed relationships + associated artifacts).

## Tool 0.3.6 lifecycle model

Tool `0.3.6` defines PTSIP `classification` as **primary lifecycle ownership**.

Canonical classifications:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

Interpretation:

- `PRODUCT` — Product runtime, user-facing behavior, Product distribution content, runtime SDK responsibility, and Product-owned quality/verification responsibility.
- `DEVELOPMENT_TOOLING` — reusable development support, verification infrastructure/test SDKs, generators, migration tooling, lint/static analysis, developer CLI/build/repository tooling.
- `DELIVERY` — release, package/container publication, signing, promotion, distribution, and deployment-to-destination responsibility.
- `OPERATIONS` — post-deployment production health, ongoing infrastructure state, backup/recovery, incident/maintenance and operational automation.
- `NEUTRAL_CONTRACT` — deliberately non-executable, non-owning, lifecycle-independent contract responsibility.

Artifact kind does not decide classification. Tests may be `PRODUCT` or `DEVELOPMENT_TOOLING` depending on ownership. Technology names such as FastAPI, Cloudflare Workers, GitHub Actions, Docker, Terraform, Python, PowerShell, Markdown, or YAML are evidence context rather than architecture authority.

## WU-01 boundary determination rules

Lifecycle classification is determined from the **governing lifecycle obligation**: why the responsibility must exist/change/remain compatible/execute/retire.

The required reasoning order is:

```text
project-owned scope
    -> coherent responsibility boundary
    -> evidence + provenance
    -> NEUTRAL_CONTRACT qualification test
    -> governing owning lifecycle
    -> mixed-lifecycle split test
    -> one classification / split / unresolved
    -> project-owner confirmation for inferred migration
```

Important frozen boundaries:

- Product-specific tests may be `PRODUCT`; reusable test SDK/framework/harness infrastructure may be `DEVELOPMENT_TOOLING`. The test target or VPMS purpose does not directly decide PTSIP ownership.
- Development/local/intermediate build support is normally `DEVELOPMENT_TOOLING`; authoritative release-unit assembly/signing/packaging for handoff is normally `DELIVERY`.
- `DELIVERY` ends at the semantic **delivery handoff** where the selected release reaches/is accepted by its destination and ordinary operation begins.
- ongoing deployed-state health/recovery/reconciliation/maintenance responsibility after handoff is `OPERATIONS`.
- `NEUTRAL_CONTRACT` requires all three semantics: non-executable architectural role, non-owning responsibility, and lifecycle-independent governance.
- a mixed component must not be classified by majority of files/jobs/steps or confidence score. If independently governable lifecycle responsibilities are separable, propose a split; if not safely resolvable, remain unresolved.
- subordinate activities from another phase do not force a split when they exist only to complete one coherent lifecycle obligation and introduce no independent lifecycle governance.

Normative rule IDs are `PTSIP-CLS-004` through `PTSIP-CLS-011`.

## Tool 0.3.5 compatibility boundary

Tool `0.3.6` must understand valid Tool `0.3.5` profiles through a legacy reader and assisted migration path.

Compatibility means:

```text
understand old profile
    -> collect evidence
    -> propose lifecycle/role/relationship/boundary migration
    -> project owner confirms
    -> write canonical 0.3.6 map safely
```

Compatibility does **not** mean retaining `TOOLCHAIN` as a canonical alias in the new schema.

A legacy `TOOLCHAIN` component must not be blindly renamed to `DEVELOPMENT_TOOLING`. It may map to `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, a component split, or unresolved clarification depending on evidence and project-owner confirmation.

Legacy `PRODUCT` and `NEUTRAL_CONTRACT` may often carry forward, but discovery may still expose split/reclassification proposals when old coarse boundaries hid different lifecycle responsibilities.

## Responsibility Map v2 direction

Tool `0.3.6` Responsibility Map v2 separates these axes:

```text
classification
    = primary lifecycle ownership

role
    = responsibility performed inside that lifecycle

relationship
    = typed semantic relationship to another responsibility/artifact

VPMS Verification Purpose
    = what a Verification Case protects/verifies
```

Supported conceptual Responsibility Map modes:

```text
explicit
template
hybrid (template + repository overrides)
```

Template selection remains explicit. Candidate discovery, migration analysis, confidence, role inference, typed-relationship inference, and component-split detection are evidence/proposals, not project architecture authority.

Responsibility Map v2 must also represent project-owned associated documentation/authority/support artifacts without requiring them to become independent components merely to express `DOCUMENTS`/`SPECIFIES`/`GOVERNS`-type semantics. Associated artifacts must not become a classification escape hatch.

The exact canonical role/relationship vocabulary and schema representation are WU-02/WU-03 work and are not frozen solely by examples in planning documents.

## VPMS boundary

PTSIP answers lifecycle ownership. VPMS answers verification purpose.

Tool `0.3.5` VPMS verification purposes currently use:

```text
PRODUCT
TOOLCHAIN
```

That `TOOLCHAIN` token is VPMS vocabulary, not a Tool `0.3.6` PTSIP lifecycle classification. Do not rename VPMS purpose vocabulary merely as an accidental side effect of PTSIP ontology migration.

PTSIP core must not depend on VPMS. VPMS may consume stable PTSIP metadata through a narrow read-only boundary. VPMS PASS is not PTSIP CONFORMANT, and PTSIP CONFORMANT is not functional verification PASS.

## Tool 0.3.6 work-unit order

Follow `planning/0.3.6.md`.

Current sequence begins:

```text
WU-00  0.3.6-draft normative baseline        COMPLETE
    ->
WU-01  lifecycle ontology/boundary rules      COMPLETE
    ->
WU-02  role + typed relationships + associated artifacts   NEXT
    ->
WU-03  canonical Responsibility Map v2 schema
```

Evidence/candidate-discovery work then feeds the legacy-reader and migration analyzer. Do not implement final migration writes before the target ontology/schema is stable enough to preserve project intent losslessly.

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

The root `ptsip.yaml`, Tool constants, canonical Specification files, embedded specdata, and release-contract evidence must agree at the release boundary.

`.github/scripts/verify_release_contract.py` is the fail-closed release gate used by release preparation and the PyPI build boundary. Do not weaken it to accommodate incomplete Specification work.

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

## Coding-agent read order

Before repository changes, read:

1. `AGENTS.md`
2. this `MEMORY.md`
3. `ptsip.yaml`
4. `src/ptsip/constants.py`
5. applicable Specification under `spec/`
6. active version plan under `planning/`

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

The `tooling-release.yml` PyPI `publish` job remains on GitHub-hosted GNU/Linux only because `pypa/gh-action-pypi-publish` is Docker-based and therefore cannot execute on the approved Windows self-hosted runner.

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