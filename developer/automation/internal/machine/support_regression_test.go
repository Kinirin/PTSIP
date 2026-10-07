package machine

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// These synthetic approvals and byte preimages live only in a temporary test
// repository. Current source approval records and their trusted hashes are never
// rewritten to make a migration or an operational verification pass.
func syntheticAuditFixture(t *testing.T) *Repository {
	t.Helper()
	source, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	r := policyTestRepo(t)
	for _, root := range []string{"src/vpms", "src/policy"} {
		origin, err := source.Path(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := filepath.WalkDir(origin, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "legacy" || entry.Name() == "__pycache__" {
					return filepath.SkipDir
				}
				return nil
			}
			ref, err := filepath.Rel(source.Root, path)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return r.AtomicWrite(filepath.ToSlash(ref), raw, nil)
		}); err != nil {
			t.Fatal(err)
		}
	}
	scope, err := r.Read(supportScopeRecord)
	if err != nil {
		t.Fatal(err)
	}
	api, err := r.Read(implementationScopeRecord)
	if err != nil {
		t.Fatal(err)
	}
	activation, err := r.Read(activationScopeRecord)
	if err != nil {
		t.Fatal(err)
	}
	retired := map[string]bool{}
	for _, raw := range List(api["subsequent_retirements"]) {
		retired[Text(Map(raw)["path"])] = true
	}
	for ref, action := range Map(activation["authorized_preservation_changes"]) {
		if action == "DELETE" {
			retired[ref] = true
		}
	}
	refs := append(Strings(scope["materialization_targets"]), Strings(api["mutation_targets"])...)
	for _, record := range []Object{scope, api} {
		for _, raw := range List(record["preserved_files"]) {
			refs = append(refs, Text(Map(raw)["path"]))
		}
	}
	for _, ref := range UniqueStrings(refs) {
		if retired[ref] {
			continue
		}
		if strings.HasPrefix(ref, "developer/automation/") && strings.HasSuffix(ref, ".py") {
			if err := r.VerifyAuditImplementationTarget(ref); err != nil {
				t.Fatal("synthetic scope requires an admitted native implementation", err)
			}
			continue
		}
		if iwpPathExists(r, ref) {
			continue
		}
		path, err := source.Path(ref)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			raw = []byte("Synthetic historical scope target; no runtime authority.\n")
			if ref == "src/policy/SFP-0006.yaml" {
				raw = []byte("status: RETIRED\n")
			}
		} else if err != nil {
			t.Fatal(err)
		}
		if err := r.AtomicWrite(ref, raw, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, record := range []Object{scope, api} {
		for _, raw := range List(record["preserved_files"]) {
			item := Map(raw)
			ref := Text(item["path"])
			if retired[ref] {
				continue
			}
			content, err := r.ReadSource(ref)
			if err != nil {
				t.Fatal(err)
			}
			if Map(activation["authorized_preservation_changes"])[ref] == "MODIFY_LIFECYCLE_ONLY" {
				content = bytes.Replace(content, []byte("status: RETIRED"), []byte("status: ACTIVE"), 1)
			}
			item["sha256"], item["lf_sha256"] = SHA256(content), LFDigest(content)
		}
	}
	if err := r.WriteJSON(supportScopeRecord, scope, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := r.ReadSource(supportScopeRecord)
	if err != nil {
		t.Fatal(err)
	}
	api["prior_materialization_sha256"] = LFDigest(raw)
	if err := r.WriteJSON(implementationScopeRecord, api, nil); err != nil {
		t.Fatal(err)
	}
	for ref := range Map(activation["prior_records"]) {
		raw, err := r.ReadSource(ref)
		if err != nil {
			t.Fatal(err)
		}
		Map(activation["prior_records"])[ref] = LFDigest(raw)
	}
	// Bind the temporary schema to these synthetic fixture preimages. The source
	// schema and original approval's const fingerprints remain untouched.
	activationSchema, err := r.Read("developer/policy/schemas/vpms-runtime-activation.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	Map(Map(activationSchema["properties"])["prior_records"])["const"] = policyClone(Map(activation["prior_records"]))
	if err := r.WriteJSON("developer/policy/schemas/vpms-runtime-activation.schema.json", activationSchema, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteJSON(activationScopeRecord, activation, nil); err != nil {
		t.Fatal(err)
	}
	return r
}

func requireAuditCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), code) {
		t.Fatalf("expected %s, got %v", code, err)
	}
}

