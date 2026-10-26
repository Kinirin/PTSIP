# PTSIP Tool T.0.3.8.b2

Source preparation: 2026-10-26. This document does not assert publication.

Adds `ptsip engines compatibility --json`, a read-only entry into the bundled
Rust Discovery diagnostic operation for the seven official engines. The selected
Python installation supplies the package, metadata and scripts roots; the CLI
does not search PATH for other copies or infer compatibility itself.

The report includes the independent Package and Tool identities, installed
operation catalog, typed inputs, request templates, separate integrity and
contract/transport checks, and source-bound declared dependency evidence.
Missing relationships or evidence remain unverified. Exit 4 preserves that
distinction from verified compatibility. Existing `engines status` is preserved.

Requires Discovery 0.3.3 with capability contract 3.1.0. Older or missing native
diagnostics produce a structured unverified result. No automatic activation,
permission grant, external-engine probing, or legacy policy validation is added.
Previously published Package 0.3.8b6 artifacts remain immutable.
