# Policy semantic review helper

This is a non-normative authoring guide. It does not own policy semantics,
approval, contract registration or execution authority. GitHub Actions inputs,
artifacts and web-agent access are documented in [ACTIONS.md](ACTIONS.md).

## Review and authoring sequence

1. Resolve the exact policy with `contract-entry --dry-run`. Use its registered
   public schema, rules schema, schema IDs, Family profile and active registry.
   Version labels alone are insufficient: historical Family contracts also
   contain a `v2` label. Unresolved selection permits historical explanation only.
2. Read the target in full. Select relevant historical sections explicitly with
   exact file paths and JSON Pointers. Preserve selection reasons for reuse.
   `source_scope` does not establish which historical rules apply.
3. Collect those sections and file SHA-256 values with this helper. Distinguish
   original text, translation, interpretation, differences and unresolved items.
   Preserve obligations, prohibitions, permissions, conditions and exceptions.
4. Present the semantic review for the user's new declaration. Translation or
   structural validity does not approve a policy change.
5. Map approved content to the fields of the actual selected schema. Current
   `Policy_Rules.contracts[]` owns rules, requirements, source items and optional
   authority declarations. Read the exact Family field profile; do not recreate
   the historical `Identity/Definition/Boundary/Evaluation` rule structure.
6. Report contract limitations before extending fields: the unrepresentable
   meaning, existing-field alternatives, responsibility separation, reusability
   within the Family, common-contract effects and the exact approval boundary.
   Do not invent authority, conditions or one-off fields to fill unresolved gaps.
7. Use the registered `policy-rules` authoring entry. Its shared selector is a
   required input to validation and generation and is returned in the report.
   Run complete `contract-entry` / `policy-check` after writing. Report schema
   conformance separately from semantic fidelity, user approval and execution.

## Responsibility separation

Consider separate documents when work areas, decision owners, conditions,
lifecycles or evidence responsibilities can be reviewed independently. Preserve
cross-document obligations and exceptions, and avoid duplicate rules or cycles.
When separation is clear, recommend the responsible document boundaries without
waiting for a separate instruction. Ask before authoring when ownership is
ambiguous or separation has significant additional cost. Do not infer final
Family ownership, new policy IDs or approval from a tentative recommendation.

## Helper boundaries

The Go helper reuses `contractentry.Resolve` and the existing YAML serialization
entry. It writes stdout only and does not change policy files.

- `--target` captures the complete document and registered selection. A failed
  selection is preserved as `selection_problem`; historical review can continue.
- `--source FILE#/pointer` extracts only the explicit section. No recursive
  `source_scope` expansion, fuzzy lookup or relevance inference occurs. Missing
  paths or pointers fail instead of expanding extraction to the entire file.
- Schema URNs and fragments are resolved against the selected rules schema;
  they are never treated as filesystem paths. Selected schemas and registered
  schema-resource files are included in the byte-hash snapshot.
- `field_findings` compares literal `properties/items/required` in the selected
  common rules schema. Unknown fields are flagged, while complex schema branches
  and object references remain `UNINSPECTED_SCHEMA`. The Family profile is
  resolved, but this inventory does not validate Family conditions or nested
  referenced contracts. Use the registered validator for full conformance.
- Open objects, scalar values, conditions, permissions, reusability and semantic
  responsibility separation remain subject to existing validation and review.
  JSON Schema allowing an extra field is not user authorization to add it.
- `--verify` checks captured file bytes only. It does not prove complete graph
  freshness, authenticity, relevance or translation fidelity. Resolve current
  contracts again for each new authoring operation.
- `NON_NORMATIVE_REVIEW`, `schema_validation: NOT_PERFORMED`, and exit code 0
  mean successful collection, including unresolved selections or review findings.
  Invalid inputs, missing sections and changed files exit 2. `go run` can expose
  that as shell exit 1 with `exit status 2`.

## Commands

Run from the repository root. `PTSIP_PYTHON` selects the existing YAML/runtime
interpreter when the system default is unsuitable.

```powershell
$env:PTSIP_PYTHON = (Resolve-Path .venv/Scripts/python.exe).Path
go -C .ptsip/engine/developer run ./policy/review/cmd/policy-review --root ../../.. --target developer/policy/RECORD/MPD-RECORD-0001.yaml --source 'developer/policy/legacy/MPD-WORK-0002.yaml#/rules/work_execution_lifecycle/execution_output'
go -C .ptsip/engine/developer run . --validator contract-entry --root ../../.. --target developer/policy/RECORD/MPD-RECORD-0001.yaml
```

Existing files under `examples/` are historical review/evidence snapshots. Their
old contract paths or fields describe those captured inputs, not current
serialization guidance. Regenerate review evidence before a new policy rewrite.
The independent [policy contract guard](../../README.md#policy-authoring-guard)
rejects current policy edits that use the old structure and additions or
modifications in the historical `Rules/` contract tree.
