# Independent PTSIP releases

Independent engine publication is admitted by the project owner's explicit
decision. The already published Tool 0.3.8a5 remains unchanged.
`tooling-release.yml` admits Engine Verification only, using the scoped
`developer/policy/contracts/engine-verification.v1.json` contract. Seven engines
can be selected independently on Windows x64, Linux x64 and macOS arm64.
Tool, Profile and Package verification are deferred. Release Gate passes only
when canonical admission, all selected engine/platform jobs and their unexpired
artifacts belong to the exact workflow source SHA. This gate reports selected
engine verification readiness; PowerShell and native publication entrypoints
remain blocked for Tool/Profile/Package and the retained legacy release surface.

`tooling-release.yml` has separate `verify` and `publish` phases. Verification
tests/builds the selected engines on all three hosts and saves original wheels
and receipts. Publication requires that completed successful verification run
ID at the exact current main SHA; it downloads those original artifacts without
rebuilding. Native Go reassembles and byte-compares every wheel, checks its
receipt, fresh source and immutable version, then creates each engine tag and
GitHub Release. The external official PyPA action uploads only Go-admitted wheels
through Trusted Publishing. Native Go confirms the public PyPI file digests.

Publication input `destination=github` creates GitHub releases while PyPI setup
is pending. Repeating publication with `destination=github-pypi` and the same
verification run reuses the exact wheel set and does not overwrite release files.
Register Pending Trusted Publishers for `ptsip-engine-context`,
`ptsip-engine-storage`, `ptsip-engine-validation`, `ptsip-engine-discovery`,
`ptsip-engine-planning`, `ptsip-engine-execution` and `ptsip-engine-projection`:
GitHub owner `Kinirin`, repository `PTSIP`, workflow `tooling-release.yml`,
environment `pypi`. Tool/Profile/Package are not publication targets of this flow.

The Linux wheel uses `manylinux_2_39_x86_64`; native Go inspects ELF architecture,
external library closure and GLIBC requirements before admitting that tag.
Packaged License bytes come from the exact source SHA's Git blob, preserving
canonical content across Windows CRLF and Unix LF worktree materialization.

Rust dependencies and debug/release compilation outputs are cached per engine,
runner, native target, compiler commit and Cargo dependency fingerprint. Exact
keys also include engine source; source changes may reuse only a compatible
dependency cache. Cargo test/build and wheel verification always run, including
on an exact cache hit. Wheels, receipts, consumer environments and prior PASS
results are never restored from the cache. The first run populates caches;
subsequent compatible runs restore them.

The retained publication implementation describes `release-contract.json`.
`run_release.ps1` selects and invokes target scripts. Native Go owns selection,
version consumption, source fingerprints, compilation checks, artifact hashes,
immutable publication and Package assembly. PowerShell owns process orchestration.

The following architecture describes retained implementation only. It does not
grant release execution permission or establish current verification readiness.

## Independent identities

- Tool: `0.3.8a5`, `0.3.8b1`, `0.3.8rc1`, or stable `0.3.8`.
- Profile: `pp.1.02`, selected from the canonical Profile registry.
- Each of seven engines: independent `MAJOR.MINOR.PATCH` versions (e.g. `0.2.0`, `0.3.0`).
- Package: `0.3.8.<six hexadecimal characters>`. Its numeric base changes only by
  explicit owner input. The suffix maps the canonical Tool/Profile/seven-engine
  version vector. It has no ordering or compatibility meaning. The complete
  SHA-256 and vector are recorded; a suffix collision fails closed.

Engine version strings have exactly three numeric segments. Two-segment
versions, aN/bN/rcN suffixes and Cargo prerelease translations are not engine
identities. Cargo uses the same three-segment version. The original Context
and Projection identities are `0.2.0`; Storage, Validation, Discovery,
Planning and Execution are `0.3.0`. The retained bootstrap version registry
records these historic starting values. Release execution never bumps a component.

## Canonical per-engine versions

Each engine owns its manually editable public version in
`src/ptsip/engines/<engine>/version.yaml`. The canonical form is
`MAJOR.MINOR.PATCH` (for example `0.2.0` or `0.3.0`). No version
is inferred from other engines and the release workflow never increments
versions automatically.

After changing a version file, synchronize the derived records:

```powershell
./.github/checks/release/run_release.ps1 -Action SyncVersions -Targets context
./.github/checks/release/run_release.ps1 -Action SyncVersions -Targets all
```

This operation updates the engine index, runtime manifest, execution bindings,
Cargo.toml, and Cargo.lock. Commit the version file and projections together.
The release Plan verifies exact agreement and fails closed on divergence.
The historical bootstrap versions are preserved separately in the release catalog.

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

## Verification gap

The engine verification exception uses native Rust test/build, a Go wheel
assembler under `engine-verification/`, and the independent Go wheel verifier.
Python is used only for the isolated consumer package installer. Verification
artifacts and successful engine jobs do not grant publication or runtime authority.

The previous verification execution map and the test enforcing its legacy
migration table have been removed. Their historical test lists do not establish
the successor release verification requirements. The common verification gap
and the release deployment gap remain OPEN. Previous PASS results must not be
used to assert readiness for the current source.

Release execution can resume only through a separately approved successor
policy and contract with its own verified implementation. Changing an execution
state projection alone cannot reactivate the retained release implementation.
