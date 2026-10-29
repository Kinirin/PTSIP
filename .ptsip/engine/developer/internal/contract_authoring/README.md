# Contract authoring

The developer CLI selects a rules contract from the exact `root_family.json`
vocabulary. Family requirements live in
`.ptsip/contracts/policy/Rules/{root_family}_rules.json`. The shared rules schema
keeps every field optional until `Policy_Identity.root_family` selects a family.
An omitted family creates no fields and no output file. Unknown families and
missing contracts fail without searching for an alternative.

Run the following commands from the repository root. Paths passed to `--input`
and `--output` are relative to `--root`; `--input -` reads JSON or YAML from stdin.
Generation prints a JSON report containing the document. An explicit `--output`
writes a new `.json`, `.yaml` or `.yml` file in an existing directory. Existing
files are never replaced.

```powershell
go -C .ptsip/engine/developer run . --root ../../.. --validator policy-rules --operation generate
go -C .ptsip/engine/developer run . --root ../../.. --validator policy-rules --operation describe --family GOV
```

The first command reports `OMITTED`. The second shows the selected contract's
field requirements. To generate a GOV rules document, supply the required scalar
values. The generator creates required object and array containers and preserves
supplied optional fields. Approval values, rule identities and role mappings
must be supplied explicitly.

```powershell
@'
Policy_Rules:
  Identity:
    contract_id: urn:ptsip:policy:rule:gov:example
    Rule_ver: '1'
  Definition:
    meaning: Example policy
  Boundary:
    scope: Example scope
'@ | go -C .ptsip/engine/developer run . --root ../../.. --validator policy-rules --operation generate --family GOV --input - --output example-policy.yaml

go -C .ptsip/engine/developer run . --root ../../.. --validator policy-rules --operation check --input example-policy.yaml
```

This checks the selected rules structure. Full policy identity and lifecycle
validation remain the shared public policy schema's responsibility.

Reusable conditions and operational authorities are declared in
`.ptsip/contracts/Responsibility-Authority/Responsibility-Authority.json`:

```json
{
  "schema_version": "ptsip-responsibility-authority/v1",
  "condition_list": [
    {"ID": "C-ID-review", "name": "Review is complete"}
  ],
  "authority_list": [
    {"ID": "OA-ID-edit", "name": "Edit policy", "action": "edit", "scope": "example"}
  ]
}
```

The repository starts with empty lists. After explicitly declaring the referenced
IDs, a mapping request can contain the following YAML. The profile's existing
`ptsip` metadata and `responsibility_map` are retained when omitted from a request.

```yaml
responsibility:
  - name: Policy editor
    ID: R-ID-editor
    condition_id:
      - C-ID-review
    Operational_Authority:
      - OA-ID-edit
    role_name: Editor
    runtime_required: false
```

```powershell
go -C .ptsip/engine/developer run . --root ../../.. --validator responsibility-authority --operation generate --input mapping-request.yaml --output mapping-profile.yaml
go -C .ptsip/engine/developer run . --root ../../.. --validator responsibility-authority --operation check --input mapping-profile.yaml
go -C .ptsip/engine/developer run . --root ../../.. --validator responsibility-authority --operation check
```

The final command checks the editable
`.ptsip/profiles/Responsibility-Authority-list.yaml`. Copy reviewed mappings into
that file manually when ready. Generation without an input creates an empty list.
Both the profile and each family contract can refer to responsibility IDs; a
policy uses `Policy_Rules.Responsibility_Refs`. Nonempty references must resolve
through the editable profile to declared conditions and authorities. IDs must
be unique and use `R-ID-`, `C-ID-`, or `OA-ID-` prefixes. `runtime_required` is a
boolean. Extension fields are allowed by the new authoring schemas.

Validation uses the installed PTSIP `validation.schema` operation, with explicit
local schema resources. Python and PyYAML are used only for serialization and
reject duplicate YAML/JSON mapping keys. No network schema resolver, retired
authority category, or legacy validator participates.

These commands validate declarations and references. Condition evaluation,
binding approval, cryptographic sealing and runtime activation retain their
separate contracts; authoring never grants execution permission. The existing
six unresolved runtime schema scaffolds remain unchanged. The `pp.2.01` list is
an authoring profile and does not activate a new distributed product operation.


## Local component responsibility metadata

The local profile catalog registers `main.ptsip.yaml` as the project source and
`Responsibility-Authority-list.yaml` as an auxiliary authoring resource. The
auxiliary list is not selected as a consumer Project Profile. Main component
classification, selectors, purpose, delivery flags and relationships remain in
the main source; `component_bindings` owns the original component `roles`,
`release_owner`, `compatibility_owner` and `runtime_required` declarations.
Every main component must have exactly one binding. Missing, duplicate, unknown
or conflicting declarations fail the local check. Removing the whole binding
map from a linked local list also fails.

These owner labels are declaration metadata. They do not manufacture condition
IDs, responsibility IDs, operational authority grants or runtime permission.
The separately authored `responsibility` list and condition/authority contracts
remain explicit. Generation without an input copies no component bindings.

```powershell
go -C .ptsip/engine/developer run . --root ../../.. --validator responsibility-authority --operation check
go -C .ptsip/engine/developer run . --root ../../.. --validator responsibility-authority --operation materialize
go -C .ptsip/engine/developer run . --root ../../.. --validator responsibility-authority --operation materialize --output .artifacts/profiles/main.ptsip.yaml
```

Materialization explicitly joins the two sources and validates the returned
pp.1.02-shaped document against the existing distributed source schema. It
preserves the authored component order and values and uses the existing
no-overwrite writer when an output is requested. The report declares
`projection_authority: NONE` and `execution_authorized: false`. The projection
is derived; neither authored source is rewritten. Generic consumer code does
not automatically inherit this developer-only composition or its owner labels.
