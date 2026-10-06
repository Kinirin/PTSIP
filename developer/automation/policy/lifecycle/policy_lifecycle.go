package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const policyIndex = "developer/policy/index.yaml"
const policySubjectRegistry = "developer/policy/registries/authority-subject-registry.yaml"
const policyApprovalSchema = "developer/policy/schemas/policy-approval-provenance.schema.json"

var policyVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
var policyReadableFamilyID = regexp.MustCompile(`^MPD-(NORM|GOV|INTENT|ARCH|INFO|CNTR|RISK|SUPPLY|REAL|ASSURE|CTRL|CHANGE|OPS|RECORD|SPEC|PLAN|WORK|VERI|MIGR|RELS)-([0-9]{4})$`)
var policyBoundaryID = regexp.MustCompile(`^MPD-BOUND-[0-9]{4}$`)

func InitialPolicyVersion() string { return "0.0" }
func policyParseVersion(version string) (int, int, error) {
	match := policyVersionPattern.FindStringSubmatch(version)
	if match == nil {
		return 0, 0, policyFailure("INVALID_POLICY_VERSION", version)
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, policyFailure("INVALID_POLICY_VERSION", version)
	}
	minor, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, policyFailure("INVALID_POLICY_VERSION", version)
	}
	return major, minor, nil
}
func ResolvePolicyVersionTransition(version, status, changeClass, target string) (Object, error) {
	if !policyContains([]string{"NON_NORMATIVE", "DRAFT_NORMATIVE", "COMPATIBLE_NORMATIVE", "INCOMPATIBLE_NORMATIVE", "LIFECYCLE_TRANSITION"}, changeClass) {
		return nil, policyFailure("UNSUPPORTED_POLICY_VERSION_CHANGE_CLASS", changeClass)
	}
	major, minor, err := policyParseVersion(version)
	if err != nil {
		return nil, err
	}
	if target == "" {
		target = status
	}
	nextMajor, nextMinor := major, minor
	valid := true
	switch changeClass {
	case "NON_NORMATIVE":
		valid = target == status
	case "DRAFT_NORMATIVE":
		valid = status == "DRAFT" && target == "DRAFT" && major == 0
		nextMinor++
	case "LIFECYCLE_TRANSITION":
		if status == "DRAFT" && target == "APPROVED" && major == 0 {
			nextMajor = 1
		} else if status == "APPROVED" && target == "ACTIVE" && major == 1 {
			nextMajor = 2
		} else if status == "ACTIVE" && policyContains([]string{"SUPERSEDED", "RETIRED"}, target) && major >= 2 {
		} else {
			valid = false
		}
	case "COMPATIBLE_NORMATIVE":
		valid = status == "ACTIVE" && target == "ACTIVE" && major >= 2
		nextMinor++
	case "INCOMPATIBLE_NORMATIVE":
		valid = status == "ACTIVE" && target == "ACTIVE" && major >= 2
		nextMajor++
		nextMinor = 0
	}
	if !valid {
		return nil, policyFailure("UNSUPPORTED_POLICY_VERSION_TRANSITION", fmt.Sprintf("%s %s -> %s (%s)", status, version, target, changeClass))
	}
	return Object{"status": "READY", "change_class": changeClass, "current_status": status, "target_status": target, "current_version": version, "next_version": fmt.Sprintf("%d.%d", nextMajor, nextMinor)}, nil
}

func policyCanonicalPath(id string) (string, error) {
	if match := policyReadableFamilyID.FindStringSubmatch(id); match != nil {
		return "developer/policy/" + match[1] + "/" + id + ".yaml", nil
	}
	if policyBoundaryID.MatchString(id) {
		return "developer/policy/" + id + ".yaml", nil
	}
	return "", policyFailure("INVALID_POLICY_ID", "exact Root Family or registered VPMS identity required: "+id)
}

type PolicyCorpus struct {
	IDs          []string
	CanonicalIDs []string
	Index        Object
	Subject      Object
	Records      map[string]Object
}

