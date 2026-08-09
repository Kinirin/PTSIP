# PTSIP Status

- Specification: `0.2.0-draft`
- Maturity: Experimental
- Canonical repository: `kwaksinwoo01/ptsip`
- Public standard status: None; project-defined specification
- External tooling model: Defined
- Reference Python tooling: `0.1.0a1` integrated in this canonical repository
- Reference Tool package name: `ptsip`
- PyPI publication: Pending
- Tool CI: Passing on Python 3.11–3.13
- Tool release namespace: `tool-v*`
- First external conformance implementation: Not yet recorded
- Reuse license: Apache License 2.0

## Release blockers for PTSIP Specification 1.0

- exercise the project profile in at least one real Consumer Repository;
- exercise automated validator rules against real dependency and packaging boundaries;
- validate the Agent Contract against real coding-agent tasks;
- stabilize rule and schema semantics;
- publish tagged specification releases;
- publish and exercise reference tooling through its independent release lifecycle.

## Reference Tool 0.1.0a1 status

Implemented:

- package/build metadata for PyPI distribution `ptsip`;
- `ptsip --version` and `ptsip spec`;
- `ptsip doctor`;
- read-only `ptsip inspect`;
- read-only `ptsip pilot` with external default state storage;
- `ptsip validate` for existing project profiles;
- embedded PTSIP 0.2.0-draft schema/registry snapshot;
- automated tests and Tool-specific GitHub Actions workflows;
- integration into the canonical `kwaksinwoo01/ptsip` repository.

Pending before public `pip install ptsip`:

- configure the PyPI Trusted Publisher for repository `kwaksinwoo01/ptsip` and workflow `tooling-release.yml`;
- publish the first `tool-v0.1.0a1` release.
