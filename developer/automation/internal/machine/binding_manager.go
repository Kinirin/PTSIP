package machine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (r *Repository) PlanIdentity(ref string) (Object, error) {
	scope, err := r.Scope(ref)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(scope, "developer/planning/") {
		return nil, fmt.Errorf("PLAN_REF_OUTSIDE_PLANNING_NAMESPACE")
	}
	payload, err := r.Read(scope)
	if err != nil {
		return nil, err
	}
	identity := Map(payload["plan_identity"])
	for _, key := range []string{"resolved_plan_id", "plan_file_id", "version", "revision"} {
		if Text(identity[key]) == "" {
			return nil, fmt.Errorf("PLAN_IDENTITY_INCOMPLETE: %s", key)
		}
	}
	return identity, nil
}
func (r *Repository) checkPlanIdentity(values Object) error {
	identity, err := r.PlanIdentity(Text(values["plan_ref"]))
	if err != nil {
		return err
	}
	for _, key := range []string{"resolved_plan_id", "plan_file_id", "version", "revision"} {
		if value := Text(values[key]); value != "" && identity[key] != value {
			return fmt.Errorf("PLAN_IDENTITY_MISMATCH: %s", key)
		}
	}
	return nil
}
func (r *Repository) CreateBinding(policy string) (Object, error) {
	snap, err := r.LoadBindingRegistry(true)
	if err != nil {
		return nil, err
	}
	rows := List(snap.Payload["bindings"])
	max := 0
	for _, raw := range rows {
		n, _ := strconv.Atoi(strings.SplitN(strings.TrimPrefix(Text(Map(raw)["binding_id"]), "PPB-"), "-", 2)[0])
		if n > max {
			max = n
		}
	}
	item := Object{"binding_id": fmt.Sprintf("PPB-%04d", max+1), "policy_ref": policy, "planning_state": "NOT_CREATED"}
	snap.Payload["bindings"] = append(rows, item)
	if _, err = r.ReplaceBindingRegistry(snap.Payload, snap.Digest, r.ValidateBindingRegistry); err != nil {
		return nil, err
	}
	return Object{"status": "CREATED", "changed": true, "binding": item}, nil
}
func (r *Repository) LinkPlan(values Object) (Object, error) {
	if err := r.checkPlanIdentity(values); err != nil {
		return nil, err
	}
	snap, err := r.LoadBindingRegistry(true)
	if err != nil {
		return nil, err
	}
	rows := List(snap.Payload["bindings"])
	matches := []Object{}
	for _, raw := range rows {
		item := Map(raw)
		if item["policy_ref"] != values["policy_ref"] {
			continue
		}
		if id := Text(values["binding_id"]); id != "" {
			if item["binding_id"] == id {
				matches = append(matches, item)
			}
		} else if item["planning_state"] == "NOT_CREATED" {
			matches = append(matches, item)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("BINDING_LINK_TARGET_AMBIGUOUS_OR_MISSING")
	}
	target := matches[0]
	fields := []string{"resolved_plan_id", "plan_file_id", "version", "revision", "plan_ref"}
	if target["planning_state"] == "CREATED" {
		same := true
		for _, key := range fields {
			same = same && target[key] == values[key]
		}
		if same {
			return Object{"status": "CURRENT", "changed": false, "binding": target}, nil
		}
		return nil, fmt.Errorf("BINDING_ALREADY_MATERIALIZED")
	}
	for _, raw := range rows {
		item := Map(raw)
		if item["binding_id"] != target["binding_id"] && item["planning_state"] == "CREATED" && item["policy_ref"] == values["policy_ref"] && item["resolved_plan_id"] == values["resolved_plan_id"] {
			return nil, fmt.Errorf("DUPLICATE_CREATED_RELATION")
		}
	}
	target["planning_state"] = "CREATED"
	for _, key := range fields {
		target[key] = values[key]
	}
	if _, err = r.ReplaceBindingRegistry(snap.Payload, snap.Digest, r.ValidateBindingRegistry); err != nil {
		return nil, err
	}
	return Object{"status": "LINKED", "changed": true, "binding": target}, nil
}
func (r *Repository) MovePlanRef(values Object) (Object, error) {
	snap, err := r.LoadBindingRegistry(true)
	if err != nil {
		return nil, err
	}
	matches := []Object{}
	for _, raw := range List(snap.Payload["bindings"]) {
		item := Map(raw)
		if item["binding_id"] == values["binding_id"] {
			matches = append(matches, item)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("BINDING_NOT_FOUND")
	}
	target := matches[0]
	if target["planning_state"] != "CREATED" {
		return nil, fmt.Errorf("PLAN_NOT_CREATED")
	}
	for _, key := range []string{"policy_ref", "resolved_plan_id", "plan_file_id"} {
		if target[key] != values[key] {
			return nil, fmt.Errorf("BINDING_IDENTITY_MISMATCH: %s", key)
		}
	}
	if target["plan_ref"] != values["from_plan_ref"] {
		return nil, fmt.Errorf("STALE_PLAN_REF")
	}
	dest := Object{"plan_ref": values["to_plan_ref"], "resolved_plan_id": values["resolved_plan_id"], "plan_file_id": values["plan_file_id"]}
	if err = r.checkPlanIdentity(dest); err != nil {
		return nil, err
	}
	if values["from_plan_ref"] == values["to_plan_ref"] {
		return Object{"status": "CURRENT", "changed": false, "binding": target}, nil
	}
	target["plan_ref"] = values["to_plan_ref"]
	if _, err = r.ReplaceBindingRegistry(snap.Payload, snap.Digest, r.ValidateBindingRegistry); err != nil {
		return nil, err
	}
	return Object{"status": "MOVED", "changed": true, "binding": target}, nil
}
func (r *Repository) TrackPlanRef(id string, apply bool) (Object, error) {
	resolution, err := r.ResolveBindings(Object{"binding_id": id})
	if err != nil {
		return nil, err
	}
	rows := List(resolution["bindings"])
	if len(rows) != 1 {
		return nil, fmt.Errorf("BINDING_NOT_FOUND_OR_AMBIGUOUS")
	}
	row := Map(rows[0])
	if row["planning_state"] != "CREATED" {
		return nil, fmt.Errorf("PLAN_NOT_CREATED")
	}
	for _, key := range []string{"policy_ref", "resolved_plan_id", "plan_file_id", "plan_ref"} {
		if Text(row[key]) == "" {
			return nil, fmt.Errorf("BINDING_IDENTITY_INCOMPLETE: %s", key)
		}
	}
	candidates := []any{}
	identities := []Object{}
	planning, err := r.Path("developer/planning")
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(planning, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		scope, e := r.Scope(path)
		if e != nil {
			return e
		}
		payload, e := r.Read(scope)
		if e != nil {
			return nil
		}
		identity := Map(payload["plan_identity"])
		if identity["plan_file_id"] == row["plan_file_id"] && Text(identity["resolved_plan_id"]) != "" {
			candidates = append(candidates, scope)
			identities = append(identities, identity)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	out := Object{"status": "UNRESOLVED", "binding_id": id, "policy_ref": row["policy_ref"], "resolved_plan_id": row["resolved_plan_id"], "plan_file_id": row["plan_file_id"], "current_plan_ref": row["plan_ref"], "discovered_plan_ref": nil, "candidates": candidates, "changed": false, "applied": false}
	if len(candidates) == 0 {
		return out, nil
	}
	if len(candidates) > 1 {
		return nil, fmt.Errorf("PLAN_FILE_ID_AMBIGUOUS")
	}
	if identities[0]["resolved_plan_id"] != row["resolved_plan_id"] {
		return nil, fmt.Errorf("PLAN_FILE_ID_CONFLICT")
	}
	out["discovered_plan_ref"] = candidates[0]
	out["status"] = "CURRENT"
	if candidates[0] == row["plan_ref"] {
		return out, nil
	}
	out["status"] = "RECONCILE_REQUIRED"
	if apply {
		_, err = r.MovePlanRef(Object{"binding_id": id, "policy_ref": row["policy_ref"], "resolved_plan_id": row["resolved_plan_id"], "plan_file_id": row["plan_file_id"], "from_plan_ref": row["plan_ref"], "to_plan_ref": candidates[0]})
		if err != nil {
			return nil, err
		}
		out["status"] = "RECONCILED"
		out["changed"] = true
		out["applied"] = true
	}
	return out, nil
}
