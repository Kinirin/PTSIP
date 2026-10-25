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
