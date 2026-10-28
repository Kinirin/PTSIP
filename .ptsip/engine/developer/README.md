# Developer validator modules

The repository developer entrypoint is `go -C .ptsip/engine/developer run .`.
`index.json` routes its registered validators to the three domain indexes:

- `Distribution/index.json`: release version tooling, verification categories,
  integration tests and future branch verification.
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
`COMPATIBILITY_BLOCKED` identifies relocated release code that still contains
legacy authority dependencies and is not a successor verification contract.
`PLANNED` modules have no implemented verification claim. Category indexes under
`release/test` retain the shared integration suite until its coupled helpers can
be separated; the self-contained version tests live under `test/tool`.

## Layout

```text
Distribution/
  index.json
  release/
    version/versiondiff.go
    verification/{releasenote,tool,engine,package,profile,admission}/
    test/{releasenote,profile,engine,package,tool}/
    controller/                     # relocated compatibility module
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
admit lifecycle transitions, or activate responsibilities. Responsibility checks
load only their five authoring schemas, declaration contract and selected
profile. A policy joins responsibility declarations through explicit
`responsibility_refs`.

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
go -C .ptsip/engine/developer/Distribution/release/test run ./releasenote --root ../../../../../.. --action verify
```

The root module does not implicitly traverse nested Go modules. Run each listed
module for migration verification. Structure validation, implementation tests,
remote CI, release admission and runtime permission are separate results.
