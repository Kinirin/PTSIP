# PTSIP Tool 0.3.1 Decision Control Plane

This document describes the Tool-level decision workflow added in PTSIP Tool 0.3.1. It does not change the PTSIP Specification version or add a new architectural classification.

## Purpose

The decision control plane exists for coding-agent sessions that reach a boundary-sensitive operation and cannot safely continue because the Consumer Repository does not declare enough architectural intent.

The workflow is deliberately **on demand**:

- no timer polls unresolved Issues;
- no scheduler reminds users about old decisions;
- no reminder is emitted when the user is not actively using a coding agent;
- a coding agent polls only when its current operation actually depends on the missing decision.

## Roles

- **Coding agent** — invokes the decision gate when the active task needs an unresolved architecture decision and reports the blocker to the user.
- **PTSIP CLI/agent client** — provides `ptsip gate` and explicit `ptsip resolve` operations.
- **PTSIP GitHub App** — creates/deduplicates clarification Issues, receives Issue comment webhooks, verifies decision authority, applies Issue-originated decisions, and closes completed Issues.
- **Decision control plane store** — records authoritative workflow state and enforces compare-and-set resolution.
- **Project Profile** — remains the repository's architecture declaration.
- **GitHub Issue** — is an asynchronous decision UI, not the authoritative state itself.

## Decision lifecycle

```text
PENDING
   |\
   | \-- explicit user decision in coding-agent chat
   |        -> RESOLVED (source=AGENT_CHAT)
   |
   \---- authorized structured GitHub Issue answer
            -> RESOLVED (source=GITHUB_ISSUE)

RESOLVED
   -> profile application
      -> APPLIED / LOCAL_APPLIED
      -> FAILED / STALE
```

The first **valid** resolution wins. Validity includes both the fixed answer invariants and compatibility with the applicable Project Profile projection. Profile-conflicting answers do not consume the `PENDING -> RESOLVED` transition.

The store performs the authoritative transition under compare-and-set semantics. A later contradictory answer receives `ALREADY_RESOLVED` / `IGNORED_TERMINAL_DECISION` and cannot replace the decision.

When the candidate selector identity changes, Tool 0.3.1 generates a new clarification identity. A newer active gate marks an older still-pending semantic request stale.

## Coding-agent polling

A coding agent does not continuously poll PTSIP. It calls:

```powershell
ptsip gate . --component <candidate-id> --json
```

only when its active boundary-sensitive task needs that component's ownership decision.

Outcomes:

- `NO_DECISION_REQUIRED` — repository declaration is already sufficient;
- `RESOLVED` — authoritative decision is available and applied;
- `DECISION_REQUIRED` — the affected work must stop until the user resolves the decision;
- `DECISION_ERROR` — the decision exists but application is stale/failed/conflicting and must be reconciled.

`DECISION_REQUIRED` uses exit code `7`. Decision/application errors use exit code `8`.

If the same semantic decision was already resolved but its profile application became `STALE` or `FAILED`, a later **active** gate rebinds only the application target snapshot to the coding agent's current branch revision. The human answer and original winning source remain unchanged. The agent may then retry the exact same authoritative answer without asking the user to decide again. A different answer still cannot replace it.

GitHub Issue open/closed state is not authoritative. If a user manually closes a still-pending Issue, PTSIP does not treat that as a decision. The next active gate that actually needs the decision reopens the Issue; there is no timer-driven reopen or reminder.

## Chat resolution

A user may decide directly in the active coding-agent conversation. The agent records only the user's explicit decision through:

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

`ptsip resolve` is an explicit write-enabled command. Its ordering is intentionally transactional:

1. fetch the authoritative decision record;
2. require the local GitHub origin, branch, and revision to match the active decision target;
3. validate the fixed answer invariants;
4. project the answer into the local profile and run the full existing Project Profile validator **without mutating the repository**;
5. only after that validation succeeds, attempt the central compare-and-set;
6. if this answer won (or is the exact stored answer being retried after a stale/failed application), atomically write the prevalidated profile;
7. report `LOCAL_APPLIED` to the control plane and complete the linked Issue.

The local write also checks that `ptsip.yaml` did not change between prevalidation and replacement. If another valid answer won the central race, the local profile is not modified.

