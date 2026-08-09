# PTSIP Reference Tooling Changelog

This changelog tracks the independently versioned PTSIP Reference Tool. Specification changes remain in [`CHANGELOG.md`](CHANGELOG.md).

## 0.2.0 — Unreleased

Evidence-v2 migration driven by the first real Consumer Repository Pilot:

- moved the Reference Tool version independently to `0.2.0` while retaining PTSIP Specification family `0.2.0-draft`;
- bound Tool 0.2.0 to immutable PTSIP Specification revision `895e12d27230af2bb99ad17a96e8df8ef41bc3e0`;
- added `ptsip-pilot-report/v2`;
- added before/after repository snapshot evidence using HEAD, Git status including ignored entries, and tracked-content fingerprints;
- invalidates inspection/pilot evidence when repository state changes during collection;
- replaced the hard-coded non-intrusion boolean with observed non-intrusion status;
- changed Git inventory to tracked-file evidence by default and records parser/read/discovery failures explicitly;
- added component candidates without auto-assigning architectural ownership;
- added typed dependency evidence for Python imports, `.csproj` `ProjectReference`, and GitHub Actions local-script invocation;
- added component-aware profile validation with deterministic selector specificity and conflict detection;
- aligned the exception profile schema with every field required by `PTSIP-EXC-001`;
- added constrained coding-agent classification schema with exactly three PTSIP classifications and separate decision statuses;
- added initial deterministic declaration-to-dependency findings without claiming complete conformance;
- distinguishes Python package-metadata support from CI-verified interpreter versions;
- expanded CI verification to Python 3.11–3.14;
- changed package summary punctuation to ASCII-safe text for legacy Windows console compatibility.

Known limitations retained intentionally in Tool 0.2.0:

- product artifact build/content inspection is not automatic yet;
- Python dependency lifecycle phase is generally `UNKNOWN` unless another adapter provides phase context;
- JavaScript/TypeScript and Go dependency adapters are not implemented yet;
- dynamic/relative import resolution is preserved as unresolved evidence when it cannot be resolved deterministically;
- component candidates are evidence for human/agent review, not automatic PTSIP ownership decisions;
- Tool 0.2.0 does not claim complete automated Enforced Conformance evaluation.

## 0.1.0a1 — 2026-08-09

Initial alpha reference-tooling release:

- added installable Python distribution named `ptsip`;
- added `ptsip --version` and `ptsip spec`;
- added `ptsip doctor` for local environment checks;
- added read-only `ptsip inspect` repository inventory;
- added read-only `ptsip pilot` with external default state/report storage;
- added `ptsip validate` for an existing PTSIP project profile;
- embedded the PTSIP 0.2.0-draft profile schema and registry snapshot;
- bound the tool to specification revision `cb4164a803678a0364ce037af4addbad1d7ecc7d`;
- updated the canonical tool/specification source after the repository rename to `kwaksinwoo01/ptsip`;
- added Python 3.11–3.13 CI tests;
- added PyPI Trusted Publishing workflow gated to `tool-v*` releases;
- added release-tag/package-version verification and `twine check` before publication;
- published prerelease `tool-v0.1.0a1` through GitHub Actions and PyPI Trusted Publishing.

### Deliberate non-goals for 0.1

- no automatic repository restructuring;
- no automatic architecture migration;
- no automatic exception approval;
- no AI-owned final classification;
- no PTSIP-specific directories created inside Consumer Repositories.
