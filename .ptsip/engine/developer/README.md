# Developer validator modules

The repository developer entrypoint is `go -C .ptsip/engine/developer run .`.
`index.json` routes its registered validators to the three domain indexes:

- `Distribution/index.json`: release version tooling, verification categories,
  integration tests and branch verification.
- `policy/index.json`: identity, Policy_status, Plan_status, conditional Rules
  and limited schema inspection.
- `Responsibility-Authority/index.json`: responsibility declarations, condition
  and operational-authority references, profile generation and validation.

These indexes are **temporary, replaceable implementation contracts** authorized
for the validator restructuring. Their `replacement_required` field records that
they must be replaced or removed when successor contracts are confirmed. They
do not establish policy lifecycle approval, release admission or runtime access.
Code authoring and local structure checks do not require a retired policy
resolver, a historical approval registry, or a completed semantic validator.
Missing current resources fail their selected check without legacy fallback.

`STRUCTURE_ONLY` reports the implemented structural scope.
Release verification uses the temporary current boundary contract; publication
admission remains `UNVERIFIED` until successor authority is registered.
`PLANNED` modules have no implemented verification claim. Category indexes under
`release/test` retain the shared integration suite until its coupled helpers can
be separated; the self-contained version tests live under `test/tool`.

## Layout

```text
Distribution/
  index.json
  release/
    version/versiondiff.go
    verification/{boundary,releasenote,tool,engine,package,profile,admission}/
    test/{releasenote,profile,engine,package,tool}/
    controller/                     # artifact controller with blocked publication
    backend/                        # relocated PEP 517 backend
    state/                          # existing release data
  branch/
policy/
  index.json
  verification/
    lifecycle/{Plan_status,Policy_status}/
    Rules/
    identity/
  test/
Responsibility-Authority/
  index.json
  verification/
  test/
internal/
  contract_authoring/                # shared authoring and reference primitives
  contractcheck/                     # explicit current schema resources
  layout/                            # index and implementation-path inspection
```

`docs/`, `release/`, `schema_check/`, `ctrl_admission/` and
`contract_authoring/` at the old developer-engine root are retired locations.
README translation code moved from `developer/automation/cmd/readme-translate`
to `Distribution/release/test/releasenote` and uses the relocated test Go module.
Private package tests remain beside their Go package where they exercise
unexported implementation details. Domain integration tests live in `test/`.

## Run checks from the repository root

```powershell
go -C .ptsip/engine/developer run . --validator developer-layout-check --root ../../..
go -C .ptsip/engine/developer run . --validator release-documentation --operation check --root ../../..
go -C .ptsip/engine/developer run . --validator policy-check --root ../../.. --input developer/policy/GOV/MPD-GOV-0006.yaml
go -C .ptsip/engine/developer run . --validator policy-identity-check --root ../../.. --input developer/policy/GOV/MPD-GOV-0006.yaml
go -C .ptsip/engine/developer run . --validator policy-status-check --root ../../.. --input developer/policy/GOV/MPD-GOV-0006.yaml
'{"status":"DRAFT"}' | go -C .ptsip/engine/developer run . --validator plan-status-check --root ../../.. --input -
go -C .ptsip/engine/developer run . --validator responsibility-authority --operation check --root ../../..
go -C .ptsip/engine/developer run . --validator policy-rules --operation generate --root ../../..
```

Policy components may be supplied directly or inside their named wrapper.
`policy-check` requires all three public policy components. The checks use the
approved schema and selected family contract and do not evaluate policy rules,
admit lifecycle transitions, or activate responsibilities. Responsibility declarations use independent condition and authority schemas.
For each of the 14 Root Families, a Policy_Rules.Definition item contains only
a registered R-ID and a registered rule constant. The R-ID keyed
Responsibility-Authority-list.yaml component_bindings map owns the rule state and
condition references. The prior component ownership projection, its materialize
operation and separate responsibility references have been removed.
These references never grant execution authority.

The installed PTSIP VALIDATION engine evaluates JSON Schema. JSON/YAML
serialization uses Python and PyYAML with explicit UTF-8. The layout check needs
only Go and the local module indexes. A default Rules generation remains
`OMITTED`; supplied scalars and an explicit family are required for generation.
See `internal/contract_authoring/README.md` for the authoring interface.

## Test modules

```powershell
go -C .ptsip/engine/developer test ./... -count=1
go -C .ptsip/engine/developer/Distribution/release/test test ./... -count=1
go -C .ptsip/engine/developer/Distribution/release/controller test ./... -count=1
go -C .ptsip/engine/developer/Distribution/release/verification/engine test ./... -count=1
go -C .ptsip/engine/developer/Distribution/release/verification/profile test ./... -count=1
go -C .ptsip/engine/developer/Distribution/release/test run ./releasenote --root ../../../../../.. --action verify
```

The root module does not implicitly traverse nested Go modules. Run each listed
module for migration verification. Structure validation, implementation tests,
remote CI, release admission and runtime permission are separate results.

## Release documentation

`.ptsip/contracts/release/Platform/Common/releasenote.json` binds the four formatting schemas and
the moved translation schemas under `.ptsip/contracts/schemas/release/releasenote/`.
Formatting schemas validate a structured Markdown projection; generated notes
carry that projection as machine-readable metadata and have separate Summary,
Version changes, Technical details and Documentation sections.

