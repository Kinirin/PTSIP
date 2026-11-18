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
	version   string
	source    string
	canonical string
	lifecycle string
}

var profileVersions = []profileVersion{
	{"pp.1.01", "specdata/ptsip-profile-pp-1.01.schema.json", "storage/profile/history/project-profile-pp.1.01.schema.json", "DEPRECATED"},
	{"pp.1.02", "specdata/ptsip-profile-pp-1.02.schema.json", "storage/profile/history/project-profile-pp.1.02.schema.json", "DEPRECATED"},
	{"pp.2.00", "profiles/profiles_schemas/ptsip-profile-pp-2.00.schema.json", "storage/profile/history/project-profile-pp.2.00.schema.json", "ACTIVE"},
}

type offlineProfileLoader struct{}

func (offlineProfileLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("unregistered profile schema resource %s", url)
}

func profileSchemaCompiler(t *testing.T, canonical bool) (*jsonschema.Compiler, map[string]*jsonschema.Schema) {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(offlineProfileLoader{})
	identities := map[string]string{}
	for _, version := range profileVersions {
		path := filepath.Join(productRoot(t), filepath.FromSlash(version.source))
		if canonical {
			path = filepath.Join(productRoot(t), "schemas", filepath.FromSlash(version.canonical))
		}
		schema := readObject(t, path)
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
			t.Fatalf("compile %s: %v", version.version, err)
		}
		compiled[version.version] = schema
	}
	return compiler, compiled
}

// Keep data-valued keywords intact while removing only schema annotations.
func profileValidationKeywords(value any) any {
	node, ok := value.(object)
	if !ok {
		return value
	}
	projected := object{}
	for keyword, child := range node {
		switch keyword {
		case "$comment", "title", "description", "default", "examples", "deprecated", "readOnly", "writeOnly":
			continue
		case "$defs", "definitions", "properties", "patternProperties", "dependentSchemas":
			children := object{}
			for name, schema := range child.(object) {
				children[name] = profileValidationKeywords(schema)
			}
			child = children
		case "allOf", "anyOf", "oneOf", "prefixItems":
			children := make(array, len(child.(array)))
			for index, schema := range child.(array) {
				children[index] = profileValidationKeywords(schema)
			}
			child = children
		case "items":
			if schemas, ok := child.(array); ok {
				children := make(array, len(schemas))
				for index, schema := range schemas {
					children[index] = profileValidationKeywords(schema)
				}
				child = children
			} else {
				child = profileValidationKeywords(child)
			}
		case "additionalProperties", "additionalItems", "contains", "propertyNames", "not", "if", "then", "else", "unevaluatedProperties", "unevaluatedItems", "contentSchema":
			child = profileValidationKeywords(child)
		case "dependencies":
			children := object{}
			for name, dependency := range child.(object) {
				children[name] = profileValidationKeywords(dependency)
			}
			child = children
		}
		projected[keyword] = child
	}
	return projected
}

func TestProjectProfileHistoricalSchemasPreserveSourceKeywords(t *testing.T) {
	for _, version := range profileVersions {
		t.Run(version.version, func(t *testing.T) {
			source := readObject(t, filepath.Join(productRoot(t), filepath.FromSlash(version.source)))
			canonical := readObject(t, filepath.Join(productRoot(t), "schemas", filepath.FromSlash(version.canonical)))
			if source["$id"] != canonical["$id"] || source["$schema"] != canonical["$schema"] {
				t.Fatalf("schema identity or dialect changed during centralization: %v, %v", source["$id"], canonical["$id"])
			}
			if !reflect.DeepEqual(profileValidationKeywords(source), profileValidationKeywords(canonical)) {
				t.Fatal("canonical profile differs from selected source validation keywords")
			}
		})
	}
	profileSchemaCompiler(t, false)
	profileSchemaCompiler(t, true)
}

func readProfileYAML(t *testing.T, path string) object {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("parse original profile YAML %s: %v", path, err)
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("original YAML is not JSON-compatible data: %s: %v", path, err)
	}
	var document object
	if err := json.Unmarshal(normalized, &document); err != nil {
		t.Fatalf("original profile must be a mapping: %s: %v", path, err)
	}
	if _, isSchema := document["$schema"]; isSchema {
		t.Fatalf("YAML instance was treated as a JSON Schema: %s", path)
	}
	return document
}

