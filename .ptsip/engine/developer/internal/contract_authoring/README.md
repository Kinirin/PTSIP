# Independent policy rules and Responsibility-Authority authoring

The fourteen contracts in `.ptsip/contracts/policy/Rules/{root_family}_rules.json`
select the same `Policy_Rules.Definition` authoring model. The selected family
requires an array of separate records with mandatory `id` and `rule`
and an optional `preconditions` list of registered C-ID condition references:

```yaml
Policy_Rules:
  Definition:
    - id: R-ID-MPD-GOV-0006-OBLIGATION-FLATTEN_POLICY_STATUS
      rule: GOV.MPD-GOV-0006.OBLIGATION.FLATTEN_POLICY_STATUS
      preconditions:
        - C-ID-POLICY_STATUS_SERIALIZATION
  Constraints: {}
```

The policy document is *not* the source of preconditions. Its list is a
non-authoritative selector and must exactly match the corresponding
`component_bindings[R-ID].condition_id` list in the Responsibility profile.
Missing or extra condition references are rejected regardless of ordering.
All 14 Family contracts have retired
`Constraints.preconditions` and `Constraints.dependencies` as authoring
fields. The `Constraints: {}` shell remains required by the currently
approved shared Family structure; removing that shell is a separate decision.


Each `id` must be registered under the exact same R-ID key in
`.ptsip/profiles/Responsibility-Authority-list.yaml`. Each `rule` must
match the registered rule constant under that key. No new rule ID, rule state,
condition or approval is generated from a plain string.

```yaml
component_bindings:
  R-ID-MPD-GOV-0006-OBLIGATION-FLATTEN_POLICY_STATUS:
    rule: GOV.MPD-GOV-0006.OBLIGATION.FLATTEN_POLICY_STATUS
    Rule_state: APPROVED.provisional
    condition_id:
      - C-ID-POLICY_STATUS_SERIALIZATION
```

The profile's `Rule_state` belongs to one registered rule constant. A null
state explicitly records a missing source state and is *not* approval. Original
rule texts in `source_text`, alongside `source_policy` and `source_rule_id`,
are provenance-only migration evidence. This metadata must not create authority.

`.ptsip/contracts/Responsibility-Authority/Responsibility-Authority.json`
declares reusable conditions and operational authorities. Condition records
refer to the independent `Condition.schema.json`; their `applies_when`
values are not duplicated in policy Definition records. The
`Operational_Authority.schema.json` contract owns the representation of
`Rule_state` used by the profile. Condition evaluation, approval, Seal and
Activation remain separate and cannot be inferred from authoring validation.

The old `responsibility_refs`, `Responsibility_Refs`, `component_id`,
`component_map`, component owner metadata and component materialization
operation are retired. No compatibility fallback recreates these declarations.

To inspect or validate contracts from the repository root:

```powershell
go -C .ptsip/engine/developer run . --root ../../.. --validator policy-rules --operation describe --family GOV
go -C .ptsip/engine/developer run . --root ../../.. --validator policy-rules --operation check --input developer/policy/GOV/MPD-GOV-0006.yaml
go -C .ptsip/engine/developer run . --root ../../.. --validator responsibility-authority --operation check
```

`generate` requires explicitly supplied rule Definition values; the tool
never invents IDs, grants or states. It creates a new output file only when
a path is provided and never overwrites an existing file. Validation uses the
installed PTSIP JSON Schema engine; Python/PyYAML is used for serialization
and rejects duplicate mapping keys.

Existing RETIRED policy sources outside GOV are historical; a family contract
change alone does not reactivate or rewrite their old policy authority.
