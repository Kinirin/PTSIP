# PTSIP release control

`composition-contract.json` is the current release selection and Package composition contract. CTRL owns publication admission, ASSURE owns the verification contract in `test/`, INFO owns independent version identities, and RECORD owns release-note routes. `release-contract.json` retains historical version syntax and bootstrap metadata; it does not reactivate the old publication surface.

Root `pyproject.toml` owns the current composition in `[tool.ptsip.release]`.
Tool binds exactly to `[project].version`; Profile and registered engine versions,
Package numeric base and immutable Specification bindings are declared centrally.
The engine-local version files, Cargo metadata, engine index, runtime constants
and Profile current registries are checked projections. Missing declarations and
conflicting projections fail; the reader never selects an old distributed source
as a substitute authority. `[tool.ptsip.release.published]` records an observed
original-source publication separately from the current source declaration.

Only `tooling-release.yml` publishes. Only `tooling-test.yml` executes verification, through modes registered in `.github/test_modes.yaml`. The test runner, test cases and mechanical verification contract are owned by `test/`. Publication downloads the original verified artifacts and does not invoke Cargo, build a wheel, or run a test suite.

Selection supports `package`, `tool`, `profiles`, and comma-separated `engine:<registered-kind>` values. Package selects Tool, Profiles and all current canonical engine registrations. The eight contract release types cover the full Package, each single component kind, and the approved mixed selections. Adding an engine changes canonical registration and its admitted implementation; it does not require another workflow.

Windows x64 and Linux x64 run in parallel and both must pass for every selected component. macOS arm64 is opt-in through `-OptionalPlatforms macos-arm64`. An omitted host produces no PASS evidence. Rust target and dependency caches are partitioned by component, host, compiler and Cargo dependency digest. Exact keys include the source SHA; compatible source changes restore only a dependency-compatible prefix. Tests and builds run on cache hits.

## Commands

Run from the checkout with Go and GitHub CLI available:

```powershell
./.github/checks/release/run_release.ps1 -Action Select -Targets package
./.github/checks/release/run_release.ps1 -Action DispatchTest -Targets engine:context,engine:projection
./.github/checks/release/run_release.ps1 -Action DispatchRelease -Targets engine:context,engine:projection -Destination github-pypi
```

The release run finds the newest successful tooling-test run with the exact source and requested selection. Run IDs and Release IDs are internal evidence, not workflow inputs. The controller verifies original Actions archive digests, selection digest, component versions, host receipts and file hashes. Different selections, expired artifacts and advanced publisher source fail admission. If no matching test exists, the action reports the separate tooling-test dispatch required. Publication does not run tests. An unchanged historical engine version reuses its original immutable files only after comparing its registered source boundaries. Changed engine source requires an explicitly assigned new version.

One source SHA has one GitHub carrier, tagged `ptsip-release-<full-source-SHA>`. Select all desired components before publishing that carrier. A retry reuses identical assets; another selection at that same source cannot replace the published carrier. Existing historical engine tags and Releases remain immutable.

Release creation returns the numeric ID used for direct reads, asset uploads and
publication. The publisher does not wait for the new draft to appear in a list.
The controller resolves an existing same-source draft automatically. A publisher
source change before publication keeps the carrier as a draft.

The owner-requested replacement of the deleted grouped Release is defined by
`publication-evidence.json`. It restores the original admitted Actions bundle
only after rechecking its successful GitHub publication, exact original test
selection, current versions, unchanged artifact producer boundaries and file
hashes. The original source tag and Package identity are preserved; the current
publisher SHA is recorded separately. It regenerates presentation assets and
does not rebuild already published Tool files or treat old tests as current-source
verification. An existing live carrier is never overwritten by this recovery.

The PyPA action is an upload adapter. Native Go compares existing version-specific PyPI file hashes before any mutation, stages only missing files, and confirms public file hashes after upload. A conflicting filename fails and requires a new component version. Tool 0.3.8a6 is published; 0.3.8a5 remains immutable. Engine Pending Trusted Publishers must match their exact project and component environment in `pypi-publishers.json` before the PyPI phase can succeed.

Publisher environment routing now comes from `pypi-publishers.json`: Tool uses
`pypi`; each engine uses `pypi-engine-<kind>`. Pending publishers have a unique
owner/repository/workflow/environment tuple, so different new projects cannot all
use the Tool's tuple. Register each engine's exact project name and matching
environment under the same single `tooling-release.yml` workflow. PyPI account
registration is an external prerequisite; the release code does not fabricate it.

## Package history and installation

The approved numeric base is `0.3.8`. Each changed successful component composition produces an opaque six-hex mapping of canonical component versions, original sources and artifact digests. The numeric base is never incremented automatically. Identical composition is idempotent; a suffix collision fails closed using the full SHA256.

Published source carriers contain the cumulative `package-registry.json` and exact snapshot. Draft carriers do not participate in latest resolution. The publication record job writes the derived registry and a separate `docs/releasenote/PTSIP_Package/<version>.md` note after observing publication. The initial installable Package requires Tool, Profiles and all seven default engines; an incomplete bootstrap composition fails installation.

```powershell
./.github/checks/release/run_release.ps1 -Action ResolvePackage -PackageVersion 0.3.8
./.github/checks/release/run_release.ps1 -Action Install -PackageVersion 0.3.8 -PythonExecutable C:/consumer/.venv/Scripts/python.exe
./.github/checks/release/run_release.ps1 -Action Install -PackageVersion 0.3.8.<SHA6> -PythonExecutable C:/consumer/.venv/Scripts/python.exe
```

The base chooses its latest published composition; an exact suffix pins its snapshot. Unknown selectors fail before installation. This is a dedicated Package command, not an extension to pip version syntax. All selected artifacts are hash-checked before pip runs. Profile distributions contain only the public catalog resources and contract registry, excluding history and Python implementation. Catalog resources are installed into the explicitly chosen Python environment. Local `.ptsip/profiles` declarations and runtime activation remain consumer-owned and are not inferred by this installer.

## Notes and verification scope

Tool notes use `docs/releasenote/tool/<version>.md`; Profile notes use `docs/releasenote/project-profile/<version>.md`. Each engine owns `docs/releasenote/engine/<kind>/<kind>-engine-<version>.md`, without a draft suffix. Multiple engines retain separate note files; the single carrier body presents their sections. Historical Specification notes retain their original content under the renamed `PTSIP_Package/` directory and are not evidence of Package publication.

`release-presentation.json` fixes the public title to `PTSIP_Package <version>`
and the section order to Release, Tool, Profile, Engine. The summary uses
`PTSIP_Tool`; each engine section includes its registered Engine ID, Capability,
Purpose and Contract changes. The new public summary omits SPEC_REVISION rather
than inventing a numeric replacement for an immutable Git revision. Full source
SHA provenance remains in the selection and shipping manifest.

The retired common verification gap remains separately open. Passing the registered release mode establishes only its selected component/source scope. Legacy scripts and their old PASS records do not become successor verification authority.
