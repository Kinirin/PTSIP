# PTSIP Tool 0.3.3 Decision Authority and Control Plane

This document describes the Tool-level decision workflow used when a coding-agent task reaches a boundary-sensitive operation and the Consumer Repository does not yet declare enough architectural intent. It does not change the bound PTSIP Specification version or add an architecture classification.

## Core ownership model

PTSIP keeps three kinds of state separate:

```text
Specification-owned rules
    -> packaged PTSIP Specification / registry

Project-owned architecture
    -> Consumer Repository ptsip.yaml

Tool-owned decision workflow state
    -> Local DecisionStore, GitHub authority ref, or hosted Control Plane
```

`ptsip.yaml` is the durable architecture declaration for a repository revision. It is intended to be committed with the Consumer Repository and should not be placed in `.gitignore` merely because PTSIP generated it.

`control-plane.sqlite3` is operational state. Local SQLite files are not a portable architecture source of truth and must not be shared through Git.

## Why Tool 0.3.3 adds GitHub coordination

A repository may be used concurrently by local IDE agents, cloud agents, CI agents, or multiple clones. Separate Local DecisionStores cannot coordinate a still-`PENDING` decision across those environments.

Continuous `ptsip.yaml` polling is not sufficient for correctness because a valid decision may exist before the profile change is committed/pushed/fetched. Tool 0.3.3 therefore treats synchronization as an **authority problem**, not as a real-time worktree-file synchronization problem.

For GitHub repositories, unresolved architecture decisions are coordinated through a dedicated Git ref:

```text
refs/heads/ptsip-policy
```

The authority ref is not the normal source branch and does not contain a shared SQLite database. It contains small JSON decision records and is mutated with non-force Git ref compare-and-swap behavior.

## Backend selection

`ptsip gate` and `ptsip resolve` select decision coordination as follows:

```text
explicit --control-plane <URL>
    -> hosted HTTP Control Plane

else explicit --coordination local
    -> Local DecisionStore

else explicit --coordination github
    -> GitHub-coordinated authority (GitHub origin required)

else GitHub origin exists
    -> GitHub-coordinated authority

else
    -> Local DecisionStore
```

`--control-plane` and `--coordination` are mutually exclusive.

`ptsip adopt --apply` uses project-owner adoption semantics. On a non-GitHub repository it does not create a Local DecisionStore record. On a GitHub repository it first establishes/reuses the GitHub authority winner for the component scope, then applies the project-owned profile.

## Local DecisionStore

The local backend stores decision workflow state under the platform PTSIP state directory. On Windows the default is equivalent to:

```text
%LOCALAPPDATA%\PTSIP\decisions\<repository-fingerprint>\control-plane.sqlite3
```

`PTSIP_HOME` may override the PTSIP state root.

The Local DecisionStore is appropriate for local-only or explicitly local coordination. Its compare-and-set rules enforce first-valid-resolution-wins within that store. It is intentionally not Git-tracked or synchronized between machines.

A GitHub repository may explicitly opt into this isolated behavior with:

```powershell
ptsip gate . --coordination local --json
```

That opt-in is useful for deliberately isolated experimentation; it is not distributed coordination.

## GitHub-coordinated authority

### Authority identity

Each GitHub repository uses one authority ref by default:

```text
refs/heads/ptsip-policy
```

The ref is bootstrapped automatically when the first write-enabled GitHub-coordinated decision operation needs it.

The root authority commit contains `authority.json`. Decision records are stored as:

```text
decisions/<global-decision-id>.json
```

### Global decision key

Local clarification IDs include local completeness facts and may differ when two clones have temporarily different Project Profile state. Using those IDs as the global lock key would recreate a coordination gap.

The GitHub authority therefore derives a `gdec-*` ID from:

- GitHub `owner/repository` identity; and
- normalized discovered component include selectors.

A different local clarification ID, branch name, or local missing-field set does not create a second authority winner for the same component scope.

### Compare-and-swap mutation

A mutation follows this sequence:

```text
1. read authority HEAD A
2. read current decision state from A
3. construct a new tree
4. create commit B whose parent is A
5. update refs/heads/ptsip-policy from A to B with force=false
```

If another environment changes the authority HEAD after step 1, the non-force ref update is rejected. PTSIP rereads the winner rather than overwriting it.

For the same component scope:

```text
Agent A reads A -> TOOLCHAIN
Agent B reads A -> PRODUCT

Agent A wins A -> B
Agent B stale write is rejected
Agent B rereads B
-> ALREADY_RESOLVED (TOOLCHAIN)
```

This promotes first-valid-resolution-wins from one local SQLite database to repository-global GitHub coordination.

### Authentication

Cloud/automation environments may provide:

```text
GH_TOKEN
or
GITHUB_TOKEN
```

Interactive developer environments may instead use an authenticated `gh` CLI.

The credential must have sufficient repository contents/ref write authority for the `ptsip-policy` ref. GitHub coordination does not silently fall back to Local DecisionStore if authentication, permission, or network access fails.

### Fail-closed behavior

