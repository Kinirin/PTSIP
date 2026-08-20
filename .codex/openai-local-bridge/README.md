# PTSIP repository-local OpenAI Local Bridge entrypoints

This directory owns the PTSIP-side invocation surface for OpenAI Local Bridge verification.

## Principle

PTSIP verification commands MUST NOT depend on the filesystem location, checkout name, active branch, or virtual environment of a separate `Kinirin/OpenAI_Local_Bridge` source checkout.

Repository-local commands may depend on the installed Bridge runtime and machine state only:

```text
%LOCALAPPDATA%\OpenAI_Local_Bridge\...
%PROGRAMDATA%\OpenAI_Local_Bridge\activation.json
```

They may also depend on PTSIP-owned files such as this directory, `.venv`, and `.codex/openai-local-bridge/tasks.toml`.

A developer moving, renaming, switching branches in, or deleting a separate OpenAI Local Bridge source checkout MUST NOT invalidate an otherwise installed/activated PTSIP verification path.

## Entrypoints

```powershell
# Read transport health without exposing the capability URL.
.\.codex\openai-local-bridge\OpenAI-Local-Bridge-Remote.ps1 status

# Recreate the quick remote transport. Requires Administrator PowerShell.
.\.codex\openai-local-bridge\OpenAI-Local-Bridge-Remote.ps1 start-quick

# Run one registered PTSIP read-only Bridge task.
.\.codex\openai-local-bridge\Test-Remote-Tasks.ps1 -TaskId wu04g-scope
```

`Test-Remote-Tasks.ps1` uses the capability URL only inside the process and its result intentionally reports `capability_url_exposed: false`.

## Boundary

These scripts do not vendor the Bridge implementation into PTSIP. The Bridge runtime remains independently installed and activated. PTSIP owns only its stable repository-local lifecycle/acceptance invocation surface so verification is not coupled to another repository checkout.
