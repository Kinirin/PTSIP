package pp

import "testing"

func TestProfileContractRegistryOwnsSingleCurrentIdentity(t *testing.T) {
	binary := ppBuildAutomation(t)

	registry, err, output := ppAutomation(t, binary, "profile-registry", "contracts")
	if err != nil {
		t.Fatalf("profile-registry contracts: %v\n%s", err, output)
	}
	if registry["current"] != "pp.1.02" {
		t.Fatalf("current contract = %#v", registry["current"])
	}

	current, err, output := ppAutomation(t, binary, "profile-registry", "current")
	if err != nil {
		t.Fatalf("profile-registry current: %v\n%s", err, output)
	}
	if current["version"] != "pp.1.02" ||
		current["lifecycle"] != "CURRENT" ||
		current["schema"] != "schemas/ptsip-profile-pp-1.02.schema.json" {
		t.Fatalf("unexpected current Project Profile contract: %#v", current)
	}
}

func TestProfileContractRegistryPreservesHistoricalLifecycle(t *testing.T) {
	binary := ppBuildAutomation(t)
	registry, err, output := ppAutomation(t, binary, "profile-registry", "contracts")
	if err != nil {
		t.Fatalf("profile-registry contracts: %v\n%s", err, output)
	}

	contracts := map[string]map[string]any{}
	for _, raw := range registry["contracts"].([]any) {
		record := raw.(map[string]any)
		contracts[record["version"].(string)] = record
	}

	legacy := contracts["pp.0.00"]
	if legacy["lifecycle"] != "LEGACY_COMPATIBILITY_ONLY" {
		t.Fatalf("pp.0.00 lifecycle = %#v", legacy["lifecycle"])
	}
	operations := legacy["operations"].([]any)
	if len(operations) != 1 || operations[0] != "IDENTIFY" {
		t.Fatalf("pp.0.00 operations = %#v", operations)
	}
	if legacy["schema"] != nil {
		t.Fatalf("pp.0.00 schema = %#v", legacy["schema"])
	}
	if contracts["pp.1.01"]["lifecycle"] != "SUPERSEDED" {
		t.Fatalf("pp.1.01 lifecycle = %#v", contracts["pp.1.01"]["lifecycle"])
	}

	transitions := registry["transitions"].([]any)
	if len(transitions) != 1 {
		t.Fatalf("transitions = %#v", transitions)
	}
	transition := transitions[0].(map[string]any)
	if transition["from"] != "pp.1.01" ||
		transition["to"] != "pp.1.02" ||
		transition["kind"] != "SEMANTIC_MIGRATION" {
		t.Fatalf("unexpected Project Profile transition: %#v", transition)
	}
}
