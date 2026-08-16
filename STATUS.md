# PTSIP Status

## Current migration state

- Canonical repository: `Kinirin/PTSIP`
- Maturity: Experimental
- Current Tool/package source version: `0.3.5`
- Tool `0.3.5` source identity: VPMS implementation, documentation, packaging, and public Python surface stabilized; repository self-profile hardening is now part of the 0.3.5 branch before merge/release
- Previous active Specification baseline on `main`: `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`
- `spec-v0.3.4-draft`: published GitHub Specification **design-release checkpoint**; tag remains unchanged
- Final `0.3.4-draft` repository-identity-migration revision: `b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Tool `0.3.5` Specification binding: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Repository identity migration: **COMPLETE**
- Tool `0.3.5` implementation state: **SELF-PROFILE HARDENING IN PROGRESS; POST-HARDENING FINAL VERIFICATION PENDING**
- Latest verified PyPI publication: `PTSIP==0.3.1`
- Tool `0.3.4` publication: **not published**; no `tool-v0.3.4`, GitHub Tool Release, or PyPI `0.3.4`
- Tool `0.3.5` publication: **not published**; source identity and earlier implementation verification do not imply release readiness after the self-profile extension
- Supported Python metadata: Python 3.11–3.14
- Routine hosted Tool CI: Python 3.14 to conserve GitHub Actions usage
- Tool release namespace: `tool-v*`
- Specification release/design namespace: `spec-v*`
- License: Apache License 2.0

The `0.3.4-draft` repository-identity-migration freeze remains the canonical active Specification snapshot. Tool `0.3.5` changes Tool behavior by introducing VPMS while intentionally retaining that Specification binding; a Tool-version change does not imply a Specification-version change.

The existing `spec-v0.3.4-draft` tag is not moved to the later normative freeze revision. Immutable tag provenance and active mutable-draft snapshot identity are separate concepts.

## Specification 0.3.4-draft activation scope

The migration combines the accepted Explicit Project Adoption design with Distributed Authority Consistency while preserving exactly three architecture classifications:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT`

The new durable structured adoption fact set is:

- `classification`
- `purpose`
- `shipped`
- `runtime_required`
- `lifecycle_owner`
- `executable`

Canonical lifecycle owners are `PRODUCT`, `DEVELOPMENT_TOOLING`, and `INDEPENDENT`. `release_owner` and `compatibility_owner` remain separate optional project metadata and are not aliases for canonical lifecycle ownership.

New stable Specification rules are:

- `PTSIP-ADP-001` — lossless explicit project adoption
- `PTSIP-AUT-001` — Decision Authority / Project Profile responsibility separation
- `PTSIP-AUT-002` — stable distributed decision identity
- `PTSIP-AUT-003` — first-valid-resolution-wins with ordered conditional mutation
- `PTSIP-AUT-004` — authority freshness
- `PTSIP-AUT-005` — safe authority/profile reconciliation
- `PTSIP-AUT-006` — fail-closed distributed coordination
- `PTSIP-AUT-007` — global decision state / local projection separation

`PTSIP-AUT-*` applies to implementations claiming distributed coordination capability. It does not create a fourth Consumer Repository architecture plane.

## Coherent migration assets

The immutable freeze `b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e` contains the coherent normative/machine-readable migration state, including:

- `spec/PTSIP-SPEC.md`
- `spec/PTSIP-CONFORMANCE.md`
- `spec/PTSIP-TERMINOLOGY.md`
- `spec/PTSIP-GOVERNANCE.md`
- `schemas/ptsip-profile.schema.json`
- `schemas/ptsip-artifact-evidence.schema.json`
- `schemas/ptsip-agent-classification.schema.json`
- `schemas/ptsip-diagnostic.schema.json`
- `registry/ptsip-registry.yaml`
- `agents/AGENT-CONTRACT.md`
- `adoption/ADOPTION-GUIDE.md`
- `reference/REFERENCE-ARCHITECTURE.md`
- `profiles/example.ptsip.yaml`
- `decisions/ADR-0005-activate-spec-0.3.4-draft.md`
- `src/ptsip/specdata/*` packaged machine-readable contracts
- Tool profile-projection behavior required for lossless structured facts

The following Tool release-readiness commit adds the Tool binding and regression fixtures that point back to that immutable snapshot.

All five canonical/embedded machine-readable pairs are required to be byte-identical by release-readiness tests:

1. Project Profile schema
2. registry
3. artifact-evidence schema
4. agent-classification schema
5. diagnostic schema

## Tool 0.3.4 distributed-authority implementation

Tool `0.3.4` already completed the distributed-authority behavior in PR `#24` (`555c528593f700a348d8da84545a62ce61291cae`) with the test identity follow-up in PR `#25` (`8cd0ddf16dc9b56f27f694138a37caae1c49bb4f`).

Implemented behavior includes:

