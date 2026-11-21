package contracts_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	usercontracts "github.com/Kinirin/PTSIP/src/ptsip/user_contracts"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

type profileVersion struct {
	version    string
	canonical  string
	lifecycle  string
	sourceBlob string
}

var profileVersions = []profileVersion{
	{"pp.1.01", "storage/profile/history/project-profile-pp.1.01.schema.json", "DEPRECATED", "e8379e2d16f2a3c30b7c192c663fa45dc4ee7fec"},
	{"pp.1.02", "storage/profile/history/project-profile-pp.1.02.schema.json", "DEPRECATED", "07fbfe8a5aa9dacc56f7b76e9ceaa3474c5599f2"},
	{"pp.2.00", "storage/profile/history/project-profile-pp.2.00.schema.json", "DEPRECATED", "14d47e469e5f12ae65d721f64b17bdc35de861d8"},
}

type offlineProfileLoader struct{}

func (offlineProfileLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("unregistered profile schema resource %s", url)
}

func profileSchemaCompiler(t *testing.T) (*jsonschema.Compiler, map[string]*jsonschema.Schema) {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(offlineProfileLoader{})
	identities := map[string]string{}
	for _, version := range profileVersions {
		schema := readObject(t, filepath.Join(productRoot(t), "schemas", filepath.FromSlash(version.canonical)))
		id := schema["$id"].(string)
		if err := compiler.AddResource(id, schema); err != nil {
			t.Fatalf("register %s: %v", version.version, err)
		}
		identities[version.version] = id
	}
	compiled := map[string]*jsonschema.Schema{}
	for _, version := range profileVersions {
		schema, err := compiler.Compile(identities[version.version])
		if err != nil {
			t.Fatalf("compile historical %s: %v", version.version, err)
		}
		compiled[version.version] = schema
	}
	return compiler, compiled
}

func TestProjectProfileHistoricalContractsRetainImmutableIdentity(t *testing.T) {
	for _, version := range profileVersions {
		t.Run(version.version, func(t *testing.T) {
			relative := "src/ptsip/schemas/" + version.canonical
			if actual := gitWorkingBlobSHA(t, relative); actual != version.sourceBlob {
				t.Fatalf("historical contract changed: %s got %s want %s", relative, actual, version.sourceBlob)
			}
			canonical := readObject(t, filepath.Join(productRoot(t), "schemas", filepath.FromSlash(version.canonical)))
			expectedID := "https://raw.githubusercontent.com/Kinirin/PTSIP/main/schemas/ptsip-profile-" + strings.Replace(version.version, "pp.", "pp-", 1) + ".schema.json"
			if canonical["$id"] != expectedID || canonical["$schema"] != dialect {
				t.Fatalf("historical schema identity/dialect changed: %v", canonical)
			}
		})
	}
	profileSchemaCompiler(t)
}

// A fresh declaration from the retained field contract, not archived legacy YAML.
func historicalProfileInstance(version string) object {
	specification := object{"source": "https://github.com/Kinirin/PTSIP", "revision": strings.Repeat("1", 40)}
	metadata := object{"version": version, "specification": specification}
	if version == "pp.1.01" {
		specification["family"] = "0.3.7-draft"
	} else {
		metadata["revision"] = "Rev.0001"
		metadata["profile_role"] = "PROJECT"
	}
	return object{
		"ptsip":              metadata,
		"responsibility_map": object{"mode": "explicit"},
		"components":         array{object{"id": "example", "classification": "PRODUCT", "include": array{"src/example/**"}, "purpose": "An independent historical contract declaration"}},
		"policies":           object{"product_to_nonproduct_runtime_dependency": "deny", "nonproduct_in_product_package": "deny", "independent_build_resolution": "required"},
	}
}