func LoadConsistentPolicyCorpus(r Repository) (*PolicyCorpus, error) {
	index, err := r.LoadNeutralPolicyIndex()
	if err != nil {
		return nil, policyFailure("INVALID_POLICY_INDEX", err.Error())
	}
	subject, err := r.Read(policySubjectRegistry)
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/developer-policy-subject-catalog.schema.json", subject); err != nil {
		return nil, policyFailure("INVALID_SUBJECT_REGISTRY", err.Error())
	}
	state := &PolicyCorpus{IDs: []string{}, CanonicalIDs: []string{}, Index: index, Subject: subject, Records: map[string]Object{}}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		id := Text(entry["id"])
		state.IDs = append(state.IDs, id)
		if entry["authority_role"] == "MIGRATION_SOURCE" {
			continue
		}
		expected, err := policyCanonicalPath(id)
		if err != nil {
			return nil, err
		}
		if entry["path"] != expected {
			return nil, policyFailure("POLICY_INDEX_PATH_MISMATCH", id)
		}
		payload, err := r.Read(expected)
		if err != nil {
			return nil, policyFailure("POLICY_FILE_NOT_FOUND", expected)
		}
		identity := Map(payload["policy"])
		if identity["id"] != id {
			return nil, policyFailure("POLICY_FILE_ID_MISMATCH", id)
		}
		if identity["status"] != entry["status"] {
			return nil, policyFailure("POLICY_STATUS_MISMATCH", id)
		}
		if payload["policy_class"] != entry["policy_class"] {
			return nil, policyFailure("POLICY_CLASS_MISMATCH", id)
		}
		state.CanonicalIDs = append(state.CanonicalIDs, id)
		state.Records[id] = payload
	}
	ids := policyStrings(Map(Map(subject["subject_identity_schemes"])["MANAGEMENT_POLICY_ID"])["registered_values"])
	if !reflect.DeepEqual(ids, state.IDs) {
		return nil, policyFailure("SUBJECT_REGISTRY_MISMATCH", "subject identity registry must exactly project catalog membership")
	}
	return state, nil
}
func policyDiscoveredIDs(r Repository) ([]string, error) {
	root, err := r.Path("developer/policy")
	if err != nil {
		return nil, err
	}
	ids := []string{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "legacy" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(entry.Name(), "MPD-") && strings.HasSuffix(entry.Name(), ".yaml") {
			ids = append(ids, strings.TrimSuffix(entry.Name(), ".yaml"))
		}
		return nil
	})
	sort.Strings(ids)
	return ids, err
}
func policyCheckDiscovery(r Repository, state *PolicyCorpus, extra string) error {
	discovered, err := policyDiscoveredIDs(r)
	if err != nil {
		return err
	}
	expected := append([]string{}, state.CanonicalIDs...)
	if extra != "" {
		expected = append(expected, extra)
	}
	sort.Strings(expected)
	if !reflect.DeepEqual(discovered, expected) {
		return policyFailure("POLICY_CORPUS_MISMATCH", "canonical files, index and subject registry must agree exactly before allocation")
	}
	return nil
}
func policyApproval(r Repository, reference string) (Object, error) {
	relative, err := r.Scope(reference)
	if err != nil {
		return nil, policyFailure("APPROVAL_PROVENANCE_OUTSIDE_REPOSITORY", err.Error())
	}
	if !strings.HasPrefix(relative, "developer/policy/approvals/") {
		return nil, policyFailure("APPROVAL_PROVENANCE_OUTSIDE_CANONICAL_ROOT", relative)
	}
	payload, err := r.Read(relative)
	if err != nil {
		return nil, policyFailure("APPROVAL_PROVENANCE_NOT_FOUND", relative)
	}
	if err := r.Validate(policyApprovalSchema, payload); err != nil {
		return nil, policyFailure("INVALID_APPROVAL_PROVENANCE", err.Error())
	}
	approval := Map(payload["approval"])
	if approval["decision"] != "APPROVED" {
		return nil, policyFailure("APPROVAL_NOT_GRANTED", "approval.decision must be APPROVED")
	}
	return approval, nil
}
func policyRequireFamilyClass(r Repository, class, family string) error {
	contract, err := r.Read("developer/policy/registries/developer-policy-catalog-contracts.json")
	if err != nil {
		return err
	}
	if !policyContains(policyStrings(Map(Map(contract["$defs"])["developer_policy_class"])["enum"]), class) {
		return policyFailure("UNKNOWN_POLICY_CLASS", class)
	}
	if class == DeveloperClass && !policyContains(policyRootFamilies, family) {
		return policyFailure("LEGACY_FAMILY_NEW_ALLOCATION_FORBIDDEN", family)
	}
	if class == "VPMS_DEVELOPER_POLICY" {
		if !policyContains(policyLegacyFamilies, family) {
			return policyFailure("VPMS_ROOT_FAMILY_NOT_AUTHORIZED", family)
		}
		execution, gate := Map(contract["application_execution"]), Map(contract["application_gate"])
		if gate["vpms_policy_materialization_authorized"] != true || execution["vpms_class_materialization_enabled"] != true || execution["m1_m7_verified"] != true {
			return policyFailure("VPMS_CLASS_MATERIALIZATION_NOT_ENABLED", "M1-M7 verification and explicit owner approval required")
		}
	}
	return nil
}
func policyNextFamilyID(ids []string, family string) (string, error) {
	maximum := 0
	for _, id := range ids {
		match := policyReadableFamilyID.FindStringSubmatch(id)
		if match != nil && match[1] == family {
			number, _ := strconv.Atoi(match[2])
			if number > maximum {
				maximum = number
			}
		}
	}
	if maximum >= 9999 {
		return "", policyFailure("POLICY_ID_SPACE_EXHAUSTED", family)
	}
	return fmt.Sprintf("MPD-%s-%04d", family, maximum+1), nil
}
func policyAnalysisGroup(r Repository, class, family, reference, groupID string) (Object, Object, error) {
	result, err := ValidateResponsibilityAnalysis(r, reference, true)
	if err != nil {
		return nil, nil, err
	}
	if result["materialization_allowed"] != true {
		return nil, nil, policyFailure("RESPONSIBILITY_MATERIALIZATION_BLOCKED", "unresolved blocking collision")
	}
	var selected Object
	matches := 0
	for _, raw := range List(result["materialization_groups"]) {
		group := Map(raw)
		if group["group_id"] == groupID {
			selected = group
			matches++
		}
	}
	if matches != 1 {
		return nil, nil, policyFailure("MATERIALIZATION_GROUP_NOT_FOUND", groupID)
	}
	if selected["family"] != family {
		return nil, nil, policyFailure("MATERIALIZATION_GROUP_FAMILY_MISMATCH", groupID)
	}
	if selected["policy_class"] != class {
		return nil, nil, policyFailure("MATERIALIZATION_GROUP_CLASS_MISMATCH", groupID)
	}
	return result, selected, nil
}
func PreflightFamilyPolicy(r Repository, class, family, approvalRef, analysisID, groupID string) (Object, error) {
	if err := policyRequireFamilyClass(r, class, family); err != nil {
		return nil, err
	}
	state, err := LoadConsistentPolicyCorpus(r)
	if err != nil {
		return nil, err
	}
	if err := policyCheckDiscovery(r, state, ""); err != nil {
		return nil, err
	}
	approval, err := policyApproval(r, approvalRef)
	if err != nil {
		return nil, err
	}
	if approval["target_status"] != "DRAFT" {
		return nil, policyFailure("NEW_POLICY_MUST_START_DRAFT", "new policy must start at DRAFT 0.0")
	}
	analysis, group, err := policyAnalysisGroup(r, class, family, analysisID, groupID)
	if err != nil {
		return nil, err
	}
	id, err := policyNextFamilyID(state.IDs, family)
	if err != nil {
		return nil, err
	}
	if approval["requested_policy_id"] != nil && approval["requested_policy_id"] != id {
		return nil, policyFailure("REQUESTED_POLICY_ID_NOT_NEXT_AVAILABLE", id)
	}
	return Object{"status": "READY", "allocated_policy_id": id, "family": family, "policy_class": class, "group_id": groupID, "analysis_id": analysis["analysis_id"], "analysis_ref": analysis["analysis_ref"], "split_required": analysis["split_required"], "target_status": approval["target_status"], "approval_id": approval["approval_id"], "approval_scope": approval["approval_scope"], "implementation_authorized": approval["implementation_authorized"], "policy_content_review_scope": approval["policy_content_review_scope"], "cohesion_key": group["cohesion_key"], "registry_mutation_required": true, "analysis_registry_mutation_required": true}, nil
}
func InspectPolicy(r Repository, id string) (Object, error) {
	state, err := LoadConsistentPolicyCorpus(r)
	if err != nil {
		return nil, err
	}
	if err := policyCheckDiscovery(r, state, ""); err != nil {
		return nil, err
	}
	payload, found := state.Records[id]
	if !found {
		return Object{"status": "NOT_FOUND", "policy_id": id}, nil
	}
	var entry Object
	for _, raw := range List(state.Index["policies"]) {
		if Map(raw)["id"] == id {
			entry = Map(raw)
		}
	}
	identity := Map(payload["policy"])
	return Object{"status": "FOUND", "policy_id": id, "title": identity["title"], "policy_status": identity["status"], "policy_version": identity["version"], "index_status": entry["status"], "subject_identity_registered": true, "operationally_resolvable": identity["status"] == "ACTIVE", "declared_runtime_authority": Map(Map(payload["rules"])["authority_semantics"])["runtime_authority"], "transition": payload["transition"], "path": entry["path"]}, nil
}
func StatusPreflight(r Repository, id, approvalRef string) (Object, error) {
	approval, err := policyApproval(r, approvalRef)
	if err != nil {
		return nil, err
	}
	if approval["requested_policy_id"] != id {
		return nil, policyFailure("APPROVAL_POLICY_ID_MISMATCH", id)
	}
	inspected, err := InspectPolicy(r, id)
	if err != nil {
		return nil, err
	}
	if inspected["status"] != "FOUND" {
		return nil, policyFailure("POLICY_NOT_FOUND", id)
	}
	return Object{"status": "READY", "policy_id": id, "current_status": inspected["policy_status"], "target_status": approval["target_status"], "approval_id": approval["approval_id"], "implementation_authorized": approval["implementation_authorized"]}, nil
}

