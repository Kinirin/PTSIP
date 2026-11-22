package contracts_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeDiscoverySource(t *testing.T, root, relative string, data []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeDiscoveryJSON(t *testing.T, root, relative string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	writeDiscoverySource(t, root, relative, data)
}

func discoverySourceOutput(t *testing.T, operation string, input object) object {
	t.Helper()
	response, err := runProfileOperation(t, "discovery", operation, input)
	if err != nil || response["status"] != "PASS" {
		t.Fatalf("native Discovery source operation failed: %v: %#v", err, response)
	}
	return response["output"].(object)
}

func discoverySourceFailure(t *testing.T, operation string, input object, diagnostic string) {
	t.Helper()
	response, err := runProfileOperation(t, "discovery", operation, input)
	if err == nil || response["status"] != "FAIL" {
		t.Fatalf("invalid Discovery source was accepted: %v: %#v", err, response)
	}
	if diagnostic != "" && !strings.Contains(fmt.Sprint(response["error"]), diagnostic) {
		t.Fatalf("failure omitted diagnostic %q: %#v", diagnostic, response)
	}
}

const discoveryProfileSupport = "engines/discovery/profile-support.json"

func copyDiscoverySelectionSources(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyJSONTree(t, filepath.Join(productRoot(t), "schemas"), filepath.Join(root, "schemas"))
	support, err := os.ReadFile(filepath.Join(productRoot(t), filepath.FromSlash(discoveryProfileSupport)))
	if err != nil {
		t.Fatal(err)
	}
	writeDiscoverySource(t, root, discoveryProfileSupport, support)
	return root
}

func TestDiscoveryProfileSourceIntegrity(t *testing.T) {
	t.Run("engine_owned_support_controls_operations", func(t *testing.T) {
		root := copyDiscoverySelectionSources(t)
		writeDiscoveryJSON(t, root, discoveryProfileSupport, object{"pp.3.00": array{"IDENTIFY"}})
		output := discoverySourceOutput(t, "discovery.profile.contract", object{"root": root})
		operations := output["operations"].(array)
		if output["version"] != "pp.3.00" || output["lifecycle"] != "ACTIVE" || len(operations) != 1 || operations[0] != "IDENTIFY" {
			t.Fatalf("separate registrations did not determine their own fields: %v", output)
		}
		writeDiscoverySource(t, root, discoveryProfileSupport, []byte(`{"pp.3.00":["IDENTIFY"],"pp.3.00":["VALIDATE"]}`))
		discoverySourceFailure(t, "discovery.profile.contract", object{"root": root}, "duplicate")
	})
	for _, tc := range []struct {
		name, schema, version string
	}{
		{"versioned_resource", "history/profile/project-profile-pp.2.00.schema.json", "pp.2.00"},
		{"current_alias", "storage/profile/project-profile.schema.json", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := copyDiscoverySelectionSources(t)
			path := filepath.Join(root, "schemas", filepath.FromSlash(tc.schema))
			doc := readObject(t, path)
			// Keep $id, version and all original validation keywords intact.
			doc["title"] = "modified resource with unchanged identity and version"
			writeDiscoveryJSON(t, root, "schemas/"+tc.schema, doc)
			input := object{"root": root}
			if tc.version != "" {
				input["contract_version"] = tc.version
			}
			discoverySourceFailure(t, "discovery.profile.contract", input, "schema SHA256 mismatch")
		})
	}
	t.Run("checkout_line_endings_preserve_canonical_pin_and_report_raw_hash", func(t *testing.T) {
		root := copyDiscoverySelectionSources(t)
		relative := "schemas/history/profile/project-profile-pp.2.00.schema.json"
		original, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		canonical := bytes.ReplaceAll(original, []byte("\r\n"), []byte("\n"))
		crlf := bytes.ReplaceAll(canonical, []byte("\n"), []byte("\r\n"))
		writeDiscoverySource(t, root, relative, crlf)
		rawDigest := fmt.Sprintf("%x", sha256.Sum256(crlf))
		output := discoverySourceOutput(t, "discovery.profile.contract", object{
			"root": root, "contract_version": "pp.2.00", "source_revision": rawDigest,
		})
		if output["source_sha256"] != rawDigest {
			t.Fatalf("source digest did not describe the actual read bytes: %#v", output)
		}
		config := readObject(t, filepath.Join(root, "schemas", "config.json"))
		registration := config["storage"].(object)["profile"].(object)["ptsip.project-profile.pp.2.00"].(object)
		if registration["schema_sha256"] != fmt.Sprintf("%x", sha256.Sum256(canonical)) {
			t.Fatal("canonical schema digest differs from registered original bytes")
		}
	})
}

func discoveryCatalogFixture(t *testing.T, root string, legacy bool, withRules bool) {
	t.Helper()
	rows := array{object{"id": "main", "resource": "main.ptsip.yaml"}}
	if withRules {
		rows = append(rows, object{"id": "rules", "resource": "rules.ptsip.yaml"})
	}
	name := ".ptsip/profiles/index.json"
	if legacy {
		name = ".ptsip/profiles/index.yaml"
	}
	// JSON is also a valid strict YAML representation, while the legacy
	// fallback case below separately exercises block YAML parsing.
	writeDiscoveryJSON(t, root, name, object{
		"schema_version": "ptsip-local-profile-catalog/v1", "default_profile": "main", "profiles": rows,
	})
}

