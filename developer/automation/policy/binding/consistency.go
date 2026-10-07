package binding

import (
	"fmt"
	"regexp"
	"strings"
)

var bindingFailureCode = regexp.MustCompile(`^[A-Z][A-Z0-9]*(_[A-Z0-9]+)+$`)

func bindingErrorCode(err error, fallback string) string {
	prefix, _, _ := strings.Cut(err.Error(), ":")
	if bindingFailureCode.MatchString(prefix) {
		return prefix
	}
	return fallback
}

func (r *Store) VerifyConsistency() (Object, error) {
	failures := []any{}
	add := func(code, message, id, reference string) {
		var bindingID, ref any
		if id != "" {
			bindingID = id
		}
		if reference != "" {
			ref = reference
		}
		failures = append(failures, Object{"code": code, "message": message, "binding_id": bindingID, "ref": ref})
	}
	report := func(total, checked int) Object {
		status := "PASS"
		if len(failures) > 0 {
			status = "FAIL"
		}
		return Object{"status": status, "binding_count": total, "checked_binding_count": checked, "failures": failures}
	}
	snapshot, err := r.LoadBindingRegistry(true)
	if err != nil {
		add(bindingErrorCode(err, "BINDING_REGISTRY_INVALID"), err.Error(), "", "")
		return report(0, 0), nil
	}
	bindings := List(snapshot.Payload["bindings"])
	resolver, err := NewResolver(r.Repository)
	if err != nil {
		add("POLICY_INDEX_INVALID", err.Error(), "", "")
		return report(len(bindings), 0), nil
	}
	known := map[string]bool{}
	for _, entry := range resolver.Index {
		if entry["authority_role"] != "MIGRATION_SOURCE" {
			known[Text(entry["id"])] = true
		}
	}
	seenIDs, relations := map[string]bool{}, map[string]bool{}
	planOwners := map[string]string{}
	checked := 0
	for _, raw := range bindings {
		checked++
		binding := Map(raw)
		id := Text(binding["binding_id"])
		policy := Text(binding["policy_ref"])
		if id == "" {
			add("BINDING_ID_INVALID", "binding_id must be a string", "", "")
			continue
		}
		if seenIDs[id] {
			add("DUPLICATE_BINDING_ID", "duplicate binding_id "+id, id, "")
		}
		seenIDs[id] = true
		if !known[policy] {
			add("UNKNOWN_POLICY_REF", "policy_ref does not resolve as canonical policy authority", id, "")
			continue
		}
		exact, err := r.ResolveBindings(Object{"binding_id": id, "policy_ref": policy})
		if err != nil {
			add(bindingErrorCode(err, "EXACT_BINDING_RESOLUTION_FAILED"), err.Error(), id, "")
			continue
		}
		if len(List(exact["bindings"])) != 1 {
			add("EXACT_BINDING_RESOLUTION_FAILED", "binding_id + policy_ref must resolve once", id, "")
			continue
		}
		if binding["planning_state"] == "NOT_CREATED" {
			continue
		}
		if binding["planning_state"] != "CREATED" {
			add("PLANNING_STATE_INVALID", "unsupported planning_state", id, "")
			continue
		}
		query := Object{"binding_id": id, "policy_ref": policy}
		complete := true
		for _, field := range []string{"resolved_plan_id", "plan_file_id", "version", "revision", "plan_ref"} {
			if Text(binding[field]) == "" {
				complete = false
			}
			if field != "version" && field != "revision" {
				query[field] = binding[field]
			}
		}
		if !complete {
			add("CREATED_BINDING_IDENTITY_INCOMPLETE", "CREATED binding identity is incomplete", id, "")
			continue
		}
		resolvedID, fileID, reference := Text(binding["resolved_plan_id"]), Text(binding["plan_file_id"]), Text(binding["plan_ref"])
		relation := createdRelation(binding)
		if relations[relation] {
			add("DUPLICATE_CREATED_RELATION", "duplicate policy to plan relation", id, "")
		}
		relations[relation] = true
		if prior, found := planOwners[fileID]; found && prior != resolvedID {
			add("PLAN_FILE_ID_CONFLICT", "plan_file_id associated with multiple resolved plans", id, "")
		} else {
			planOwners[fileID] = resolvedID
		}
		exact, err = r.ResolveBindings(query)
		if err != nil {
			add(bindingErrorCode(err, "EXACT_PLAN_BINDING_RESOLUTION_FAILED"), err.Error(), id, reference)
			continue
		}
		if len(List(exact["bindings"])) != 1 {
			add("EXACT_PLAN_BINDING_RESOLUTION_FAILED", "full binding identity must resolve once", id, reference)
			continue
		}
		tracking, err := r.TrackPlanRef(id, false)
		if err != nil {
			add(bindingErrorCode(err, "PLAN_REF_TRACKING_FAILED"), err.Error(), id, reference)
			continue
		}
		target := Text(tracking["discovered_plan_ref"])
		switch tracking["status"] {
		case "CURRENT":
		case "RECONCILE_REQUIRED":
			add("PLAN_REF_RECONCILE_REQUIRED", "registered plan_ref must be reconciled", id, target)
		case "UNRESOLVED":
			add("PLAN_REF_UNRESOLVED", "no planning document resolves plan_file_id", id, reference)
			target = ""
		default:
			add("PLAN_REF_TRACKING_STATE_INVALID", "unexpected tracking state", id, reference)
			target = ""
		}
		if target == "" {
			continue
		}
		identity, err := r.PlanIdentity(target)
		if err != nil {
			add(bindingErrorCode(err, "PLAN_DOCUMENT_INVALID"), err.Error(), id, target)
			continue
		}
		for _, mismatch := range []struct{ field, code string }{{"resolved_plan_id", "RESOLVED_PLAN_ID_MISMATCH"}, {"plan_file_id", "PLAN_FILE_ID_MISMATCH"}, {"version", "PLAN_VERSION_MISMATCH"}, {"revision", "PLAN_REVISION_MISMATCH"}} {
			field, code := mismatch.field, mismatch.code
			if identity[field] != binding[field] {
				add(code, fmt.Sprintf("%s expected %v, got %v", field, binding[field], identity[field]), id, target)
			}
		}
	}
	return report(len(bindings), checked), nil
}
