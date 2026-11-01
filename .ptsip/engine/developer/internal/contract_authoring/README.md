# Current contract authoring entry

The repository-local entry is `.ptsip/index.json` → `.ptsip/contracts/index.json`.
Policy selection follows `policy/boundary.json`, `policy/routing/policy_routes.json`,
`policy/candidate.index.json` and `policy/Policy_Rules/Family_Selection.json`.
The declared `Policy_Identity.policy_kind` and `root_family` select the route.

Current schemas are owned by `.ptsip/contracts/policy/` and
`.ptsip/contracts/compile-sealed-rbac/`. The former
`contracts/schemas/Policy/` and `contracts/schemas/compile-sealed-rbac/` trees
are historical copies and are excluded from current loading and scanning.

## Policy documents

The public v2 schema owns `Policy_Identity`, `Policy_Lifecycle` and `Policy_Rules`.
`Policy_Rules.contracts[]` groups contract identity/version, independent rules,
requirements, original source items and optional authority declarations.
Family field profiles belong to `Public_Policy_Rules.schema.json`; the public
envelope delegates once. Read the selected schemas for exact required fields.

A rule identifies its responsibility with `id` (R-ID) and `rule`. Its
`source_ref` is an array of local `source_items[].item_id` values. Its optional
`authority_ref` is an array of IDs in the same contract's `authority_ref[]`.
Original authority coordinates or unresolved selection bindings are recorded
in that contract catalog. ID matches never establish an authority grant.
Requirements use `requirement_id`; empty optional fields are omitted and
pending fields remain explicitly unresolved.

The `policy-rules` authoring interface validates supplied complete v2 policy
content. `generate` preserves that content and does not synthesize missing
scalars, empty containers, authority, responsibilities, or semantic decisions.
An empty request without a Family selection returns `OMITTED` and writes no
file. Missing or conflicting declarations fail closed.

```powershell
go -C .ptsip/engine/developer run . --validator policy-rules --root ../../.. --operation describe --family GOV
go -C .ptsip/engine/developer run . --validator contract-entry --root ../../.. --target developer/policy/GOV/MPD-GOV-0001.yaml --dry-run
go -C .ptsip/engine/developer run . --validator contract-entry --root ../../.. --target developer/policy/GOV/MPD-GOV-0001.yaml
```

`contract-entry` / `policy-check` performs the current schema and declared local
reference checks. Structural validity is separate from source equivalence,
relation applicability, requirement fulfillment, lifecycle approval and
runtime authorization. Historical four-block rules and v7 schemas are not
fallbacks for an unresolved current contract.

## Responsibility and authority

The profile binding remains `.ptsip/profiles/Responsibility-Authority-list.yaml`.
Its schema is `contracts/compile-sealed-rbac/Responsibility/Responsibilities_List.schema.json`.
Independent condition and authority owners are in `Conditions_exercise/` and
`Authority/`; exact IDs and vocabularies are read from those machine contracts.
The `responsibility-authority` interface resolves independently of policy
Family selection. Missing authorization registrations fail closed; policy
status is not a substitute for a responsibility status or an authorization.

```powershell
go -C .ptsip/engine/developer run . --validator responsibility-authority --root ../../.. --operation check
```
