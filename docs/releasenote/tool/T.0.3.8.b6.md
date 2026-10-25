# PTSIP Tool T.0.3.8.b6 — source preparation (not published)

Expected PyPI prerelease identity: `0.3.8b6`. The canonical Package/Tool composition
has **not** been migrated or verified for this candidate. The current `main`
`pyproject.toml` still carries a separate older source composition, while
published `0.3.8b5` was built from a different release line. Do not infer
b6 readiness or publication from this note.

## Prepared on main

- Canonical six Compile-Sealed RBAC schema identities:
  `.ptsip/contracts/schemas/compile-sealed-rbac/*.schema.json`.
- Distribution resources:
  `src/ptsip/specdata/compile-sealed-rbac/*.schema.json`.
- Canonical central RBAC registry field requirement contract:
  `registry/compile-sealed-rbac-registry-fields.json`, mirrored in
  `src/ptsip/specdata/compile-sealed-rbac-registry-fields.json`.
- `setup.py`, `pyproject.toml`, and `MANIFEST.in` package the resources
  and check source/embedded file equality.
- `src/ptsip/specdata/ptsip-registry.yaml` matches its canonical
  `registry/ptsip-registry.yaml` source.

All six original canonical schema files remain **fail-closed scaffolds**.
The non-normative draft companion files below are available for field-shape
experimentation and tests; they are NOT replacements for the effective schemas:
- `central-registry.draft.schema.json`: approved four kinds and path syntax,
  plus provisional ID/revision/metadata shape.
- `responsibility.candidate.schema.json`, `condition-atom.candidate.schema.json`,
  `permission-atom.candidate.schema.json`, `binding.candidate.schema.json`.
- `sealed-contract.candidate.schema.json`, `activation.candidate.schema.json`
  (derived artifact candidates, NOT central registry source entries).
- `candidate-index.json`: machine-readable implementation-only mapping,
  explicitly `effective_policy_override=false` and
  `rbac_authorization_permitted=false`.
- `schemas/ptsip-profile-pp-2.00.candidate.schema.json`: opt-in declaration
  candidate. pp.1.02 remains the current registered profile.

Rust `rbac.rs` now includes a draft JSON registry source preflight which
rejects duplicate registered IDs, unapproved kinds, escaping/missing paths,
source ID/kind mismatches and revision mismatches. YAML source paths are
legal but fail closed during content verification until a native YAML decoder
is available. These source checks do not validate approval evidence, authentic
Seal, Activation or executable authority. The RBAC operation remains unregistered.



## Implementation and verification record (provisional)

- GitHub source/embedded parity checks: PASS for the six candidate schema pairs;
  PASS for the six existing fail-closed schema pairs; PASS for the draft central
  registry and candidate index pairs.
- Repository read-only static assertions: PASS (17 assertions) for registered
  semantics, candidate packaging declarations, Rust draft identity comparisons,
  existing pp.1.02 current binding, and publication guard preservation.
- An **offline reconstruction** of the candidate JSON Schema structures was
  validated with `jsonschema` and pytest: **8 tests passed**. This was NOT an
  execution of the checked-out GitHub test suite and cannot be submitted as
  official exact-source release evidence.
- Rust cargo tests: **COMMAND REPORTED, PASS/FAIL OUTPUT NOT SUPPLIED**.
  The user listed the intended `cargo test` command alongside a successful
  package build; the exact test result was not supplied. Rust/Cargo tests
  were not executed by this assistant in the current environment.
- Repository pytest including
  `src/tests/ptsip/contracts/test_rbac_candidate_contracts.py`:
  **COMMAND REPORTED, PASS/FAIL OUTPUT NOT SUPPLIED**. No checked-out
  source pytest logs or complete summary were shared.
