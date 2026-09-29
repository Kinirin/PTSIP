"""Compile validated migration candidates, then resolve exact declared graphs."""
from __future__ import annotations

from copy import deepcopy
from importlib.resources import files

from .candidate import pointer, read_json, resource, validate_candidate_assets


class MachineContractResolver:
    def __init__(self, root=None):
        self.root = root or files("agent_contracts")
        # Whole-set validation belongs to compilation, not identity selection.
        validate_candidate_assets(self.root)
        self.index = read_json(self.root, "index.json")
        self.group = read_json(self.root, self.index["contract_ref"])
        refs = {"index.json", "index.yaml", "bindings/current.yaml", self.index["contract_ref"],
                self.index["conformance_ref"], *self.index["schemas"].values(),
                "schemas/candidate-index.schema.json", "schemas/candidate-group.schema.json",
                "schemas/candidate-vectors.schema.json"}
        refs.update(ref for by_id in self.index["migration_sources"].values() for ref in by_id.values())
        self._compiled_bytes = {ref: resource(self.root, ref).read_bytes() for ref in sorted(refs)}

    def resolve(self, operation_id: str) -> dict:
        def unresolved(reason):
            return {"status": "UNRESOLVED", "operation_id": operation_id, "reason": reason,
                    "mutation_authorized": False, "normative_authority": False}
        if not isinstance(operation_id, str) or operation_id not in self.index["primitives"]:
            return unresolved("UNKNOWN_EXACT_IDENTITY")
        try:
            if any(resource(self.root, ref).read_bytes() != expected
                   for ref, expected in self._compiled_bytes.items()):
                return unresolved("REVALIDATION_REQUIRED")
            payloads = self.group["payloads"]
            op = payloads["operations"][operation_id]
            primitive = self.index["primitives"][operation_id]
            actions, gates, conditions, schemas = {}, {}, {}, {}
            pending = [item["condition_ref"] for item in op["preconditions"]]
            io_ids = {op["input_schema_ref"], op["output_schema_ref"]}
            for step in op["steps"]:
                if step["type"] == "CONDITION":
                    pending.append(step["condition_ref"])
                    continue
                action_id = step["action_ref"]
                action = payloads["actions"][action_id]
                actions[action_id] = action
                io_ids.update((action["input_schema_ref"], action["output_schema_ref"]))
                pending.extend(check["condition_ref"] for field in ("preconditions", "postconditions")
                               for check in action[field])
                if "mutation_gate_ref" in step:
                    gate_id = step["mutation_gate_ref"]
                    gate = payloads["gates"][gate_id]
                    gates[gate_id] = gate
                    io_ids.add(gate["input_schema_ref"])
                    pending.extend(check["condition_ref"] for check in gate["requirements"])
            while pending:
                identity = pending.pop()
                if identity in conditions:
                    continue
                condition = payloads["conditions"][identity]
                conditions[identity] = condition
                io_ids.add(condition["input_schema_ref"])
                evaluation = condition["evaluation"]
                pending.extend(evaluation.get("condition_refs", []))
                if "condition_ref" in evaluation:
                    pending.append(evaluation["condition_ref"])
            for identity in sorted(io_ids):
                schemas[identity] = payloads["io_schemas"][identity]
            rules = [pointer(self.group, self.index["rule_index"][rule_id]) for rule_id in op["rule_refs"]]
            result = {"status": "RESOLVED", "operation_id": operation_id, "primitive": primitive,
                      "operation": op, "rules": rules, "actions": actions, "gates": gates,
                      "conditions": {k: conditions[k] for k in sorted(conditions)}, "io_schemas": schemas,
                      "vocabularies": {k: payloads["vocabularies"][k] for k in op["vocabulary_refs"]},
                      "semantic_source": "VALIDATED_CURRENT_COMPATIBILITY_SOURCE",
                      "normative_authority": False, "mutation_authorized": False}
            return deepcopy(result)
        except (OSError, KeyError, ValueError, TypeError) as exc:
            return unresolved(f"UNRESOLVED_REQUIRED_REFERENCE: {exc}")