func TestProjectProfileHistoricalContractSelection(t *testing.T) {
	_, historical := profileSchemaCompiler(t)
	_, current := officialPP300Schema(t)
	for _, version := range profileVersions {
		t.Run(version.version, func(t *testing.T) {
			instance := historicalProfileInstance(version.version)
			if err := historical[version.version].Validate(instance); err != nil {
				t.Fatalf("canonical %s rejected its field-contract declaration: %v", version.version, err)
			}
			for _, other := range profileVersions {
				if other.version != version.version && historical[other.version].Validate(instance) == nil {
					t.Errorf("historical %s accepted declaration for %s", other.version, version.version)
				}
			}
			if current.Validate(instance) == nil {
				t.Fatal("official pp.3.00 accepted a historical version declaration")
			}
			missingMap := clone(t, instance)
			delete(missingMap, "responsibility_map")
			if historical[version.version].Validate(missingMap) == nil {
				t.Fatal("historical contract dropped its required responsibility map")
			}
		})
	}
	if err := current.Validate(officialPP300Instance()); err != nil {
		t.Fatalf("current pp.3.00 declaration rejected: %v", err)
	}
}

func TestProjectProfileRecordedFixtureEvidence(t *testing.T) {
	root := repositoryRoot(t)
	plan := readObject(t, filepath.Join(root, "src", "planning", "0.3.8", "ptsip-schema-centralization.plan.json"))
	retirement := plan["profile_directory_retirement"].(object)
	original := retirement["original_head"].(string)
	git := func(args ...string) ([]byte, error) {
		command := exec.Command("git", args...)
		command.Dir = root
		return command.Output()
	}
	if _, err := git("cat-file", "-e", original+"^{commit}"); err != nil {
		shallow, shallowErr := git("rev-parse", "--is-shallow-repository")
		if shallowErr == nil && strings.TrimSpace(string(shallow)) == "true" {
			t.Skip("recorded fixture source commit is outside this shallow checkout")
		}
		t.Fatalf("recorded fixture source commit %s is unavailable: %v", original, err)
	}
	_, contracts := profileSchemaCompiler(t)
	checked := 0
	for _, row := range retirement["removal_manifest"].(array) {
		fixture := row.(object)
		group := fixture["role_group"]
		if group != "HISTORICAL_YAML_INSTANCE_FIXTURE" && group != "CURRENT_YAML_DISTRIBUTED_EXAMPLE" {
			continue
		}
		checked++
		path := fixture["source_path"].(string)
		t.Run(path, func(t *testing.T) {
			blob := fixture["git_blob_sha"].(string)
			resolved, err := git("rev-parse", "--verify", original+":"+path)
			if err != nil || strings.TrimSpace(string(resolved)) != blob {
				t.Fatalf("recorded fixture does not bind the original Git path: %s: %v", resolved, err)
			}
			raw, err := git("cat-file", "blob", blob)
			if err != nil || float64(len(raw)) != fixture["size_bytes"] {
				t.Fatalf("recorded fixture blob unavailable or size differs: %d: %v", len(raw), err)
			}
			var instance object
			if err := yaml.Unmarshal(raw, &instance); err != nil {
				t.Fatal(err)
			}
			identity, ok := instance["ptsip"].(object)
			if !ok {
				t.Fatal("fixture is missing its Profile identity object")
			}
			version, _ := identity["version"].(string)
			contract := contracts[version]
			if contract == nil {
				t.Fatalf("fixture has no registered historical schema: %q", version)
			}
			if err := contract.Validate(instance); err != nil {
				t.Fatalf("fixture differs from its registered %s schema: %v", version, err)
			}
		})
	}
	if checked != 12 {
		t.Fatalf("expected 12 recorded Profile fixtures, found %d", checked)
	}
}

func gitWorkingBlobSHA(t *testing.T, relative string) string {
	t.Helper()
	path := filepath.ToSlash(relative)
	command := exec.Command("git", "hash-object", "--path="+path, path)
	command.Dir = repositoryRoot(t)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("hash canonical Git blob %s: %v: %s", path, err, output)
	}
	return strings.TrimSpace(string(output))
}