`release-documentation --operation preview|generate --base <SHA> --head <SHA>`
compares the existing version declarations at two exact Git revisions. The head
must be checked out and the source/version tree must be clean. It creates notes
only for actual declaration changes, preserving previous and current values,
source paths and revision/digest provenance. It never selects a version bump.
No version changes produce `NO_VERSION_CHANGES` without a documentation write.

An optional `--input` JSON/YAML document supplies a `notes` map keyed by changed
component ID. Each entry may contain `summary`, `technical_details`, and a
`documentation` list of objects with `title` and repository-relative `path`.
Absent details use factual version/source comparisons and the registered
technical document. Unchanged or unknown component overrides fail validation.

English notes are classified below `docs/releasenote/`. Engine filenames are
`engine/{engine}/{engine}-engine-{version}.md`, profiles use
`project-profile/pp.{version}.md`, Tool notes use `tool/T.X.Y.Z[-STATE].md`, and
Package notes use `PTSIP_Package/{version}.md`. Existing historical filenames are
retained. Archives allow creation or identical replay, and conflicting contents
fail before the current README changes. `engine/README.md` is withdrawn.

The `readme-translation` workflow runs English generation first, then updates
only current targets declared in `docs/translated/index.yaml`. Translations have
no versioned archive. The existing Go translation command exposes
`documentation-check`, `documentation-preview` and `documentation-generate`
with the same contracts; its closed schema adapter requires no retired policy
resolver. The developer root CLI validates through installed PTSIP VALIDATION.

## Branch module

Branch implementation, HTTP reader, command dispatch and regression tests live
under `Distribution/branch/`. Its local `index.json` owns the temporary closed
command surface. The module has no dependency on `developer/automation`, the
retired policy resolver, policy registries or historical approval files.

```powershell
go -C .ptsip/engine/developer run . --validator branch-check --root ../../..
go -C .ptsip/engine/developer run ./Distribution/branch/cmd/ptsip-branch branch-control commands --repository ../../..
go -C .ptsip/engine/developer run ./Distribution/branch/cmd/ptsip-branch branch-control inspect --github-repository Kinirin/PTSIP --branch main --repository ../../..
```

`branch-guard validate` checks Git reference syntax and the exact request fields,
returning `STRUCTURE_VALID` with `authorization_status: UNVERIFIED` and
`branch_creation_authorized: false`. It does not recover the retired development
branch naming or retention rules. Classification and profile-transition
authority remain `UNVERIFIED` (CLI exit 4). Creation, recreation and HTTP mutation
methods are blocked before any request until a successor authority contract is
defined. The old `ptsip-dev branch-*` entry forwards to this module before any
legacy command-catalog resolution. The commit hook's PP checks are unchanged.

## Release verification boundary

`.ptsip/contracts/release/Platform/Common/verification-boundary.json` permits only contract
structure checks, local engine verification and source-bound test evidence.
The boundary has no policy authority, no verification waivers, and no permission
to publish or activate runtime behavior. Missing or changed bindings fail closed.
The official engine inventory stays limited to the seven declared engines.

`release-admission-check` returns `CONTRACT_STRUCTURE_VALID` (exit 0) for the
current structure. `ctrl-release-admission-check` returns `UNVERIFIED` (exit 4)
with `SUCCESSOR_AUTHORITY_UNRESOLVED`. Controller and engine publication entry
points reject before GitHub or PyPI publication. Exact checkout SHA, verification
run, host, receipt, digest and immutable artifact checks remain required in the
artifact implementations; structure checks do not assert those results.

Profile verification lives in `release/verification/profile`, with a small
forwarder for the shared test runner. The developer snapshot and schema resources
are under `.ptsip/contracts/release/Platform/Common/profile/` and
`.ptsip/contracts/schemas/release/profile/`. Verification compares the snapshot
against the distributed source projection and checks catalog, versions, schemas
and immutable profile baselines. Existing `schemas/...` values in the snapshot
are wire identifiers matched against that source projection; the closed loader
reads only the new schema storage directory and never resolves those values as
root paths or downloads. This snapshot does not change consumer profile authority.

Every Package selection requires the full Package Join on each selected host.
The former lightweight CLI returns `LIGHTWEIGHT_VERIFICATION_RETIRED` without
writing artifacts; the empty compatibility matrix cannot waive a Package Join.
Historical migration inventories remain audit data and are never admission input.

## Release platform contract layout

`.ptsip/contracts/release/index.json` catalogs distribution-service platforms.
`Platform/Common/` holds shared composition, source-bound verification, version,
profile and documentation contracts. `Platform/GitHub/` holds carrier evidence and
release presentation; `Platform/PyPI/` holds publisher routing. Windows, Linux and
optional macOS hosts remain declared by the shared verification contract.

Each platform has a `{Platform_name}_validation.json` schema for its catalog
routing record. These schemas describe the current resource paths and do not
assert artifact verification or publication authority. The existing release
contract schema is `Platform/Common/release-contract_validation.json`, and shared
structural rules are `Platform/Common/release_validation.json`.

All local readers use the platform paths. Historical immutable Profile ZIPs may
retain their former snapshot member name; the archive reader accepts that name
as transport data only and rejects duplicate current/historical members. It does
not read a contract from the former local directory.
