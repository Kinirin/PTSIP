package machine

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

func planningEqual(a, b any) bool {
	left, _ := CanonicalJSON(a)
	right, _ := CanonicalJSON(b)
	return string(left) == string(right)
}
func planningGovernanceErrors(payload any, registry Object, label string) []string {
	constants := Map(registry["constants"])
	if constants == nil {
		return []string{"governance source constants must be mapping"}
	}
	roles := map[string]string{"decision_source": "DECISION_SOURCE", "approval_source": "APPROVAL_SOURCE", "authorization_source": "AUTHORIZATION_SOURCE", "transition_source": "TRANSITION_SOURCE"}
	errors := []string{}
	var visit func(any, string)
	visit = func(value any, path string) {
		if object := Map(value); object != nil {
			keys := []string{}
			for key := range object {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				child := object[key]
				childPath := path + "." + key
				if role := roles[key]; role != "" {
					source := Map(constants[Text(child)])
					valid := false
					for _, raw := range List(source["allowed_roles"]) {
						valid = valid || raw == role
					}
					if !valid {
						errors = append(errors, label+": "+childPath+" must resolve a governance source allowed for "+role)
					}
				} else if key == "source" && (child == "DIRECT_PROJECT_OWNER_INSTRUCTION" || child == "DIRECT_PROJECT_OWNER_TEMPORARY_APPROVAL") {
					errors = append(errors, label+": legacy governance source requires explicit role")
				}
				visit(child, childPath)
			}
		} else {
			for i, child := range List(value) {
				visit(child, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
	visit(payload, "")
	return errors
}
func (r *Repository) ValidatePlanning() []string {
	errors := []string{}
	root, err := r.Read(PlanningRootIndex)
	if err != nil {
		return []string{err.Error()}
	}
	if err = r.Validate("developer/planning/schemas/planning-root-index.schema.json", root); err != nil {
		errors = append(errors, err.Error())
	}
	sources, err := r.Read("developer/policy/registries/governance-source-registry.yaml")
	if err != nil {
		return append(errors, err.Error())
	}
	planningRoot, err := r.Path("developer/planning")
	if err != nil {
		return append(errors, err.Error())
	}
	walkErr := filepath.WalkDir(planningRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}
		scope, err := r.Scope(path)
		if err != nil {
			return err
		}
		payload, err := r.Read(scope)
		if err != nil {
			errors = append(errors, err.Error())
			return nil
		}
		errors = append(errors, planningGovernanceErrors(payload, sources, scope)...)
		return nil
	})
	if walkErr != nil {
		errors = append(errors, walkErr.Error())
	}
	for _, raw := range List(root["plans"]) {
		entry := Map(raw)
		path := Text(entry["path"])
		if path == "" {
			continue
		}
		plan, err := r.Read(path)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		prerelease := entry["schema_version"] == "ptsip-prerelease-plan-entry/v1alpha1"
		schema := "developer/planning/schemas/planning-index.schema.json"
		wuSchema := "developer/planning/schemas/work-unit.schema.json"
		if prerelease {
			schema = "developer/planning/schemas/prerelease-planning-index.schema.json"
			wuSchema = "developer/planning/schemas/prerelease-work-unit.schema.json"
		}
		if err = r.Validate(schema, plan); err != nil {
			errors = append(errors, path+": "+err.Error())
			continue
		}
		identity := Map(plan["plan"])
		routing := Map(entry["entry_routing"])
		planRouting := Map(plan["responsibility_routing"])
		indexed, err := planningIndexed(plan)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		docs := map[string]Object{}
		gate := Text(identity["current_gate"])
		if prerelease {
			gate = Text(Map(plan["execution_model"])["current_gate"])
			expected := "developer/planning/" + Text(identity["development_line"]) + "/" + Text(identity["prerelease"]) + "/index.yaml"
			if path != expected || identity["canonical_location"] != path {
				errors = append(errors, path+": prerelease canonical location mismatch")
			}
			for _, key := range []string{"integration_branch", "status"} {
				if entry[key] != identity[key] {
					errors = append(errors, path+": root registration "+key+" mismatch")
				}
			}
			if entry["plan_version"] != identity["prerelease"] {
				errors = append(errors, path+": root prerelease mismatch")
			}
			for _, key := range []string{"resolved_plan_id", "plan_file_id", "version", "revision"} {
				if Text(Map(plan["plan_identity"])[key]) == "" {
					errors = append(errors, path+": plan_identity."+key+" missing")
				}
			}
			if err := r.validatePlanningFormalIdentity(Map(plan["plan_identity"])); err != nil {
				errors = append(errors, path+": "+err.Error())
			}
		} else if routing != nil {
			if planRouting == nil {
				errors = append(errors, path+": responsibility_routing missing")
			} else {
				for _, key := range []string{"model", "canonical_plan", "branch_creation_parent"} {
					if !planningEqual(routing[key], planRouting[key]) {
						errors = append(errors, path+": routing "+key+" mismatch")
					}
				}
				if routing["responsibility_control_plane"] != path || planRouting["branch_creation_parent"] != identity["integration_branch"] {
					errors = append(errors, path+": routing location mismatch")
				}
				left, right := Map(routing["dependency_bearing_convergence"]), Map(planRouting["dependency_bearing_convergence"])
				if left["id"] != right["id"] || left["responsibility"] != right["responsibility"] {
					errors = append(errors, path+": convergence routing mismatch")
				}
				if !planningEqual(planningLeafTuples(routing), planningLeafTuples(planRouting)) {
					errors = append(errors, path+": independent leaf routing mismatch")
				}
				if convergence := indexed[Text(right["id"])]; convergence == nil || !planningEqual(convergence["depends_on"], right["expected_depends_on"]) {
					errors = append(errors, path+": convergence dependencies mismatch")
				}
				for _, raw := range List(planRouting["independent_leaf_work_units"]) {
					leaf := Map(raw)
					row := indexed[Text(leaf["id"])]
					if row == nil || len(List(row["depends_on"])) != 0 {
						errors = append(errors, path+": independent leaf must be indexed with empty dependencies")
					}
				}
			}
		}
		if routing != nil {
			errors = append(errors, planningRoutingErrors(entry, plan, indexed)...)
		}
		gateResolved := false
		for _, raw := range List(plan["work_units"]) {
			row := Map(raw)
			id, ref := Text(row["id"]), Text(row["path"])
			if id == gate {
				gateResolved = true
			}
			if ref == "" {
				continue
			}
			payload, err := r.Read(ref)
			if err != nil {
				errors = append(errors, err.Error())
				continue
			}
			if err = r.Validate(wuSchema, payload); err != nil {
				errors = append(errors, ref+": "+err.Error())
				continue
			}
			wu := Map(payload["work_unit"])
			docs[id] = payload
			if wu["id"] != id {
				errors = append(errors, ref+": work unit identity mismatch")
			}
			if prerelease {
				if payload["plan_id"] != identity["id"] || ref != strings.TrimSuffix(path, "index.yaml")+id+"/"+id+".yaml" {
					errors = append(errors, ref+": canonical prerelease work unit identity mismatch")
				}
			} else {
				for _, key := range []string{"lifecycle", "approval", "implementation_authorization"} {
					actual, registered := Map(wu[key]), Map(row[key])
					if actual["status"] != registered["status"] {
						errors = append(errors, ref+": "+key+" status mismatch")
					}
					sourceKey := ""
					if key == "approval" {
						sourceKey = "approval_source"
					}
					if key == "implementation_authorization" {
						sourceKey = "authorization_source"
					}
					if sourceKey != "" && actual[sourceKey] != registered[sourceKey] {
						errors = append(errors, ref+": "+sourceKey+" mismatch")
					}
				}
			}
			if !planningEqual(wu["depends_on"], row["depends_on"]) {
				errors = append(errors, ref+": dependencies mismatch")
			}
			if planningStatus(wu) == "COMPLETE" {
				if len(List(payload["completion_evidence"])) == 0 {
					errors = append(errors, ref+": COMPLETE work unit requires completion_evidence")
				} else if !prerelease && !planningEqual(payload["completion_evidence"], row["completion_evidence"]) {
					errors = append(errors, ref+": completion evidence mismatch")
				}
			}
			for _, raw := range List(payload["extensions"]) {
				extRow := Map(raw)
				extRef, extID := Text(extRow["path"]), Text(extRow["id"])
				extension, err := r.Read(extRef)
				if err != nil {
					errors = append(errors, err.Error())
					continue
				}
				if err = r.Validate("developer/planning/schemas/plan-extension.schema.json", extension); err != nil {
					errors = append(errors, err.Error())
				}
				errors = append(errors, ExtensionParentConsistency(payload, extension, extID, extRef)...)
				status := planningStatus(Map(extension["extension"]))
				if ExtensionMachineReady(extension) && status != "COMPLETE" {
					errors = append(errors, extRef+": machine-ready extension remains nonterminal")
				}
				if Map(extension["extension"])["id"] == gate {
					gateResolved = true
					if status == "COMPLETE" || status == "SUPERSEDED" || status == "CANCELLED" {
						errors = append(errors, path+": current gate points to terminal extension")
					}
				}
			}
		}
		if !gateResolved {
			errors = append(errors, path+": current gate does not resolve to registered work unit or extension")
		}
		if prerelease {
			errors = append(errors, planningPrereleaseDependencies(plan, docs)...)
		} else if Map(routing["merge_reconciliation"]) != nil {
			state, err := r.BuildPlanningMaterializedState(entry, plan, docs)
			if err != nil {
				errors = append(errors, err.Error())
			} else if !planningEqual(entry["materialized_state"], state) {
				errors = append(errors, PlanningRootIndex+": materialized_state stale")
			}
		}
	}
	return errors
}
func planningLeafTuples(routing Object) []string {
	out := []string{}
	for _, raw := range List(routing["independent_leaf_work_units"]) {
		row := Map(raw)
		value := []any{}
		for _, key := range []string{"id", "branch", "responsibility", "state", "continuation_branch"} {
			value = append(value, row[key])
		}
		encoded, _ := CanonicalJSON(value)
		out = append(out, string(encoded))
	}
	sort.Strings(out)
	return out
}
func (r *Repository) validatePlanningFormalIdentity(identity Object) error {
	key := "urn:ptsip:go:formal-plan-identity:v2"
	compiler, err := r.Compiler()
	if err != nil {
		return err
	}
	compiled := r.schemas[key]
	if compiled == nil {
		binding, err := r.Read(BindingSchemaPath)
		if err != nil {
			return err
		}
		bindingProperties := Map(Map(Map(binding["$defs"])["binding"])["properties"])
		properties := Object{}
		fields := []any{"resolved_plan_id", "plan_file_id", "version", "revision"}
		for _, field := range fields {
			properties[Text(field)] = bindingProperties[Text(field)]
		}
		schema := Object{"$schema": "https://json-schema.org/draft/2020-12/schema", "$id": key, "type": "object", "required": fields, "properties": properties, "additionalProperties": false}
		if err = compiler.AddResource(key, schema); err != nil {
			return err
		}
		compiled, err = compiler.Compile(key)
		if err != nil {
			return err
		}
		r.schemas[key] = compiled
	}
	return compiled.Validate(identity)
}
func planningRoutingErrors(entry, plan Object, indexed map[string]Object) []string {
	errors := []string{}
	path := Text(entry["path"])
	routing := Map(entry["entry_routing"])
	resolver, entryResolution := Map(routing["resolver"]), Map(plan["entry_resolution"])
	if entry["schema_version"] != "ptsip-prerelease-plan-entry/v1alpha1" {
		for _, pair := range [][2]string{{"module", "resolver_module"}, {"match_mode", "match_mode"}, {"unmatched_behavior", "unknown_branch_behavior"}} {
			if resolver[pair[0]] != entryResolution[pair[1]] {
				errors = append(errors, path+": resolver "+pair[0]+" mismatch")
			}
		}
		merge := Map(routing["merge_reconciliation"])
		if merge["target_branch"] != entry["integration_branch"] || merge["leaf_shared_index_mutation"] != "FORBIDDEN" || merge["state_source"] != "WORK_UNIT_DOCUMENTS" {
			errors = append(errors, path+": invalid merge reconciliation contract")
		}
	}
	branches := map[string]bool{}
	expected := map[string]bool{Text(entry["integration_branch"]): true}
	integration := 0
	for _, raw := range List(routing["independent_leaf_work_units"]) {
		expected[Text(Map(raw)["branch"])] = true
	}
	migration := Map(entry["branch_identity_migration"])
	pending := migration["status"] == "RENAME_PENDING"
	legacy := Text(migration["legacy_branch"])
	if pending {
		expected[legacy] = true
	}
	aliases := 0
	if migration != nil && (migration["canonical_branch"] != entry["integration_branch"] || legacy == "" || legacy == Text(entry["integration_branch"])) {
		errors = append(errors, path+": invalid branch identity migration")
	}
	for _, raw := range List(routing["branch_entrypoints"]) {
		row := Map(raw)
		branch := Text(row["branch"])
		if branches[branch] {
			errors = append(errors, path+": duplicate branch entrypoint")
		}
		branches[branch] = true
		if branch == entry["integration_branch"] {
			integration++
			if row["entry_document"] != path {
				errors = append(errors, path+": invalid integration entrypoint")
			}
			if entry["schema_version"] != "ptsip-prerelease-plan-entry/v1alpha1" && (row["role"] != "INTEGRATION_CONTROL_PLANE" || row["state"] != "ACTIVE" || row["work_unit"] != nil) {
				errors = append(errors, path+": invalid integration role/state")
			}
		}
		if row["role"] == "BRANCH_RENAME_SOURCE_ALIAS" {
			aliases++
			if !pending || branch != legacy || row["canonical_branch"] != entry["integration_branch"] || row["entry_document"] != path || row["state"] != "ACTIVE" {
				errors = append(errors, path+": invalid branch rename alias")
			}
		}
		if row["role"] == "INDEPENDENT_LEAF" {
			id := Text(row["work_unit"])
			registered := indexed[id]
			if registered == nil || row["entry_document"] != registered["path"] {
				errors = append(errors, path+": leaf work unit identity mismatch")
			}
			if row["state"] == "MERGED" {
				if row["merged_into"] != entry["integration_branch"] && row["merged_into"] != legacy {
					errors = append(errors, path+": invalid merged leaf target")
				}
				if continuation := row["continuation_branch"]; continuation != nil && continuation != entry["integration_branch"] {
					errors = append(errors, path+": invalid leaf continuation branch")
				}
			} else if row["state"] != "ACTIVE" {
				errors = append(errors, path+": invalid leaf entrypoint state")
			}
		}
	}
	if integration != 1 {
		errors = append(errors, path+": integration branch must resolve exactly once")
	}
	if entry["schema_version"] != "ptsip-prerelease-plan-entry/v1alpha1" && !reflect.DeepEqual(branches, expected) {
		errors = append(errors, path+": branch entrypoint set mismatch")
	}
	if pending && aliases != 1 || !pending && aliases != 0 {
		errors = append(errors, path+": source alias count mismatch")
	}
	for _, raw := range List(routing["independent_leaf_work_units"]) {
		route := Map(raw)
		matches := []Object{}
		for _, raw := range List(routing["branch_entrypoints"]) {
			row := Map(raw)
			if row["branch"] == route["branch"] {
				matches = append(matches, row)
			}
		}
		if len(matches) != 1 || matches[0]["role"] != "INDEPENDENT_LEAF" || matches[0]["work_unit"] != route["id"] {
			errors = append(errors, path+": independent leaf entrypoint identity mismatch")
		}
	}
	return errors
}
func planningPrereleaseDependencies(plan Object, docs map[string]Object) []string {
	errors := []string{}
	seen := map[string]bool{}
	preceding := map[string]bool{}
	for _, rawBatch := range List(Map(plan["execution_model"])["dependency_order"]) {
		batch := List(rawBatch)
		for _, raw := range batch {
			id := Text(raw)
			if seen[id] || docs[id] == nil {
				errors = append(errors, "dependency order must contain each registered work unit exactly once")
			}
			seen[id] = true
			for _, dep := range List(Map(docs[id]["work_unit"])["depends_on"]) {
				if !preceding[Text(dep)] {
					errors = append(errors, "dependency order invalid for "+id)
				}
			}
		}
		for _, raw := range batch {
			preceding[Text(raw)] = true
		}
	}
	if len(seen) != len(List(plan["work_units"])) {
		errors = append(errors, "dependency order coverage mismatch")
	}
	gate := Text(Map(plan["execution_model"])["current_gate"])
	for _, dep := range List(Map(docs[gate]["work_unit"])["depends_on"]) {
		if planningStatus(Map(docs[Text(dep)]["work_unit"])) != "COMPLETE" {
			errors = append(errors, "current gate dependency incomplete: "+Text(dep))
		}
	}
	return errors
}