- Go developer policy/release tests: **COMMANDS REPORTED, PASS/FAIL OUTPUT NOT SUPPLIED**.
- Local wheel/sdist build: **USER-REPORTED COMPLETION (NOT INDEPENDENTLY VERIFIED)**.
  User executed `python -m build --sdist --wheel` from
  `C:\\Users\\rhkrt\\Documents\\GitHub\\ptsip` and reported:
  - `dist/ptsip-0.3.8.1b1-py3-none-any.whl`
  - `dist/ptsip-0.3.8.1b1.tar.gz`
  These are current development-source `0.3.8.1b1` artifacts, **not**
  proposed release `0.3.8b6` and not proof of seven native engines in
  a correct Windows/Linux platform wheel.
  Source snapshot SHA, hashes, package file list and build logs not supplied.
- Windows/Linux native engine composition, wheel installation, exact-source
  multi-platform GitHub Actions: **NOT VERIFIED**.
- Release admission: **BLOCKED** by current effective release control plane
  (`.github/workflows/tooling-release.yml` is design-only and explicitly
  fails publication; `.ptsip/contracts/release/admission.json` has
  `publication_authorized=false`).
- The `main` `pyproject.toml` still declares `0.3.8.1b1`, not `0.3.8b6`;
  a b5-to-b6 package/engine composition reconciliation has not been verified.

### Commands for exact-source verification after checkout

```bash
python -m pytest -q src/tests/ptsip/contracts/test_rbac_candidate_contracts.py src/tests/ptsip/contracts/test_embedded_specdata.py
cargo test --manifest-path src/ptsip/engines/validation/Cargo.toml
go -C developer/tests test ./schema_validation -count=1
go -C .ptsip/engine/developer test ./release -count=1
python -m build --sdist --wheel
```

These commands are a verification checklist, not a release admission grant.
Native Windows and Linux wheel integration and the registered GitHub
`tooling-test` mode must also be verified from the exact source SHA before
release policy can admit publication. No distribution was published as part
of these changes.

## Historical b1 dependency resolver failure (source dependencies since retired)

The user inspected a **previously built** `0.3.8.1b1` pure-Python wheel:
it contains 13 RBAC schema resources and requires seven separate native engine
pip distributions. That wheel was created **before** the integrated delivery
source update and remains historical evidence; it is not the new Wheel output.

**The earlier b1 dependency dry run failed.** The user ran
`python -m pip install --dry-run --only-binary=:all:
dist/ptsip-0.3.8.1b1-py3-none-any.whl`, and pip reported:

```text
ERROR: Could not find a version that satisfies the requirement
ptsip-engine-context==0.2.0 (from ptsip) (from versions: 0.2.1)
ERROR: No matching distribution found for ptsip-engine-context==0.2.0
```

The canonical `main` source composition still specifies Context/Projection
0.2.0 and the other five engines 0.3.0. The published b5 complete Package
instead identifies Context/Projection 0.2.1, Storage/Discovery/Planning/
Execution 0.3.1 and Validation 0.3.3. This is a source-version reconciliation
problem, not evidence of missing RBAC schema packaging. The pip resolver
stopped at Context, so no success or failure is yet asserted for resolving the
other six `main` pins.

**No automatic version repair.** The approved source-to-version contract
(`.ptsip/contracts/release/composition-contract.json`) requires an explicit
new version for a changed native engine source, and
`developer/policy/INFO/MPD-INFO-0005.yaml` forbids engine version
auto-increment. In particular, `main`'s modified Rust Validation Engine
source cannot be represented as the already-published Validation 0.3.3
binary by merely changing `pyproject.toml` dependency pins. Align the
package's selected engine versions, canonical engine registry, installation
plan, per-engine projections and exact-source artifact evidence under the
existing release contract. Do not mutate immutable historical b5 records.

User-reported results are input evidence only; this failure does not attest
an exact-source, multi-platform release build or authorize publication.

## Published b5 comparator (user-observed pip dry run and repository source comparison)

