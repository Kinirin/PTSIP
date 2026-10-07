package machine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const supportScopeRecord = "developer/policy/registries/vpms-contract-materialization.json"
const activationScopeRecord = "developer/policy/registries/vpms-runtime-activation.json"
const implementationScopeRecord = "developer/policy/registries/vpms-api-implementation.json"

func (r *Repository) ValidateResourceSet(schemaPath string, value any, resources []string) error {
	compiler := jsonschema.NewCompiler()
	compiler.UseRegexpEngine(compileSchemaRegexp)
	compiler.UseLoader(closedLoader{})
	for _, path := range UniqueStrings(append(resources, schemaPath)) {
		document, err := r.Read(path)
		if err != nil {
			return err
		}
		id := Text(document["$id"])
		if id == "" {
			return Fail("SCHEMA_ID_REQUIRED", path)
		}
		if err := compiler.AddResource(id, document); err != nil {
			return err
		}
	}
	schema, err := r.Read(schemaPath)
	if err != nil {
		return err
	}
	compiled, err := compiler.Compile(Text(schema["$id"]))
	if err != nil {
		return err
	}
	return compiled.Validate(value)
}
func (r *Repository) ScopeRecord() (Object, error) {
	record, err := r.Read(supportScopeRecord)
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/vpms-contract-materialization.schema.json", record); err != nil {
		return nil, err
	}
	return record, nil
}
func (r *Repository) ActivationRecord() (Object, error) {
	path, err := r.Path(activationScopeRecord)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	record, err := r.Read(activationScopeRecord)
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/vpms-runtime-activation.schema.json", record); err != nil {
		return nil, err
	}
	for relative, expected := range Map(record["prior_records"]) {
		content, err := r.ReadSource(relative)
		if err != nil {
			return nil, err
		}
		if LFDigest(content) != Text(expected) {
			return nil, Fail("ACTIVATION_PRIOR_PROVENANCE_CHANGED", relative)
		}
	}
	return record, nil
}
func (r *Repository) ProductContract(id string, requireActive bool) (Object, error) {
	identity, err := r.Read("src/vpms/contracts/identity-registry.json")
	if err != nil {
		return nil, err
	}
	resources := []string{"src/vpms/contracts/identity-registry.json"}
	for _, path := range Strings(identity["schema_resources"]) {
		if strings.Contains(path, "..") || strings.Contains(path, "\\") || filepath.IsAbs(path) {
			return nil, Fail("UNSAFE_REGISTERED_PATH", path)
		}
		resources = append(resources, "src/vpms/contracts/"+path)
	}
	catalog, err := r.Read("src/vpms/contracts/index.json")
	if err != nil {
		return nil, err
	}
	if err := r.ValidateResourceSet("src/vpms/contracts/schemas/catalog.schema.json", catalog, resources); err != nil {
		return nil, err
	}
	entry := Map(Map(catalog["contracts"])[id])
	if entry == nil {
		return nil, Fail("UNKNOWN_CONTRACT_ID", id)
	}
	relative := Text(entry["path"])
	if strings.Contains(relative, "..") || strings.Contains(relative, "\\") || filepath.IsAbs(relative) {
		return nil, Fail("UNSAFE_REGISTERED_PATH", relative)
	}
	payload, err := r.Read("src/vpms/contracts/" + relative)
	if err != nil {
		return nil, err
	}
	if err := r.ValidateResourceSet("src/vpms/contracts/schemas/product-contract.schema.json", payload, resources); err != nil {
		return nil, err
	}
	for _, field := range []string{"id", "status", "responsibility", "runtime_enabled"} {
		expected := entry[field]
		if field == "id" {
			expected = id
		}
		if payload[field] != expected {
			return nil, Fail("CONTRACT_INDEX_MISMATCH", field)
		}
	}
	if payload["contract_class"] != identity["contract_class"] {
		return nil, Fail("CONTRACT_CLASS_MISMATCH", id)
	}
	if requireActive && (catalog["capability"] != "ACTIVE" || payload["status"] != "ACTIVE" || payload["runtime_enabled"] != true) {
		return nil, Fail("CONTRACT_NOT_ACTIVE", id)
	}
	return payload, nil
}
func (r *Repository) ValidateSelectionDocument(kind string, payload Object) (Object, error) {
	file := map[string]string{"request": "selection-request", "result": "selection-result", "rule": "selection-rule"}[kind]
	if file == "" {
		return nil, Fail("UNKNOWN_SELECTION_DOCUMENT", kind)
	}
	identity, err := r.Read("src/vpms/contracts/identity-registry.json")
	if err != nil {
		return nil, err
	}
	resources := []string{"src/vpms/contracts/identity-registry.json"}
	for _, path := range Strings(identity["schema_resources"]) {
		resources = append(resources, "src/vpms/contracts/"+path)
	}
	if err := r.ValidateResourceSet("src/vpms/contracts/schemas/"+file+".schema.json", payload, resources); err != nil {
		return nil, err
	}
	if kind == "result" {
		catalog, err := r.Read("src/vpms/contracts/index.json")
		if err != nil {
			return nil, err
		}
		selection, err := r.ProductContract(Text(Map(catalog["entrypoints"])["selection"]), false)
		if err != nil {
			return nil, err
		}
		semantics := Map(selection["semantics"])
		if semantics["ordering"] != "CASE_ID_ASCENDING" || !reflect.DeepEqual(Strings(semantics["diagnostic_order"]), []string{"location", "code", "reference"}) {
			return nil, Fail("UNSUPPORTED_SELECTION_ORDER", "selection contract ordering is not registered")
		}
		cases := Strings(payload["case_ids"])
		ordered := append([]string{}, cases...)
		sort.Strings(ordered)
		if !reflect.DeepEqual(cases, ordered) {
			return nil, Fail("NONDETERMINISTIC_SELECTION_ORDER", "case IDs must be ascending")
		}
		previous := ""
		for _, raw := range List(payload["diagnostics"]) {
			item := Map(raw)
			key := Text(item["location"]) + "\x00" + Text(item["code"]) + "\x00" + Text(item["reference"])
			if key < previous {
				return nil, Fail("NONDETERMINISTIC_DIAGNOSTIC_ORDER", "diagnostics must follow registered fields")
			}
			previous = key
		}
	}
	return Object{"status": "PASS", "kind": kind, "execution_performed": false}, nil
}

