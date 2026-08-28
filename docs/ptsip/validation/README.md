# Local validation captures

Use the explicit capture command when a requested local validation run needs a
repository-local evidence log:

```text
ptsip validation capture -- <command> [args...]
```

The capture command executes the literal argument vector exactly once without a
shell, records both output streams and the child exit code in a timestamped
`.log` file in this directory, and creates a local commit containing only that
generated log. A nonzero child exit still produces and commits the log, and the
same exit code is returned to the caller.

The command does not infer a validation command, create a summary or manifest,
include unrelated staged or untracked work, or push the resulting commit.
