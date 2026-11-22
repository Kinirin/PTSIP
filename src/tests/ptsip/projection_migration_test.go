package contracts_test

import "testing"

func TestProjectionMigrationTemplateAndCoverage(t *testing.T) {
	source := object{"family": "TOOL_0.3.6_PROFILE_0.3.6_DRAFT",
		"family_semantics": object{"kind": "V036", "responsibility_map_mode": "template",
			"template_id": "python-package-library", "template_revision": "sha256:409acd1cd9907a60761a3cf26a051185d40b5e926e6952131b641b10bccc5c9b",
			"remove_component_ids": array{}, "remove_associated_artifact_ids": array{}, "remove_relationship_ids": array{}},
		"components": array{}, "associated_artifacts": array{}, "relationships": array{}}
	projected := repositoryOperation(t, "projection", "projection.migration.source", source)
	coverages := projected["coverages"].(array)
	if len(projected["issues"].(array)) != 0 || len(coverages) < 2 {
		t.Fatalf("historical template projection rejected or incomplete: %v", projected)
	}
	for _, row := range coverages {
		if row.(object)["origin"] != "TEMPLATE_EFFECTIVE" {
			t.Fatalf("template output lost historical provenance: %v", row)
		}
	}
	source["family_semantics"].(object)["template_revision"] = "sha256:" + "0000000000000000000000000000000000000000000000000000000000000000"
	if result := repositoryOperation(t, "projection", "projection.migration.source", source); len(result["issues"].(array)) != 1 || len(result["coverages"].(array)) != 0 {
		t.Fatalf("unresolved template revision returned coverage: %v", result)
	}

	component := func(id, selector, pointer string) object {
		return object{"id": id, "source_classification": "PRODUCT", "include": array{selector}, "exclude": array{},
			"purpose": "verification fixture", "source_pointer": pointer}
	}
	declared := object{"family": "TOOL_0.3.5_PROFILE_0.3.4_DRAFT",
		"family_semantics":     object{"kind": "V034", "declaration_form": "COMPONENTS", "boundaries": array{}},
		"components":           array{component("broad", "src/**", "/components/0"), component("application", "src/app.py", "/components/1")},
		"associated_artifacts": array{}, "relationships": array{}}
	explicit := repositoryOperation(t, "projection", "projection.migration.source", declared)
	if len(explicit["issues"].(array)) != 0 || len(explicit["coverages"].(array)) != 2 {
		t.Fatalf("explicit component source projection did not preserve declarations: %v", explicit)
	}
	paths := array{"src/app.py", "notes.txt"}
	matched := repositoryOperation(t, "projection", "projection.migration.coverage", object{"paths": paths, "coverages": explicit["coverages"]})
	existing := matched["existing"].(array)
	if len(existing) != 1 || len(matched["uncovered"].(array)) != 1 {
		t.Fatalf("exact selector lost to wildcard or inventory changed: %v", matched)
	}
	match := existing[0].(object)
	coverage := match["coverage"].(object)
	if match["path"] != "src/app.py" || coverage["declaration_id"] != "application" || coverage["origin"] != "SOURCE_COMPONENT" || coverage["source_pointer"] != "/components/1" {
		t.Fatalf("explicit source identity was lost across native projection operations: %v", match)
	}
	target := object{"draft_version": "pp.3.00", "specification_revision": "fixture", "relationships": array{}, "associated_artifacts": array{},
		"components": array{object{"id": "application", "classification": "PRODUCT", "include": array{"src/app.py"}, "exclude": array{}}}}
	compatibility := repositoryOperation(t, "projection", "projection.migration.compatibility", object{"existing": existing, "target": target, "classifications": array{"PRODUCT"}})
	if compatibility["results"].(array)[0].(object)["target_status"] != "ALREADY_SATISFIED" {
		t.Fatalf("exact target ownership was not preserved: %v", compatibility)
	}
	declared["components"] = array{component("a", "src/**", "/components/0"), component("b", "src/**", "/components/1")}
	competing := repositoryOperation(t, "projection", "projection.migration.source", declared)
	conflict := repositoryOperation(t, "projection", "projection.migration.coverage", object{"paths": paths, "coverages": competing["coverages"]})
	if len(conflict["ambiguous"].(array)) != 1 || len(conflict["existing"].(array)) != 0 {
		t.Fatalf("equal-specificity conflict was silently resolved: %v", conflict)
	}
}