The user executed:
`python -m pip install --dry-run --ignore-installed --only-binary=:all: "PTSIP==0.3.8b5"`
on Windows. Pip selected cached
`ptsip-0.3.8b5-py3-none-win_amd64.whl` and completed dependency resolution:
`Would install PTSIP-0.3.8b5 PyYAML-6.0.3 attrs-26.1.0 jsonschema-4.26.0 jsonschema-specifications-2025.9.1 referencing-0.37.0 rpds-py-2026.9.1`.
No independent `ptsip-engine-*` distributions were in the reported resolver
plan. This is a **successful dry-run only**, not evidence of installed or
executed engines.

GitHub comparison against original b5 delivery branch
`dev/0.3.8b2` shows:
- `pyproject.toml`: project `0.3.8b5` with PyYAML/jsonschema runtime
  dependencies, no seven `ptsip-engine-*` requirements.
- `.github/workflows/tooling-release.yml`: host-tagged, integrated PTSIP
  wheel and source distribution checks for the b5 single-project release.
- Earlier `main` before the integrated-delivery migration: project
  `0.3.8.1b1` required seven independent engine pip distributions.
- Current `main` after the migration: those independent engine pip dependencies
  were removed; the versioned seven native component identities remain in
  `[tool.ptsip.release.engines]`, and publication is still blocked.

These are two different distribution mechanisms, not interchangeable
versions of the same wheel layout. The user-requested b6 goal requires
source/version/artifact composition reconciliation with the effective
release policy before any new publication workflow can be activated.
Neither the historical b5 delivery branch nor its release evidence is
modified by this record.

## Integrated target Wheel source migration (not a release)

The owner explicitly selected reconciliation: one target-specific PTSIP Wheel
per supported platform, retaining seven native components with independent
versions, verification and source-bound identity. Release verification remains
under `.ptsip/engine/developer/release/`. The superseded
`.github/checks/release/controller/` is **not** restored.

Implementation currently committed to `main`:
- `src/policy/CNTR/SFP-CNTR-0005.yaml`: active integrated consumer delivery
  normalized, obsolete bundle-forbidden flag retired; immutable historic
  distribution choice left intact.
- `pyproject.toml`: removed seven `ptsip-engine-*` runtime pip dependencies;
  PTSIP and every engine version are unchanged (`0.3.8.1b1`,
  Context/Projection 0.2.0, other five 0.3.0).
- `src/ptsip/engines/contracts/installation-plan.json`: v2 integrated
  seven-component plan, with complete host Wheel, exact-source verification,
  no standalone engine pip install and no publication authority.
- `src/ptsip/engines/planning/src/installation.rs`: integrated candidate
  selection checks component identity, host evidence and native artifact
  hashes. The previous published `planning.select-installation` operation
  is **fail-closed** pending an explicitly versioned operation cutover.
  The candidate itself cannot verify complete assembled Wheel or admit release.
- `src/ptsip/engine_installation.py`: read-only integrated bundle origin
  inspection, no fallback to external engine distributions.
- `src/tests/ptsip/cli/test_engine_installation.py` and
  `src/tests/ptsip/contracts/test_integrated_installation_contract.py`:
  consumer negative checks and source contract alignment assertions.

The b5 single-project native packaging and verification modules have now
been **ported as implementation** into the current developer release owner,
not linked to or restored under `.github/checks/release/controller/`:

- `.ptsip/engine/developer/release/controller/package_wheel.go`:
  `BundledPackageWheel`, all-seven engine origin/digest checks, Windows/Linux
  wheel tags, executable permissions, no engine `Requires-Dist`, wheel RECORD.
- `controller/pypi_archive.go`: PyPI-safe ZIP header normalization and
  source-bound normalization helper.
- `controller/package_build.go`: host-specific complete native Wheel and
  `ptsip-pip-package-test-receipt/v2` tied to current published component
  registry and exact source; no independent engine Wheel upload from this join.
- `controller/package_admission.go`: verified receipts for every requested
  native host plus one source distribution required before admitting a PTSIP
  Package carrier.
- `controller/composition.go`: rejects obsolete `ptsip-engine-*`
  runtime pip dependencies.
