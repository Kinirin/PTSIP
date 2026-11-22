package contracts_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const officialProfileID = "urn:ptsip:storage:contract:profile:official"

func officialPP300Schema(t *testing.T) (object, *jsonschema.Schema) {
	t.Helper()
	document := readObject(t, filepath.Join(productRoot(t), "schemas", "storage", "profile", "project-profile.schema.json"))
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(offlineProfileLoader{})
	if err := compiler.AddResource(officialProfileID, document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(officialProfileID)
	if err != nil {
		t.Fatalf("official pp.3.00 does not compile independently offline: %v", err)
	}
	return document, schema
}

// This independent pp.3.00 instance is not a renamed historical YAML fixture.
func officialPP300Instance() object {
	return object{
		"ptsip": object{
			"version": "pp.3.00", "revision": "Rev.0001", "profile_role": "PROJECT",
			"specification": object{"source": "https://github.com/Kinirin/PTSIP", "revision": strings.Repeat("1", 40)},
		},
		"responsibility_map": object{"mode": "DEFAULT"},
		"components": array{
			object{
				"id": "application", "classification": "PRODUCT", "include": array{"src/application/**"}, "purpose": "Application implementation",
				"dependency": object{"default": "deny", "allow": array{object{"from": "application", "to": "api"}}, "deny": array{}},
			},
			object{
				"id": "api", "classification": "NEUTRAL_CONTRACT", "include": array{"contracts/**"}, "purpose": "Shared interface declarations", "executable": false,
				"dependency": object{"default": "deny"},
			},
		},
		"associated_artifacts": array{object{"id": "application-docs", "anchor": "application", "include": array{"docs/**"}, "purpose": "Application documentation"}},
		"relationships":        array{object{"id": "application-api", "from": "application", "to": "api", "type": "IMPORTS"}},
		"policies":             object{"product_to_nonproduct_runtime_dependency": "deny", "nonproduct_in_product_package": "deny", "independent_build_resolution": "required"},
	}
}

func firstPP300Component(instance object) object {
	return instance["components"].(array)[0].(object)
}

func TestOfficialPP300ContractAndDependencySource(t *testing.T) {
	document, schema := officialPP300Schema(t)
	if document["$id"] != officialProfileID || document["title"] != "PTSIP official Current Project Profile" || document["$schema"] != dialect {
		t.Fatalf("unexpected official pp.3.00 identity: %v", document)
	}
	var inspect func(any)
	inspect = func(value any) {
		switch node := value.(type) {
		case object:
			if _, exists := node["$comment"]; exists {
				t.Fatal("new official contract contains a lifecycle $comment")
			}
			if ref, exists := node["$ref"]; exists && !strings.HasPrefix(ref.(string), "#/$defs/") {
				t.Fatalf("pp.3.00 requires an external schema resource: %v", ref)
			}
			for _, child := range node {
				inspect(child)
			}
		case array:
			for _, child := range node {
				inspect(child)
			}
		}
	}
	inspect(document)
	definitions := document["$defs"].(object)
	component := definitions["component"].(object)
	if !reflect.DeepEqual(component["required"], array{"id", "classification", "include", "purpose", "dependency"}) {
		t.Fatalf("component required fields differ from the approved five: %v", component["required"])
	}
	if component["properties"].(object)["dependency"].(object)["$ref"] != "#/$defs/componentDependencyPolicy" {
		t.Fatal("component dependency does not use the actual field contract")
	}
	history := readObject(t, filepath.Join(productRoot(t), "schemas", "history", "profile", "project-profile-pp.2.00.schema.json"))
	for _, name := range []string{"componentDependencyPolicy", "componentDependencyRelation"} {
		if !reflect.DeepEqual(definitions[name], history["$defs"].(object)[name]) {
			t.Fatalf("%s changed instead of reusing the declared dependency field contract", name)
		}
	}
	for name, expected := range map[string]string{
		"pp.1.01": "e8379e2d16f2a3c30b7c192c663fa45dc4ee7fec",
		"pp.1.02": "07fbfe8a5aa9dacc56f7b76e9ceaa3474c5599f2",
		"pp.2.00": "14d47e469e5f12ae65d721f64b17bdc35de861d8",
	} {
		path := "src/ptsip/schemas/history/profile/project-profile-" + name + ".schema.json"
		if actual := gitWorkingBlobSHA(t, path); actual != expected {
			t.Fatalf("historical contract %s changed: got %s, want %s", name, actual, expected)
		}
	}
	if err := schema.Validate(officialPP300Instance()); err != nil {
		t.Fatalf("independent valid pp.3.00 declaration rejected: %v", err)
	}
}

func TestOfficialPP300RequiredFieldsAndClosedStructure(t *testing.T) {
	_, schema := officialPP300Schema(t)
	cases := []struct {
		name   string
		mutate func(object)
	}{
		{"missing-components", func(v object) { delete(v, "components") }},
		{"empty-components", func(v object) { v["components"] = array{} }},
		{"old-parallel-dependency", func(v object) { v["component_dependency_policy"] = object{"default": "deny"} }},
		{"unknown-root-field", func(v object) { v["unapproved"] = true }},
		{"unknown-component-field", func(v object) { firstPP300Component(v)["unapproved"] = true }},
		{"wrong-version-pp2", func(v object) { v["ptsip"].(object)["version"] = "pp.2.00" }},
		{"wrong-version-shape", func(v object) { v["ptsip"].(object)["version"] = "pp.3.0" }},
		{"explicit-mode", func(v object) { v["responsibility_map"].(object)["mode"] = "explicit" }},
		{"hybrid-mode", func(v object) { v["responsibility_map"].(object)["mode"] = "hybrid" }},
		{"template-mode", func(v object) { v["responsibility_map"].(object)["mode"] = "template" }},
		{"default-mode-case", func(v object) { v["responsibility_map"].(object)["mode"] = "default" }},
		{"unapproved-template", func(v object) {
			v["responsibility_map"].(object)["template"] = object{"id": "example", "revision": "1"}
		}},
		{"revision-zero", func(v object) { v["ptsip"].(object)["revision"] = "Rev.0000" }},
		{"revision-malformed", func(v object) { v["ptsip"].(object)["revision"] = "Revx0001" }},
		{"revision-limit", func(v object) { v["ptsip"].(object)["revision"] = "Rev.10000" }},
	}
	for _, name := range []string{"id", "classification", "include", "purpose", "dependency"} {
		name := name
		cases = append(cases, struct {
			name   string
			mutate func(object)
		}{"missing-component-" + name, func(v object) { delete(firstPP300Component(v), name) }})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			instance := officialPP300Instance()
			test.mutate(instance)
			if err := schema.Validate(instance); err == nil {
				t.Fatalf("unapproved pp.3.00 declaration accepted: %s", test.name)
			}
		})
	}
}

