# Independent PTSIP releases

`tooling-release.yml` consumes the admitted contract in `release-contract.json`.
`run_release.ps1` selects and invokes target scripts. Native Go owns selection,
version consumption, source fingerprints, compilation checks, artifact hashes,
immutable publication and Package assembly. PowerShell owns process orchestration.

The release Go module moved from `developer/automation/release/`. Its existing Go
import path and `ptsip-dev release ...` command compatibility remain available
through explicit module replacements; the physical owner is this directory.

## Independent identities

- Tool: `0.3.8a5`, `0.3.8b1`, `0.3.8rc1`, or stable `0.3.8`.
- Profile: `pp.1.02`, selected from the canonical Profile registry.
- Each of seven engines: `0.2a1`, `0.2b1`, `0.2rc1`, or stable `0.2`.
- Package: `0.3.8.<six hexadecimal characters>`. Its numeric base changes only by
  explicit owner input. The suffix maps the canonical Tool/Profile/seven-engine
  version vector. It has no ordering or compatibility meaning. The complete
  SHA-256 and vector are recorded; a suffix collision fails closed.

Existing engine IDs `0.2.0` and `0.3.0` are preserved as exact bootstrap IDs.
Future public engine versions use the approved two-number format. Cargo's
build projection is explicit: `0.2a1` maps to `0.2.0-alpha.1`, `0.2rc1` to
`0.2.0-rc.1`, and stable `0.2` to `0.2.0`. Update the public manifest/index/binding
and its Cargo projection together. Release execution never bumps a component.

## Selection and execution

Preview one engine locally without asserting that it has been published:

```powershell
./.github/checks/release/run_release.ps1 -Action Plan -Targets context -Offline
```

Compile the selected engine for the local host:

```powershell
./.github/checks/release/run_release.ps1 -Action Build -Target context
```

`-Action Build` without a single `-Target` builds the selected components in
parallel on the local host. This is local build evidence; publishing an engine
requires all admitted platform receipts and observed successful workflow jobs.

Dispatch exact-main verification, or explicitly select publication:

```powershell
./.github/checks/release/run_release.ps1 -Action Dispatch -ReleasePhase verify -Targets context
./.github/checks/release/run_release.ps1 -Action Dispatch -ReleasePhase publish -Targets context
./.github/checks/release/run_release.ps1 -Action Dispatch -ReleasePhase verify -Targets all -DisabledTargets tool
```

Changed selected components build independently in the Actions matrix. Linux
x64, Windows x64 and macOS arm64 are currently admitted engine platforms.
Unselected, disabled and unchanged published components do not build. An existing
component version with changed source fails closed instead of overwriting a tag
or silently incrementing Tool/Profile/another engine. Individual publications
proceed only after that component's required platform jobs succeed. A failed
component blocks itself and any Package requiring it.

`contracts_release.ps1` projects the selected engine's neutral contracts; it is
a packaging helper and does not define an eighth engine.

## Distribution and Package join

Engine GitHub Releases contain platform ZIP files, native release executables,
their registered neutral contracts and source/hash/platform receipts. Profile
Releases contain the current schema, registry and public examples; history is
excluded. Tool publication retains wheel/sdist verification and PyPI Trusted
Publishing, and consumes exactly the already verified build artifact.

The Package job joins the verified component artifacts after their independent
work. Unchanged components are reused from immutable releases; the historical
Tool bootstrap is also checked against the version-specific PyPI hashes. A
Package never silently compiles an unselected missing component. Publication
also requires each referenced component publication to have completed.

The Python wheel remains a Tool distribution. Native executables ship in their
independent engine archives and the aggregate Package. Engine installation,
consumer discovery/activation and public API retirement are separate scopes.

## Verification boundaries

The retired common `tooling-test` gap remains OPEN. These selected release
component jobs do not reissue former common CI contexts or claim repository-wide
verification. Publication requires observed exact-SHA jobs from this workflow;
a local receipt saying PASS is insufficient. Existing Tool artifact boundary,
snapshot binding, installed-wheel, Specification, Profile transition, metadata
and distribution checks are preserved in the Tool target.