- `controller/tool_project.go`: Tool-only intermediate uses setuptools;
  its build cannot accidentally invoke the complete PTSIP native backend.
- `controller/package.go`: read-only installed PTSIP host bundle origin
  and source version verification, not standalone pip engine inspection.
- `controller/publication.go`: when the complete PTSIP Package is selected,
  the derived PyPI upload project list contains only PTSIP, not intermediate
  engine Wheel artifacts.
- `.ptsip/engine/developer/release/backend/ptsip_build_backend.py`:
  b5-style native Rust `--locked` wheel build and exact-source sdist rebuild.
  `pyproject.toml` and `MANIFEST.in` include this relocated backend and
  native crate/lockfile source files.
- `.ptsip/engine/developer/release/test/`: b5 positive/negative native
  bundle tests, host/source receipt tests, PyPI ZIP normalization tests,
  source rebuild/install verifier and native PTSIP pip-install checks.
- `developer/tests/engines/independent_engine_wheel_test.go`:
  keeps independent native *build evidence* while reading the integrated
  v2 component plan rather than retired consumer pip dependencies.

**Verification status (updated, user-reported Windows local run):**
- `go -C .ptsip/engine/developer/release/test test ./... -count=1`:
  `ok .../release/test 11.872s`; CLI test package returned `[no test files]`.
  The user reported successful local execution after the test repoRoot
  correction in main commit `3c324ed5c8e7f01d4f0a437fbbfa120f414efabb`.
- `python -m build --sdist --wheel`:
  `Successfully built ptsip-0.3.8.1b1.tar.gz and
  ptsip-0.3.8.1b1-py3-none-win_amd64.whl`. This confirms a Windows
  platform-tagged Wheel and sdist were produced via the native backend.
- **Windows complete Wheel binary inventory and content hashes:
  USER-REPORTED PASS.** The user inspected
  `dist/ptsip-0.3.8.1b1-py3-none-win_amd64.whl` with Python `zipfile`
  and `hashlib.sha256`. Each of the seven `ptsip-engine-*.exe` script
  members (context, storage, validation, discovery, planning, execution,
  projection) matched its `ptsip-engines.json` recorded `binary_sha256`.
  The test also asserted the exact seven-engine key set,
  `execution_authorized=false`, and the `win_amd64` WHEEL tag.
  The user reported `PASS: 7 native engines verified`.
- **Clean Windows consumer install: USER-REPORTED PASS.**
  The user created a temporary virtualenv, installed the built local
  `ptsip-0.3.8.1b1-py3-none-win_amd64.whl` with `pip` (exit code 0),
  and called `ptsip.engine_installation.inspect()` from that environment.
  All seven native engines (CONTEXT, STORAGE, VALIDATION, DISCOVERY,
  PLANNING, EXECUTION, PROJECTION) returned
  `INSTALLED_UNATTESTED_RELEASE`. `ALL_INSTALLED=True`,
  `EXECUTION_AUTHORIZED=False`, `RELEASE_ATTESTED=False`;
  the script printed `PASS: CLEAN_WINDOWS_WHEEL_INSTALL`.
  This is an installed-file/native-source-and-digest inspection; it does
  not independently attest runnable engine operations or official release.
- **Linux release verification: PENDING WORKFLOW RUN.**
  The existing read-only verification executor
  `.github/workflows/tooling-test.yml` handles registered
  `release-verification`, `targets=package` across required
  `windows-x64` and `linux-x64` runners. Its developer release
  controller/test roots are under `.ptsip/engine/developer/release/`.
  The `registered-join` Linux job runs complete PTSIP wheel installation
  plus Linux sdist native rebuild/installation and `twine check`.
  No workflow run has been reported for the current tested source.
- The Windows local Wheel SHA256 and exact source checkout SHA were
  not separately supplied, and CI attestation/record inspection remain open.
  Remaining Go/controller/developer tests and Python/Rust tests outside
  the reported local suite are not asserted as PASS.