// The unqualified entry point now delegates through the exact requested Root
// identity and the same responsibility gate as explicit Family registration.
func PreflightNewPolicy(r Repository, approvalRef, analysisID, groupID string) (Object, error) {
	approval, err := policyApproval(r, approvalRef)
	if err != nil {
		return nil, err
	}
	id := Text(approval["requested_policy_id"])
	match := rootID.FindStringSubmatch(id)
	if match == nil {
		return nil, policyFailure("ROOT_FAMILY_ID_REQUIRED", "new Developer authority requires an exact requested Root Family identity")
	}
	return PreflightFamilyPolicy(r, DeveloperClass, match[1], approvalRef, analysisID, groupID)
}
func RegisterPolicy(r Repository, approvalRef, analysisID, groupID, policyFile string) (Object, error) {
	approval, err := policyApproval(r, approvalRef)
	if err != nil {
		return nil, err
	}
	id := Text(approval["requested_policy_id"])
	match := rootID.FindStringSubmatch(id)
	if match == nil {
		return nil, policyFailure("ROOT_FAMILY_ID_REQUIRED", "new Developer authority requires an exact requested Root Family identity")
	}
	return RegisterFamilyPolicy(r, DeveloperClass, match[1], approvalRef, analysisID, groupID, policyFile)
}