func TestProjectProfileRegistryPinsSelectedSourceAndCurrentVersion(t *testing.T) {
	product := productRoot(t)
	config := readObject(t, filepath.Join(product, "schemas", "config.json"))
	profiles := config["storage"].(object)["profile"].(object)
	if len(profiles) != 4 {
		t.Fatalf("expected current selector and three version contracts, got %v", profiles)
	}
	for _, version := range profileVersions {
		key := "ptsip.project-profile." + version.version
		entry, ok := profiles[key].(object)
		if !ok {
			t.Fatalf("exact version registration missing: %s", key)
		}
		schema := readObject(t, filepath.Join(product, "schemas", filepath.FromSlash(version.canonical)))
		if entry["$id"] != schema["$id"] || entry["version"] != version.version || entry["path"] != version.canonical {
			t.Fatalf("version registration does not identify exact canonical schema: %s: %v", key, entry)
		}
		if entry["registered_lifecycle"] != version.lifecycle {
			t.Errorf("wrong registered lifecycle for %s: %v", key, entry)
		}
		if expected := version.sourceBlob; entry["source_blob_sha"] != expected {
			t.Errorf("immutable original-source blob differs for %s: got %v want %s", key, entry["source_blob_sha"], expected)
		}
		if expected := gitWorkingBlobSHA(t, "src/ptsip/schemas/"+version.canonical); entry["schema_blob_sha"] != expected {
			t.Errorf("canonical schema blob differs for %s: got %v want %s", key, entry["schema_blob_sha"], expected)
		}
	}
	entry, ok := profiles["ptsip.project-profile"].(object)
	if !ok {
		t.Fatal("current project-profile selector is missing")
	}
	selector := readObject(t, filepath.Join(product, "schemas", "storage", "profile", "project-profile.schema.json"))
	if selector["$id"] != officialProfileID || selector["title"] != "PTSIP official Current Project Profile" || selector["$ref"] != nil {
		t.Fatalf("current resource must be the direct official pp.3.00 contract: %v", selector)
	}
	if selector["properties"].(object)["ptsip"].(object)["properties"].(object)["version"].(object)["const"] != "pp.3.00" {
		t.Fatal("current resource has wrong official version")
	}
	if entry["$id"] != selector["$id"] || entry["version"] != "pp.3.00" || entry["path"] != "storage/profile/project-profile.schema.json" {
		t.Fatalf("current registration differs from official contract: %v", entry)
	}
	if entry["registered_lifecycle"] != "ACTIVE" {
		t.Errorf("current official contract must have ACTIVE lifecycle: %v", entry)
	}
	if expected := gitWorkingBlobSHA(t, "src/ptsip/schemas/storage/profile/project-profile.schema.json"); entry["schema_blob_sha"] != expected {
		t.Errorf("current official schema blob differs from file: %v", entry)
	}
	for _, denied := range []string{
		"ptsip.project-profile.pp.1.00", "ptsip.project-profile.candidate", "ptsip.mode-profile", "ptsip.user-profile", "ptsip.source-profile",
	} {
		if _, found := profiles[denied]; found {
			t.Errorf("unapproved profile target was registered: %s", denied)
		}
	}
	for _, relative := range []string{
		"storage/profile/history/project-profile-pp.1.00.schema.json", "storage/profile/project-profile-candidate.schema.json",
		"storage/profile/mode-profile.schema.json", "storage/profile/user-profile.schema.json", "storage/profile/source-profile.schema.json",
	} {
		if _, err := os.Stat(filepath.Join(product, "schemas", filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("unapproved target exists or cannot be checked: %s: %v", relative, err)
		}
	}
}

func TestProjectProfileRegisteredLifecycleUsesClosedEnum(t *testing.T) {
	cases := []struct {
		name     string
		value    any
		accepted bool
	}{
		{"active", "ACTIVE", true},
		{"legacy", "LEGACY", true},
		{"deprecated", "DEPRECATED", true},
		{"retired-current", "CURRENT", false},
		{"retired-superseded", "SUPERSEDED", false},
		{"lowercase", "active", false},
		{"unknown", "UNREGISTERED", false},
		{"empty", "", false},
		{"null", nil, false},
		{"boolean", true, false},
		{"number", 1, false},
		{"array", array{"ACTIVE"}, false},
		{"object", object{"status": "ACTIVE"}, false},
	}
	for _, key := range []string{"ptsip.project-profile", "ptsip.project-profile.pp.1.01"} {
		t.Run(key, func(t *testing.T) {
			for _, test := range cases {
				t.Run(test.name, func(t *testing.T) {
					root := t.TempDir()
					schemas := filepath.Join(root, "schemas")
					copyJSONTree(t, filepath.Join(productRoot(t), "schemas"), schemas)
					path := filepath.Join(schemas, "config.json")
					config := readObject(t, path)
					profiles := config["storage"].(object)["profile"].(object)
					profiles[key].(object)["registered_lifecycle"] = test.value
					payload, err := json.Marshal(config)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, payload, 0o644); err != nil {
						t.Fatal(err)
					}
					_, err = usercontracts.New(root)
					if (err == nil) != test.accepted {
						t.Fatalf("registered lifecycle %v accepted=%v, want %v: %v", test.value, err == nil, test.accepted, err)
					}
				})
			}
		})
	}
}

