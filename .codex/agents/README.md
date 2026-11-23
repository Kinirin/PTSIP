# PTSIP project agents

Codex loads the checked-in TOML definitions in this directory for the trusted PTSIP checkout. Open a new Codex session from the repository root after pulling changes. You can delegate by the stable `name`, for example “Use `profile_analysis` to trace the active Project Profile registration.” A project agent uses its own definition and the current task context. It can run from any checkout of this repository.

[`project-agents.json`](project-agents.json) maps the 21 schema-centralization agents to their original task names, historical display names, and runnable definitions. Use `agent_id` as the stable identity in results. Preserve the existing IDs when refining an agent’s instructions. Other project agents in this directory remain available under their existing names.

When creating a reusable PTSIP agent, add a TOML definition here with a unique `name`, a targeted `description`, and `developer_instructions` for its responsibility, workflow, boundaries, and expected output. Add its stable identity to the catalog and use a fresh project session to check that Codex can delegate to it by name. Keep machine-specific paths and credentials out of its instructions. Give analysis-only agents a read-only sandbox.

Store reusable results under [the project analysis directory](../analysis/project-agents/README.md). Review and commit the configuration and shareable evidence so another checkout can use both.
