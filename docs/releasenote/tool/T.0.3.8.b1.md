# PTSIP Tool T.0.3.8.b1

Tool identity uses `T.major.minor.micro.state` to distinguish it from the complete
Package's independent PyPI version. `T.0.3.8.b1` projects to `0.3.8b1` when a Python
distribution version is required. Existing published `PTSIP 0.3.8b1` remains an
immutable historical Tool distribution; it is not a complete Package.

The root `PTSIP` project now declares Package version `0.3.8.1b1`. Tool code,
Profile catalog and all seven registered engine dependencies are mandatory in the
complete Package. Standalone Tool builds use a temporary `ptsip-tool` metadata
projection. No engine or Profile version is incremented by the Tool prefix change.

Consumer authority admission and runtime activation remain separate from
installation. Current engine PyPI publication requires project authority beyond
the owner's existing PTSIP-only API token. A failed full Package request must not
be downgraded to a Tool-only publication.
