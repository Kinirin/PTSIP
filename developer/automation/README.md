# Developer automation

Run Developer automation through the native Go command surface from the repository root:

```powershell
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-resolver resolve --scope developer/automation --operation READ
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-validator validate
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-plan-consistency verify
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev context-projection check
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev agent-instruction-materializer status
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev agent-context-migration verify --stage AUTO
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev automation-migration inspect
```

The build tags select the registered grammar subset used by CI. Python source inspection
uses a native parser; it does not execute or import Python automation.

`agent-integration status` reads an existing `.agent/index.yaml`. Use the materializer
preview above before initializing an Agent surface through the registered materialization
or bootstrap workflow. Preview does not modify `AGENTS.md` or create execution authority.

For repeated calls, build the CLI once. The output path is relative to the Go module:

```powershell
go -C developer/automation build -tags grammar_subset,grammar_subset_python -o ../../.artifacts/ptsip-dev.exe ./cmd/ptsip-dev
& ./.artifacts/ptsip-dev.exe --repository . policy-validator validate
```

The closed command vocabulary, required inputs, and nonfatal statuses are registered in
[`go-automation-cutover.v1.json`](../policy/contracts/go-automation-cutover.v1.json).
Policy resolution uses canonical Root policy IDs and exact sections. Historical policy
IDs and pre-retirement protocol fixtures provide audit evidence only.

The migration inventory records the original Python module and its explicit Go targets.
`automation-migration inspect` reports source retirement separately from final cutover
verification. A source-ready report does not grant release or operational approval.

Responsibility packages include `branch`, `release`, `planning`, `policy/binding`, and
`policy/lifecycle`. Existing Agent, IWP, PP, and shared repository adapters remain under
`internal/machine` while their package ownership is resolved through the approved
responsibility-domain plan. Package placement does not transfer policy authority.

Release build orchestration and product regression runners can invoke the product's
Python build and test tools. Those tools belong to the product runtime and test
toolchain; Developer automation itself has no Python implementation entrypoint.

Run native regression checks with:

```powershell
go -C developer/automation test -tags grammar_subset,grammar_subset_python -count=1 ./...
go -C developer/tests test -tags grammar_subset,grammar_subset_python -count=1 ./...
```
