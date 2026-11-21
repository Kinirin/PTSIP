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

	coverage := func(id, selector string) object {
		return object{"declaration_id": id, "kind": "COMPONENT", "source_classification": "PRODUCT", "include": array{selector}, "exclude": array{},
			"purpose": "verification fixture", "source_pointer": "/components/0", "source_family": "TOOL_0.3.6_PROFILE_0.3.6_DRAFT", "origin": "PROJECT_EXPLICIT"}
	}
	paths := array{"src/app.py", "notes.txt"}
	matched := repositoryOperation(t, "projection", "projection.migration.coverage", object{"paths": paths, "coverages": array{coverage("broad", "src/**"), coverage("application", "src/app.py")}})
	existing := matched["existing"].(array)
	if len(existing) != 1 || existing[0].(object)["coverage"].(object)["declaration_id"] != "application" || len(matched["uncovered"].(array)) != 1 {
		t.Fatalf("exact selector lost to wildcard or inventory changed: %v", matched)
	}
	target := object{"draft_version": "pp.3.00", "specification_revision": "fixture", "relationships": array{}, "associated_artifacts": array{},
		"components": array{object{"id": "application", "classification": "PRODUCT", "include": array{"src/app.py"}, "exclude": array{}}}}
	compatibility := repositoryOperation(t, "projection", "projection.migration.compatibility", object{"existing": existing, "target": target, "classifications": array{"PRODUCT"}})
	if compatibility["results"].(array)[0].(object)["target_status"] != "ALREADY_SATISFIED" {
		t.Fatalf("exact target ownership was not preserved: %v", compatibility)
	}
	conflict := repositoryOperation(t, "projection", "projection.migration.coverage", object{"paths": paths, "coverages": array{coverage("a", "src/**"), coverage("b", "src/**")}})
	if len(conflict["ambiguous"].(array)) != 1 || len(conflict["existing"].(array)) != 0 {
		t.Fatalf("equal-specificity conflict was silently resolved: %v", conflict)
	}
}
