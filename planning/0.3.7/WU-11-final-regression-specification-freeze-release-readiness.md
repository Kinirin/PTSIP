# WU-11 — Tool 0.3.7 Final Regression, Specification Freeze, and Release Readiness

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-10 — Project Profile Compatibility, Migration, and Adoption (`COMPLETE / VERIFIED`)  
> **Architecture authorities:** ADR-0017, ADR-0019, ADR-0020  
> **Exact entry baseline:** `d7a19de22721b546c62809984b67c2ba1c723e7d`  
> **Predecessor verification authority:** `9a9159685a4f5de103d79e9d1c38bdbbada25d4c` via `tooling-test` run `33081718965`, job `98550433277`  
> **Release role:** final integration and release-readiness authority

## 0. Purpose

Perform the final Tool `0.3.7` integration and release-readiness verification only after the independent PP identity layer and PP migration/compatibility layers are complete.

WU-11 is deliberately downstream of WU-09/WU-10 so its exact-SHA evidence represents the final integrated code and contract state rather than an earlier baseline.

WU-11 entered `ACTIVE` state automatically under the standing successor-entry rule after WU-10 was documented `COMPLETE / VERIFIED`. This status change is an entry-state transition only; it does not by itself authorize WU-11 implementation, Specification freeze, release handoff, or release execution.

## 1. Final Tool identity

Finalize Tool package identity independently from Project Profile identity.

Required distinction:

```text
PTSIP Tool version
    0.3.7

Supported Project Profile contract identities
    explicit compatibility declaration
    e.g. pp.1.01 plus supported historical sources
```

Updating Tool package/runtime version to `0.3.7` MUST NOT rewrite Project Profile contract identities for numeric symmetry.

## 2. Final contract synchronization

Review and synchronize only surfaces that actually own completed Tool/PP behavior, including as applicable:

```text
src/ptsip/constants.py
pyproject.toml
schemas/
src/ptsip/specdata/
spec/
releasenote/
profiles/
adoption/ADOPTION-GUIDE.md
agents/AGENT-CONTRACT.md
README.md
README.ko.md
STATUS.md
MEMORY.md
AGENTS.md
.github/workflows/
```

Do not mechanically bump consumer/example Project Profiles. Their PP identity changes only when their own declaration contract requires and authorizes migration.

## 3. Specification freeze

Freeze the final Tool `0.3.7` normative Specification after implementation is complete.

Required chain:

```text
final implemented Tool behavior
    ↓
final Tool 0.3.7 normative Specification
    ↓
immutable SPEC_REVISION
    ↓
Tool/Specification binding
    ↓
explicit supported PP compatibility declaration
```

No floating Specification revision may be used as final release authority.

## 4. Full regression

Run the complete repository regression against the exact final source SHA.

At minimum verify:

- all focused WU-09 identity tests;
- all focused WU-10 compatibility/migration tests;
- all existing migration state/execution/recovery fixtures;
- repository self-profile validation;
- PP compatibility diagnostics;
- Tool/PP version independence;
- VPMS boundary;
- complete `python -m pytest` suite.

Any later code or contract change invalidates the final-regression authority and requires a new exact-SHA verification.

## 5. Package/distribution verification

Verify the final built wheel/sdist, including:

- package metadata reports `PTSIP` Tool `0.3.7`;
- CLI/runtime reports Tool `0.3.7`;
- PP contract identity is reported separately where relevant;
- wheel artifact evidence preserves ADR-0018 subsystem ownership;
- `ptsip-evidence`, `ptsip-source-compat`, and `ptsip-migration` are present and attributed correctly;
- embedded schemas/specdata are synchronized;
- install/reinstall smoke succeeds;
- transition/migration capability is packaged;
- no fixture-only temporary profile or checkpoint/ledger artifact is included;
- no Project Profile is bumped merely to match Tool `0.3.7`.

## 6. Exact-SHA workflow gate

The final required workflow run must:

- use the self-hosted runner policy;
- checkout and verify the exact dispatched source SHA;
- run the final full regression;
- build and verify distributions;
- verify artifact evidence and exact snapshot binding;
- smoke the installed wheel;
- write the successful exact-SHA status only if every required step passes.

Earlier successful runs remain historical evidence for their verified SHAs but cannot substitute for the final WU-11 SHA.

## 7. Release readiness boundary

WU-11 may declare Tool `0.3.7` release-ready only when all final evidence agrees on:

```text
Tool version
Specification revision
source SHA
built artifact identity
supported PP compatibility
repository validation state
```

Release readiness does not mean every supported repository must adopt the latest PP contract. Compatibility and migration requirements remain repository-specific and PP-authorized.

## 8. Non-goals

WU-11 must not:

- redesign PP version grammar;
- introduce new historical compatibility mappings except to fix a verified WU-10 defect;
- use release pressure to bypass migration authority;
- create a Tool-numbered Project Profile;
- collapse Tool and PP version numbers back into one identity.

Unexpected architecture changes discovered in WU-11 should normally reopen the responsible earlier WU boundary rather than be patched invisibly into release code.

## 9. Completion gate

WU-11 is complete only when:

- Tool runtime/package identity is `0.3.7`;
- final Tool Specification is frozen to an immutable revision;
- Tool-to-PP compatibility declaration is synchronized with implemented behavior;
- public/embedded schemas and specdata are synchronized;
- complete regression passes at the exact final SHA;
- package/distribution verification passes at that SHA;
- wheel artifact evidence reflects the declared Product component architecture;
- installed-wheel smoke passes;
- final self-hosted exact-SHA workflow succeeds;
- release documentation reflects actual Tool and PP identities without numeric coupling;
- no unauthorized Project Profile migration occurred;
- final release handoff evidence is recorded.
