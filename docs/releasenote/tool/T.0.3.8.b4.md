# PTSIP Tool T.0.3.8.b4

Emergency PTSIP Package **0.3.8b4** includes Validation engine **0.3.2** with
capability contract **3.1.0**. Profile pp.1.02, Context/Projection 0.2.1 and
Storage/Discovery/Planning/Execution 0.3.1 retain their independent versions.

Validation automatically connects external `$ref` URIs to `$id` values in the
explicitly supplied local `schema_resources` set. Callers provide schema contents;
the engine builds the identifier index. Relative and transitive external
references and fragments in a different resource are supported.

**Internal `$ref` and `$dynamicRef` are unsupported.** The engine now names these
limitations explicitly in errors and `ptsip-engine-validation --help`. It does
not download schemas or scan the filesystem. See the
[support table and request example](https://github.com/Kinirin/PTSIP/blob/codex/validation-b4/docs/validation-schema-references.md).

Only **PTSIP** is published to PyPI: one complete Windows x64 wheel, one complete
Linux x64 wheel and one sdist with all Rust sources and lockfiles. Each wheel
contains the Tool, public Profiles and all seven engines. Linux source rebuild
and fresh installation remain mandatory before publication. Installation does
not grant runtime activation or PTSIP conformance authority.

The bounded hotfix branch is `codex/validation-b4`, derived from the published b3
branch. Historical releases are preserved and this work does not merge to main.
Software verification does not stand in for the retired Developer Policy Resolver.
