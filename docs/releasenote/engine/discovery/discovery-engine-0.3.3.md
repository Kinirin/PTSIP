# Discovery Engine 0.3.3

Engine ID: `urn:uuid:e51f2248-0db7-4f5c-aa4b-e278a133bc50`

Capability: `urn:uuid:6cc343f5-a3af-404e-a22f-8d57f861bb97`, version `3.1.0`.

Source preparation: 2026-10-26. Publication is a separate action.

### Purpose

Adds `discovery.check-environment` for the seven official engines in an explicitly
selected installation. Rust checks the bundle origins, binary hashes, engine and
capability identities, current contract/transport versions and declared
dependency evidence. It does not run inspected engines or provider operations.

### Contract changes

The additive operation returns separate diagnostic dimensions and the installed
operation catalog with typed I/O and request templates. Empty dependency lists,
missing evidence and stale evidence do not become compatibility PASS. Known
dependency version mismatches are incompatible. A failed component does not
hide independent diagnostic rows. Source files are rechecked before reporting.

Existing Discovery operations keep their behavior under an explicitly versioned
binding. The previous 3.0.0 contract is retained as a historical reference.
Planning and Storage responsibilities, execution authorization, source owner
legitimacy and release attestation remain separate.
