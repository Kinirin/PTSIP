# PTSIP Tool T.0.3.8.b5

PTSIP Package **0.3.8b5** includes Validation engine **0.3.3**, capability contract
**4.0.0**. Profile pp.1.02, Context/Projection 0.2.1 and
Storage/Discovery/Planning/Execution 0.3.1 retain independent versions.

Validation now supports internal `$ref`, recursive schemas and `$dynamicRef` with
dynamic anchors. Callers can configure recursive local JSON schema discovery and
opt-in HTTP(S) reference retrieval with explicit directory/origin permissions.
Results record the acquired schemas' URIs and SHA256 hashes. See the
[complete guide](https://github.com/Kinirin/PTSIP/blob/dev/0.3.8b2/docs/validation-schema-references.md).

Only the PTSIP project is published to PyPI: complete Windows x64 and Linux x64
wheels plus one source distribution. All seven native engines ship in each wheel.
Source builds require Rust 1.88+. Exact-source native component, complete wheel
installation and Linux source rebuild/installation verification precede publication.
Public PyPI digests must match the approved artifacts before recording publication.

This release continues on dev/0.3.8b2 from completed b4, without incorporating main's
policy reconstruction. Historical releases remain immutable. Runtime activation,
legacy public API cutover and PTSIP conformance authority are unchanged.
