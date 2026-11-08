# PTSIP Tool 0.4.0 — OPTIONAL / NON-BLOCKING Planning Map

> **Status:** DRAFT / OPTIONAL PLANNING INDEX  
> **Integration line:** Tool `0.4.0`  
> **Release classification:** `OPTIONAL / NON-BLOCKING`  
> **CORE planning:** `planning/0.4.0/`

## 1. Directory responsibility

`planning/0.4.0-op/` is the planning and session-document namespace for work classified as:

```text
OPTIONAL / NON-BLOCKING
```

It may also retain explicitly deferred/out-of-release ideas as planning context when that helps future work, but such records do not authorize implementation.

The governing release rule is:

```text
optional work complete        → 0.4.0 may release
optional work incomplete      → 0.4.0 may release
optional work deferred        → 0.4.0 may release
optional work abandoned       → 0.4.0 may release
```

No file, session, prototype, branch, or successful experiment in this directory is a Tool 0.4.0 release prerequisite.

## 2. Placement rule

Classification comes before directory placement:

```text
UNCLASSIFIED
    ↓
Release Dependency Review
    ↓
OPTIONAL / NON-BLOCKING
    ↓
planning/0.4.0-op/
```

Accidentally placing an item here does not override an already approved CORE classification. Likewise, moving a file to `planning/0.4.0/` does not promote it by itself.

## 3. Optional session document map

| Document | Responsibility | Release relationship |
| --- | --- | --- |
| `OP-01-experimental-and-deferred-directions.md` | AI advisory extensions, history, workflow convenience, advanced optional directions, promotion discipline | OPTIONAL / NON-BLOCKING or DEFERRED as marked |
| `OP-02-ptsip-backed-test-mode-productization.md` | productize reusable `ptsip.yaml`-backed development Test Mode orchestration based on the proven `Kinirin/turbo-system` pattern | OPTIONAL / NON-BLOCKING |

Future optional sessions should use an `OP-` prefix or another explicit non-blocking identifier so they cannot be mistaken for CORE WUs.

## 4. Optional branch relationship

Optional work may be isolated when useful:

```text
dev/0.4.0
  ├─ optional/<experiment>
  └─ optional/<non-blocking-extension>
```

Branch existence does not imply merge obligation. Merge into `dev/0.4.0` requires an explicit decision and must not weaken CORE contracts or release qualification.

## 5. Promotion firewall

Promotion to CORE requires all of the following:

1. identify the approved 0.4.0 objective, safety invariant, or release proof that fails without the item;
2. show that safe deferral is no longer possible;
3. review compatibility, authority, ownership, and verification consequences;
4. obtain explicit project-owner approval;
5. create or move the controlling planning record into `planning/0.4.0/` only after reclassification.

Implementation progress, sunk cost, provider availability, or convenience is not a valid substitute for this review.

## 6. CORE non-dependency rule

OPTIONAL code, templates, providers, or workflows must not become imports, runtime dependencies, required CI statuses, or release checks of the CORE unless promoted explicitly.

Where an optional feature consumes CORE contracts, dependency direction should remain:

```text
CORE contract
    ↓
OPTIONAL adapter / extension
```

not:

```text
CORE
    ↓
OPTIONAL implementation required to function
```

## 7. Current state

```text
Pre-WU-00A release boundary
    = COMPLETE

CORE planning
    = planning/0.4.0/

OPTIONAL planning
    = planning/0.4.0-op/

OPTIONAL implementation authorization
    = only by separately approved optional session/WU

0.4.0 release dependency
    = NONE from this directory unless explicitly promoted
```