// SupportField consumes an exact class-owned Root ref registered in the Go audit contract.
func (r *Repository) SupportField(name string) (any, error) {
	refs, err := r.Read("developer/policy/contracts/go-support-audit-refs.v1.json")
	if err != nil {
		return nil, err
	}
	reference := Map(Map(refs["fields"])[name])
	if reference == nil {
		return nil, Fail("SUPPORT_AUDIT_REF_UNREGISTERED", name)
	}
	id := Text(reference["policy_id"])
	path := Text(reference["policy_path"])
	index, err := r.Read("src/policy/index.yaml")
	if err != nil {
		return nil, err
	}
	selected := []Object{}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		if entry["id"] == id {
			selected = append(selected, entry)
		}
	}
	if len(selected) != 1 || "src/policy/"+Text(selected[0]["path"]) != path {
		return nil, Fail("SUPPORT_ROOT_METADATA_MISMATCH", id)
	}
	policy, err := r.Read(path)
	if err != nil {
		return nil, err
	}
	if policy["policy_class"] != "PTSIP_SUPPORT_FEATURE" || Map(policy["policy"])["id"] != id || Map(policy["policy"])["status"] != selected[0]["status"] {
		return nil, Fail("SUPPORT_ROOT_METADATA_MISMATCH", id)
	}
	value, exists := Map(policy["authority_semantics"])[Text(reference["section"])]
	if !exists {
		return nil, Fail("SUPPORT_ROOT_SECTION_MISSING", name)
	}
	for _, field := range Strings(reference["value_path"]) {
		mapped := Map(value)
		var ok bool
		value, ok = mapped[field]
		if !ok {
			return nil, Fail("SUPPORT_ROOT_FIELD_MISSING", field)
		}
	}
	return value, nil
}
func (r *Repository) VerifyPreservedFiles(preserved []any, activation Object) error {
	changes := Map(activation["authorized_preservation_changes"])
	for _, raw := range preserved {
		record := Map(raw)
		relative := Text(record["path"])
		operation := Text(changes[relative])
		path, err := r.Path(relative)
		if err != nil {
			return err
		}
		if operation == "DELETE" {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				return Fail("RETIRED_SOURCE_PRESENT", relative)
			}
			continue
		}
		if operation == "MODIFY" {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if operation == "MODIFY_LIFECYCLE_ONLY" {
			content = bytes.Replace(content, []byte("status: RETIRED"), []byte("status: ACTIVE"), 1)
		}
		if LFDigest(content) != Text(record["lf_sha256"]) {
			return Fail("PRESERVED_SOURCE_CHANGED", relative)
		}
	}
	return nil
}
func (r *Repository) ExactAuditScope(record Object, expected []string, check bool) error {
	if !check {
		return nil
	}
	head, err := r.GitOutput("rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head != record["base_head"] {
		return Fail("BASE_HEAD_CHANGED", "audit scope base differs from HEAD")
	}
	changed, err := r.GitOutput("diff", "--name-only", "HEAD")
	if err != nil {
		return err
	}
	untracked, err := r.GitOutput("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return err
	}
	actual := []string{}
	for _, path := range strings.Split(changed+"\n"+untracked, "\n") {
		if path != "" {
			actual = append(actual, path)
		}
	}
	actual = UniqueStrings(actual)
	expected = UniqueStrings(expected)
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		return Fail("EXACT_CHANGE_SCOPE_MISMATCH", "worktree paths differ from the approved audit scope")
	}
	return nil
}
func (r *Repository) VerifyContractRegistration(check bool) (Object, error) {
	activation, err := r.ActivationRecord()
	if err != nil {
		return nil, err
	}
	record, err := r.ScopeRecord()
	if err != nil {
		return nil, err
	}
	allocation := Map(record["allocation"])
	if allocation == nil {
		return nil, Fail("PREFLIGHT_ALLOCATION_REQUIRED", "allocation must already exist")
	}
	identity, err := r.Read("src/vpms/contracts/identity-registry.json")
	if err != nil {
		return nil, err
	}
	catalog, err := r.Read("src/vpms/contracts/index.json")
	if err != nil {
		return nil, err
	}
	expected := Map(allocation["product_contract_ids"])
	if !reflect.DeepEqual(expected, Map(catalog["entrypoints"])) || len(Map(catalog["contracts"])) != 3 {
		return nil, Fail("ALLOCATION_CATALOG_MISMATCH", "product entrypoints differ from the allocated identities")
	}
	seen := map[string]bool{}
	active := activation != nil
	desired := "APPROVED"
	if active {
		desired = "ACTIVE"
	}
	design := Map(record["approved_design"])
	if identity["contract_class"] != design["contract_class"] {
		return nil, Fail("UNAPPROVED_CONTRACT_CLASS", "contract class differs from approval")
	}
	for role, value := range expected {
		id := Text(value)
		if seen[id] {
			return nil, Fail("CONTRACT_ID_COLLISION", id)
		}
		seen[id] = true
		payload, err := r.ProductContract(id, active)
		if err != nil {
			return nil, err
		}
		if payload["status"] != desired || payload["runtime_enabled"] != active {
			return nil, Fail("UNAUTHORIZED_RUNTIME_ACTIVATION", id)
		}
		if payload["owner_approval"] != "USER_EXPLICIT" {
			return nil, Fail("OWNER_APPROVAL_REQUIRED", id)
		}
		for _, owned := range Strings(payload["owns"]) {
			if Has(Strings(payload["excludes"]), owned) {
				return nil, Fail("OWN_EXCLUDE_OVERLAP", owned)
			}
		}
		semantics := Map(payload["semantics"])
		if role == "protocol" {
			for _, field := range []string{"expanded_identity", "governing_question"} {
				if Map(semantics["identity"])[field] != design[field] {
					return nil, Fail("UNAPPROVED_PROTOCOL_IDENTITY", field)
				}
			}
		}
		if role == "selection" {
			for field, approved := range map[string]string{"request_kinds": "selection_request_kinds", "rule_kind": "selection_rule_kind", "ordering": "ordering", "invalid_request": "duplicate_unknown_empty", "failure_atomicity": "failure_atomicity", "execution_side_effects": "selection_execution_side_effects"} {
				if !reflect.DeepEqual(semantics[field], design[approved]) {
					return nil, Fail("UNAPPROVED_SELECTION_SEMANTICS", field)
				}
			}
		}
		if role == "execution_composition" && payload["responsibility"] != "VPMS_EXECUTION_COMPOSITION" {
			return nil, Fail("EXECUTION_COMPOSITION_OWNER_MISMATCH", id)
		}
	}
	invariants, err := r.Read("src/vpms/contracts/invariants.json")
	if err != nil {
		return nil, err
	}
	if invariants["activation_requires_explicit_owner_approval"] != true {
		return nil, Fail("ACTIVATION_GATE_MISSING", "explicit owner activation invariant is missing")
	}
	root, _ := r.Path("src/vpms/contracts")
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte("MPD-")) || bytes.Contains(data, []byte("developer/")) {
			return Fail("PRODUCT_DEVELOPER_POLICY_DEPENDENCY", path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	runtimeSurface, err := r.SupportField("integration_runtime_surface")
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(Strings(runtimeSurface), []string{"src/vpms/integration/ptsip_bridge.py"}) {
		return nil, Fail("INTEGRATION_SCOPE_EXPANDED", "support runtime surface differs from approved boundary")
	}
	if err := r.VerifyPreservedFiles(List(record["preserved_files"]), activation); err != nil {
		return nil, err
	}
	expectedPaths := Strings(record["materialization_targets"])
	for _, relative := range expectedPaths {
		path, err := r.Path(relative)
		if err != nil {
			return nil, err
		}
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			return nil, Fail("MISSING_SCOPE_TARGET", relative)
		}
	}
	if check && active {
		return nil, Fail("ORIGINAL_SCOPE_SUPERSEDED_BY_ACTIVATION", "original worktree scope is no longer applicable")
	}
	for _, raw := range List(record["preserved_files"]) {
		path := Text(Map(raw)["path"])
		if strings.HasPrefix(path, "developer/") {
			expectedPaths = append(expectedPaths, path)
		}
	}
	if err := r.ExactAuditScope(record, expectedPaths, check); err != nil {
		return nil, err
	}
	supportStatus := "DRAFT"
	status := "REGISTERED_NON_ACTIVE"
	if active {
		supportStatus = "ACTIVE"
		status = "REGISTERED_ACTIVE"
	}
	return Object{"status": status, "support_policy_id": allocation["support_policy_id"], "support_status": supportStatus, "product_contract_count": 3, "product_status": desired, "runtime_enabled": active, "scope_target_count": len(List(record["materialization_targets"])), "preserved_file_count": len(List(record["preserved_files"]))}, nil
}
func (r *Repository) SupportRegistrationPreflight() (Object, error) {
	record, err := r.ScopeRecord()
	if err != nil {
		return nil, err
	}
	if allocation := Map(record["allocation"]); allocation != nil {
		result := Object{"status": "ALLOCATED"}
		for key, value := range allocation {
			result[key] = value
		}
		return result, nil
	}
	return nil, Fail("ROOT_FAMILY_REGISTRATION_REQUIRED", "generic SFP numeric allocation has retired; use explicit class + Root Family entry and lifecycle admission")
}
func (r *Repository) VerifyAPIImplementation(check bool) (Object, error) {
	activation, err := r.ActivationRecord()
	if err != nil {
		return nil, err
	}
	record, err := r.Read(implementationScopeRecord)
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/vpms-api-implementation.schema.json", record); err != nil {
		return nil, err
	}
	prior, err := r.ReadSource(Text(record["prior_materialization_ref"]))
	if err != nil {
		return nil, err
	}
	if LFDigest(prior) != record["prior_materialization_sha256"] {
		return nil, Fail("PRIOR_MATERIALIZATION_CHANGED", "prior materialization fingerprint differs")
	}
	if _, err := r.VerifyContractRegistration(false); err != nil {
		return nil, err
	}
	if err := r.VerifyPreservedFiles(List(record["preserved_files"]), activation); err != nil {
		return nil, err
	}
	retired := []string{}
	for _, raw := range List(record["subsequent_retirements"]) {
		retired = append(retired, Text(Map(raw)["path"]))
	}
	targets := Strings(record["mutation_targets"])
	for _, path := range retired {
		if !Has(targets, path) {
			return nil, Fail("RETIREMENT_OUTSIDE_ORIGINAL_SCOPE", path)
		}
		candidate, err := r.Path(path)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(candidate); !os.IsNotExist(err) {
			return nil, Fail("RETIRED_IMPLEMENTATION_TARGET_PRESENT", path)
		}
	}
	for _, path := range targets {
		if Has(retired, path) {
			continue
		}
		candidate, err := r.Path(path)
		if err != nil {
			return nil, err
		}
		if info, err := os.Stat(candidate); err != nil || info.IsDir() {
			return nil, Fail("MISSING_IMPLEMENTATION_TARGET", path)
		}
	}
	for _, path := range Strings(record["removed_uncommitted_paths"]) {
		candidate, err := r.Path(path)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(candidate); !os.IsNotExist(err) {
			return nil, Fail("RETIRED_WRITE_PATH_PRESENT", path)
		}
	}
	if record["support_policy_generation_root"] != "src/policy" {
		return nil, Fail("INVALID_SUPPORT_POLICY_WRITE_ROOT", "support generation root differs")
	}
	desired := "IMPLEMENTED_INACTIVE"
	if activation != nil {
		desired = "IMPLEMENTED_ACTIVE"
	}
	for _, raw := range List(record["bindings"]) {
		binding := Map(raw)
		file := map[string]string{"protocol": "protocol.json", "selection": "selection.json", "execution_composition": "execution-composition.json"}[Text(binding["contract"])]
		if file == "" {
			return nil, Fail("UNKNOWN_PRODUCT_CONTRACT_ROLE", Text(binding["contract"]))
		}
		payload, err := r.Read("src/vpms/contracts/" + file)
		if err != nil {
			return nil, err
		}
		matches := []Object{}
		for _, raw := range List(payload["planned_bindings"]) {
			item := Map(raw)
			if item["api"] == binding["api"] {
				matches = append(matches, item)
			}
		}
		if len(matches) != 1 || matches[0]["source"] != binding["source"] || matches[0]["availability"] != desired {
			return nil, Fail("IMPLEMENTATION_BINDING_MISMATCH", Text(binding["api"]))
		}
	}
	if check && (activation != nil || len(retired) > 0) {
		return nil, Fail("ORIGINAL_SCOPE_SUPERSEDED", "historical scope superseded by activation or retirement")
	}
	priorRecord := Object{}
	if err := json.Unmarshal(prior, &priorRecord); err != nil {
		return nil, err
	}
	expected := append(Strings(priorRecord["materialization_targets"]), targets...)
	for _, raw := range List(record["preserved_files"]) {
		path := Text(Map(raw)["path"])
		if strings.HasPrefix(path, "developer/") {
			expected = append(expected, path)
		}
	}
	if err := r.ExactAuditScope(record, expected, check); err != nil {
		return nil, err
	}
	return Object{"status": desired, "runtime_activation": activation != nil, "commit_push_authorized": true, "implementation_target_count": len(targets), "combined_changed_path_count": len(UniqueStrings(expected)), "preserved_file_count": len(List(record["preserved_files"])), "retired_target_count": len(retired)}, nil
}
func init() {
	RegisterOperations("support-contract", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		switch command {
		case "preflight":
			return r.SupportRegistrationPreflight()
		case "verify":
			return r.VerifyContractRegistration(BoolOption(opts, "check-worktree"))
		case "inspect":
			return r.ProductContract(args[0], BoolOption(opts, "require-active"))
		case "validate-selection":
			payload, err := r.Read(opts["--input"])
			if err != nil {
				return nil, err
			}
			return r.ValidateSelectionDocument(opts["--kind"], payload)
		}
		return nil, fmt.Errorf("unregistered support-contract operation")
	})
	RegisterOperations("vpms-api", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		return r.VerifyAPIImplementation(BoolOption(opts, "check-worktree"))
	})
}