## GitHub Issue resolution

Clarification Issues request the fixed `ptsip-clarification-answer/v1` structure:

```yaml
format: ptsip-clarification-answer/v1
decision:
  classification: TOOLCHAIN
  purpose: Repository migration tooling
  shipped: NO
  runtime_required: NO
  lifecycle_owner: DEVELOPMENT_TOOLING
  executable: YES
```

Issue comments are accepted only when:

- the decision is still pending;
- the comment parses as the fixed structured contract;
- the user has repository write-level authority accepted by the reference service;
- the answer passes deterministic consistency validation;
- the answer can be projected against the Project Profile at the exact recorded gate revision without conflicting with an existing declaration.

Free-form prose is not interpreted by an LLM.

The App validates the profile projection **before** attempting the central compare-and-set. After an Issue answer wins, the App applies the already validated `ptsip.yaml` content against the exact recorded branch revision. The write uses Git data objects and a non-force ref update so a concurrent branch move cannot silently receive a stale decision.

If the branch changed after the last active gate, application becomes `STALE`; the accepted human decision remains authoritative, the Issue is completed, and the next coding-agent gate can rebind that same decision to the current snapshot for application without reopening the architecture question.

## Late Issue answers

If a coding-agent chat already resolved the decision, the Issue is closed after local profile application is reported. Any later Issue comment is ignored because the authoritative decision is no longer `PENDING`.

Likewise, an Issue answer that wins first prevents a later contradictory chat answer from changing the decision. GitHub Issue open/closed state is a UI state only and cannot replace the control-plane decision state.

## Control plane service

Tool 0.3.1 includes the reference service entry point:

```powershell
pip install "ptsip[github-app]"
ptsip-app --host 127.0.0.1 --port 8080 --db ptsip-control-plane.sqlite3
```

Required environment:

```text
PTSIP_CONTROL_PLANE_TOKEN
PTSIP_GITHUB_WEBHOOK_SECRET
PTSIP_GITHUB_APP_ID
PTSIP_GITHUB_PRIVATE_KEY or PTSIP_GITHUB_PRIVATE_KEY_PATH
```

Remote coding-agent clients select this legacy service only by passing an explicit base URL:

```powershell
ptsip gate . --control-plane https://control-plane.example --json
```

The bearer token remains configured separately through `PTSIP_CONTROL_PLANE_TOKEN`.

The reference service exposes:

- `POST /v1/gate` — register/poll an on-demand decision and create/reopen the Issue if needed;
- `POST /v1/decision` — fetch the authoritative decision before local profile prevalidation;
- `POST /v1/resolve` — compare-and-set an explicit chat decision or retry the exact authoritative answer after a stale/failed application;
- `POST /v1/application` — record coding-agent application state (`LOCAL_APPLIED`, `FAILED`, or `STALE`);
- `POST /github/webhook` — GitHub App webhook receiver with `X-Hub-Signature-256` verification;
- `GET /healthz` — health check.

Agent API requests use the configured bearer token. Request bodies are bounded by the reference server. The reference SQLite store is Tool-owned service state. It is not a replacement for the Consumer Repository's `ptsip.yaml` declaration.

## Installation discovery

Installation/repository webhook events are used to cache GitHub App installation IDs when available. The cache is not authoritative: if a gate needs a repository whose installation mapping is absent (for example after deploying the service with a fresh database while the App is already installed), the reference service resolves the repository installation on demand with GitHub App JWT authentication and stores the recovered mapping.

This recovery is triggered by an active request; it is not background polling.

## GitHub App permissions/events

The reference workflow should be deployed with the minimum GitHub App repository permissions required by the enabled operations:

- **Issues: write** — create, comment on, reopen, and close clarification Issues;
- **Contents: write** — read `ptsip.yaml` and create an exact-parent Git commit/ref update for Issue-originated profile application;
- **Metadata: read** — inspect a comment author's repository permission before accepting an Issue decision.

The App receives `issue_comment` webhooks for asynchronous answers. Installation/repository events are useful for proactively caching installation mappings but are not required for correctness because active requests can recover the repository installation on demand.

No scheduled reconciliation or reminder worker is part of Tool 0.3.1.