func TestRegisteredProfileLifecycleSelectionDecisions(t *testing.T) {
	catalog := validator(t)
	for _, key := range []string{"ptsip.project-profile"} {
		for _, newRegistration := range []bool{false, true} {
			selection, err := catalog.SelectRegisteredSchema(key, newRegistration)
			if err != nil || !selection.Allowed || selection.RegisteredLifecycle != usercontracts.Active {
				t.Fatalf("ACTIVE selection must allow existing and new use: %+v, %v", selection, err)
			}
			if selection.SchemaKey != key || selection.Version != "pp.3.00" || selection.SchemaPath == "" || len(selection.Notifications) != 0 {
				t.Fatalf("ACTIVE selection lost its registration identity or emitted a notice: %+v", selection)
			}
		}
	}
	for _, version := range profileVersions {
		key := "ptsip.project-profile." + version.version
		for _, newRegistration := range []bool{false, true} {
			selection, err := catalog.SelectRegisteredSchema(key, newRegistration)
			if err != nil || !selection.Allowed || selection.RegisteredLifecycle != usercontracts.Deprecated {
				t.Fatalf("DEPRECATED selection must remain available with an upgrade notice: %+v, %v", selection, err)
			}
			if selection.SchemaKey != key || selection.Version != version.version || selection.SchemaPath != version.canonical {
				t.Fatalf("DEPRECATED selection changed the requested contract: %+v", selection)
			}
			if len(selection.Notifications) != 1 || selection.Notifications[0].Code != "UPGRADE_REQUIRED" ||
				!selection.Notifications[0].Mandatory || selection.Notifications[0].Message == "" {
				t.Fatalf("DEPRECATED selection omitted the mandatory upgrade notice: %+v", selection)
			}
		}
	}

	root := t.TempDir()
	schemas := filepath.Join(root, "schemas")
	copyJSONTree(t, filepath.Join(productRoot(t), "schemas"), schemas)
	path := filepath.Join(schemas, "config.json")
	config := readObject(t, path)
	key := "ptsip.project-profile.pp.1.01"
	config["storage"].(object)["profile"].(object)[key].(object)["registered_lifecycle"] = "LEGACY"
	payload, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	legacyCatalog, err := usercontracts.New(root)
	if err != nil {
		t.Fatal(err)
	}
	existing, err := legacyCatalog.SelectRegisteredSchema(key, false)
	if err != nil || !existing.Allowed || existing.RegisteredLifecycle != usercontracts.Legacy || len(existing.Notifications) != 0 {
		t.Fatalf("LEGACY existing use must remain allowed: %+v, %v", existing, err)
	}
	admission, err := legacyCatalog.SelectRegisteredSchema(key, true)
	if err == nil || admission.Allowed || admission.RegisteredLifecycle != usercontracts.Legacy {
		t.Fatalf("LEGACY new registration must be denied: %+v, %v", admission, err)
	}
}