func TestDiscoveryProfileCatalogSources(t *testing.T) {
	t.Run("legacy_yaml_fallback_and_explicit_selection", func(t *testing.T) {
		root := t.TempDir()
		writeDiscoverySource(t, root, ".ptsip/profiles/index.yaml", []byte("schema_version: ptsip-local-profile-catalog/v1\ndefault_profile: main\nprofiles:\n  - id: main\n    resource: main.ptsip.yaml\n"))
		writeDiscoveryJSON(t, root, ".ptsip/profiles/main.ptsip.yaml", object{"ptsip": object{"version": "pp.2.00"}})
		output := discoverySourceOutput(t, "discovery.profile.catalog", object{"repository_root": root})
		if output["selected_id"] != "main" || output["catalog_valid"] != true {
			t.Fatalf("legacy catalog was not selected: %#v", output)
		}
		output = discoverySourceOutput(t, "discovery.profile.catalog", object{"repository_root": root, "catalog_ref": ".ptsip/profiles/index.json"})
		if output["catalog_valid"] != false || output["resource_path"] != nil {
			t.Fatalf("missing explicit JSON catalog silently fell back to YAML: %#v", output)
		}
		discoveryCatalogFixture(t, root, false, false)
		discoverySourceFailure(t, "discovery.profile.catalog", object{"repository_root": root}, "catalogs conflict")
	})
	for _, tc := range []struct{ name, relative, payload string }{
		{"duplicate_json_key", ".ptsip/profiles/index.json", `{"schema_version":"ptsip-local-profile-catalog/v1","default_profile":"main","default_profile":"main","profiles":[{"id":"main","resource":"main.ptsip.yaml"}]}`},
		{"duplicate_yaml_key", ".ptsip/profiles/index.yaml", "schema_version: ptsip-local-profile-catalog/v1\ndefault_profile: main\ndefault_profile: main\nprofiles:\n  - id: main\n    resource: main.ptsip.yaml\n"},
		{"yaml_alias", ".ptsip/profiles/index.yaml", "schema_version: ptsip-local-profile-catalog/v1\ndefault_profile: main\nprofiles: &profiles []\nother: *profiles\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeDiscoverySource(t, root, tc.relative, []byte(tc.payload))
			writeDiscoveryJSON(t, root, ".ptsip/profiles/main.ptsip.yaml", object{})
			discoverySourceFailure(t, "discovery.profile.catalog", object{"repository_root": root}, "")
		})
	}
	t.Run("ordinary_file_ancestor_is_not_a_missing_catalog", func(t *testing.T) {
		root := t.TempDir()
		writeDiscoverySource(t, root, ".ptsip", []byte("not a directory"))
		discoverySourceFailure(t, "discovery.profile.catalog", object{"repository_root": root}, "")
	})
}