When a GitHub repository is using GitHub coordination and the authority is unavailable:

```text
existing committed ptsip.yaml
    -> still readable

new/change architecture decision
    -> blocked with coordination error
```

PTSIP must not create a separate local winner simply because GitHub is temporarily unreachable. That would create split-brain authority.

## Action-time synchronization

PTSIP does not continuously poll the authority ref.

A coding agent calls:

```powershell
ptsip gate . --component <candidate-id> --json
```

when its current task depends on the component boundary.

If the selected Project Profile already declares sufficient intent, the gate returns `NO_DECISION_REQUIRED` without contacting the decision backend.

If the local profile is missing the decision but the GitHub authority already contains a resolved winner, the gate applies that authoritative answer to the selected local profile and returns a resolved result. This allows a stale clone to converge before another environment's `ptsip.yaml` commit has propagated.

This is **action-time synchronization**, not background polling.

## Decision lifecycle

The conceptual lifecycle remains:

```text
PENDING
   -> explicit valid human/project-owner answer
   -> RESOLVED
   -> local profile application
```

The first valid authoritative resolution wins. A later contradictory answer cannot replace it.

For GitHub coordination, profile application status is clone-local. One clone may have projected the decision while another has not, so the global authority does not pretend that one `application_status` describes every worktree.

For the Local and hosted backends, their existing application bookkeeping remains backend-specific.

## Explicit project adoption

`ptsip adopt` is distinct from coding-agent `resolve` semantics.

```powershell
ptsip adopt . `
  --component tools `
  --classification TOOLCHAIN `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --lifecycle-owner DEVELOPMENT_TOOLING `
  --executable yes `
  --json
```

The default is a dry-run. It performs candidate discovery, answer validation, profile projection, stale-evidence checks, and full profile validation without writing the Consumer Repository or bootstrapping the GitHub authority.

Application requires `--apply`.

For GitHub repositories the application ordering is:

```text
candidate/evidence validation
    -> GitHub authority gate
    -> global first-winner resolution/reuse
    -> local prepared-profile write
    -> profile validation
```

A different existing GitHub winner returns `ALREADY_RESOLVED`; the local profile is not overwritten with the losing answer.

## Coding-agent resolution

When `ptsip gate` returns `DECISION_REQUIRED`, an active coding-agent session records only the user's explicit facts:

```powershell
ptsip resolve . `
  --decision <decision-id> `
  --classification TOOLCHAIN `
  --purpose "Repository migration tooling" `
  --shipped no `
  --runtime-required no `
  --lifecycle-owner DEVELOPMENT_TOOLING `
  --executable yes
```

Answer consistency is deterministic. Free-form architecture inference is not performed by the decision backend.

Local/hosted decisions retain their existing branch/revision application protections. GitHub-coordinated decision authority is repository-global; a resolved `gdec-*` winner can therefore be reconciled by another branch/clone, while local profile projection still refuses conflicting existing project declarations and concurrent profile changes.

## Result semantics

The gate exposes the existing high-level states:

- `NO_DECISION_REQUIRED` — selected profile is already sufficient;
- `RESOLVED` — the needed decision is authoritative and available/applied for the active operation;
- `DECISION_REQUIRED` — the affected work must stop until a human/project owner supplies the decision;
- `DECISION_ERROR` — authority/application reconciliation failed or conflicted.

`DECISION_REQUIRED` uses exit code `7`. Decision/application errors use exit code `8`.

Adoption uses `ptsip-adoption/v1` with statuses including:

- `ADOPTION_PLAN`;
- `ADOPTED`;
- `ALREADY_DECLARED`;
- `CONFLICT`;
- `UNKNOWN_COMPONENT`;
- `STALE_EVIDENCE`;
- `ALREADY_RESOLVED` when another GitHub-coordinated architecture answer already won.

## Hosted HTTP Control Plane

The GitHub App-backed service introduced in Tool 0.3.1 remains available as an explicit hosted backend. It is no longer selected implicitly.

```powershell
pip install "ptsip[github-app]"
ptsip-app --host 127.0.0.1 --port 8080 --db ptsip-control-plane.sqlite3
```

Remote clients select it only with:

```powershell
ptsip gate . --control-plane https://control-plane.example --json
```

The bearer token remains configured through `PTSIP_CONTROL_PLANE_TOKEN`.

The hosted service exposes:

- `POST /v1/gate`;
- `POST /v1/decision`;
- `POST /v1/resolve`;
- `POST /v1/application`;
- `POST /github/webhook`;
- `GET /healthz`.

Its GitHub Issue interface remains an optional asynchronous human UI. GitHub Issue open/closed state is not itself authoritative decision state.

## Non-goals

Tool 0.3.3 GitHub coordination does not:

- put SQLite files in the Consumer Repository;
- require `.PTSIP/` or `.ptsip/` worktree directories;
- continuously poll GitHub;
- infer `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT` from directory names;
- make the coding agent the architecture authority;
- replace `ptsip.yaml` as the project architecture declaration;
- change the bound PTSIP Specification revision.
