# Support Policy

PTSIP Support Feature Policy has one canonical repository authority and one installed-package projection.

Repository authority:

```text
docs/Support_policy/
├─ automation/
└─ policy/
   ├─ index.yaml
   ├─ SFP-*.yaml
   ├─ schemas/
   └─ registries/
```

- `policy/` is the canonical Support Feature Policy authority.
- `automation/` is repository-side Support Policy management/projection support and is not policy authority.
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
