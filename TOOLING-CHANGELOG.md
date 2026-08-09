# PTSIP Reference Tooling Changelog

This changelog tracks the independently versioned PTSIP Reference Tool. Specification changes remain in [`CHANGELOG.md`](CHANGELOG.md).

## 0.1.0a1 — Unreleased

Initial alpha reference-tooling implementation:

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
- added PyPI Trusted Publishing workflow gated to `tool-v*` releases.

### Deliberate non-goals for 0.1

- no automatic repository restructuring;
- no automatic architecture migration;
- no automatic exception approval;
- no AI-owned final classification;
- no PTSIP-specific directories created inside Consumer Repositories.