func TestOfficialPP300DependencyPayload(t *testing.T) {
	_, schema := officialPP300Schema(t)
	for _, dependency := range []object{
		{"default": "allow"},
		{"default": "deny"},
		{"default": "deny", "allow": array{object{"from": "application", "to": "api"}}},
		{"default": "allow", "deny": array{object{"from": "api", "to": "application"}}},
	} {
		instance := officialPP300Instance()
		firstPP300Component(instance)["dependency"] = dependency
		if err := schema.Validate(instance); err != nil {
			t.Fatalf("actual dependency field contract rejected %v: %v", dependency, err)
		}
	}
	invalid := []struct {
		name  string
		value any
	}{
		{"null", nil}, {"array", array{}}, {"string", "deny"}, {"missing-default", object{}},
		{"unknown-default", object{"default": "inherit"}},
		{"invented-targets", object{"default": "deny", "targets": array{"api"}}},
		{"invented-types", object{"default": "deny", "types": array{"IMPORTS"}}},
		{"allow-object", object{"default": "deny", "allow": object{"from": "application", "to": "api"}}},
		{"relation-missing-from", object{"default": "deny", "allow": array{object{"to": "api"}}}},
		{"relation-missing-to", object{"default": "deny", "allow": array{object{"from": "application"}}}},
		{"relation-invented-type", object{"default": "deny", "deny": array{object{"from": "application", "to": "api", "type": "IMPORTS"}}}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			instance := officialPP300Instance()
			firstPP300Component(instance)["dependency"] = test.value
			if err := schema.Validate(instance); err == nil {
				t.Fatalf("dependency payload outside the field contract accepted: %v", test.value)
			}
		})
	}
	for _, revision := range []string{"Rev.0001", "Rev.0002", "Rev.0100", "Rev.9999"} {
		instance := officialPP300Instance()
		instance["ptsip"].(object)["revision"] = revision
		if err := schema.Validate(instance); err != nil {
			t.Fatalf("planned revision format %s rejected: %v", revision, err)
		}
	}
}
