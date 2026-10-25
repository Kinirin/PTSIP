# Project Profile pp.2.00

State: Current registered Project Profile contract
Transition: pp.1.02 -> pp.2.00 / IDENTITY_ONLY
Specification binding: 0.3.7-draft @ 3c47816770d194ae42f98faedc911d980db0e62a

## Purpose

The project owner explicitly chose `pp.2.00` as the new independently versioned Project Profile distribution. Current source and native release payload differ from those published under `pp.1.02`; the latter remains immutable.

## Contract changes

The new registered schema identity, embedded schema projection, catalog, distributed examples, immutable per-version baseline, and both contract registries consistently use `pp.2.00`. The existing `pp.1.02` structural contract and path/role authorities are preserved. The non-normative experimental pp.2.00 RBAC candidate schema is **not** activated by this version-only release decision.

## Release verification

Full Tool + Project Profiles + seven native engines remain mandatory for the integrated PTSIP Package. The one-release `0.3.8.b6` lightweight verification route replaces repeat full Package Join runs, not source SHA binding, component integrity, or version validation. This note does not assert completed publication.