func TestProjectProfileAllOriginalYAMLVariants(t *testing.T) {
	_, sourceSchemas := profileSchemaCompiler(t, false)
	compiler, canonicalSchemas := profileSchemaCompiler(t, true)
	selector := readObject(t, filepath.Join(productRoot(t), "schemas", "storage", "profile", "project-profile.schema.json"))
	if err := compiler.AddResource(selector["$id"].(string), selector); err != nil {
		t.Fatal(err)
	}
	current, err := compiler.Compile(selector["$id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	variants := []struct{ name, mode string }{
		{"example.ptsip.yaml", "explicit"}, {"hybrid-python-package.ptsip.yaml", "hybrid"}, {"template-python-package.ptsip.yaml", "template"},
	}
	for _, version := range profileVersions {
		roots := []string{filepath.Join(productRoot(t), "profiles", "history", version.version)}
		if version.version == "pp.2.00" {
			roots = append(roots, filepath.Join(productRoot(t), "profiles"))
		}
		for _, root := range roots {
			for _, variant := range variants {
				path := filepath.Join(root, variant.name)
				name, _ := filepath.Rel(productRoot(t), path)
				t.Run(filepath.ToSlash(name), func(t *testing.T) {
					document := readProfileYAML(t, path)
					if document["ptsip"].(object)["version"] != version.version || document["responsibility_map"].(object)["mode"] != variant.mode {
						t.Fatalf("fixture identity or variant changed: %s", path)
					}
					if err := sourceSchemas[version.version].Validate(document); err != nil {
						t.Fatalf("selected source rejected original %s: %v", version.version, err)
					}
					if err := canonicalSchemas[version.version].Validate(document); err != nil {
						t.Fatalf("canonical schema rejected original %s: %v", version.version, err)
					}
					for _, other := range profileVersions {
						if other.version != version.version && canonicalSchemas[other.version].Validate(document) == nil {
							t.Errorf("wrong contract %s accepted %s fixture", other.version, version.version)
						}
					}
					if got := current.Validate(document) == nil; got != (version.version == "pp.2.00") {
						t.Errorf("current selector accepted=%v for %s", got, version.version)
					}
					missingMap := clone(t, document)
					delete(missingMap, "responsibility_map")
					if canonicalSchemas[version.version].Validate(missingMap) == nil {
						t.Error("canonical schema dropped a required profile responsibility map")
					}
				})
			}
		}
	}
}

func gitBlobSHA(t *testing.T, relative string) string {
	t.Helper()
	command := exec.Command("git", "rev-parse", "HEAD:"+filepath.ToSlash(relative))
	command.Dir = repositoryRoot(t)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve original Git blob %s: %v: %s", relative, err, output)
	}
	return strings.TrimSpace(string(output))
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
		if expected := gitBlobSHA(t, "src/ptsip/"+version.source); entry["source_blob_sha"] != expected {
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
	pp2 := readObject(t, filepath.Join(product, "schemas", filepath.FromSlash(profileVersions[2].canonical)))
	if selector["$ref"] != pp2["$id"] || selector["$id"] != "urn:ptsip:contract:project-profile:current" {
		t.Fatalf("current selector must reference only exact pp.2.00: %v", selector)
	}
	if entry["$id"] != selector["$id"] || entry["version"] != "pp.2.00" || entry["path"] != "storage/profile/project-profile.schema.json" {
		t.Fatalf("current registration differs from selector: %v", entry)
	}
	if entry["registered_lifecycle"] != "ACTIVE" {
		t.Errorf("current selector must have ACTIVE lifecycle: %v", entry)
	}
	if expected := gitBlobSHA(t, "src/ptsip/"+profileVersions[2].source); entry["source_blob_sha"] != expected {
		t.Errorf("current selector provenance does not use selected pp2 source: %v", entry)
	}
	if expected := gitWorkingBlobSHA(t, "src/ptsip/schemas/storage/profile/project-profile.schema.json"); entry["schema_blob_sha"] != expected {
		t.Errorf("current selector schema blob differs from canonical file: %v", entry)
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
	for _, key := range []string{"ptsip.project-profile", "ptsip.project-profile.pp.2.00"} {
		for _, newRegistration := range []bool{false, true} {
			selection, err := catalog.SelectRegisteredSchema(key, newRegistration)
			if err != nil || !selection.Allowed || selection.RegisteredLifecycle != usercontracts.Active {
				t.Fatalf("ACTIVE selection must allow existing and new use: %+v, %v", selection, err)
			}
			if selection.SchemaKey != key || selection.Version != "pp.2.00" || selection.SchemaPath == "" || len(selection.Notifications) != 0 {
				t.Fatalf("ACTIVE selection lost its registration identity or emitted a notice: %+v", selection)
			}
		}
	}
	for _, version := range profileVersions[:2] {
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
	if assignment["pp.2.00"] != "ACTIVE" || assignment["other_registered_project_profile_versions"] != "DEPRECATED" {
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
		if entry["version"] == "pp.2.00" {
			expected = assignment["pp.2.00"]
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
	decisions := plan["current_user_decisions_2026_11_18"].(object)
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
