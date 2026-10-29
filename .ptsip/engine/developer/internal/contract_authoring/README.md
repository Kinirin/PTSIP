# Responsibility rule constants and authorization contracts

## Family rule declarations

All fourteen Family contracts use `Policy_Rules.Definition[]`. Each entry
binds an opaque, nonsemantic R-ID to a **named rule constant** in uppercase
letters and hyphens, at most 20 characters. A rule constant is not a random
hash, document number, path or version.

```yaml
Policy_Rules:
  Definition:
    - id: R-ID-8D95436233BE6DCD5E90B2AAA986052C
      rule: FLATTEN-POLICY-STATU
      preconditions:
        - C-ID-514BC6B03401A855DF7B7C157645C79B
```

Policy documents select responsibility and optional registered conditions.
The policy document never owns condition semantics. `Constraints` is retired
and rejected in all fourteen Family contracts.

## Responsibility-Authority profile

`.ptsip/profiles/Responsibility-Authority-list.yaml` declares responsibilities
and their condition/authorization links only; it stores **no policy file ID,
path or ownership mapping**. One R-ID can be referenced by more than one
policy document without edits to the profile.

```yaml
component_bindings:
  R-ID-8D95436233BE6DCD5E90B2AAA986052C:
    rule: FLATTEN-POLICY-STATU
    status: OBLIGATION
    condition_id:
      - C-ID-514BC6B03401A855DF7B7C157645C79B
    condition_name: POLICY-STATUS
```

The approved independent lifecycle statuses are `OBLIGATION` and
`PROHIBITION`, not policy document approval states. The profile rejects
`live_policy`, `source_policy`, `source_rule_id`, `source_text` and
`Rule_state`. An optional `Authorization_ID` must resolve through the
independent authorization registry, with the same `condition_name`.

## Authorization rules

`.ptsip/contracts/schemas/compile-sealed-rbac/authority/Authorization_intention.json`
registers explicit authorization rules. Each record has **five required
fields**: `ID`, `resource`, `action`, `effect`, and `condition_name`.
`ID` matches `^Auth-[A-Z]+# Responsibility rule constants and authorization contracts

## Family rule declarations

All fourteen Family contracts use `Policy_Rules.Definition[]`. Each entry
binds an opaque, nonsemantic R-ID to a **named rule constant** in uppercase
letters and hyphens, at most 20 characters. A rule constant is not a random
hash, document number, path or version.

```yaml
Policy_Rules:
  Definition:
    - id: R-ID-8D95436233BE6DCD5E90B2AAA986052C
      rule: FLATTEN-POLICY-STATU
      preconditions:
        - C-ID-514BC6B03401A855DF7B7C157645C79B
```

Policy documents select responsibility and optional registered conditions.
The policy document never owns condition semantics. `Constraints` is retired
and rejected in all fourteen Family contracts.

## Responsibility-Authority profile

`.ptsip/profiles/Responsibility-Authority-list.yaml` declares responsibilities
and their condition/authorization links only; it stores **no policy file ID,
path or ownership mapping**. One R-ID can be referenced by more than one
policy document without edits to the profile.

```yaml
component_bindings:
  R-ID-8D95436233BE6DCD5E90B2AAA986052C:
    rule: FLATTEN-POLICY-STATU
    status: OBLIGATION
    condition_id:
      - C-ID-514BC6B03401A855DF7B7C157645C79B
    condition_name: POLICY-STATUS
```

The approved independent lifecycle statuses are `OBLIGATION` and
`PROHIBITION`, not policy document approval states. The profile rejects
`live_policy`, `source_policy`, `source_rule_id`, `source_text` and
`Rule_state`. An optional `Authorization_ID` must resolve through the
independent authorization registry, with the same `condition_name`.

## Authorization rules

`.ptsip/contracts/schemas/compile-sealed-rbac/authority/Authorization_intention.json`
registers explicit authorization rules. Each record has **five required
fields**: `ID`, `resource`, `action`, `effect`, and `condition_name`.
. The `action` field resolves against the
**twelve explicitly approved codes** in `lifecycle/keywords.json`:

| Code | Meaning | Code | Meaning |
| --- | --- | --- | --- |
| `REA` | READ | `LIS` | LIST |
| `CRE` | CREATE | `WRI` | WRITE |
| `EDI` | EDIT | `DEL` | DELETE |
| `VAL` | VALIDATE | `EXE` | EXECUTE |
| `APR` | APPROVE | `PUB` | PUBLISH |
| `GRA` | GRANT | `REV` | REVOKE |

`WRI` records/writes data; `EDI` modifies an existing target. The
registered `APR`, `PUB`, `GRA`, and `REV` names are vocabulary, not
implicit authority to approve policy, publish, grant, or revoke rights.
Actual authorization rules require independently approved resources,
conditions and effects. `effect` is `ALLOW` or `DENY`. The condition name must be registered
in the independent condition catalog.

Example of **schema shape only** (this rule is *not* registered):

```json
{
  "ID": "Auth-READ",
  "resource": "EXAMPLE-RESOURCE",
  "action": "REA",
  "effect": "DENY",
  "condition_name": "POLICY-STATUS"
}
```

Use explicit lookups through `responsibility-authority` with
`--operation lookup-intention --input Auth-READ`; this currently fails
closed because the registry is empty. A matched ALLOW entry does **not**
by itself grant runtime execution, Seal or Activation.

Historical policy prose and previous mapping IDs remain in Git history for
audit only. They do not create authorization intentions or privilege grants.

## Approved authorization decision model

`.ptsip/contracts/Responsibility-Authority/Authorization_decision.json` is
the machine-readable **C: explicit ALLOW/DENY and unregistered block** policy.
It is validated against
`authority/Authorization_decision.schema.json` whenever contract/profile
checks run. The decision outcomes are:

| Registration / binding | Machine outcome | Persisted effect |
| --- | --- | --- |
| Missing authorization | `BLOCK_UNRESOLVED` | None |
| Missing or invalid binding | `BLOCK_UNRESOLVED` | None |
| Registered explicit `DENY` | `BLOCK_EXPLICIT` | Explicit DENY only |
| Registered explicit `ALLOW` | `ALLOW_CANDIDATE` | Explicit ALLOW only |

`ALLOW_CANDIDATE` is **not** permission to execute. Higher-policy
authorization and action-time gates remain separately required. Resource
identity matching, condition evaluation, actor bindings, and
simultaneous/conflicting rule precedence are `UNRESOLVED`; do not invent
these rules in an implementation. Unknown registration is blocked but is
never materialized as a permanent DENY entry. The actual
`Authorization_intention.json` still contains no permission records.

## Validation

```powershell
go -C .ptsip/engine/developer run . --root ../../.. --validator policy-rules --operation check --input developer/policy/GOV/MPD-GOV-0006.yaml
go -C .ptsip/engine/developer run . --root ../../.. --validator responsibility-authority --operation check
```
