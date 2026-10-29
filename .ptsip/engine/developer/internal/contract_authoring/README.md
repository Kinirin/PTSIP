# Opaque responsibility and authorization contracts

The fourteen `.ptsip/contracts/policy/Rules/{root_family}_rules.json`
contracts select independent Policy_Rules.Definition entries. Each item refers
to an opaque R-ID, an opaque rule constant, and optionally an opaque C-ID
reference. No policy name, path or version is encoded into these identifiers.

```yaml
Policy_Rules:
  Definition:
    - id: R-ID-8D95436233BE6DCD5E90B2AAA986052C
      rule: RULE-FD125E8425C39349304D0BC566F5701E
      preconditions:
        - C-ID-514BC6B03401A855DF7B7C157645C79B
```

A policy document is not allowed to define or override preconditions. The
registered list at `component_bindings[R-ID].condition_id` in
`.ptsip/profiles/Responsibility-Authority-list.yaml` must match exactly.
The empty `Constraints` field is retired in the common schema and all
fourteen Family contracts; explicit use is rejected.

Responsibility profiles use an independent lifecycle type, not a policy
approval state. The entrypoint
`.ptsip/contracts/schemas/compile-sealed-rbac/lifecycle/index.json`
loads the `status.json` contract; only `OBLIGATION` and `PROHIBITION`
are currently allowed. A previous `APPROVED.*` Rule_state cannot be
interpreted as this type. The profile maps opaque rule IDs as follows:

```yaml
component_bindings:
  R-ID-8D95436233BE6DCD5E90B2AAA986052C:
    rule: RULE-FD125E8425C39349304D0BC566F5701E
    status: OBLIGATION
    condition_id:
      - C-ID-514BC6B03401A855DF7B7C157645C79B
    condition_name: POLICY-STATUS
    live_policy:
      - MPD-GOV-0006
```

`condition_name` is an English uppercase hyphenated description, no longer
than 20 characters; it must match its declared condition name. `live_policy`
is a nonempty array when present and can hold multiple policy references.

An optional `Authorization_ID` uses a keyword from
`lifecycle/keywords.json`: `REA` (Read), `WRI` (Write), or `EDI` (Edit),
followed by a hyphen and 15 alphanumeric characters. The explicit
`authority/Authorization_intention.json` registry holds
`Authorization_ID` / `intention` records. The lookup operation resolves
only an explicitly registered and profile-linked ID; it never fabricates
a permission:

```powershell
go -C .ptsip/engine/developer run . --root ../../.. --validator responsibility-authority --operation lookup-intention --input REA-AAAAAAAAAAAAAAA
```

The example lookup ID is not registered, so it **fails closed**. No
Read/Write/Edit mappings were inferred during the opaque-ID migration;
an action needs explicit user authorization and a real intention record.

The active responsibility profile no longer contains `source_text`,
`source_policy`, `source_rule_id`, or `Rule_state`. The previous
natural-language rules remain available in exact prior Git commits for
historical audit only; none becomes a new authorization intention without
approval. No Profile check, rule lookup, approval or status declaration
activates a responsibility or authorizes execution.

Validation examples:

```powershell
go -C .ptsip/engine/developer run . --root ../../.. --validator policy-rules --operation check --input developer/policy/GOV/MPD-GOV-0006.yaml
go -C .ptsip/engine/developer run . --root ../../.. --validator responsibility-authority --operation check
```
