# Support Policy

PTSIP Support Feature Policy has one canonical repository authority and one installed-package projection.

Repository authority:

```text
src/policy/
├─ index.yaml
├─ SFP-*.yaml
├─ schemas/
└─ registries/
```

- `src/policy/` is the canonical Support Feature Policy authority.
- `docs/Support_policy/` retains human guidance only.
- `docs/Support_policy/automation/` is repository-side Support Policy management/projection guidance and is not policy authority.
- `developer/` is a separate PTSIP repository-development policy boundary.

Installed distribution projection:

```text
ptsip/support/
├─ schemas/
├─ registries/
└─ policy/
   ├─ index.yaml
   └─ SFP-*.yaml
```

The installed projection is generated deterministically during package build. It is not a second authoring authority. Repository-source execution reads the canonical source; an installed PTSIP distribution reads the shipped `ptsip/support/` projection.

Schema `$id` URIs retain their existing identity across this source relocation. They are not repository lookup paths; the canonical registry and layout resolve the files under `src/policy/`. Policy provenance `source_ref` uses the new canonical source path.