- authority freshness even when local Project Profile declaration is complete;
- read-only authority absence lookup without fabricated history;
- first-valid-resolution-wins with non-force Git-ref CAS;
- deterministic missing/equivalent/conflicting reconciliation;
- `AUTHORITY_PROFILE_CONFLICT` without silent overwrite;
- stale repository/profile refusal;
- fail-closed `COORDINATION_UNAVAILABLE` instead of implicit Local DecisionStore fallback;
- global decision state separate from clone-local `LOCAL_PROJECTION` application state;
- action-time synchronization rather than continuous polling.

## Verification history

### Tool 0.3.4 pre-rebind verification

GitHub Actions run `31471025526` on Python `3.14.6` verified the original Tool `0.3.4` source while it was still bound to `0.2.0-draft @ a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`:

- `134 passed`
- Tool identity smoke passed
- exact then-current Specification binding passed
- build passed
- wheel/sdist `twine check` passed
- installed-wheel smoke passed

That run remains valid evidence for the distributed-authority implementation, but it predates the repository identity migration and is **not** final release evidence.

### Repository-identity-migration verification

Final local and hosted verification was not used as a Tool `0.3.4` publication boundary because Tool `0.3.4` was not published. Its behavior and immutable Specification binding are carried forward into Tool `0.3.5`.

### Tool 0.3.5 pre-self-profile implementation-completion checkpoint

GitHub Actions `tooling-test` run `31930110185` (job `95123510309`) verified the Tool `0.3.5` implementation on Python `3.14.7` before the repository self-profile hardening was added. The one-off verification branch contained the same product, test, and packaging blobs as the then-current `tool-0.3.5-vpms`; its only intentional extra surface was the temporary workflow trigger and final-gate smoke steps.

Verified results at that checkpoint:

- full repository pytest: `240 passed in 10.25s`;
- source/editable identity: distribution metadata, `TOOL_VERSION`, and `ptsip.__version__` all `0.3.5`;
- Specification identity remains `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`;
- `ptsip --version` -> `PTSIP Tool 0.3.5`;
- `ptsip spec` reports `tool_version: 0.3.5` while retaining the `0.3.4-draft` Specification identity;
- `ptsip conform --help` passes before and after wheel installation;
- `python -m build` produced `ptsip-0.3.5.tar.gz` and `ptsip-0.3.5-py3-none-any.whl`;
- `python -m twine check dist/*` passed for both artifacts;
- sdist and wheel contain all current `vpms`, `vpms.domain`, `vpms.execution`, `vpms.execution.adapters`, and `vpms.integration` Python files;
- built wheel was force-reinstalled successfully;
- installed-wheel smoke verified `vpms` imports from `site-packages`, PRODUCT/TOOLCHAIN purpose identities remain available, and Tool/Specification identity remains correct.

This remains valid historical evidence for the code and distribution boundary it tested, but it is no longer the final verification of the latest branch HEAD because the 0.3.5 branch has been extended with repository self-profile hardening.

### Tool 0.3.5 repository self-profile hardening

Coding-agent evaluation at branch commit `c560f977e06e526389b68cdc8d48398046103455` found that the PTSIP repository itself had no root `ptsip.yaml`. As a result, repository inspection could discover candidates but PTSIP had no project-owned responsibility declaration for its own implementation, verification, release automation, and specification contracts.

The 0.3.5 branch therefore continues without merge and now includes:

- repository self-profile commit `eee53f609af4437d63554383d61addccccf89737`;
- `ptsip.yaml` bound to `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`;
- explicit PRODUCT ownership for `ptsip-core`, `vpms`, `ptsip-distribution`, and product documentation;
- explicit NEUTRAL_CONTRACT ownership for embedded/canonical Specification contracts, governance contracts, and the repository architecture declaration;
- explicit TOOLCHAIN ownership for repository verification, release automation, CI/documentation automation, and repository maintenance;
- regression-test commit `78080449a2234600391b34169ad2f3ce9a021d4e` adding `tests/ptsip/test_repository_self_profile_035.py`;
- VPMS self-adoption target IDs `ptsip-distribution` and `repository-release-automation` now resolve against actual PTSIP component metadata.

Static exact-blob checks confirmed the committed `ptsip.yaml` blob and the locally reconstructed content both hash to `88dfceaca3815c745236434b9adfe30219a5b6ee`. Schema-semantic constraints and the five candidate selectors observed during the coding-agent evaluation were checked for unique declared coverage. The container cannot resolve `github.com`, so exact-checkout pytest/CLI verification remains pending on a local execution environment rather than being replaced by an unnecessary hosted Actions run.

A post-self-profile final verification must supersede the earlier WU-18 checkpoint before branch merge or release preparation.

## Tool lineage and publication policy

- Tool `0.3.0`: published
- Tool `0.3.1`: published as `tool-v0.3.1` / PyPI
- Tool `0.3.2`: source-only migration
- Tool `0.3.3`: permanently source-only; never tag, GitHub Release, or PyPI
- Tool `0.3.4`: implementation completed; repository identity migration complete; not published
- Tool `0.3.5`: VPMS implementation and earlier WU-18 checkpoint completed; repository self-profile hardening is now in progress and requires a post-hardening final verification before merge/release preparation

No Tool publication is implied by Specification activation, source-version stabilization, or a superseded verification checkpoint.
