package pp

import "testing"

func TestPublicProfileCatalogDoesNotOwnCurrentContractAuthority(t *testing.T) {
	binary := ppBuildAutomation(t)
	catalog, err, output := ppAutomation(t, binary, "profile-registry", "catalog")
	if err != nil {
		t.Fatalf("profile-registry catalog: %v\n%s", err, output)
	}
	if _, exists := catalog["current"]; exists {
		t.Fatal("public catalog duplicates current contract authority")
	}
	if _, exists := catalog["current_contract"]; exists {
		t.Fatal("public catalog duplicates current_contract authority")
	}
}

func TestPublicProfileCatalogExactlyDescribesDistributionAssets(t *testing.T) {
	binary := ppBuildAutomation(t)
	catalog, err, output := ppAutomation(t, binary, "profile-registry", "catalog")
	if err != nil {
		t.Fatalf("profile-registry catalog: %v\n%s", err, output)
	}

	expected := []map[string]any{
		{
			"id": "example",
			"resource": "example.ptsip.yaml",
			"contract": "pp.1.02",
			"responsibility_mode": "explicit",
			"profile_role": "DISTRIBUTED_EXAMPLE",
			"materialization": "PROJECT_PATH_RESOLUTION_REQUIRED",
		},
		{
			"id": "hybrid-python-package",
			"resource": "hybrid-python-package.ptsip.yaml",
			"contract": "pp.1.02",
			"responsibility_mode": "hybrid",
			"profile_role": "DISTRIBUTED_EXAMPLE",
			"materialization": "PROJECT_PATH_RESOLUTION_REQUIRED",
		},
		{
			"id": "template-python-package",
			"resource": "template-python-package.ptsip.yaml",
			"contract": "pp.1.02",
			"responsibility_mode": "template",
			"profile_role": "DISTRIBUTED_EXAMPLE",
			"materialization": "PROJECT_PATH_RESOLUTION_REQUIRED",
		},
	}

	profiles := catalog["profiles"].([]any)
	if len(profiles) != len(expected) {
		t.Fatalf("catalog profile count = %d", len(profiles))
	}
	for i, want := range expected {
		got := profiles[i].(map[string]any)
		for key, value := range want {
			if got[key] != value {
				t.Fatalf("profile[%d].%s = %#v, want %#v", i, key, got[key], value)
			}
		}
	}
}
