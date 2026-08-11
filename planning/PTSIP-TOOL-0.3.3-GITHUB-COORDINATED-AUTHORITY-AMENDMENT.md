# PTSIP Tool 0.3.3 — GitHub-Coordinated Authority Scope Amendment

> **Approved:** 2026-08-11  
> **Depends on:** Local DecisionStore implementation from the Tool 0.3.3 workstream  
> **Target Tool version:** `0.3.3`  
> **Bound Specification:** `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`

## 1. Why this amendment exists

The original explicit-adoption plan correctly kept Tool-owned Local DecisionStore state outside Consumer Repositories and treated repository `ptsip.yaml` as the durable project-owned architecture declaration.

That model is sufficient after a decision has propagated through Git, but a coordination gap remains while a decision is still pending or has been accepted in one environment before its `ptsip.yaml` projection has been committed, pushed, fetched, or otherwise observed by another environment.

Separate local SQLite databases cannot provide repository-global first-winner semantics across local IDE agents, cloud agents, CI agents, or multiple clones.

The project therefore approved the following dependency order:

```text
Local DecisionStore
    -> GitHub-coordinated authority
    -> explicit ptsip adopt integration
    -> profile reconciliation
    -> Tool 0.3.3 integration/release verification
```

## 2. Scope amendment

The original plan's non-goal excluding a new distributed decision authority is superseded for Tool 0.3.3.

Tool 0.3.3 now includes a GitHub-native coordination authority implemented without sharing SQLite and without requiring a separate hosted PTSIP service.

The following original non-goals remain unchanged:

- no new PTSIP architecture classification;
- no change to conformance outcome semantics;
- no automatic Product/Toolchain/Neutral classification from directory names;
- no silent Project Profile schema widening;
- no continuous polling/reminder worker;
- no mandatory GitHub Projects board;
- no requirement to place PTSIP state in `.PTSIP/` inside the Consumer Repository;
- no Tool 0.3.3 publication/tag as part of migration implementation;
- global `gate --json` exception-envelope redesign and aggregate gate-precondition diagnostics remain separate follow-up work.

## 3. State ownership contract

Tool 0.3.3 uses three distinct state layers:

```text
PTSIP Specification / packaged registry
    -> normative/machine policy owned by PTSIP

Consumer Repository ptsip.yaml
    -> project-owned architecture declaration

Decision coordination state
    -> Local SQLite, GitHub authority ref, or explicit hosted Control Plane
```

`ptsip.yaml` is intended to be committed and shared with the Consumer Repository.

`control-plane.sqlite3` remains Tool-owned operational state under `PTSIP_HOME` and must not be Git-shared.

## 4. GitHub authority contract

GitHub repositories use the dedicated ref:

```text
refs/heads/ptsip-policy
```

The authority is bootstrapped automatically by a write-enabled coordinated operation.

The root authority commit contains an `authority.json` manifest. Existing branches with that name are accepted only if the manifest matches the PTSIP authority format, repository identity, and ref identity. Otherwise the Tool fails closed and refuses to modify the branch.

Decision documents use:

```text
decisions/<gdec-id>.json
```

The global decision ID is derived from:

- GitHub `owner/repository` identity; and
- normalized discovered component include selectors.

It does not depend on one clone's local clarification ID, branch, or missing-field list.

## 5. Compare-and-swap rule

Every authority mutation is serialized through Git history:

```text
read authority HEAD A
    -> construct mutation
    -> create commit B with parent A
    -> update ptsip-policy A -> B with force=false
```

A stale writer is rejected and must reread the winning authority state. A later contradictory answer cannot replace the first valid resolution.

This provides repository-global coordination without requiring all environments to continuously hold the same worktree snapshot.

## 6. Action-time synchronization

PTSIP does not continuously poll the authority.

A coding agent checks the authority only when a boundary-sensitive operation actually produces a clarification request through `ptsip gate`.

If the local Project Profile is missing a declaration but the authority already contains a resolved winner, the gate projects the authoritative answer into the selected local profile rather than asking the user again.

Thus correctness does not depend on immediate `ptsip.yaml` commit/push propagation.

## 7. Backend selection

```text
explicit --control-plane <URL>
    -> hosted HTTP Control Plane

explicit --coordination local
    -> embedded Local DecisionStore

explicit --coordination github
    -> GitHub authority

otherwise GitHub origin present
    -> GitHub authority

otherwise
    -> Local DecisionStore
```

`--control-plane` and `--coordination` are mutually exclusive.

GitHub coordination accepts `GH_TOKEN` or `GITHUB_TOKEN`, with authenticated `gh` CLI fallback for interactive environments.

When GitHub coordination is selected but unavailable, the write-enabled decision operation fails. It must not silently fall back to Local SQLite.

## 8. `ptsip adopt` integration

`ptsip adopt` remains the explicit project-owner path:

```text
inspect/discover candidate
    -> project owner supplies fixed architecture facts
    -> dry-run adoption plan
    -> --apply
```

For non-GitHub repositories, adoption directly writes the validated project declaration and does not fabricate a Local DecisionStore workflow record.

For GitHub repositories, `adopt --apply` first creates/reuses the repository-global authority decision and then projects the winning answer into the local Project Profile.

A different existing winner produces `ALREADY_RESOLVED` and the losing answer is not written locally.

## 9. Specification-binding constraint

`runtime_required` remains a required DecisionAnswer/adoption fact and may be retained in Tool-owned authority records.

Tool 0.3.3 must not add that field to the Project Profile schema while remaining bound to immutable Specification revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`.

If durable Project Profile representation of that fact becomes normative, it requires an explicit future Specification revision.

## 10. Completion boundary

This amendment is complete when:

1. GitHub authority bootstrap and manifest ownership are guarded;
2. global component-scope decision identity is deterministic;
3. stale writes cannot replace the authority HEAD;
4. a second clone can reconcile an already resolved decision without seeing another clone's Local SQLite or pushed `ptsip.yaml`;
5. GitHub authority unavailability does not create Local fallback winners;
6. `ptsip adopt` dry-run/apply and explicit profile handling work end-to-end;
7. existing Local and hosted HTTP backend behavior remains available through explicit selection rules;
8. Tool/package identity is `0.3.3` while Specification identity remains unchanged;
9. complete pytest, CLI smoke, package build, and `twine check` pass before merging to `main`;
10. no tag, GitHub Release, or PyPI publication is created by the migration merge itself.