// policyWriteTransaction validates each target with repository path and CAS checks,
// and restores exact previous bytes on an interrupted multi-record write.
func policyWriteTransaction(r Repository, updates map[string]Object) error {
	paths := []string{}
	original := map[string][]byte{}
	digests := map[string]string{}
	for path := range updates {
		absolute, err := r.Path(path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(absolute)
		if err != nil {
			return err
		}
		original[path] = data
		digests[path] = SHA256(data)
		paths = append(paths, path)
	}
	sort.Strings(paths)
	written := []string{}
	for _, path := range paths {
		digest := digests[path]
		if err := r.WriteYAML(path, updates[path], &digest); err != nil {
			rollbackErrors := []string{}
			for i := len(written) - 1; i >= 0; i-- {
				if rollbackErr := r.AtomicWrite(written[i], original[written[i]], nil); rollbackErr != nil {
					rollbackErrors = append(rollbackErrors, rollbackErr.Error())
				}
			}
			if len(rollbackErrors) > 0 {
				return fmt.Errorf("%w; rollback: %s", err, strings.Join(rollbackErrors, "; "))
			}
			return err
		}
		written = append(written, path)
	}
	return nil
}
func RegisterFamilyPolicy(r Repository, class, family, approvalRef, analysisID, groupID, policyFile string) (Object, error) {
	if err := policyRequireFamilyClass(r, class, family); err != nil {
		return nil, err
	}
	state, err := LoadConsistentPolicyCorpus(r)
	if err != nil {
		return nil, err
	}
	approval, err := policyApproval(r, approvalRef)
	if err != nil {
		return nil, err
	}
	if approval["target_status"] != "DRAFT" {
		return nil, policyFailure("NEW_POLICY_MUST_START_DRAFT", "DRAFT 0.0 required")
	}
	analysis, _, err := policyAnalysisGroup(r, class, family, analysisID, groupID)
	if err != nil {
		return nil, err
	}
	id, err := policyNextFamilyID(state.IDs, family)
	if err != nil {
		return nil, err
	}
	if approval["requested_policy_id"] != id {
		return nil, policyFailure("REQUESTED_POLICY_ID_NOT_NEXT_AVAILABLE", id)
	}
	expected, err := policyCanonicalPath(id)
	if err != nil {
		return nil, err
	}
	relative, err := r.Scope(policyFile)
	if err != nil {
		return nil, policyFailure("POLICY_FILE_OUTSIDE_REPOSITORY", err.Error())
	}
	if relative != expected {
		return nil, policyFailure("POLICY_FILE_PATH_MISMATCH", expected)
	}
	if err := policyCheckDiscovery(r, state, id); err != nil {
		return nil, err
	}
	payload, err := r.Read(expected)
	if err != nil {
		return nil, policyFailure("POLICY_FILE_NOT_FOUND", expected)
	}
	schema := "developer/policy/schemas/management-policy.schema.json"
	if class == DeveloperClass {
		schema = "developer/policy/schemas/root-family-policy.schema.json"
	}
	if err := r.Validate(schema, payload); err != nil {
		return nil, policyFailure("INVALID_POLICY_FILE", err.Error())
	}
	identity := Map(payload["policy"])
	if payload["policy_class"] != class {
		return nil, policyFailure("POLICY_CLASS_MISMATCH", id)
	}
	if class == DeveloperClass && payload["responsibility_family"] != family {
		return nil, policyFailure("RESPONSIBILITY_FAMILY_MISMATCH", family)
	}
	if identity["id"] != id {
		return nil, policyFailure("POLICY_FILE_ID_MISMATCH", id)
	}
	if identity["status"] != approval["target_status"] {
		return nil, policyFailure("APPROVED_STATUS_MISMATCH", id)
	}
	if identity["version"] != InitialPolicyVersion() {
		return nil, policyFailure("NEW_POLICY_VERSION_MISMATCH", id)
	}
	registry, err := r.Read(policyAnalysisRegistry)
	if err != nil {
		return nil, err
	}
	if err := r.Validate(policyAnalysisRegistrySchema, registry); err != nil {
		return nil, policyFailure("INVALID_ANALYSIS_REGISTRY", err.Error())
	}
	for _, raw := range List(registry["bindings"]) {
		if Map(raw)["policy_id"] == id {
			return nil, policyFailure("DUPLICATE_ANALYSIS_BINDING", id)
		}
	}
	index := policyClone(state.Index)
	entries := append(List(index["policies"]), Object{"id": id, "policy_class": class, "path": expected, "status": approval["target_status"]})
	sort.Slice(entries, func(i, j int) bool { return Text(Map(entries[i])["id"]) < Text(Map(entries[j])["id"]) })
	index["policies"] = entries
	subject := policyClone(state.Subject)
	values := append(policyStrings(Map(Map(subject["subject_identity_schemes"])["MANAGEMENT_POLICY_ID"])["registered_values"]), id)
	sort.Strings(values)
	anyValues := []any{}
	for _, value := range values {
		anyValues = append(anyValues, value)
	}
	Map(Map(subject["subject_identity_schemes"])["MANAGEMENT_POLICY_ID"])["registered_values"] = anyValues
	bindings := append(List(registry["bindings"]), Object{"policy_id": id, "policy_class": class, "family": family, "analysis_id": analysis["analysis_id"], "group_id": groupID})
	sort.Slice(bindings, func(i, j int) bool { return Text(Map(bindings[i])["policy_id"]) < Text(Map(bindings[j])["policy_id"]) })
	registry["bindings"] = bindings
	for path, value := range map[string]Object{"developer/policy/schemas/developer-policy-catalog.schema.json": index, "developer/policy/schemas/developer-policy-subject-catalog.schema.json": subject, policyAnalysisRegistrySchema: registry} {
		if err := r.Validate(path, value); err != nil {
			return nil, err
		}
	}
	if err := policyWriteTransaction(r, map[string]Object{policyIndex: index, policySubjectRegistry: subject, policyAnalysisRegistry: registry}); err != nil {
		return nil, err
	}
	return Object{"status": "REGISTERED", "policy_id": id, "family": family, "policy_class": class, "group_id": groupID, "analysis_id": analysis["analysis_id"], "policy_status": approval["target_status"], "approval_id": approval["approval_id"], "index": policyIndex, "subject_registry": policySubjectRegistry, "analysis_registry": policyAnalysisRegistry}, nil
}