func TestSupportRegistrationScopeAndAllocationHaveExplicitSeparateApproval(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	scope, err := r.ScopeRecord()
	if err != nil {
		t.Fatal(err)
	}
	approval := Map(scope["approval"])
	if approval["decision"] != "APPROVED" || approval["decision_source"] != "USER_EXPLICIT" {
		t.Fatal(approval)
	}
	for _, field := range []string{"runtime_activation_authorized", "selector_retirement_authorized", "sfp_0006_retirement_authorized", "commit_push_authorized"} {
		if approval[field] != false {
			t.Fatal(field, approval)
		}
	}
	first, err := r.SupportRegistrationPreflight()
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.SupportRegistrationPreflight()
	if err != nil || !reflect.DeepEqual(first, second) || first["status"] != "ALLOCATED" || !reflect.DeepEqual(first["product_contract_ids"], Map(scope["allocation"])["product_contract_ids"]) {
		t.Fatal(first, second, err)
	}
}

func TestSupportRegistrationAndAPIHappyPathsUseSyntheticApprovedFixture(t *testing.T) {
	r := syntheticAuditFixture(t)
	result, err := r.VerifyContractRegistration(false)
	if err != nil || result["status"] != "REGISTERED_ACTIVE" || result["support_policy_id"] != "SFP-0023" || agentInt(result["product_contract_count"]) != 3 || result["runtime_enabled"] != true {
		t.Fatal(result, err)
	}
	api, err := r.VerifyAPIImplementation(false)
	if err != nil || api["status"] != "IMPLEMENTED_ACTIVE" || api["commit_push_authorized"] != true || api["runtime_activation"] != true || agentInt(api["retired_target_count"]) != 1 {
		t.Fatal(api, err)
	}
}

func TestSupportContractLookupRejectsUnknownInactiveAndMismatchedIdentity(t *testing.T) {
	r := syntheticAuditFixture(t)
	catalog, err := r.Read("src/vpms/contracts/index.json")
	if err != nil {
		t.Fatal(err)
	}
	catalog["capability"] = "REGISTERED_NON_ACTIVE_ONLY"
	for id, raw := range Map(catalog["contracts"]) {
		entry := Map(raw)
		entry["status"], entry["runtime_enabled"] = "APPROVED", false
		ref := "src/vpms/contracts/" + Text(entry["path"])
		payload, err := r.Read(ref)
		if err != nil {
			t.Fatal(err)
		}
		payload["status"], payload["runtime_enabled"] = "APPROVED", false
		if err := r.WriteJSON(ref, payload, nil); err != nil {
			t.Fatal(err)
		}
		if id == "" {
			t.Fatal("empty identity")
		}
	}
	if err := r.WriteJSON("src/vpms/contracts/index.json", catalog, nil); err != nil {
		t.Fatal(err)
	}
	_, err = r.ProductContract("protocol", false)
	requireAuditCode(t, err, "UNKNOWN_CONTRACT_ID")
	for _, raw := range Map(catalog["entrypoints"]) {
		_, err := r.ProductContract(Text(raw), true)
		requireAuditCode(t, err, "CONTRACT_NOT_ACTIVE")
	}
	id := Text(Map(catalog["entrypoints"])["protocol"])
	payload, err := r.Read("src/vpms/contracts/protocol.json")
	if err != nil {
		t.Fatal(err)
	}
	payload["status"], payload["runtime_enabled"] = "ACTIVE", true
	if err := r.WriteJSON("src/vpms/contracts/protocol.json", payload, nil); err != nil {
		t.Fatal(err)
	}
	_, err = r.ProductContract(id, false)
	requireAuditCode(t, err, "CONTRACT_INDEX_MISMATCH")
	entry := Map(Map(catalog["contracts"])[id])
	entry["status"], entry["runtime_enabled"] = "ACTIVE", true
	if err := r.WriteJSON("src/vpms/contracts/index.json", catalog, nil); err != nil {
		t.Fatal(err)
	}
	_, err = r.ProductContract(id, true)
	requireAuditCode(t, err, "CONTRACT_NOT_ACTIVE")
}

