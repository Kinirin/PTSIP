# PTSIP Validation Engine 0.3.3

Engine ID: `urn:uuid:44258a79-af23-4ce9-94c9-6d23de29bd21`
Capability: `urn:uuid:450659b1-1e89-4138-91d3-8dcbf6d6dc1a`

### Purpose

Validate schemas, contracts, typed I/O, preconditions and Cases independently or
compose checks without asserting PTSIP conformance or granting execution authority.

### Contract changes

Capability contract **4.0.0** supports internal `$ref` (JSON Pointer, anchors and
recursion) and `$dynamicRef`/`$dynamicAnchor` with dynamic evaluation scope. External
resources still resolve by `$id`. Optional `schema_resolution` configures recursive
JSON discovery in explicit directories, an anonymous schema's `base_uri`, and
opt-in HTTP(S) retrieval from exact `allowed_origins`. Transitive references use
the same permission scope. Redirects, ambient proxies and out-of-root files are
rejected. Output adds `schema_resources_used` with URI, source and SHA256 evidence.

The complete resource set is validated, including unused supplied/acquired schemas.
Acquisition has document/byte/node/time limits and performs no persistent writes.
See the [support guide and complete request](https://github.com/Kinirin/PTSIP/blob/dev/0.3.8b2/docs/validation-schema-references.md).
The prior 3.1.0 contract remains historical; native requests select 0.3.3 / 4.0.0.

Pinned official Draft 2020-12 ref/dynamicRef fixtures exercise 108 offline examples;
15 cases requiring separate localhost fixtures are excluded from that count.
Dedicated native tests cover scoped file/HTTP acquisition, transitive references,
dynamic recursive extension, denied origins, path escape and redirect rejection.
This is bounded reference verification, not a claim of complete JSON Schema or
PTSIP conformance. Source builds require Rust 1.88+ for the locked dependencies.
