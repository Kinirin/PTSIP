# PTSIP Reference Tooling Changelog

This changelog tracks the independently versioned PTSIP Reference Tool. Specification changes remain in [`CHANGELOG.md`](CHANGELOG.md).

## 0.3.0 — Release candidate source, not yet published

Conformance-capability migration from the source-only Tool 0.2.3 baseline, bound to PTSIP Specification `0.2.0-draft` revision `14a0c2f54bb486de6a109979224f998b04fd04a3`:

- adds `ptsip conform` with `CONFORMANT`, `NON_CONFORMANT`, and `INCOMPLETE` outcomes and distinct CLI exit codes;
- emits deterministic `ptsip-diagnostic/v1`-shaped diagnostics with instance identity separate from PTSIP rule identity;
- evaluates rule-relative blocking coverage instead of treating empty findings as conformance;
- accepts explicit read-only `ptsip-artifact-evidence/v1` inputs and evaluates Product packaging isolation without conflating artifact producer and artifact owner;
- adds an independent build-resolution evaluator over explicitly declared component manifests, supporting Python, npm, .NET project, and Go module manifest evidence while blocking ambiguous cross-plane shared-manifest evidence;
- adds lifecycle evidence evaluation that requires declared release/compatibility ownership and positive path-scoped release evidence, while treating workflow triggers alone as insufficient proof of lifecycle coupling or independence;
- adds deterministic JavaScript/TypeScript source dependency evidence for static imports, CommonJS `require`, and dynamic `import()`/`require()` uncertainty;
- adds npm manifest evidence for runtime/build dependency declarations and local package-to-package resolution;
- adds Go source import evidence with local-module, standard-library/platform, and external-module scope resolution;
- adds source-level .NET `using` evidence correlated with local project namespaces and deterministic `PackageReference` matches while retaining `.csproj` `ProjectReference` evidence;
- routes `inspect`, `pilot`, `clarify`, and `conform` through one Tool 0.3.0 composite dependency evidence graph;
- keeps `ptsip pilot` at `conformance.status=NOT_EVALUATED`; strict outcomes are produced only by `ptsip conform`;
- evaluates optional project-specific component dependency policy separately from universal PTSIP diagnostics, consistent with `PTSIP-POL-001`;
- ingests bound-schema agent classification decisions as review evidence without allowing them to overwrite Project Profile declarations;
- accepts explicit `ptsip-external-evidence/v1` dependency evidence only when producer metadata and Consumer Repository identity/revision can be verified, preserving input SHA-256 and canonical provenance;
- permits trusted external evidence to supplement native unresolved evidence, but blocks contradictory resolved external/native evidence instead of silently overriding either source;
- performs a final conformance-report audit against the embedded diagnostic schema, active rule registry, coverage-gap identity, and evaluator-state contract before allowing `CONFORMANT`;
- retains Tool 0.2.2 Human Clarification's deterministic, zero-LLM semantics and Tool 0.2.3 evidence-correctness behavior as regression boundaries.

The Tool 0.3.0 source is not yet tagged or published. Release readiness requires supported-Python verification, package build and `twine check`, release-tag/package-version wiring verification, documentation review, and final PR review before `tool-v0.3.0` publication.

## 0.2.3 — Source-only migration, not published

Evidence-correctness migration bound to PTSIP Specification `0.2.0-draft` revision `14a0c2f54bb486de6a109979224f998b04fd04a3`:

- uses Python source encoding detection so UTF-8 BOM and valid source encoding declarations do not become false read failures;
- resolves relative Python imports deterministically when package/repository evidence identifies the target;
- adds evidence-node scope values `PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, and `UNRESOLVED_TARGET` without expanding the three PTSIP architectural classifications;
- records dependency provenance independently from target scope;
- represents dynamic Python import relationships as `LOADS` and retains `DYNAMIC` resolution when the target identity is not statically known;
- recognizes direct declared Python dependencies as external evidence when declaration/import naming matches deterministically;
- resolves GitHub Actions local scripts from effective workflow/job/step `working-directory`;
- stops converting arbitrary GitHub Actions `run:` commands into unresolved local-script invocation edges;
- reports declared dependency evaluator state as `RAN` or `BLOCKED` so `findings: []` cannot imply an evaluator ran when it did not;
- synchronizes embedded profile schema and registry resources to the Tool-bound immutable Specification snapshot;
- aligns Project Profile validation with `boundaries XOR components`, retired mandatory-rule waiver semantics, exact revision binding, and project component-policy references;
- retains Tool 0.2.2 deterministic Human Clarification behavior as a regression boundary;
- intentionally does not add `ptsip conform`, Product Artifact adapters, JS/TS/npm/Go adapters, stable diagnostic emission, agent-decision ingestion, or external evidence import.

Tool 0.2.3 remains an evidence/validation implementation and does not claim complete automated PTSIP Conformance Evaluation.

Tool 0.2.3 is intentionally not published to PyPI and receives no `tool-v0.2.3` release/tag. Its source migration becomes the implementation baseline for Tool 0.3.0.

## 0.2.2 — Unreleased

Deterministic human clarification support for missing architectural intent:

- added `ptsip clarify` to stop at missing intent instead of expanding speculative classification inference;
- clarification analysis uses deterministic repository evidence and fixed completeness rules only, and reports `llm_calls: 0` in machine-readable output;
- added fixed English (`en`) and Korean (`ko`) question templates without model-based translation;
- language selection follows explicit `--lang`, then `PTSIP_LANG`, then the operating-system locale, with English as the fallback;
- added Git `origin` discovery and deterministic GitHub HTTPS/SSH normalization to `owner/repository`;
- added explicit `--publish github-issue` transport, with optional `--repo owner/repository` override;
- GitHub publishing requires the external `gh` CLI only when publishing is explicitly requested; ordinary clarification remains read-only and network-free;
- stores GitHub clarification publication state under `PTSIP_HOME/clarifications` outside the Consumer Repository to prevent duplicate issue creation;
- does not collect, interpret, or automatically classify from Issue answers in Tool 0.2.2;
- retains the existing `0.2.0-draft` Specification family and immutable Specification revision; this Tool capability does not change normative PTSIP semantics or schemas;
- keeps the separately planned pilot-driven Tool 0.2.3/0.3.0 work independent from this release.

## 0.2.0 — 2026-08-09

Evidence-v2 release driven by the first real Consumer Repository Pilot:

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
- aligned the profile schema with the Tool 0.2.0 bound specification revision;
- added constrained coding-agent classification schema with exactly three PTSIP classifications and separate decision statuses;
- added initial deterministic declaration-to-dependency findings without claiming complete conformance;
- distinguishes Python package-metadata support from CI-verified interpreter versions;
- expanded CI verification to Python 3.11–3.14;
- changed package summary punctuation to ASCII-safe text for legacy Windows console compatibility;
- published `tool-v0.2.0` through GitHub Actions and PyPI Trusted Publishing;
- verified public installation with `pip install ptsip==0.2.0`.

Known limitations retained intentionally in Tool 0.2.0:

- product artifact build/content inspection is not automatic yet;
- Python dependency lifecycle phase is generally `UNKNOWN` unless another adapter provides phase context;
- JavaScript/TypeScript and Go dependency adapters are not implemented yet;
- dynamic/relative import resolution is preserved as unresolved evidence when it cannot be resolved deterministically;
- component candidates are evidence for human/agent review, not automatic PTSIP ownership decisions;
- Tool 0.2.0 does not claim complete automated Enforced Conformance evaluation;
- Tool 0.2.0 remains bound to specification revision `895e12d27230af2bb99ad17a96e8df8ef41bc3e0` even when the `0.2.0-draft` family evolves to a newer immutable snapshot.

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
