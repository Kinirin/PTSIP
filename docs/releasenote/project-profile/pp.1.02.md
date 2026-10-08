# Project Profile pp.1.02

State: Current Project Profile contract
Transition: pp.1.01 -> pp.1.02 / SEMANTIC_MIGRATION
Specification binding: 0.3.7-draft @ 3c47816770d194ae42f98faedc911d980db0e62a

## Purpose

This contract separates developer-owned Project Profile generation from
user-owned profile lineage and removes path-shaped examples that could be
misread as repository architecture authority.

## Contract changes

- ptsip.revision is now required and uses canonical Rev.#### form.
- Developer-distributed baselines begin at Rev.0001.
- ptsip.specification.family is no longer serialized in current Project
  Profiles. Specification identity remains exact through source plus the
  immutable Git revision.
- ptsip.profile_role distinguishes repository authority (PROJECT) from
  distributed examples (DISTRIBUTED_EXAMPLE).
- Distributed examples require project-path materialization and must not be
  treated as canonical repository layout.
- New repository-local profile storage uses .ptsip/profiles/index.yaml with
  an explicit default_profile and one or more *.ptsip.yaml resources.
- New adoption defaults to .ptsip/profiles/main.ptsip.yaml.
- Repository-root ptsip.yaml remains a compatibility/migration input rather
  than the default target for new profile creation.

## Example-path safety

Public examples use unresolved project-owned selector placeholders instead of
prescribing paths such as product/app/** or a developer's private tooling
layout. A project or agent must resolve those selectors against repository
evidence and explicit project authority before materializing a PROJECT profile.

## Authority boundaries

This Project Profile transition does not change the frozen Specification
family or its immutable revision. It also does not rewrite historical Tool
release notes. Tool SemVer, Project Profile contract identity,
ptsip.revision, and Specification revision remain separate axes.