func TestDiscoveryLinkedRuleProfileSources(t *testing.T) {
	records := array{
		object{"ID": "root 자유", "rule_value": object{"root_id": "child with spaces"}},
		object{"ID": "child with spaces", "parent_id": "root 자유", "rule_value": array{1, "opaque", false}},
	}
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("catalog_legacy_%v", legacy), func(t *testing.T) {
			root := t.TempDir()
			discoveryCatalogFixture(t, root, legacy, true)
			writeDiscoverySource(t, root, ".ptsip/profiles/main.ptsip.yaml", []byte("ptsip:\n  version: pp.2.00\n  rule_profile: rules.ptsip.yaml\n"))
			writeDiscoveryJSON(t, root, ".ptsip/profiles/rules.ptsip.yaml", object{"rules": records})
			output := discoverySourceOutput(t, "discovery.profile.rules", object{"repository_root": root, "rule_id": "child with spaces"})
			if output["source_ref"] != ".ptsip/profiles/rules.ptsip.yaml" || output["status"] != "RESOLVED" || output["validation_verdict"] != "NOT_ASSERTED" {
				t.Fatalf("linked exact lookup lost its source or claimed validation: %#v", output)
			}
			inline := discoverySourceOutput(t, "discovery.profile.rules", object{"records": records})
			if inline["records_sha256"] != output["records_sha256"] || inline["source_ref"] != nil {
				t.Fatalf("inline and linked discovery did not preserve the same records: %#v / %#v", inline, output)
			}
			validated := nativeProfileOutput(t, "validation.profile.rules", object{"records": output["records"]})
			raw, err := json.Marshal(validated["records"])
			if err != nil || validated["valid"] != true || fmt.Sprintf("%x", sha256.Sum256(raw)) != output["records_sha256"] {
				t.Fatalf("cross-engine records handoff changed bytes or graph result: %v: %#v / %#v", err, output, validated)
			}
		})
	}
	t.Run("explicit_main_payload_does_not_require_a_main_file", func(t *testing.T) {
		root := t.TempDir()
		discoveryCatalogFixture(t, root, false, true)
		writeDiscoveryJSON(t, root, ".ptsip/profiles/rules.ptsip.yaml", object{"rules": records})
		output := discoverySourceOutput(t, "discovery.profile.rules", object{
			"repository_root": root, "main_payload": object{"ptsip": object{"rule_profile": "rules.ptsip.yaml"}},
		})
		if output["source_ref"] != ".ptsip/profiles/rules.ptsip.yaml" || len(output["records"].(array)) != 2 {
			t.Fatalf("explicit main payload was not used: %#v", output)
		}
	})
	t.Run("absent_catalog_or_link_produces_empty_unvalidated_records", func(t *testing.T) {
		for _, input := range []object{
			{"repository_root": t.TempDir()},
			{"repository_root": t.TempDir(), "main_payload": object{"ptsip": object{"version": "pp.2.00"}}},
		} {
			output := discoverySourceOutput(t, "discovery.profile.rules", input)
			if len(output["records"].(array)) != 0 || output["source_ref"] != nil || output["validation_verdict"] != "NOT_ASSERTED" {
				t.Fatal(output)
			}
		}
	})
	t.Run("input_branches_are_exclusive", func(t *testing.T) {
		for _, input := range []object{
			{}, {"records": records, "repository_root": t.TempDir()},
			{"records": records, "main_payload": object{}}, {"main_payload": object{}},
		} {
			discoverySourceFailure(t, "discovery.profile.rules", input, "")
		}
	})
	t.Run("unregistered_or_ambiguous_link_fails", func(t *testing.T) {
		for _, row := range []object{
			{"id": "other", "resource": "rules.ptsip.yaml"},
			{"id": "rules", "resource": "other.ptsip.yaml"},
		} {
			root := t.TempDir()
			writeDiscoveryJSON(t, root, ".ptsip/profiles/index.json", object{
				"schema_version": "ptsip-local-profile-catalog/v1", "default_profile": "main",
				"profiles": array{object{"id": "main", "resource": "main.ptsip.yaml"}, row},
			})
			discoverySourceFailure(t, "discovery.profile.rules", object{
				"repository_root": root, "main_payload": object{"ptsip": object{"rule_profile": "rules.ptsip.yaml"}},
			}, "catalog registration")
		}
		discoverySourceFailure(t, "discovery.profile.rules", object{
			"repository_root": t.TempDir(), "main_payload": object{"ptsip": object{"rule_profile": "other.ptsip.yaml"}},
		}, "unregistered linked")
	})
	for _, tc := range []struct{ name, payload string }{
		{"missing_rules_array", `{"rules":{}}`},
		{"missing_record_identity", `{"rules":[{"rule_value":"opaque"}]}`},
		{"duplicate_exact_identity", `{"rules":[{"ID":"same"},{"ID":"same"}]}`},
		{"duplicate_yaml_key", "rules:\n  - ID: first\n    ID: second\n"},
		{"yaml_alias", "rules:\n  - &record {ID: first}\n  - *record\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			discoveryCatalogFixture(t, root, false, true)
			writeDiscoverySource(t, root, ".ptsip/profiles/rules.ptsip.yaml", []byte(tc.payload))
			discoverySourceFailure(t, "discovery.profile.rules", object{
				"repository_root": root, "main_payload": object{"ptsip": object{"rule_profile": "rules.ptsip.yaml"}},
			}, "")
		})
	}
	t.Run("graph_validation_has_its_own_engine_owner", func(t *testing.T) {
		discovered := discoverySourceOutput(t, "discovery.profile.rules", object{"records": array{object{"ID": "opaque child", "parent_id": "missing"}}})
		validated := nativeProfileOutput(t, "validation.profile.rules", object{"records": discovered["records"]})
		if discovered["validation_verdict"] != "NOT_ASSERTED" || validated["valid"] != false {
			t.Fatalf("Discovery or Validation crossed its responsibility boundary: %#v / %#v", discovered, validated)
		}
	})
}

func discoveryDirectoryAlias(t *testing.T, link, target string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
		if err != nil {
			t.Skipf("junction creation unavailable: %v: %s", err, output)
		}
		return
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("directory symlink creation unavailable: %v", err)
	}
}

func TestDiscoveryProfileSourceAliases(t *testing.T) {
	t.Run("selected_root_alias_is_rejected", func(t *testing.T) {
		link := filepath.Join(t.TempDir(), "root-alias")
		discoveryDirectoryAlias(t, link, t.TempDir())
		discoverySourceFailure(t, "discovery.profile.catalog", object{"repository_root": link}, "alias")
	})
	t.Run("alias_ancestor_is_rejected_when_catalog_leaf_is_missing", func(t *testing.T) {
		root := t.TempDir()
		discoveryDirectoryAlias(t, filepath.Join(root, ".ptsip"), t.TempDir())
		discoverySourceFailure(t, "discovery.profile.catalog", object{"repository_root": root}, "alias")
	})
}
