# PTSIP Validation Engine 0.3.2

Engine ID: `urn:uuid:44258a79-af23-4ce9-94c9-6d23de29bd21`
Capability: `urn:uuid:450659b1-1e89-4138-91d3-8dcbf6d6dc1a`

### Purpose

Validate schemas, contracts, typed I/O, preconditions and Cases through five
independent operations, or combine checks with `validation.compose`. The separate
handoff and case-preparation operations retain their registered responsibilities.

### Contract changes

Capability contract **3.1.0** adds optional `schema_resources` to the five generic
validation operations and to the top level of `validation.compose`. The engine
automatically indexes the supplied schemas' `$id` values and connects external
`$ref` targets. Relative external references, transitive references and fragments
in a different registered resource are supported. Embedded schema resources with
their own `$id` are also indexed.

**Internal `$ref` (`#`, `#/$defs/...`, or an absolute self-reference) and
`$dynamicRef` remain unsupported.** They fail explicitly with
`INTERNAL_REF_UNSUPPORTED` or `DYNAMIC_REF_UNSUPPORTED`. Missing resources and
duplicate IDs fail closed. There is no network retrieval or automatic filesystem
search. Callers supply the JSON contents of permitted local schemas.

See the [reference support table and complete request example](https://github.com/Kinirin/PTSIP/blob/codex/validation-b4/docs/validation-schema-references.md).
The previous 3.0.0 contract is retained as historical data; requests must select
the current exact engine and contract versions. Outputs still do not assert PTSIP
conformance or grant execution authority.

PTSIP 0.3.8b4 embeds Validation 0.3.2 and all six other engines in each complete
Windows x64 and Linux x64 wheel. Only the PTSIP project is published to PyPI.
