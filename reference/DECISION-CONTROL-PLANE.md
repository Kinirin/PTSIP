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

The first valid resolution wins. The store performs the transition only while the decision is `PENDING`. A later answer receives `ALREADY_RESOLVED` / `IGNORED_TERMINAL_DECISION` and cannot replace the decision.

When the candidate selector identity changes, Tool 0.3.1 generates a new clarification identity. A newer gate marks the older still-pending decision stale.

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

`ptsip resolve` is an explicit write-enabled command. It first validates the fixed answer invariants, then wins the central compare-and-set only if the decision is still pending, projects the decision into `ptsip.yaml`, validates the resulting local profile, and finally reports the application state to the control plane.

If another valid answer already won, the command does not modify the profile.

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
- the user has repository `write`, `maintain`, or `admin` permission;
- the answer passes deterministic consistency validation.

Free-form prose is not interpreted by an LLM.

After an Issue answer wins, the App applies `ptsip.yaml` against the exact recorded branch revision. The write uses Git data objects and a non-force ref update so a concurrent branch move cannot silently receive a stale decision. If the branch changed, application becomes `STALE`; the resolved decision is not reinterpreted against the new snapshot.

## Late Issue answers

If a coding-agent chat already resolved the decision, the Issue is closed after local profile application is reported. Any later Issue comment is ignored because the authoritative decision is no longer `PENDING`.

GitHub Issue open/closed state is not authoritative. Manually closing an Issue does not create a PTSIP decision.

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

Coding-agent clients configure:

```text
PTSIP_CONTROL_PLANE_URL
PTSIP_CONTROL_PLANE_TOKEN
```

The reference service exposes:

- `POST /v1/gate` — register/poll an on-demand decision and create the Issue if needed;
- `POST /v1/resolve` — compare-and-set an explicit chat decision;
- `POST /v1/application` — record local/remote profile application state;
- `POST /github/webhook` — GitHub App webhook receiver with `X-Hub-Signature-256` verification;
- `GET /healthz` — health check.

The reference SQLite store is Tool-owned service state. It is not a replacement for the Consumer Repository's `ptsip.yaml` declaration.

## GitHub App permissions/events

The deployment should grant only the permissions needed by the enabled workflow. The reference implementation uses repository Issues, repository contents/Git data, and collaborator-permission lookup. Webhooks should include installation/repository events needed to map installations and `issue_comment` for asynchronous answers.

No scheduled reconciliation or reminder worker is part of Tool 0.3.1.
