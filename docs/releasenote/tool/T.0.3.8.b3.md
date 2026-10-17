# PTSIP Tool T.0.3.8.b3

PTSIP release `0.3.8.b3` has normalized PyPI Package version `0.3.8b3`.
The Package installs the Tool, public Project Profile pp.1.02 and all seven
independently versioned native Rust engines inside one complete platform wheel.

Context and Projection are 0.2.1. Storage, Validation, Discovery, Planning and
Execution are 0.3.1. Engine capability contracts are unchanged.

Only the PTSIP project is uploaded to PyPI, with complete Windows x64 and Linux
x64 wheels and one source distribution containing Rust sources and lockfiles.
Each platform wheel is installed in a fresh environment. The Linux source
distribution is independently rebuilt and installed before publication.
Installed Profile history is excluded; source-only baselines remain build inputs.
Installation grants no runtime activation authority.

This release remains isolated on dev/0.3.8b2 and must never merge to main.
The opaque Package composition identity remains 0.3.8.<SHA6>. Retired Developer
policy validators remain disabled; software verification is not successor policy PASS.