func TestSupportContractRegisteredPathsCannotEscapeTheirRoot(t *testing.T) {
	for _, path := range []string{"../policy/SFP-0006.yaml", "/absolute", "nested/../../escape"} {
		t.Run(path, func(t *testing.T) {
			r := syntheticAuditFixture(t)
			catalog, err := r.Read("src/vpms/contracts/index.json")
			if err != nil {
				t.Fatal(err)
			}
			id := Text(Map(catalog["entrypoints"])["protocol"])
			Map(Map(catalog["contracts"])[id])["path"] = path
			if err := r.WriteJSON("src/vpms/contracts/index.json", catalog, nil); err != nil {
				t.Fatal(err)
			}
			_, err = r.ProductContract(id, false)
			if err == nil {
				t.Fatal("unsafe registered path admitted")
			}
		})
	}
}

func TestSupportDuplicateJSONKeysFailClosed(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	for _, source := range []string{`{"id":"one","id":"two"}`, `{"nested":{"id":"one","id":"two"}}`} {
		if err := r.AtomicWrite("duplicate.json", []byte(source), nil); err != nil {
			t.Fatal(err)
		}
		_, err := r.Read("duplicate.json")
		requireAuditCode(t, err, "DUPLICATE_JSON_KEY")
	}
}

func TestSupportApprovalSchemaDoesNotInferRuntimeOrRetirementFlags(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	scope, err := r.ScopeRecord()
	if err != nil {
		t.Fatal(err)
	}
	Map(scope["approval"])["runtime_activation_authorized"] = true
	if err := r.Validate("developer/policy/schemas/vpms-contract-materialization.schema.json", scope); err == nil {
		t.Fatal("materialization approval escalated to runtime activation")
	}
	activation, err := r.Read(activationScopeRecord)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"runtime_activation_authorized", "selector_retirement_authorized", "sfp_0006_retirement_authorized", "sfp_0023_activation_authorized"} {
		fixture := policyClone(activation)
		Map(fixture["approval"])[field] = false
		if err := r.Validate("developer/policy/schemas/vpms-runtime-activation.schema.json", fixture); err == nil {
			t.Fatal("missing explicit successor approval accepted", field)
		}
	}
}

func TestSupportPreservationAllowsOnlyCRLFTextConversionAgainstOriginalPreimages(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	scope, err := r.ScopeRecord()
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range List(scope["preserved_files"]) {
		item := Map(raw)
		ref := Text(item["path"])
		t.Run(ref, func(t *testing.T) {
			command := exec.Command("git", "-C", r.Root, "show", "7dd0adc:"+ref)
			bytes, err := command.Output()
			if err != nil {
				t.Fatal(err)
			}
			lf := bytesReplaceCRLF(bytes)
			if LFDigest(lf) != item["lf_sha256"] || LFDigest(bytesReplaceLF(lf)) != item["lf_sha256"] || LFDigest(append(lf, []byte("# semantic/source change\n")...)) == item["lf_sha256"] {
				t.Fatal("immutable preimage preservation mismatch", ref)
			}
		})
	}
	if LFDigest([]byte("A\rB\r")) == LFDigest([]byte("A\nB\n")) || LFDigest([]byte("Ａ\nB\n")) == LFDigest([]byte("A\nB\n")) {
		t.Fatal("lone CR or Unicode alteration accepted as EOL conversion")
	}
}

