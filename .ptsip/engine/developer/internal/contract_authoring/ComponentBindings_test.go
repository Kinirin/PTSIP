package contract_authoring_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	schemas "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
)

func componentBindingFixture(t *testing.T) (*authoring.Catalog, authoring.Validator, authoring.Document) {
	t.Helper()
	source := repository(t)
	root := t.TempDir()
	for _, relative := range []string{
		authoring.ProfilePath, authoring.DefinitionsPath, authoring.MainProfilePath, authoring.LocalProfileCatalogPath,
		"src/ptsip/specdata/ptsip-profile-pp-1.02.schema.json",
		authoring.RBACSchemaRoot + "Conditions_exercise/Condition.schema.json",
		authoring.RBACSchemaRoot + "authority/Operational_Authority.schema.json",
		authoring.RBACSchemaRoot + "responsibility/Responsibility.schema.json",
		authoring.RBACSchemaRoot + "Responsibilities_List.schema.json",
		authoring.RBACSchemaRoot + "Responsibility_Authority_Contract.schema.json",
	} {
		raw, err := os.ReadFile(filepath.Join(source, relative))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := authoring.LoadResponsibilities(root)
	if err != nil {
		t.Fatal(err)
	}
	validate, err := schemas.InstalledValidator()
	if err != nil {
		t.Skipf("installed PTSIP unavailable: %v", err)
	}
	profile, err := authoring.ReadDocument(root, authoring.ProfilePath)
	if err != nil {
		t.Fatal(err)
	}
	return catalog, validate, profile
}

func TestSeparatedComponentMetadataMaterializesExactDeclaredValues(t *testing.T) {
	catalog, validate, profile := componentBindingFixture(t)
	projection, err := catalog.MaterializeMain(validate, profile)
	if err != nil {
		t.Fatal(err)
	}
	source, err := authoring.ReadDocument(catalog.Root, authoring.MainProfilePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection["components"].([]any)) != len(profile["component_bindings"].([]any)) {
		t.Fatal("component assignment lost")
	}
	fields := []string{"roles", "release_owner", "compatibility_owner", "runtime_required"}
	bindings := map[string]map[string]any{}
	for _, raw := range profile["component_bindings"].([]any) {
		row := raw.(map[string]any)
		bindings[row["component_id"].(string)] = row
	}
	for _, raw := range source["components"].([]any) {
		row := raw.(map[string]any)
		for _, field := range fields {
			if _, present := row[field]; present {
				t.Fatalf("source mixes %s", field)
			}
		}
	}
	for _, raw := range projection["components"].([]any) {
		row := raw.(map[string]any)
		binding := bindings[row["id"].(string)]
		for _, field := range fields {
			if !reflect.DeepEqual(row[field], binding[field]) {
				t.Fatalf("declared %s changed", field)
			}
			delete(row, field)
		}
	}
	if !reflect.DeepEqual(projection, source) {
		t.Fatal("architecture, selectors or relationships changed during join")
	}
	if len(profile["responsibility"].([]any)) != 0 || len(catalog.Definitions["condition_list"].([]any)) != 0 || len(catalog.Definitions["authority_list"].([]any)) != 0 {
		t.Fatal("component owner labels synthesized execution declarations")
	}
}

func TestComponentBindingFailuresDoNotProduceAProjection(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*testing.T, *authoring.Catalog, authoring.Document)
	}{
		{"missing", func(t *testing.T, c *authoring.Catalog, p authoring.Document) {
			p["component_bindings"] = p["component_bindings"].([]any)[1:]
		}},
		{"duplicate", func(t *testing.T, c *authoring.Catalog, p authoring.Document) {
			rows := p["component_bindings"].([]any)
			p["component_bindings"] = append(rows, rows[0])
		}},
		{"unknown", func(t *testing.T, c *authoring.Catalog, p authoring.Document) {
			p["component_bindings"].([]any)[0].(map[string]any)["component_id"] = "missing-component"
		}},
		{"grant", func(t *testing.T, c *authoring.Catalog, p authoring.Document) {
			p["component_map"].(map[string]any)["execution_authorized"] = true
		}},
		{"missing_main", func(t *testing.T, c *authoring.Catalog, p authoring.Document) {
			if err := os.Remove(filepath.Join(c.Root, authoring.MainProfilePath)); err != nil {
				t.Fatal(err)
			}
		}},
		{"mixed_source", func(t *testing.T, c *authoring.Catalog, p authoring.Document) {
			main, err := authoring.ReadDocument(c.Root, authoring.MainProfilePath)
			if err != nil {
				t.Fatal(err)
			}
			main["components"].([]any)[0].(map[string]any)["release_owner"] = "conflicting-owner"
			raw, err := json.Marshal(main)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(c.Root, authoring.MainProfilePath), raw, 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{"catalog_route", func(t *testing.T, c *authoring.Catalog, p authoring.Document) {
			file := filepath.Join(c.Root, authoring.LocalProfileCatalogPath)
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var data map[string]any
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatal(err)
			}
			data["profiles"].([]any)[0].(map[string]any)["responsibility_authority_ref"] = "unregistered"
			raw, err = json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, raw, 0644); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			catalog, validate, profile := componentBindingFixture(t)
			item.mutate(t, catalog, profile)
			if projection, err := catalog.MaterializeMain(validate, profile); err == nil || projection != nil {
				t.Fatalf("invalid binding materialized: %v", err)
			}
		})
	}
}

func TestExplicitMaterializationReportsProjectionWithoutWritingSources(t *testing.T) {
	catalog, validate, _ := componentBindingFixture(t)
	before, err := os.ReadFile(filepath.Join(catalog.Root, authoring.MainProfilePath))
	if err != nil {
		t.Fatal(err)
	}
	report, code := authoring.Run(catalog.Root, "responsibility-authority", "materialize", "", "", "", validate)
	result := report.(map[string]any)
	if code != 0 || result["status"] != "MATERIALIZED" || result["projection_authority"] != "NONE" || result["execution_authorized"] != false {
		t.Fatalf("materialization report: %v", report)
	}
	after, err := os.ReadFile(filepath.Join(catalog.Root, authoring.MainProfilePath))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("materialization modified authored source")
	}
}

func TestDefaultLinkedProfileCannotDropAllComponentMappings(t *testing.T) {
	catalog, validate, profile := componentBindingFixture(t)
	delete(profile, "component_map")
	delete(profile, "component_bindings")
	raw, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(catalog.Root, authoring.ProfilePath), raw, 0644); err != nil {
		t.Fatal(err)
	}
	report, code := authoring.Run(catalog.Root, "responsibility-authority", "check", "", "", "", validate)
	if code == 0 || report.(map[string]any)["status"] != "FAIL" {
		t.Fatalf("entire linked mapping disappeared silently: %v", report)
	}
	if err := catalog.CheckProfile(validate, profile); err != nil {
		t.Fatalf("standalone unlinked candidate was incorrectly disabled: %v", err)
	}
}
