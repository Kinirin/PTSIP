# Validation schema references in PTSIP 0.3.8b5

PTSIP 0.3.8b5 ships Validation engine **0.3.3**, capability contract **4.0.0**.
Reference behavior follows the selected JSON Schema dialect; the default is Draft
2020-12. This release verifies the declared reference features, not complete JSON
Schema conformance or PTSIP conformance. Results grant no execution authority.

| Reference or acquisition feature | Support |
| --- | --- |
| Internal `$ref`: `#/$defs/...`, `#anchor`, absolute self-reference | Supported, including recursive data structures |
| `$dynamicRef` and `$dynamicAnchor` | Supported with dynamic evaluation scope |
| External `$ref`, relative URIs, transitive refs and fragments | Supported through registered IDs and scoped acquisition |
| Explicit `schema_resources` | Automatically indexed by `$id`; each explicit resource requires an absolute fragment-free `$id` |
| Local file discovery | Recursively loads `.json` files only from `schema_resolution.directories` |
| HTTP(S) acquisition | Automatically fetches missing referenced resources only when `allow_network` is true and the exact origin is listed |

The five generic operations (`validation.contract`, `validation.schema`,
`validation.typed-io`, `validation.precondition`, `validation.case`) accept optional
`schema_resources` and `schema_resolution`. For `validation.compose`, provide these
once at the top level alongside `checks`; each check uses that configuration.
Separate Case preparation and handoff operations retain their registered inputs.

This complete native-engine request reuses an internal definition:

```json
{
  "engine_id": "urn:uuid:44258a79-af23-4ce9-94c9-6d23de29bd21",
  "engine_version": "0.3.3",
  "contract_id": "urn:uuid:450659b1-1e89-4138-91d3-8dcbf6d6dc1a",
  "contract_version": "4.0.0",
  "operation_id": "validation.schema",
  "input": {
    "schema": {
      "type": "object",
      "properties": {"name": {"$ref": "#/$defs/name"}},
      "required": ["name"],
      "$defs": {"name": {"type": "string", "minLength": 2}}
    },
    "instance": {"name": "Alice"}
  }
}
```

Pipe this JSON to `ptsip-engine-validation`. It returns `PASS` and `output.valid:
true`. A name with the wrong type returns `PASS` and `output.valid: false` with
diagnostics. An unresolved reference, invalid schema or disallowed acquisition
returns `FAIL` and exit code 2, without a validation verdict.

## Discover schema IDs from local files

For example, save `{"$id":"urn:example:address","type":"object"}` in a JSON file
inside `./schemas`. The following operation input finds it by `$id` automatically:

```json
{
  "schema": {"$ref": "urn:example:address"},
  "instance": {},
  "schema_resolution": {"directories": ["./schemas"]}
}
```

Paths are relative to the engine process working directory unless absolute. Only
explicit directories and their descendants are scanned, in deterministic file
order. Symlink entries are skipped and canonical paths outside those roots are
rejected. Use directories containing JSON schemas: invalid JSON/schema files fail
the request, including unused discovered files. Identical acquired schemas may
deduplicate; conflicting IDs fail with `DUPLICATE_SCHEMA_ID`. Duplicate explicit
`schema_resources` IDs remain errors.

For a relative file reference such as `"$ref":"address.json"` in an anonymous
input schema, also set `base_uri` to an absolute retrieval URI, for example
`file:///C:/project/schemas/root.json` on Windows or
`file:///workspace/schemas/root.json` on Linux. The referenced file must still be
inside a configured directory. An HTTP-shaped `$id` can resolve from a local file;
it does not require downloading when its resource is already indexed.

## Retrieve referenced schemas over HTTP(S)

```json
{
  "schema": {"$ref": "https://schemas.example.com/address.json"},
  "instance": {},
  "schema_resolution": {
    "allow_network": true,
    "allowed_origins": ["https://schemas.example.com"]
  }
}
```

An origin includes the scheme, host and effective port. The engine retrieves
missing reference targets, follows their relative/transitive references within
the same explicit permissions, and evaluates internal references in each loaded
resource. It does not enumerate arbitrary web directories. Different origins need
their own entries. HTTP is available for explicitly permitted origins, including
local test servers. HTTPS uses certificate validation; redirects, URL credentials
and ambient proxy settings are disabled. No persistent download cache or project
file writes are performed. Unconfigured network access stays disabled.

## Dynamic references and reproducible inputs

`$dynamicRef` uses `$dynamicAnchor` and the evaluator's dynamic scope, including
recursive schema extension. It is not substituted with an ordinary `$ref`, and
does not require enabling file or network access. Static, embedded and explicitly
supplied resources can provide all required schemas offline.

Generic results include `output.schema_resources_used`; composed results include
it for each check. Entries contain `uri`, `source` and `sha256`. `INPUT_JSON_COMPACT`
hashes the parsed input schema serialized as compact JSON by the engine;
`FILE_BYTES` and `NETWORK_BYTES` hash the exact acquired bytes. Record these
entries to detect a changed local/remote schema. Built-in dialect meta-schemas
come from the locked evaluator dependency.

Limits per schema evaluation: 128 total schema documents; 16,384 schema nodes;
8 configured directories; 4,096 scanned directory entries; 16 allowed origins;
1 MiB per acquired document and 8 MiB total acquired bytes. Each HTTP request has
a maximum 5-second timeout and shares a 20-second acquisition deadline. The
existing stdin request limit remains 1 MiB. Composition evaluates checks separately
and records their separate resource evidence.

Unresolved references, conflicting IDs, malformed schemas, byte/count/time limits,
disallowed files and denied origins all fail closed. Annotation data inside
`examples`, `default` and `const` does not trigger reference resolution.

The previous 3.1.0 contract remains historical data. Native callers must use the
current exact engine/contract versions. Run `ptsip-engine-validation --help` for
the invocation summary. The independent source build now requires Rust 1.88+;
complete platform wheels already contain all seven native executables.

Reference semantics are described in the [JSON Schema Core specification](https://json-schema.org/draft/2020-12/json-schema-core).