func bytesReplaceCRLF(raw []byte) []byte { return bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n")) }
func bytesReplaceLF(raw []byte) []byte   { return bytes.ReplaceAll(raw, []byte("\n"), []byte("\r\n")) }

func TestSupportSuccessorApprovalDoesNotRewriteImplementationOnlyScope(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	activation, err := r.Read(activationScopeRecord)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := r.ScopeRecord()
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := r.Read(implementationScopeRecord)
	if err != nil {
		t.Fatal(err)
	}
	if Map(activation["approval"])["runtime_activation_authorized"] != true || Map(scope["approval"])["runtime_activation_authorized"] != false {
		t.Fatal("approval planes collapsed")
	}
	approval := Map(implementation["approval"])
	if approval["decision_source"] != "USER_EXPLICIT" || approval["automatic_commit_push_authorized"] != true || approval["selector_retirement_authorized"] != false || approval["sfp_0006_retirement_authorized"] != false || implementation["task_context_status"] != "UNREGISTERED" || implementation["support_policy_generation_root"] != "src/policy" {
		t.Fatal(implementation)
	}
}

func TestSupportRetirementCannotWaiveMissingTargetsOrAdmitReintroducedFiles(t *testing.T) {
	for _, mode := range []string{"missing_retirement_declaration", "retired_reintroduced", "live_missing", "selector_reintroduced", "preserved_semantics_changed", "successor_missing"} {
		t.Run(mode, func(t *testing.T) {
			r := syntheticAuditFixture(t)
			scope, err := r.ScopeRecord()
			if err != nil {
				t.Fatal(err)
			}
			activation, err := r.ActivationRecord()
			if err != nil {
				t.Fatal(err)
			}
			ref, code := "", ""
			switch mode {
			case "missing_retirement_declaration":
				record, err := r.Read(implementationScopeRecord)
				if err != nil {
					t.Fatal(err)
				}
				delete(record, "subsequent_retirements")
				if err := r.WriteJSON(implementationScopeRecord, record, nil); err != nil {
					t.Fatal(err)
				}
				raw, _ := r.ReadSource(implementationScopeRecord)
				Map(activation["prior_records"])[implementationScopeRecord] = LFDigest(raw)
				if err := r.WriteJSON(activationScopeRecord, activation, nil); err != nil {
					t.Fatal(err)
				}
				schema, err := r.Read("developer/policy/schemas/vpms-runtime-activation.schema.json")
				if err != nil {
					t.Fatal(err)
				}
				Map(Map(schema["properties"])["prior_records"])["const"] = policyClone(Map(activation["prior_records"]))
				if err := r.WriteJSON("developer/policy/schemas/vpms-runtime-activation.schema.json", schema, nil); err != nil {
					t.Fatal(err)
				}
				r = &Repository{Root: r.Root}
				_, err = r.VerifyAPIImplementation(false)
				requireAuditCode(t, err, "MISSING_IMPLEMENTATION_TARGET")
				return
			case "retired_reintroduced":
				ref, code = "docs/Support_policy/automation/README.md", "RETIRED_IMPLEMENTATION_TARGET_PRESENT"
			case "live_missing":
				ref, code = "src/vpms/domain/snapshot.py", "MISSING_IMPLEMENTATION_TARGET"
				path, _ := r.Path(ref)
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				_, err = r.VerifyAPIImplementation(false)
				requireAuditCode(t, err, code)
				return
			case "selector_reintroduced":
				ref, code = "src/vpms/domain/selector.py", "RETIRED_SOURCE_PRESENT"
			case "preserved_semantics_changed":
				ref, code = "src/policy/SFP-0006.yaml", "PRESERVED_SOURCE_CHANGED"
			case "successor_missing":
				path, _ := r.Path(activationScopeRecord)
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if result, err := r.VerifyContractRegistration(false); err == nil || result != nil {
					t.Fatal("active contract admitted without successor approval", result, err)
				}
				return
			}
			if err := r.AtomicWrite(ref, []byte("unauthorized fixture mutation\n"), nil); err != nil {
				t.Fatal(err)
			}
			if mode == "retired_reintroduced" {
				_, err := r.VerifyAPIImplementation(false)
				requireAuditCode(t, err, code)
			} else {
				requireAuditCode(t, r.VerifyPreservedFiles(List(scope["preserved_files"]), activation), code)
			}
		})
	}
}
