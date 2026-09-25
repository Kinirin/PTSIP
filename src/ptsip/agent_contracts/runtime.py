from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from .validator import (
    AgentContractValidationError,
    agent_contract_resource_size,
    load_agent_contract_yaml,
    validate_agent_contract_plane,
)


class AgentOperationLoadError(RuntimeError):
    """Stable failure for operation-scoped agent contract loading."""


@dataclass(frozen=True)
class AgentOperationBundle:
    operation_key: str
    operation_ref: str
    binding_ref: str
    binding: dict[str, Any]
    operation: dict[str, Any]
    specs: tuple[dict[str, Any], ...]
    vocabularies: tuple[dict[str, Any], ...]
    resource_refs: tuple[str, ...]
    context_bytes: int
    context_budget_bytes: int

    def as_dict(self) -> dict[str, Any]:
        return {
            "operation_key": self.operation_key,
            "operation_ref": self.operation_ref,
            "binding_ref": self.binding_ref,
            "binding": self.binding,
            "operation": self.operation,
            "specs": list(self.specs),
            "vocabularies": list(self.vocabularies),
            "resource_refs": list(self.resource_refs),
            "context_bytes": self.context_bytes,
            "context_budget_bytes": self.context_budget_bytes,
        }


def load_operation_contract(operation_key: str) -> AgentOperationBundle:
    if not isinstance(operation_key, str) or not operation_key.strip():
        raise AgentOperationLoadError("operation_key must be a non-empty string")
    key = operation_key.strip()

    try:
        validate_agent_contract_plane()
        index = load_agent_contract_yaml("index.yaml")
        matches = [entry for entry in index["operations"] if entry["id"] == key]
        if len(matches) != 1:
            raise AgentOperationLoadError(f"Unknown agent operation: {key}")

        operation_ref = matches[0]["ref"]
        binding_ref = index["binding"]["current_ref"]
        binding = load_agent_contract_yaml(binding_ref)
        if operation_ref not in binding["active_operations"]:
            raise AgentOperationLoadError(f"Inactive agent operation: {key}")

        operation = load_agent_contract_yaml(operation_ref)
        spec_refs = tuple(operation["spec_refs"])
        vocabulary_refs = tuple(operation["vocabulary_refs"])

        inactive_specs = sorted(set(spec_refs) - set(binding["active_specs"]))
        inactive_vocabularies = sorted(
            set(vocabulary_refs) - set(binding["active_vocabularies"])
        )
        if inactive_specs or inactive_vocabularies:
            raise AgentOperationLoadError(
                "Operation requires inactive machine resources: "
                f"specs={inactive_specs}, vocabularies={inactive_vocabularies}"
            )

        specs = tuple(load_agent_contract_yaml(ref) for ref in spec_refs)
        vocabularies = tuple(load_agent_contract_yaml(ref) for ref in vocabulary_refs)
        resource_refs = (binding_ref, operation_ref, *spec_refs, *vocabulary_refs)
        context_bytes = sum(agent_contract_resource_size(ref) for ref in resource_refs)
        budget = int(binding["loading"]["max_context_bytes"])
        if context_bytes > budget:
            raise AgentOperationLoadError(
                f"Operation-scoped context budget exceeded for {key}: "
                f"{context_bytes} > {budget}"
            )

        return AgentOperationBundle(
            operation_key=key,
            operation_ref=operation_ref,
            binding_ref=binding_ref,
            binding=binding,
            operation=operation,
            specs=specs,
            vocabularies=vocabularies,
            resource_refs=resource_refs,
            context_bytes=context_bytes,
            context_budget_bytes=budget,
        )
    except AgentContractValidationError as exc:
        raise AgentOperationLoadError(str(exc)) from exc
