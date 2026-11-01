# Policy review through GitHub Actions

This is a non-normative usage guide for the read-only review helper. It does not
declare a new validator, Test Mode, policy contract, approval, or runtime grant.

`.github/workflows/policy-review.yml` runs the existing Go helper on an explicitly
selected checkout and publishes its results as a GitHub Actions artifact. It
supports manual dispatch and reuse by another workflow in the same repository.
It does not run automatically on every policy edit.

The separate `Policy contract guard` workflow runs on pull requests and pushes
to `main`. It rejects writes into the historical `Rules/` contract tree and
validates changed current policy documents through registered contract entry.
Its structural pass/fail is distinct from the review helper's collection status.
The helper now uses the shared current selector, resolves Family schema URNs
against registered resources, and retains complex inventory branches as
uninspected rather than interpreting an obsolete Family `structure` field.

## Inputs and checkout selection

| Input | Meaning |
| --- | --- |
| `targets` | A JSON array of exact `developer/policy/**.yaml` or `.yml` paths. Manual dispatch defaults to the four RECORD policies; reusable callers must supply their selection. |
| `sources` | A JSON array of explicitly selected historical `FILE#/json/pointer` sections. Default: `[]`. |
| `expected_sha` | Optional exact 40-character commit SHA. A mismatch with the checkout fails before the Go helper runs. |

Each array accepts at most 64 distinct, nonempty strings. Paths must be canonical
repository-relative POSIX paths. The adapter passes every item as a separate
process argument without evaluating a shell command. The Go helper retains its
existing path, symlink, exact-pointer and contract-selection checks.

The workflow checks out `github.sha`, records the actual `git rev-parse HEAD`,
and verifies that they match. A real Actions checkout must be clean. The manual
dispatch branch selects the event revision; `expected_sha` can detect a branch
moving after a caller selected a revision. The wrapper does not switch to an
arbitrary ref or mutate policy files.

## Manual execution and web agents

After the workflow is committed and pushed to the default branch, open
**Actions > Policy review helper > Run workflow**. Choose the branch, supply
the arrays, and optionally set `expected_sha`.

A web agent without a repository shell can use GitHub's workflow-dispatch API
when its GitHub integration has the necessary Actions permission. The presence
of a workflow does not add dispatch capabilities to an agent or connector that
does not expose them. An agent with a checkout and a shell can also keep using
the existing local Go command directly.

An authenticated GitHub CLI caller can dispatch the same workflow. For example,
from PowerShell:

```powershell
$reviewSha = gh api repos/Kinirin/PTSIP/commits/main --jq '.sha'
$request = @{
  ref = 'main'
  inputs = @{
    targets = '["developer/policy/RECORD/MPD-RECORD-0001.yaml","developer/policy/RECORD/MPD-RECORD-0002.yaml"]'
    sources = '["developer/policy/legacy/MPD-WORK-0002.yaml#/rules/work_execution_lifecycle/execution_output"]'
    expected_sha = $reviewSha
  }
} | ConvertTo-Json -Depth 4
$request | gh api --method POST repos/Kinirin/PTSIP/actions/workflows/policy-review.yml/dispatches --input -

gh run list --repo Kinirin/PTSIP --workflow policy-review.yml --event workflow_dispatch --commit $reviewSha --limit 5 --json databaseId,status,conclusion,url
gh run watch <RUN_ID> --repo Kinirin/PTSIP --exit-status
gh run download <RUN_ID> --repo Kinirin/PTSIP --dir <NEW_OUTPUT_DIRECTORY>
```

For an agent with a direct REST tool, the endpoint is
`POST /repos/Kinirin/PTSIP/actions/workflows/policy-review.yml/dispatches` and the
JSON body has the same `ref` and `inputs` as `$request` above. The array inputs
are JSON-encoded **strings**, not nested arrays in the dispatch body.

Match a returned run using its workflow, event, source SHA and run ID; do not
assume that the newest unrelated run belongs to this request. Artifacts can be
listed through `GET /repos/Kinirin/PTSIP/actions/runs/{run_id}/artifacts` and
downloaded through the returned artifact ID. Artifact access requires the
appropriate repository and Actions access.

## Reuse from a workflow in this repository

```yaml
name: Selected policy review
on:
  workflow_dispatch:
permissions:
  contents: read
jobs:
  selected-review:
    uses: ./.github/workflows/policy-review.yml
    with:
      targets: '["developer/policy/RECORD/MPD-RECORD-0002.yaml"]'
      sources: '[]'
      expected_sha: ${{ github.sha }}
```

The called workflow exposes `source_sha`, `review_status`, `artifact_id`, and
`artifact_url` through `needs.selected-review.outputs` in subsequent jobs.
Each invocation generates a unique artifact name so multiple callers in one
run do not overwrite or collide with one another.

## Artifact contents and result interpretation

Read `run.json` first. Artifact files are created outside the checkout in the
runner's temporary directory and retained for 14 days.

| File | Contents |
| --- | --- |
| `run.json` | Checkout SHA, repository/run identity, dirty-state observation, collection status, review counts, helper exit code and packet digest when available. |
| `request.json` | The accepted explicit target/source selections and optional expected SHA. |
| `review.json` | Exact UTF-8 stdout from the Go helper, including selected source values and captured file hashes. It may contain a failure response or incomplete output on helper failure. |
| `stderr.log` | Helper diagnostics when the helper was invoked. |

Artifact upload is attempted even when collection fails. Setup failures before
the adapter starts may have no review artifact; inspect the failed setup step.
No access token or secret input is required by this workflow, and its repository
permission is limited to `contents: read`. Checkout credentials are not persisted.

- `NON_NORMATIVE_REVIEW`: collection succeeded without selection problems or
  declared-field inventory findings. It is not schema conformance or approval.
- `REVIEW_REQUIRED`: collection succeeded, with unresolved selections or field
  findings preserved for review. The job can succeed because it collected the
  requested evidence, while the review remains unresolved.
- `FAIL`: invalid inputs, SHA mismatch, an unexpected authority claim, or helper
  execution failure. The adapter returns a nonzero exit code and retains the
  available diagnostics.

The job summary repeats the status, source SHA and counts. It never claims
policy approval, semantic equivalence, execution authority or schema validation.
Unicode helper output is preserved as bytes; metadata is UTF-8 JSON with ASCII
escapes to avoid console encoding loss.

## Dependencies and verification integration

The collection job needs Go from the existing developer `go.mod`, Python 3.12,
and PyYAML 6.0.3 for the existing YAML serialization adapter. It does not install
the PTSIP VALIDATION runtime because this helper does not perform schema
validation. The existing Go contract selector remains responsible for selection.

The existing `tooling-test.yml` job selected with
`mode: developer-schema-contracts` runs the Go review tests and the transport
unit tests. Its existing retirement-gap reporting remains unchanged; adding
these tests does not revive retired Test Mode resolvers or assert registry
conformance.

Local checks, from the repository root:

```powershell
$env:PTSIP_PYTHON = (Resolve-Path .venv/Scripts/python.exe).Path
go -C .ptsip/engine/developer test ./policy/review/... ./internal/contractentry -count=1
go -C .ptsip/engine/developer vet ./policy/review/... ./internal/contractentry
& .venv/Scripts/python.exe -m unittest discover -s .ptsip/engine/developer/policy/review -p workflow_test.py -v
```

The original helper and local usage remain documented in `README.md`.
GitHub's [manual-dispatch documentation](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manually-run-a-workflow)
and [workflow syntax reference](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax)
describe the platform behavior.
