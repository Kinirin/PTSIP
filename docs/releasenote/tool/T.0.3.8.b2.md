# PTSIP Tool T.0.3.8.b2

The PTSIP 0.3.8b2 Package installs the Tool, the public Project Profile catalog
pp.1.02 and all seven independently versioned native Rust engines. Engine
dependencies are mandatory and exact; no extra or platform marker omits an engine.

Tool identity is T.0.3.8.b2. Its standalone build metadata projects to 0.3.8b2;
the PTSIP PyPI project now represents the complete Package rather than Tool alone.
The opaque Package composition identity remains 0.3.8.<SHA6>.

Context and Projection remain 0.2.0. Storage, Validation, Discovery, Planning
and Execution remain 0.3.0. Previously published engine artifacts retain their
original source and hashes after current-source verification.

Windows x64 and Linux x64 are verified independently. Profile history is excluded.
Installing executables does not activate engines or grant runtime authority.

This release is isolated on dev/0.3.8b2 and must not merge to main. Retired
Developer policies, schemas and validators remain retired. The bounded release
instruction and software tests do not establish successor policy-validation PASS.