func TestProjectProfileLifecycleAndWorkAuthorizationFollowCurrentPlan(t *testing.T) {
	plan := readObject(t, filepath.Join(repositoryRoot(t), "src", "planning", "0.3.8", "ptsip-schema-centralization.plan.json"))
	lifecycle := plan["target_registry"].(object)["registered_lifecycle"].(object)
	allowed := map[string]bool{}
	for _, value := range lifecycle["allowed_constants"].(array) {
		state := value.(string)
		if allowed[state] {
			t.Fatalf("duplicate lifecycle constant in plan: %s", state)
		}
		allowed[state] = true
	}
	if !reflect.DeepEqual(allowed, map[string]bool{"ACTIVE": true, "LEGACY": true, "DEPRECATED": true}) {
		t.Fatalf("plan lifecycle vocabulary differs from the approved closed enum: %v", allowed)
	}
	assignment := lifecycle["current_assignment"].(object)
	if assignment["pp.3.00"] != "ACTIVE" || assignment["other_registered_project_profile_versions"] != "DEPRECATED" {
		t.Fatalf("plan assigns the wrong current or older profile lifecycle: %v", assignment)
	}
	definitions := lifecycle["definitions"].(object)
	active, legacy, deprecated := definitions["ACTIVE"].(object), definitions["LEGACY"].(object), definitions["DEPRECATED"].(object)
	if active["meaning"] != "CURRENT_STANDARD_NEW_CONTRACT_STRUCTURE" || active["new_registration"] != "ALLOWED" ||
		legacy["new_registration"] != "DENIED" || legacy["existing_use"] != "ALLOWED" ||
		deprecated["notification"] != "MANDATORY_TRANSITION_OR_UPGRADE_NOTICE_WHEN_SELECTED" {
		t.Fatalf("plan lifecycle behavior differs from approved selection semantics: %v", definitions)
	}
	config := readObject(t, filepath.Join(productRoot(t), "schemas", "config.json"))
	for key, value := range config["storage"].(object)["profile"].(object) {
		entry := value.(object)
		expected := assignment["other_registered_project_profile_versions"]
		if entry["version"] == "pp.3.00" {
			expected = assignment["pp.3.00"]
		}
		if entry["registered_lifecycle"] != expected {
			t.Errorf("registry %s differs from the current plan lifecycle: %v", key, entry)
		}
	}
	scope := plan["release_scope"].(object)
	if scope["full_038_work_units_approved"] != true ||
		scope["implementation_authorization_scope"] != "WU_01_THROUGH_WU_14_WITH_EXPLICIT_DECISION_GATES" ||
		scope["execution_order"] != "ACTUAL_DEPENDENCIES_NOT_WORK_UNIT_NUMBER" {
		t.Fatalf("plan lost continuous dependency-driven authorization: %v", scope)
	}
	workUnits, authorized := map[string]bool{}, map[string]bool{}
	for _, value := range plan["work_units"].(array) {
		id := value.(object)["id"].(string)
		if workUnits[id] {
			t.Fatalf("duplicate work unit: %s", id)
		}
		workUnits[id] = true
	}
	for _, value := range scope["authorized_work_unit_ids"].(array) {
		id := value.(string)
		if authorized[id] {
			t.Fatalf("duplicate authorized work unit: %s", id)
		}
		authorized[id] = true
	}
	if len(workUnits) != 14 || !reflect.DeepEqual(workUnits, authorized) {
		t.Fatalf("authorized work units differ from the complete plan: units=%v authorized=%v", workUnits, authorized)
	}
	for index := 1; index <= 14; index++ {
		if !authorized[fmt.Sprintf("WU-%02d", index)] {
			t.Errorf("approved work unit missing: WU-%02d", index)
		}
	}
}

func TestProjectProfileTargetsFollowApprovedPlanPaths(t *testing.T) {
	root := repositoryRoot(t)
	plan := readObject(t, filepath.Join(root, "src", "planning", "0.3.8", "ptsip-schema-centralization.plan.json"))
	director := plan["current_user_decisions"].(object)["Director_Decision"].(object)
	decisions, found := director["decisions_0001"].(object)
	if !found {
		t.Fatal("Profile path decision is missing from Director_Decision.decisions_0001")
	}
	denied := map[string]bool{}
	for _, candidate := range decisions["unapproved_paths"].(object)["denied_candidates"].(array) {
		denied[candidate.(string)] = true
	}
	migration := plan["profile_history_migration"].(object)
	approved := map[string]bool{migration["canonical_current_resource"].(string): true}
	for _, source := range migration["historical_schema_files"].(array) {
		approved[source.(object)["target"].(string)] = true
	}
	if len(approved) != 4 {
		t.Fatalf("expected exactly the approved selector and three historical targets: %v", approved)
	}
	created := []string{"src/ptsip/schemas/storage/profile/project-profile.schema.json"}
	for _, version := range profileVersions {
		created = append(created, "src/ptsip/schemas/"+version.canonical)
	}
	for _, path := range created {
		if !approved[path] || denied[path] {
			t.Errorf("new profile schema target lacks approval or is explicitly denied: %s", path)
		}
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil || !info.Mode().IsRegular() {
			t.Errorf("approved canonical profile schema is missing: %s: %v", path, err)
		}
	}
}
