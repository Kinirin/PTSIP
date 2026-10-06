package planning

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func extensionStatusRecords(payload Object) []Object {
	records := []Object{}
	for _, raw := range Map(payload["migration_stages"]) {
		row := Map(raw)
		if Text(row["status"]) != "" {
			records = append(records, row)
		}
	}
	for key, raw := range payload {
		if !strings.HasSuffix(key, "_execution_plan") {
			continue
		}
		for _, value := range List(Map(raw)["execution_order"]) {
			row := Map(value)
			if Text(row["status"]) != "" {
				records = append(records, row)
			}
		}
	}
	return records
}
func extensionValidationComplete(record Object) bool {
	if record["validation"] == nil {
		return true
	}
	validation := Map(record["validation"])
	if validation == nil {
		return false
	}
	saw := false
	for _, key := range []string{"status", "result", "state"} {
		value := validation[key]
		if value == nil {
			continue
		}
		saw = true
		if value != "PASS" && value != "AUTHORIZED" && value != "COMPLETE" {
			return false
		}
	}
	return saw
}
func ExtensionMachineReady(payload Object) bool {
	if payload["schema_version"] != "ptsip-plan-extension/v1" {
		return false
	}
	extension := Map(payload["extension"])
	authorization := Map(payload["implementation_authorization"])
	blockers := List(payload["current_known_blockers"])
	status := planningStatus(extension)
	if extension == nil || authorization == nil || (status != "ACTIVE" && status != "COMPLETE") || (authorization["status"] != "AUTHORIZED" && authorization["status"] != "COMPLETE") || blockers == nil || len(blockers) > 0 {
		return false
	}
	records := extensionStatusRecords(payload)
	if len(records) == 0 {
		return false
	}
	for _, row := range records {
		if row["status"] != "COMPLETE" || !extensionValidationComplete(row) {
			return false
		}
	}
	return true
}
func ExtensionParentConsistency(parent, extension Object, id, ref string) []string {
	errors := []string{}
	metadata := Map(extension["extension"])
	if metadata == nil {
		return []string{ref + ": extension metadata missing"}
	}
	if metadata["id"] != id {
		errors = append(errors, ref+": extension.id mismatch")
	}
	if metadata["parent"] != Map(parent["work_unit"])["id"] {
		errors = append(errors, ref+": extension.parent mismatch")
	}
	matches := []Object{}
	for _, raw := range List(parent["extensions"]) {
		row := Map(raw)
		if row["id"] == id {
			matches = append(matches, row)
		}
	}
	if len(matches) != 1 {
		return append(errors, ref+": parent must declare extension exactly once")
	}
	if matches[0]["path"] != ref {
		errors = append(errors, ref+": parent extension path mismatch")
	}
	if matches[0]["status"] != planningStatus(metadata) {
		errors = append(errors, ref+": parent extension status mismatch")
	}
	return errors
}
func extensionContext(r Repository, payload Object, ref string) (string, string, string, error) {
	extension := Map(payload["extension"])
	id, parent, version := Text(extension["id"]), Text(extension["parent"]), Text(payload["plan_version"])
	if id == "" || parent == "" || version == "" {
		return "", "", "", fmt.Errorf("EXTENSION_IDENTITY_INVALID")
	}
	root, err := r.Read(PlanningRootIndex)
	if err != nil {
		return "", "", "", err
	}
	matches := []Object{}
	for _, raw := range List(root["plans"]) {
		row := Map(raw)
		if row["plan_version"] == version {
			matches = append(matches, row)
		}
	}
	if len(matches) != 1 {
		return "", "", "", fmt.Errorf("PLAN_VERSION_UNRESOLVED")
	}
	entry := matches[0]
	versionRef, branch := Text(entry["path"]), Text(entry["integration_branch"])
	index, err := r.Read(versionRef)
	if err != nil {
		return "", "", "", err
	}
	rows, err := planningIndexed(index)
	if err != nil {
		return "", "", "", err
	}
	parentRef := Text(rows[parent]["path"])
	parentPayload, err := r.Read(parentRef)
	if err != nil {
		return "", "", "", err
	}
	failures := ExtensionParentConsistency(parentPayload, payload, id, ref)
	if len(failures) > 0 {
		recoverable := false
		for _, raw := range List(parentPayload["extensions"]) {
			row := Map(raw)
			if row["id"] == id && row["path"] == ref && row["status"] == "ACTIVE" && planningStatus(extension) == "COMPLETE" {
				recoverable = true
			}
		}
		if !recoverable {
			return "", "", "", fmt.Errorf("%s", strings.Join(failures, "; "))
		}
	}
	return parentRef, versionRef, branch, nil
}
func FinalizeExtension(r Repository, ref string) (Object, error) {
	payload, err := r.Read(ref)
	if err != nil {
		return nil, err
	}
	extension := Map(payload["extension"])
	id := Text(extension["id"])
	out := Object{"extension_id": ppOptional(id), "eligible": false, "finalized": false, "current_gate_before": nil, "current_gate_after": nil, "failures": []string{}}
	if payload["schema_version"] != "ptsip-plan-extension/v1" {
		return out, nil
	}
	if id == "" {
		out["failures"] = []string{"extension.id missing"}
		return out, nil
	}
	if !ExtensionMachineReady(payload) {
		return out, nil
	}
	out["eligible"] = true
	parentRef, versionRef, branch, err := extensionContext(r, payload, ref)
	if err != nil {
		out["failures"] = []string{err.Error()}
		return out, nil
	}
	parent, err := r.Read(parentRef)
	if err != nil {
		return nil, err
	}
	originals := map[string][]byte{}
	for _, file := range []string{ref, parentRef, PlanningRootIndex, versionRef} {
		path, e := r.Path(file)
		if e != nil {
			return nil, e
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			return nil, e
		}
		originals[file] = raw
	}
	restore := func() {
		for file, raw := range originals {
			path, _ := r.Path(file)
			planningAtomicWrite(path, raw)
		}
	}
	changed := planningStatus(extension) == "ACTIVE" || Map(payload["implementation_authorization"])["status"] == "AUTHORIZED"
	extensionText, err := planningReplaceExtensionStatuses(string(originals[ref]), payload)
	if err != nil {
		out["failures"] = []string{err.Error()}
		return out, nil
	}
	parentText := strings.ReplaceAll(string(originals[parentRef]), "\r\n", "\n")
	for _, raw := range List(parent["extensions"]) {
		row := Map(raw)
		if row["id"] == id && row["status"] == "ACTIVE" {
			parentText, err = planningReplaceStageStatus(parentText, id, "ACTIVE", "COMPLETE")
			if err != nil {
				out["failures"] = []string{err.Error()}
				return out, nil
			}
		}
	}
	Map(extension["lifecycle"])["status"] = "COMPLETE"
	Map(payload["implementation_authorization"])["status"] = "COMPLETE"
	for _, raw := range List(parent["extensions"]) {
		row := Map(raw)
		if row["id"] == id {
			changed = changed || row["status"] != "COMPLETE"
			row["status"] = "COMPLETE"
		}
	}
	for file, content := range map[string]string{ref: extensionText, parentRef: parentText} {
		digest := SHA256(originals[file])
		if err = r.AtomicWrite(file, []byte(content), &digest); err != nil {
			restore()
			out["failures"] = []string{err.Error()}
			return out, nil
		}
	}
	if actual, e := r.Read(ref); e != nil || !planningEqual(actual, payload) {
		restore()
		out["failures"] = []string{"extension text mutation does not match selected machine transition"}
		return out, nil
	}
	if actual, e := r.Read(parentRef); e != nil || !planningEqual(actual, parent) {
		restore()
		out["failures"] = []string{"parent text mutation does not match selected machine transition"}
		return out, nil
	}
	if err != nil {
		out["failures"] = []string{err.Error()}
		return out, nil
	}
	reconciliation, err := ReconcilePlanning(r, branch, "", true)
	if err == nil && reconciliation["current_gate_before"] == id && reconciliation["current_gate_after"] != extension["parent"] {
		err = fmt.Errorf("EXTENSION_GATE_NOT_RELEASED")
	}
	if err == nil {
		if failures := ValidatePlanning(r); len(failures) > 0 {
			err = fmt.Errorf("%s", strings.Join(failures, "; "))
		}
	}
	if err != nil {
		restore()
		out["failures"] = []string{err.Error()}
		return out, nil
	}
	out["finalized"] = changed || reconciliation["changed"] == true
	out["current_gate_before"] = reconciliation["current_gate_before"]
	out["current_gate_after"] = reconciliation["current_gate_after"]
	return out, nil
}
func planningFindStage(payload any, id string) (Object, error) {
	matches := []Object{}
	var visit func(any)
	visit = func(value any) {
		if object := Map(value); object != nil {
			if object["id"] == id && object["status"] != nil {
				matches = append(matches, object)
			}
			for _, child := range object {
				visit(child)
			}
		} else {
			for _, child := range List(value) {
				visit(child)
			}
		}
	}
	visit(payload)
	if len(matches) != 1 {
		return nil, fmt.Errorf("STAGE_UNRESOLVED: %s occurs %d times", id, len(matches))
	}
	return matches[0], nil
}
func planningPromotePayload(payload Object, id string, automatic Object) error {
	stage, err := planningFindStage(payload, id)
	if err != nil {
		return err
	}
	if stage["status"] != "IMPLEMENTED_VALIDATION_PENDING" {
		return fmt.Errorf("STAGE_STATUS_MISMATCH")
	}
	stage["status"] = "COMPLETE"
	if validation := Map(stage["validation"]); validation["status"] == "PENDING" {
		validation["status"] = "PASS"
	}
	if next := Map(automatic["next_stage"]); next != nil {
		nextStage, err := planningFindStage(payload, Text(next["id"]))
		if err != nil {
			return err
		}
		if nextStage["status"] != next["from_status"] {
			return fmt.Errorf("NEXT_STAGE_STATUS_MISMATCH")
		}
		nextStage["status"] = next["to_status"]
	}
	updates := Map(automatic["document_updates"])
	for _, raw := range List(updates["mapping_scalars"]) {
		row := Map(raw)
		target := Map(Map(payload[Text(row["section"])])[Text(row["key"])])
		field := Text(row["field"])
		if target == nil || target[field] != row["from_value"] {
			return fmt.Errorf("DOCUMENT_UPDATE_SCALAR_MISMATCH")
		}
		target[field] = row["to_value"]
	}
	for _, raw := range List(updates["list_removals"]) {
		row := Map(raw)
		section := Text(row["section"])
		list := List(payload[section])
		count := 0
		out := []any{}
		for _, item := range list {
			if item == row["value"] {
				count++
			} else {
				out = append(out, item)
			}
		}
		if count != 1 {
			return fmt.Errorf("DOCUMENT_UPDATE_LIST_ITEM_MISMATCH")
		}
		payload[section] = out
	}
	return nil
}
func RunPlanningRegression(r Repository, targets []any, goTargets []any) []string {
	failures := []string{}
	if len(targets) > 0 {
		python := "python"
		candidate := filepath.Join(r.RootDir(), ".venv", "Scripts", "python.exe")
		if _, err := os.Stat(candidate); err == nil {
			python = candidate
		}
		args := []string{"-m", "pytest"}
		for _, raw := range targets {
			target := Text(raw)
			if target == "" {
				return []string{"pytest_targets must be nonempty strings"}
			}
			file := strings.SplitN(target, "::", 2)[0]
			if _, err := r.Path(file); err != nil {
				return []string{err.Error()}
			}
			args = append(args, target)
		}
		args = append(args, "-q")
		command := exec.Command(python, args...)
		command.Dir = r.RootDir()
		output, err := command.CombinedOutput()
		if err != nil {
			failures = append(failures, "pytest validation failed: "+string(output))
		}
	}
	for _, raw := range goTargets {
		target := Text(raw)
		scope, err := r.Scope(target)
		if err != nil || !strings.HasPrefix(scope, "developer/tests/") {
			failures = append(failures, "Go validation target must be registered developer test module")
			continue
		}
		command := exec.Command("go", "-C", target, "test", "-count=1", "./...")
		command.Dir = r.RootDir()
		output, err := command.CombinedOutput()
		if err != nil {
			failures = append(failures, "Go validation failed: "+string(output))
		}
	}
	return failures
}
func planningRegisteredCheck(r Repository, check string) []string {
	switch check {
	case "PLANNING_VALIDATION":
		return ValidatePlanning(r)
	case "POLICY_VALIDATION":
		result, err := r.DispatchOperation("policy-validator", "validate", map[string]string{}, nil)
		if err != nil {
			return []string{err.Error()}
		}
		payload := Map(result)
		if payload["status"] != "PASS" {
			return []string{fmt.Sprint(result)}
		}
		return nil
	case "CURRENT_LEGACY_DEPENDENCY_ZERO":
		result, err := r.DispatchOperation("current-dependency-gate", "validate", map[string]string{}, nil)
		if err != nil {
			return []string{err.Error()}
		}
		payload := Map(result)
		if payload["status"] != "PASS" {
			return []string{fmt.Sprint(result)}
		}
		return nil
	default:
		return []string{"unknown automatic completion check: " + check}
	}
}
func FinalizePlanningStage(r Repository, ref, id string) (Object, error) {
	payload, err := r.Read(ref)
	if err != nil {
		return nil, err
	}
	out := Object{"stage_id": id, "promoted": false, "failures": []string{}, "extension_finalized": false}
	stage, err := planningFindStage(payload, id)
	if err != nil {
		out["failures"] = []string{err.Error()}
		return out, nil
	}
	if stage["status"] == "COMPLETE" {
		extension, err := FinalizeExtension(r, ref)
		if err != nil {
			return nil, err
		}
		out["extension_finalized"] = extension["finalized"]
		out["failures"] = extension["failures"]
		return out, nil
	}
	if stage["status"] != "IMPLEMENTED_VALIDATION_PENDING" {
		out["failures"] = []string{"stage status must be IMPLEMENTED_VALIDATION_PENDING"}
		return out, nil
	}
	automatic := Map(stage["automatic_completion"])
	if automatic["from_status"] != "IMPLEMENTED_VALIDATION_PENDING" || automatic["to_status"] != "COMPLETE" {
		out["failures"] = []string{"automatic completion transition contract invalid"}
		return out, nil
	}
	failures := RunPlanningRegression(r, List(automatic["pytest_targets"]), List(automatic["go_targets"]))
	checks := List(automatic["required_checks"])
	if len(checks) == 0 {
		failures = append(failures, "automatic completion required_checks must not be empty")
	}
	for _, raw := range checks {
		failures = append(failures, planningRegisteredCheck(r, Text(raw))...)
	}
	if len(failures) > 0 {
		out["failures"] = failures
		return out, nil
	}
	path, err := r.Path(ref)
	if err != nil {
		return nil, err
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err = planningPromotePayload(payload, id, automatic); err != nil {
		out["failures"] = []string{err.Error()}
		return out, nil
	}
	promotedText, err := PlanningPromoteStageText(string(original), id, automatic)
	if err != nil {
		out["failures"] = []string{err.Error()}
		return out, nil
	}
	rendered := []byte(promotedText)
	check, err := ppYAML(rendered, ref, true)
	if err != nil || !planningEqual(check, payload) {
		out["failures"] = []string{"stage text mutation does not match selected machine transition"}
		return out, nil
	}
	digest := SHA256(original)
	if err = r.AtomicWrite(ref, rendered, &digest); err != nil {
		return nil, err
	}
	extension, err := FinalizeExtension(r, ref)
	if err != nil {
		planningAtomicWrite(path, original)
		return nil, err
	}
	if len(Strings(extension["failures"])) > 0 {
		planningAtomicWrite(path, original)
		out["failures"] = extension["failures"]
		return out, nil
	}
	if extension["finalized"] != true {
		if failures = ValidatePlanning(r); len(failures) > 0 {
			planningAtomicWrite(path, original)
			out["failures"] = failures
			return out, nil
		}
	}
	out["promoted"] = true
	out["extension_finalized"] = extension["finalized"]
	return out, nil
}
