# Validation schema references in PTSIP 0.3.8b4

PTSIP 0.3.8b4 ships Validation engine **0.3.2**, capability contract **3.1.0**.
Schema validation is a declared subset of JSON Schema. It does not establish
PTSIP conformance or grant execution authority.

| Reference form | Support |
| --- | --- |
| External `$ref` to another schema's `$id` | Supported when that resource is supplied in `schema_resources` or embedded in the input schema |
| Relative external `$ref` | Resolved against the current resource's absolute `$id` |
| External URI with a JSON Pointer or `$anchor` fragment | Supported for a different registered resource |
| Internal `$ref`, including `#`, `#/$defs/...` and an absolute reference to the current resource | **Unsupported**, rejected with `INTERNAL_REF_UNSUPPORTED` |
| `$dynamicRef` | **Unsupported**, rejected with `DYNAMIC_REF_UNSUPPORTED` |
| Automatic HTTP retrieval or filesystem search | **Unsupported**; URI identifiers do not cause downloads or directory scans |

The engine automatically indexes `$id` values in the **explicitly supplied local
resource set** and resolves references by URI. Callers do not need to construct a
URI-to-schema mapping. Callers must load their permitted schema files and include
their JSON contents in `schema_resources`; merely placing a file in a repository
does not register its `$id`.

`schema_resources` is optional for `validation.contract`, `validation.schema`,
`validation.typed-io`, `validation.precondition`, and `validation.case`. For
`validation.compose`, supply it once at the top level alongside `checks`; the
resources are available to every check in that request. Existing requests without
references can omit it. The case-catalog preparation and handoff operations retain
their separate registered input contracts; they do not accept this new field.

This is a complete native-engine stdin request:

```json
{
  "engine_id": "urn:uuid:44258a79-af23-4ce9-94c9-6d23de29bd21",
  "engine_version": "0.3.2",
  "contract_id": "urn:uuid:450659b1-1e89-4138-91d3-8dcbf6d6dc1a",
  "contract_version": "3.1.0",
  "operation_id": "validation.schema",
  "input": {
    "schema": {
      "$id": "https://example.invalid/schemas/customer",
      "type": "object",
      "properties": {"address": {"$ref": "address"}},
      "required": ["address"]
    },
    "schema_resources": [
      {
        "$id": "https://example.invalid/schemas/address",
        "type": "object",
        "properties": {"zip": {"type": "integer"}},
        "required": ["zip"]
      }
    ],
    "instance": {"address": {"zip": 12345}}
  }
}
```

Run `ptsip-engine-validation --help` for the transport and reference limits, or
pipe the JSON request to `ptsip-engine-validation`. The example returns a `PASS`
operation result with `output.valid: true`. Changing `zip` to a string returns
`output.valid: false` with validation diagnostics. A missing resource or unsupported
reference returns a `FAIL` result and exit code 2, without a validation verdict.

Each supplied resource requires an absolute, fragment-free `$id`. Nested resource
IDs may resolve relative to their enclosing `$id`. Duplicate IDs, missing targets,
invalid resources and conflicting identities fail closed. The entire supplied set
is checked, including resources that are not used by the selected schema.

The resource set is scoped to one request and is discarded afterward. The limits
are 128 supplied resources, 16,384 schema nodes and the existing 1 MiB transport
request limit. Reference-looking keys inside annotation values such as `examples`,
`default` or `const` are data and do not activate reference resolution.

The support table describes PTSIP's capability boundary. JSON Schema itself defines
additional reference behavior; see the [JSON Schema structuring documentation](https://json-schema.org/understanding-json-schema/structuring).