- The registered workflow still intentionally rejects publication.
  `planning.select-installation` needs a versioned operation cutover.
  Historical b5 assets remain immutable.

**This is user-reported local verification evidence, not release admission or
publication.** Candidate/beta6 source preparation is not a completed release.

Required local tests and release preflight:

```powershell
git pull --ff-only origin main
go -C .ptsip/engine/developer/release/controller test ./... -count=1
go -C .ptsip/engine/developer/release/test test ./... -count=1
go -C developer/tests test ./engines -count=1
python -m pytest -q src/tests/ptsip/cli/test_engine_installation.py src/tests/ptsip/contracts/test_integrated_installation_contract.py
cargo test --manifest-path src/ptsip/engines/planning/Cargo.toml
python -m build --sdist --wheel
```

`python -m build` now selects the native backend and requires Rust/Cargo,
a supported platform and a clean exact source. Its local Wheel must include
seven native executables and the distribution's `ptsip-engines.json`; merely
observing a `py3-none-any` Tool intermediate is not complete Package proof.

## 2026-10-26 workflow diagnostics — read-only verification cutover

GitHub Actions `tooling-test` run 40224599600, mode
`release-verification`, targets `package`, source
`d0cbca451d01fa97ae9033653410ddf1102b2ad1`: **FAILED
before Linux/Windows matrix launch.** Job
`mode-selection` 121263190756 failed in
`Resolve registered mode and execution contract`. Log:
`developer/automation/release@v0.0.0` replaced to deleted
`../../.github/checks/release`; missing `.github/checks/release/go.mod`.
All subsequent registered verification and join jobs were skipped.
This is NOT native Linux engine failure evidence.

Current main implements a bounded fix:
- `.ptsip/engine/developer/release/test/runner.go` does not invoke the
  retired `MPD-ASSURE-0001` policy resolver. It checks existing
  `.github/test_modes.yaml`, read-only
  `.ptsip/contracts/release/verification-contract.json`, and the
  independent publication-denied
  `.ptsip/contracts/release/admission.json` before selecting exact-source
  Windows/Linux test jobs.
- Derived selection output explicitly reports
  `policy_authority=NOT_ASSERTED`,
  `verification_scope=SOURCE_BOUND_TEST_EVIDENCE_ONLY`, and
  `publication_authorized=false`. This is **non-normative test
  execution**, not retroactive ASSURE policy activation or release
  admission. Any missing/different contract or publication authorization
  fails closed.
- Obsolete `developer/automation/release` module imports/replacements
  removed from `developer/automation/go.mod`,
  `developer/automation/cmd/ptsip-dev/main.go`, and
  `developer/tests/go.mod`. The deleted `.github/checks/release/`
  controller was NOT reinstated.
- `.ptsip/engine/developer/release/test/verification_route_test.go`
  adds three regression tests for valid nonnormative selection,
  missing/changed admission, and legacy/publication denial.

Static remote source checks of nine migration invariants passed.
**No post-fix end-to-end tooling-test run or Linux artifact evidence
has yet been received.** The historical failed run cannot validate
the newer source; a fresh `tooling-test` run on updated main is
required. All official publication gates remain disabled.

## Release blockers / explicit scope limits

- Approve complete field/lookup/serialization contracts before enabling
  `pp.2.00`, registering `validation.rbac-preflight`, or claiming RBAC support.
- Do not mark `pp.2.00` as current in `project-profile-contracts.yaml`
  until its full schema and compatible runtime/profile migration exist.
- Existing `src/ptsip/repository/schemas/engine-state.schema.json` and
  `repository-index.schema.json` do not own these RBAC contracts.
- Resolve the b5-to-b6 package and seven native-engine composition, then
  run source, wheel, sdist, installation and workflow verification against
  the exact release source. Publication is a separate verified action.

**Status: SOURCE_ASSETS_STAGED; RELEASE_NOT_VERIFIED; NOT_PUBLISHED.**
